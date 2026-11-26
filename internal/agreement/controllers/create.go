package controllers

import (
	"errors"
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	leadmodels "lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AgreementPersonalInput struct {
	FullName    string `json:"fullName" binding:"required,min=2,max=120"`
	PANNumber   string `json:"panNumber" binding:"omitempty,max=10"`
	DOB         string `json:"dob" binding:"omitempty,max=32"`
	Phone       string `json:"phone" binding:"required,max=20"`
	Email       string `json:"email" binding:"omitempty,email,max=160"`
	FullAddress string `json:"fullAddress" binding:"omitempty,max=500"`
}

type AgreementDetailsInput struct {
	Duration       string     `json:"duration" binding:"required,max=80"`
	ConsultingFees float64    `json:"consultingFees" binding:"gte=0"`
	StartDate      *time.Time `json:"startDate"`
	ExecutedDate   *time.Time `json:"executedDate"`
	MonthlyIncome  float64    `json:"monthlyIncome" binding:"gte=0"`
}

type CreateAgreementRequest struct {
	LeadID    string                 `json:"leadId" binding:"required,max=24"`
	Personal  AgreementPersonalInput `json:"personal" binding:"required"`
	Lenders   []leadmodels.Lender    `json:"lenders" binding:"omitempty,max=50,dive"`
	Agreement AgreementDetailsInput  `json:"agreement" binding:"required"`
}

// Create saves agreement details, personal info, selected lenders, and monthly income onto the lead.
func Create(c *gin.Context) {
	var req CreateAgreementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	leadObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.LeadID))
	if err != nil || leadObjID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	fullName := strings.TrimSpace(req.Personal.FullName)
	if len(fullName) < 2 {
		response.BadRequest(c, "Full name must be at least 2 characters")
		return
	}

	duration := strings.TrimSpace(req.Agreement.Duration)
	if duration == "" {
		response.BadRequest(c, "Agreement duration is required")
		return
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	filter := bson.M{
		"_id":        leadObjID,
		"deleted_at": bson.M{"$exists": false},
	}

	var existing leadmodels.Lead
	if err := col.FindOne(c.Request.Context(), filter).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [AgreementCreate] Lead lookup failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	var preparedByID *primitive.ObjectID
	preparedByName := "System"
	if emp, ok := utils.GetAuthEmployee(c); ok && emp != nil {
		preparedByName = emp.Name
		if oid, err := primitive.ObjectIDFromHex(emp.EmployeeID); err == nil && !oid.IsZero() {
			preparedByID = &oid
		}
	}

	lenders := make([]leadmodels.Lender, 0, len(req.Lenders))
	for _, l := range req.Lenders {
		name := strings.TrimSpace(l.LenderName)
		loanType := strings.TrimSpace(l.LoanType)
		if name == "" || loanType == "" {
			response.BadRequest(c, "Lender name and loan type cannot be empty")
			return
		}
		lenders = append(lenders, leadmodels.Lender{
			LenderName:        name,
			LoanType:          loanType,
			OutstandingAmount: l.OutstandingAmount,
		})
	}

	now := time.Now().UTC()
	setFields := bson.M{
		"personal.full_name":           fullName,
		"personal.pan_number":          strings.ToUpper(strings.TrimSpace(req.Personal.PANNumber)),
		"personal.dob":                 strings.TrimSpace(req.Personal.DOB),
		"personal.phone":               strings.TrimSpace(req.Personal.Phone),
		"personal.email":               strings.ToLower(strings.TrimSpace(req.Personal.Email)),
		"personal.full_address":        strings.TrimSpace(req.Personal.FullAddress),
		"financial.monthly_income":     req.Agreement.MonthlyIncome,
		"financial.lenders":            lenders,
		"agreement.duration":           duration,
		"agreement.consulting_fees":    req.Agreement.ConsultingFees,
		"agreement.start_date":         req.Agreement.StartDate,
		"agreement.executed_date":      req.Agreement.ExecutedDate,
		"agreement.prepared_by":        preparedByID,
		"agreement.prepared_by_name":   preparedByName,
		"updated_at":                   now,
	}

	if _, err := col.UpdateOne(c.Request.Context(), filter, bson.M{"$set": setFields}); err != nil {
		log.Printf("[ERROR] [AgreementCreate] Update failed: %v", err)
		response.InternalServerError(c, "Failed to save agreement")
		return
	}

	response.Success(c, "Agreement saved successfully", gin.H{
		"id":         leadObjID.Hex(),
		"leadNumber": existing.LeadNumber,
	})
}
