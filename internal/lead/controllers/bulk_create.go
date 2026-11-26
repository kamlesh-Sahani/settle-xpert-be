package controllers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	employeemodels "lead-crm/backend/internal/employee/models"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BulkLeadItem struct {
	FullName               string                  `json:"fullName"`
	Phone                  string                  `json:"phone"`
	Email                  string                  `json:"email"`
	DOB                    string                  `json:"dob"`
	PANNumber              string                  `json:"panNumber"`
	MaritalStatus          string                  `json:"maritalStatus"`
	FullAddress            string                  `json:"fullAddress"`
	City                   string                  `json:"city"`
	State                  string                  `json:"state"`
	EmploymentStatus       models.EmploymentStatus `json:"employmentStatus"`
	EmploymentType         string                  `json:"employmentType"`
	TotalOutstandingAmount float64                 `json:"totalOutstandingAmount"`
	CreditCardDue          float64                 `json:"creditCardDue"`
	PersonalDue            float64                 `json:"personalDue"`
	MonthlyIncome          float64                 `json:"monthlyIncome"`
	MonthlyEMI             float64                 `json:"monthlyEmi"`
	Harassment             models.HarassmentStatus `json:"harassment"`
	ServiceType            string                  `json:"serviceType"`
	AssignedTo             string                  `json:"assignedTo"`
}

type BulkCreateLeadsRequest struct {
	DefaultAssignedTo string         `json:"defaultAssignedTo" binding:"omitempty,max=24"`
	Leads             []BulkLeadItem `json:"leads" binding:"required,min=1,max=2000"`
}

