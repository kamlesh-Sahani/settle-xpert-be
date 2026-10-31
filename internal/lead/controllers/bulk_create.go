package controllers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var bulkDigitCleaner = regexp.MustCompile(`\D`)

type BulkLeadItem struct {
	FullName           string            `json:"fullName"`
	Phone              string            `json:"phone"`
	Email              string            `json:"email"`
	ServiceRequired    string            `json:"serviceRequired"`
	Source             models.LeadSource `json:"source"`
	City               string            `json:"city"`
	State              string            `json:"state"`
	TotalDebtAmount    float64           `json:"totalDebtAmount"`
	OutstandingAmount  float64           `json:"outstandingAmount"`
	MonthlyIncome      float64           `json:"monthlyIncome"`
	CurrentEMIAmount   float64           `json:"currentEmiAmount"`
	QualificationNotes string            `json:"qualificationNotes"`
	AssignedTo         string            `json:"assignedTo"`
}

type BulkCreateLeadsRequest struct {
	DefaultSource     models.LeadSource `json:"defaultSource"`
	DefaultAssignedTo string            `json:"defaultAssignedTo"`
	Leads             []BulkLeadItem    `json:"leads" binding:"required,min=1"`
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

	col := database.GetCollection("leads")
	empCol := database.GetCollection("employees")
	ctx := c.Request.Context()

	creatorName := "System"
	if empCtx, ok := utils.GetAuthEmployee(c); ok && empCtx != nil && empCtx.Name != "" {
		creatorName = empCtx.Name
	}

	// 1. Gather all normalized phone numbers for CRM deduplication check
	cleanPhones := make([]string, 0, len(req.Leads))
	for _, item := range req.Leads {
		raw := strings.TrimSuffix(strings.TrimSpace(item.Phone), ".0")
		p := bulkDigitCleaner.ReplaceAllString(raw, "")
		if len(p) > 10 {
			p = p[len(p)-10:]
		}
		if len(p) == 10 {
			cleanPhones = append(cleanPhones, p)
		}
	}

	// 2. Query existing phone numbers in CRM
	existingPhones := make(map[string]bool)
	if len(cleanPhones) > 0 {
		dupFilter := bson.M{
			"deleted_at": bson.M{"$exists": false},
			"phone":      bson.M{"$in": cleanPhones},
		}
		cursor, err := col.Find(ctx, dupFilter, options.Find().SetProjection(bson.M{"phone": 1}))
		if err == nil {
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var doc struct {
					Phone string `bson:"phone"`
				}
				if err := cursor.Decode(&doc); err == nil && doc.Phone != "" {
					existingPhones[doc.Phone] = true
				}
			}
		}
	}

	empCache := make(map[string]string)
	now := time.Now()

	documents := make([]interface{}, 0, len(req.Leads))
	skippedLeads := make([]SkippedLeadInfo, 0)
	seenInBatch := make(map[string]bool)

	for i, item := range req.Leads {
		rowNum := i + 1
		fullName := strings.TrimSpace(item.FullName)
		rawPhone := strings.TrimSuffix(strings.TrimSpace(item.Phone), ".0")
		cleanedPhone := bulkDigitCleaner.ReplaceAllString(rawPhone, "")
		if len(cleanedPhone) > 10 {
			cleanedPhone = cleanedPhone[len(cleanedPhone)-10:]
		}

		// Validation: Missing name or invalid phone
		if fullName == "" || len(cleanedPhone) < 10 {
			skippedLeads = append(skippedLeads, SkippedLeadInfo{
				Row:    rowNum,
				Phone:  rawPhone,
				Name:   fullName,
				Reason: "Invalid phone number (must be 10 digits) or missing borrower name",
			})
			continue
		}

		// Deduplication: Duplicate within the same file batch
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

		// Deduplication: Already exists in CRM
		if existingPhones[cleanedPhone] {
			skippedLeads = append(skippedLeads, SkippedLeadInfo{
				Row:    rowNum,
				Phone:  cleanedPhone,
				Name:   fullName,
				Reason: "Lead with this phone already exists in your CRM",
			})
			continue
		}

		source := models.SourceExcelImport
		if item.Source != "" {
			source = item.Source
		} else if req.DefaultSource != "" {
			source = req.DefaultSource
		}

		assignedToRaw := strings.TrimSpace(item.AssignedTo)
		if assignedToRaw == "" && req.DefaultAssignedTo != "" {
			assignedToRaw = strings.TrimSpace(req.DefaultAssignedTo)
		}

		status := models.LeadStatusNew
		if assignedToRaw != "" {
			status = models.LeadStatusInProgress
		}

		var assignedToObjID *primitive.ObjectID
		assignedToName := ""

		if assignedToRaw != "" {
			if objID, err := primitive.ObjectIDFromHex(assignedToRaw); err == nil {
				assignedToObjID = &objID
				if name, ok := empCache[assignedToRaw]; ok {
					assignedToName = name
				} else {
					var emp struct {
						Name string `bson:"name"`
					}
					if err := empCol.FindOne(ctx, bson.M{"_id": objID}).Decode(&emp); err == nil {
						assignedToName = emp.Name
						empCache[assignedToRaw] = emp.Name
					}
				}
			}
		}

		leadNumber := fmt.Sprintf("LD-%d-%02d", (now.UnixNano()/1000)%10000000, rowNum)
		totalDebt := item.TotalDebtAmount
		outstanding := item.OutstandingAmount
		if outstanding == 0 {
			outstanding = totalDebt
		}

		serviceReq := strings.TrimSpace(item.ServiceRequired)
		if serviceReq == "" {
			serviceReq = "LOAN_RESTRUCTURING"
		}

		leadTime := now.Add(time.Duration(i) * time.Millisecond)
		lead := models.Lead{
			ID:                 primitive.NewObjectID(),
			LeadNumber:         leadNumber,
			FullName:           fullName,
			Phone:              cleanedPhone,
			Email:              strings.TrimSpace(item.Email),
			City:               strings.TrimSpace(item.City),
			State:              strings.TrimSpace(item.State),
			ServiceRequired:    serviceReq,
			Source:             source,
			TotalDebtAmount:    totalDebt,
			OutstandingAmount:  outstanding,
			Status:             status,
			QualificationNotes: strings.TrimSpace(item.QualificationNotes),
			FinancialAssessment: &models.FinancialAssessment{
				MonthlyIncome:   item.MonthlyIncome,
				TotalMonthlyEMI: item.CurrentEMIAmount,
				TotalDebtAmount: totalDebt,
				Debts:           []models.DebtItem{},
			},
			AssignedTo:     assignedToObjID,
			AssignedToName: assignedToName,
			Debts:          []models.DebtItem{},
			Enquiries:      []models.EnquiryItem{},
			Conversations:  []models.ConversationItem{},
			FollowUps:      []models.FollowUp{},
			ActivityTimeline: []models.ActivityItem{
				{
					ID:              primitive.NewObjectID(),
					Type:            "LEAD_CREATED",
					Description:     "Lead imported in bulk from spreadsheet",
					PerformedByName: creatorName,
					CreatedAt:       leadTime,
				},
			},
			CreatedAt: leadTime,
			UpdatedAt: leadTime,
		}

		documents = append(documents, lead)
	}

	if len(documents) == 0 {
		response.BadRequest(c, "No valid leads could be imported. Please verify that borrower names and 10-digit phone numbers are provided.")
		return
	}

	if _, err := col.InsertMany(ctx, documents); err != nil {
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
