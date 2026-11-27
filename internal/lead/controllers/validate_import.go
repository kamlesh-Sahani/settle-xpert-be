package controllers

import (
	"log"
	"strings"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ValidateImportRequest struct {
	Leads []BulkLeadItem `json:"leads" binding:"required,min=1,max=2000"`
}

type ValidatedLeadItem struct {
	Row                    int                     `json:"row"`
	FullName               string                  `json:"fullName"`
	Phone                  string                  `json:"phone"`
	Email                  string                  `json:"email"`
	DOB                    string                  `json:"dob"`
	PANNumber              string                  `json:"panNumber"`
	MaritalStatus          string                  `json:"maritalStatus"`
	FullAddress            string                  `json:"fullAddress"`
	City                   string                  `json:"city"`
	State                  string                  `json:"state"`
	EmploymentStatus       models.EmploymentStatus `json:"employmentStatus,omitempty"`
	EmploymentType         string                  `json:"employmentType"`
	ServiceType            string                  `json:"serviceType"`
	TotalOutstandingAmount float64                 `json:"totalOutstandingAmount"`
	CreditCardDue          float64                 `json:"creditCardDue"`
	PersonalDue            float64                 `json:"personalDue"`
	MonthlyIncome          float64                 `json:"monthlyIncome"`
	MonthlyEMI             float64                 `json:"monthlyEmi"`
	Harassment             models.HarassmentStatus `json:"harassment,omitempty"`
	IsValid                bool                    `json:"isValid"`
	Status                 string                  `json:"status"` // VALID, INVALID, DUPLICATE_FILE, DUPLICATE_CRM
	ErrorReason            string                  `json:"errorReason"`
}

type ValidateImportResponse struct {
	TotalRows      int                 `json:"totalRows"`
	ValidCount     int                 `json:"validCount"`
	InvalidCount   int                 `json:"invalidCount"`
	DuplicateCount int                 `json:"duplicateCount"`
	ValidatedLeads []ValidatedLeadItem `json:"validatedLeads"`
}

func ValidateImport(c *gin.Context) {
	var req ValidateImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}
	ctx := c.Request.Context()

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
		cursor, err := col.Find(ctx, dupFilter, options.Find().SetProjection(bson.M{"personal.phone": 1}))
		if err != nil {
			log.Printf("[ERROR] [LeadValidateImport] Existing phone lookup failed: %v", err)
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

	seenInBatch := make(map[string]bool, len(req.Leads))
	validatedList := make([]ValidatedLeadItem, 0, len(req.Leads))
	validCount, invalidCount, duplicateCount := 0, 0, 0

	for i, item := range req.Leads {
		rowNum := i + 1
		fullName, cleanedPhone, _, validationErr := validateBulkLeadRow(item)
		if fullName == "" {
			fullName = strings.TrimSpace(item.FullName)
		}

		vLead := ValidatedLeadItem{
			Row:                    rowNum,
			FullName:               fullName,
			Phone:                  cleanedPhone,
			Email:                  strings.ToLower(strings.TrimSpace(item.Email)),
			DOB:                    strings.TrimSpace(item.DOB),
			PANNumber:              strings.ToUpper(strings.TrimSpace(item.PANNumber)),
			MaritalStatus:          strings.TrimSpace(item.MaritalStatus),
			FullAddress:            strings.TrimSpace(item.FullAddress),
			City:                   strings.TrimSpace(item.City),
			State:                  strings.TrimSpace(item.State),
			EmploymentStatus:       item.EmploymentStatus,
			EmploymentType:         strings.TrimSpace(item.EmploymentType),
			ServiceType:            strings.TrimSpace(item.ServiceType),
			TotalOutstandingAmount: item.TotalOutstandingAmount,
			CreditCardDue:          item.CreditCardDue,
			PersonalDue:            item.PersonalDue,
			MonthlyIncome:          item.MonthlyIncome,
			MonthlyEMI:             item.MonthlyEMI,
			Harassment:             item.Harassment,
		}
		if vLead.Phone == "" {
			vLead.Phone = strings.TrimSpace(item.Phone)
		}

		if validationErr != "" {
			vLead.Status = "INVALID"
			vLead.ErrorReason = validationErr
			invalidCount++
			validatedList = append(validatedList, vLead)
			continue
		}

		if seenInBatch[cleanedPhone] {
			vLead.Status = "DUPLICATE_FILE"
			vLead.ErrorReason = "Duplicate phone number in uploaded file"
			duplicateCount++
			validatedList = append(validatedList, vLead)
			continue
		}
		seenInBatch[cleanedPhone] = true

		if existingPhones[cleanedPhone] {
			vLead.Status = "DUPLICATE_CRM"
			vLead.ErrorReason = "Lead already exists in CRM"
			duplicateCount++
			validatedList = append(validatedList, vLead)
			continue
		}

		vLead.IsValid = true
		vLead.Status = "VALID"
		validCount++
		validatedList = append(validatedList, vLead)
	}

	response.Success(c, "Import leads validated successfully", ValidateImportResponse{
		TotalRows:      len(req.Leads),
		ValidCount:     validCount,
		InvalidCount:   invalidCount,
		DuplicateCount: duplicateCount,
		ValidatedLeads: validatedList,
	})
}
