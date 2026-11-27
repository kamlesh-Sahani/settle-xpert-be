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
	Duration           int        `json:"duration" binding:"required,gte=1"`
	ResolutionDuration int        `json:"resolutionDuration" binding:"required,gte=1"`
	ConsultingFees     float64    `json:"consultingFees" binding:"gte=0"`
	StartDate          *time.Time `json:"startDate"`
	ExecutedDate       *time.Time `json:"executedDate"`
	MonthlyIncome      float64    `json:"monthlyIncome" binding:"gte=0"`
}

type CreateAgreementRequest struct {
	LeadID    string                 `json:"leadId" binding:"required,max=24"`
	Personal  AgreementPersonalInput `json:"personal" binding:"required"`
	Lenders   []leadmodels.Lender    `json:"lenders" binding:"omitempty,max=50,dive"`
	Agreement AgreementDetailsInput  `json:"agreement" binding:"required"`
}

// Create saves agreement details, personal info, selected lenders, and monthly income onto the lead.
func Create(c *gin.Context) {
	emp, ok := utils.GetAuthEmployee(c)
	if !ok || emp == nil {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	preparedByID, err := primitive.ObjectIDFromHex(strings.TrimSpace(emp.EmployeeID))
	if err != nil || preparedByID.IsZero() {
		response.Unauthorized(c, "Invalid authenticated employee")
		return
	}
	preparedByName := strings.TrimSpace(emp.Name)
	if preparedByName == "" {
		response.Unauthorized(c, "Authenticated employee name is missing")
		return
	}

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

	phone := strings.TrimSpace(req.Personal.Phone)
	if phone == "" {
		response.BadRequest(c, "Phone number is required")
		return
	}

	if req.Agreement.Duration <= 0 {
		response.BadRequest(c, "Agreement duration must be at least 1 month")
		return
	}

	if req.Agreement.ResolutionDuration <= 0 {
		response.BadRequest(c, "Resolution duration must be at least 1 month")
		return
	}

	lenders := make([]leadmodels.Lender, 0, len(req.Lenders))
	var totalOutstanding float64
	for _, l := range req.Lenders {
		name := strings.TrimSpace(l.LenderName)
		loanType := strings.TrimSpace(l.LoanType)
		if name == "" || loanType == "" {
			response.BadRequest(c, "Lender name and loan type cannot be empty")
			return
		}
		if l.OutstandingAmount < 0 {
			response.BadRequest(c, "Lender outstanding amount cannot be negative")
			return
		}
		lenders = append(lenders, leadmodels.Lender{
			LenderName:        name,
			LoanType:          loanType,
			OutstandingAmount: l.OutstandingAmount,
		})
		totalOutstanding += l.OutstandingAmount
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	ctx := c.Request.Context()
	filter := bson.M{
		"_id":        leadObjID,
		"deleted_at": bson.M{"$exists": false},
	}

	var existing leadmodels.Lead
	if err := col.FindOne(ctx, filter).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [AgreementCreate] Lead lookup failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	now := time.Now().UTC()
	setFields := bson.M{
		"personal.full_name":                  fullName,
		"personal.pan_number":                 strings.ToUpper(strings.TrimSpace(req.Personal.PANNumber)),
		"personal.dob":                        strings.TrimSpace(req.Personal.DOB),
		"personal.phone":                      phone,
		"personal.email":                      strings.ToLower(strings.TrimSpace(req.Personal.Email)),
		"personal.full_address":               strings.TrimSpace(req.Personal.FullAddress),
		"financial.monthly_income":            req.Agreement.MonthlyIncome,
		"financial.lenders":                   lenders,
		"financial.total_outstanding_amount":  totalOutstanding,
		"agreement.duration":                  req.Agreement.Duration,
		"agreement.resolution_duration":       req.Agreement.ResolutionDuration,
		"agreement.consulting_fees":           req.Agreement.ConsultingFees,
		"agreement.start_date":                req.Agreement.StartDate,
		"agreement.executed_date":             req.Agreement.ExecutedDate,
		"agreement.prepared_by":               &preparedByID,
		"agreement.prepared_by_name":          preparedByName,
		"updated_at":                          now,
	}

	res, err := col.UpdateOne(ctx, filter, bson.M{"$set": setFields})
	if err != nil {
		log.Printf("[ERROR] [AgreementCreate] Update failed: %v", err)
		response.InternalServerError(c, "Failed to save agreement")
		return
	}
	if res.MatchedCount == 0 {
		response.NotFound(c, "Lead not found")
		return
	}

	response.Success(c, "Agreement saved successfully", gin.H{
		"id":         leadObjID.Hex(),
		"leadNumber": existing.LeadNumber,
	})
}
