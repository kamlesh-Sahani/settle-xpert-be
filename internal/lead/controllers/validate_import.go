package controllers

import (
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
	DefaultSource models.LeadSource `json:"defaultSource"`
	Leads         []BulkLeadItem    `json:"leads" binding:"required,min=1"`
}

type ValidatedLeadItem struct {
	Row                int               `json:"row"`
	FullName           string            `json:"fullName"`
	Phone              string            `json:"phone"`
	Email              string            `json:"email"`
	ServiceRequired    string            `json:"serviceRequired"`
	Source             models.LeadSource `json:"source"`
	City               string            `json:"city"`
	State              string            `json:"state"`
	TotalDebtAmount    float64           `json:"totalDebtAmount"`
	MonthlyIncome      float64           `json:"monthlyIncome"`
	CurrentEMIAmount   float64           `json:"currentEmiAmount"`
	QualificationNotes string            `json:"qualificationNotes"`
	IsValid            bool              `json:"isValid"`
	Status             string            `json:"status"` // "VALID", "INVALID", "DUPLICATE_FILE", "DUPLICATE_CRM"
	ErrorReason        string            `json:"errorReason"`
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
	ctx := c.Request.Context()

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

	seenInBatch := make(map[string]bool)
	validatedList := make([]ValidatedLeadItem, 0, len(req.Leads))
	validCount := 0
	invalidCount := 0
	duplicateCount := 0

	for i, item := range req.Leads {
		rowNum := i + 1
		fullName := strings.TrimSpace(item.FullName)
		rawPhone := strings.TrimSuffix(strings.TrimSpace(item.Phone), ".0")
		cleanedPhone := bulkDigitCleaner.ReplaceAllString(rawPhone, "")
		if len(cleanedPhone) > 10 {
			cleanedPhone = cleanedPhone[len(cleanedPhone)-10:]
		}

		source := models.SourceExcelImport
		if item.Source != "" {
			source = item.Source
		} else if req.DefaultSource != "" {
			source = req.DefaultSource
		}

		vLead := ValidatedLeadItem{
			Row:                rowNum,
			FullName:           fullName,
			Phone:              cleanedPhone,
			Email:              strings.TrimSpace(item.Email),
			ServiceRequired:    item.ServiceRequired,
			Source:             source,
			City:               strings.TrimSpace(item.City),
			State:              strings.TrimSpace(item.State),
			TotalDebtAmount:    item.TotalDebtAmount,
			MonthlyIncome:      item.MonthlyIncome,
			CurrentEMIAmount:   item.CurrentEMIAmount,
			QualificationNotes: strings.TrimSpace(item.QualificationNotes),
			IsValid:            false,
		}
		if vLead.Phone == "" {
			vLead.Phone = rawPhone
		}

		if fullName == "" {
			vLead.Status = "INVALID"
			vLead.ErrorReason = "Borrower Full Name is missing"
			invalidCount++
			validatedList = append(validatedList, vLead)
			continue
		}

		if len(cleanedPhone) != 10 {
			vLead.Status = "INVALID"
			vLead.ErrorReason = "Invalid phone number (must be 10 digits)"
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