type SkippedLeadInfo struct {
	Row    int    `json:"row"`
	Phone  string `json:"phone"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

func BulkCreate(c *gin.Context) {
	var req BulkCreateLeadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	leadCol := database.GetCollection("leads")
	empCol := database.GetCollection("employees")
	if leadCol == nil || empCol == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}
	ctx := c.Request.Context()

	// 1. Collect requested assignees & enforce lead:assign if any assignment is provided
	assigneeIDs := make(map[string]primitive.ObjectID)
	hasAssignment := strings.TrimSpace(req.DefaultAssignedTo) != ""
	if !hasAssignment {
		for _, item := range req.Leads {
			if strings.TrimSpace(item.AssignedTo) != "" {
				hasAssignment = true
				break
			}
		}
	}
	if hasAssignment {
		if !hasLeadPermission(c, "lead:assign") {
			response.Forbidden(c, "You do not have permission to assign leads")
			return
		}

		var candidateOIDs []primitive.ObjectID
		collectID := func(raw string) {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				return
			}
			if oid, err := primitive.ObjectIDFromHex(raw); err == nil && !oid.IsZero() {
				if _, seen := assigneeIDs[raw]; !seen {
					assigneeIDs[raw] = oid
					candidateOIDs = append(candidateOIDs, oid)
				}
			}
		}
		collectID(req.DefaultAssignedTo)
		for _, item := range req.Leads {
			collectID(item.AssignedTo)
		}

		activeSet := make(map[primitive.ObjectID]bool)
		if len(candidateOIDs) > 0 {
			empCursor, err := empCol.Find(ctx, bson.M{
				"_id":        bson.M{"$in": candidateOIDs},
				"status":     employeemodels.EmployeeStatusActive,
				"deleted_at": bson.M{"$exists": false},
			}, options.Find().SetProjection(bson.M{"_id": 1}))
			if err != nil {
				log.Printf("[ERROR] [LeadBulkCreate] Employee verification failed: %v", err)
				response.InternalServerError(c, "Failed to verify assigned employees")
				return
			}
			defer empCursor.Close(ctx)
			for empCursor.Next(ctx) {
				var doc struct {
					ID primitive.ObjectID `bson:"_id"`
				}
				if err := empCursor.Decode(&doc); err == nil {
					activeSet[doc.ID] = true
				}
			}
		}
		for k, oid := range assigneeIDs {
			if !activeSet[oid] {
				delete(assigneeIDs, k)
			}
		}

		if defID := strings.TrimSpace(req.DefaultAssignedTo); defID != "" {
			if _, ok := assigneeIDs[defID]; !ok {
				response.BadRequest(c, "Default assigned employee is invalid or inactive")
				return
			}
		}
	}

	// 2. Gather valid normalized phone numbers for CRM deduplication
	cleanPhones := make([]string, 0, len(req.Leads))
	for _, item := range req.Leads {
		if phone, ok := normalizeLeadPhone(item.Phone); ok {
			cleanPhones = append(cleanPhones, phone)
		}
	}

	existingPhones := make(map[string]bool)
	if len(cleanPhones) > 0 {
		dupFilter := bson.M{
			"deleted_at":     bson.M{"$exists": false},
			"personal.phone": bson.M{"$in": cleanPhones},
		}
		cursor, err := leadCol.Find(ctx, dupFilter, options.Find().SetProjection(bson.M{"personal.phone": 1}))
		if err != nil {
			log.Printf("[ERROR] [LeadBulkCreate] Duplicate lookup failed: %v", err)
			response.InternalServerError(c, "Failed to validate existing leads")
			return
		}
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var doc struct {
				Personal struct {
					Phone string `bson:"phone"`
				} `bson:"personal"`
			}
			if err := cursor.Decode(&doc); err == nil && doc.Personal.Phone != "" {
				existingPhones[doc.Personal.Phone] = true
			}
		}
	}

	// 3. Validate rows and build lead structs
	now := time.Now().UTC()
	validLeads := make([]models.Lead, 0, len(req.Leads))
	skippedLeads := make([]SkippedLeadInfo, 0)
	seenInBatch := make(map[string]bool, len(req.Leads))

	for i, item := range req.Leads {
		rowNum := i + 1
		fullName, cleanedPhone, pan, validationErr := validateBulkLeadRow(item)
		if validationErr != "" {
			displayPhone := cleanedPhone
			if displayPhone == "" {
				displayPhone = strings.TrimSpace(item.Phone)
			}
			skippedLeads = append(skippedLeads, SkippedLeadInfo{
				Row:    rowNum,
				Phone:  displayPhone,
				Name:   strings.TrimSpace(item.FullName),
				Reason: validationErr,
			})
			continue
		}

		if seenInBatch[cleanedPhone] {
			skippedLeads = append(skippedLeads, SkippedLeadInfo{
				Row:    rowNum,
				Phone:  cleanedPhone,
				Name:   fullName,
				Reason: "Duplicate phone number within uploaded spreadsheet",
			})
			continue
		}
		seenInBatch[cleanedPhone] = true

		if existingPhones[cleanedPhone] {
			skippedLeads = append(skippedLeads, SkippedLeadInfo{
				Row:    rowNum,
				Phone:  cleanedPhone,
				Name:   fullName,
				Reason: "Lead with this phone already exists in your CRM",
			})
			continue
		}

		var assignedToObjID *primitive.ObjectID
		rowAssignedRaw := strings.TrimSpace(item.AssignedTo)
		effectiveAssignedRaw := rowAssignedRaw
		if effectiveAssignedRaw == "" {
			effectiveAssignedRaw = strings.TrimSpace(req.DefaultAssignedTo)
		}
		if effectiveAssignedRaw != "" {
			oid, ok := assigneeIDs[effectiveAssignedRaw]
			if !ok {
				skippedLeads = append(skippedLeads, SkippedLeadInfo{
					Row:    rowNum,
					Phone:  cleanedPhone,
					Name:   fullName,
					Reason: "Assigned employee ID is invalid or inactive",
				})
				continue
			}
			oidCopy := oid
			assignedToObjID = &oidCopy
		}

		var finInfo *models.FinancialInfo
		if item.TotalOutstandingAmount > 0 || item.CreditCardDue > 0 || item.PersonalDue > 0 ||
			item.MonthlyIncome > 0 || item.MonthlyEMI > 0 || item.Harassment != "" {
			finInfo = &models.FinancialInfo{
				TotalOutstandingAmount: item.TotalOutstandingAmount,
				CreditCardDue:          item.CreditCardDue,
				PersonalDue:            item.PersonalDue,
				MonthlyIncome:          item.MonthlyIncome,
				MonthlyEMI:             item.MonthlyEMI,
				Harassment:             item.Harassment,
			}
		}

		leadTime := now.Add(time.Duration(i) * time.Millisecond)
		validLeads = append(validLeads, models.Lead{
			ID:     primitive.NewObjectID(),
			Status: models.LeadStatusNew,
			Personal: models.PersonalInfo{
				FullName:         fullName,
				Phone:            cleanedPhone,
				Email:            strings.ToLower(strings.TrimSpace(item.Email)),
				DOB:              strings.TrimSpace(item.DOB),
				PANNumber:        pan,
				MaritalStatus:    strings.TrimSpace(item.MaritalStatus),
				FullAddress:      strings.TrimSpace(item.FullAddress),
				City:             strings.TrimSpace(item.City),
				State:            strings.TrimSpace(item.State),
				EmploymentStatus: item.EmploymentStatus,
				EmploymentType:   strings.TrimSpace(item.EmploymentType),
			},
			Financial:   finInfo,
			ServiceType: strings.TrimSpace(item.ServiceType),
			AssignedTo:  assignedToObjID,
			CreatedAt:       leadTime,
			UpdatedAt:       leadTime,
		})
	}

	if len(validLeads) == 0 {
		response.BadRequest(c, "No valid leads could be imported. Please verify that borrower names and 10-digit phone numbers are provided.")
		return
	}

	// 4. Reserve sequential lead numbers atomically for the batch
	startSeq, err := reserveSequenceRange(ctx, "lead_number", leadCol, int64(len(validLeads)))
	if err != nil {
		log.Printf("[ERROR] [LeadBulkCreate] Lead number reservation failed: %v", err)
		response.InternalServerError(c, "Failed to generate lead numbers")
		return
	}

	documents := make([]any, len(validLeads))
	for i := range validLeads {
		validLeads[i].LeadNumber = fmt.Sprintf("LEAD-%04d", startSeq+int64(i))
		documents[i] = validLeads[i]
	}

	if _, err := leadCol.InsertMany(ctx, documents); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			response.Conflict(c, "One or more leads duplicate an existing phone number or lead number")
			return
		}
		log.Printf("[ERROR] [LeadBulkCreate] InsertMany failed: %v", err)
		response.InternalServerError(c, "Failed to import leads into database")
		return
	}

	response.Success(c, fmt.Sprintf("Successfully imported %d leads", len(documents)), gin.H{
		"totalSubmitted": len(req.Leads),
		"createdCount":   len(documents),
		"skippedCount":   len(skippedLeads),
		"skippedLeads":   skippedLeads,
	})
}
