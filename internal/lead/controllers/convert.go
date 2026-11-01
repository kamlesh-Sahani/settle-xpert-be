package controllers

import (
	"errors"
	"fmt"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/audit"
	auditmodels "lead-crm/backend/internal/audit/models"
	customermodels "lead-crm/backend/internal/customer/models"
	"lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConvertLeadRequest struct {
	LeadID string `json:"leadId" binding:"required"`
}

func ConvertLeadToCustomer(c *gin.Context) {
	var req ConvertLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	leadObjID, err := primitive.ObjectIDFromHex(req.LeadID)
	if err != nil {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	leadCol := database.GetCollection("leads")
	custCol := database.GetCollection("customers")
	ctx := c.Request.Context()

	var lead models.Lead
	if err := leadCol.FindOne(ctx, bson.M{"_id": leadObjID, "deleted_at": bson.M{"$exists": false}}).Decode(&lead); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	if lead.CustomerID != nil && !lead.CustomerID.IsZero() {
		response.Conflict(c, "This lead has already been converted to a customer")
		return
	}

	var convertedByID *primitive.ObjectID
	convertedByName := "System"
	if emp, ok := utils.GetAuthEmployee(c); ok {
		convertedByName = emp.Name
		if id, err := primitive.ObjectIDFromHex(emp.EmployeeID); err == nil {
			convertedByID = &id
		}
	}

	now := time.Now()

	count, _ := custCol.CountDocuments(ctx, bson.M{})
	seq := count + 1
	customerNumber := fmt.Sprintf("CUST-%04d", seq)
	for {
		existsCount, _ := custCol.CountDocuments(ctx, bson.M{"customer_number": customerNumber})
		if existsCount == 0 {
			break
		}
		seq++
		customerNumber = fmt.Sprintf("CUST-%04d", seq)
	}

	custID := primitive.NewObjectID()
	caseNumber := fmt.Sprintf("CASE-%04d", seq)
	loans := make([]customermodels.LoanSettlementItem, len(lead.Debts))
	for i, d := range lead.Debts {
		loans[i] = customermodels.LoanSettlementItem{
			ID:                primitive.NewObjectID(),
			Lender:            d.Lender,
			LoanType:          d.LoanType,
			OriginalAmount:    d.OutstandingAmount,
			OutstandingAmount: d.OutstandingAmount,
			Status:            "PENDING",
		}
	}

	customer := customermodels.Customer{
		ID:                  custID,
		CustomerNumber:      customerNumber,
		CaseNumber:          caseNumber,
		CaseStatus:          customermodels.CaseStatusOnboarding,
		LeadID:              &leadObjID,
		LeadNumber:          lead.LeadNumber,
		FullName:            lead.FullName,
		Email:               lead.Email,
		Phone:               lead.Phone,
		AlternatePhone:      lead.AlternatePhone,
		DOB:                 lead.DOB,
		PANCard:             lead.PANCard,
		Address:             lead.Address,
		City:                lead.City,
		State:               lead.State,
		Status:              customermodels.CustomerStatusActive,
		TotalDebtAmount:     lead.TotalDebtAmount,
		OutstandingAmount:   lead.OutstandingAmount,
		FinancialAssessment: lead.FinancialAssessment,
		Debts:               lead.Debts,
		Loans:               loans,
		AssignedTo:          lead.AssignedTo,
		AssignedToName:      lead.AssignedToName,
		AssignedTeam:        lead.AssignedTeam,
		ConvertedBy:         convertedByID,
		ConvertedByName:     convertedByName,
		ConvertedAt:         now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if _, err := custCol.InsertOne(ctx, customer); err != nil {
		response.InternalServerError(c, "Failed to create customer record")
		return
	}

	activity := models.ActivityItem{
		ID:              primitive.NewObjectID(),
		Type:            "LEAD_CONVERTED",
		Description:     fmt.Sprintf("Lead converted to customer %s", customerNumber),
		PerformedByName: convertedByName,
		CreatedAt:       now,
	}

	leadUpdate := bson.M{
		"$set": bson.M{
			"status":          models.LeadStatusConverted,
			"converted_at":    now,
			"customer_id":     custID,
			"customer_number": customerNumber,
			"updated_at":      now,
		},
		"$push": bson.M{
			"activity_timeline": activity,
		},
	}

	if _, err := leadCol.UpdateOne(ctx, bson.M{"_id": leadObjID}, leadUpdate); err != nil {
		response.InternalServerError(c, "Customer record created, but failed to update lead status")
		return
	}

	audit.Record(c, audit.RecordParams{
		EntityID:   leadObjID.Hex(),
		EntityType: "lead",
		Action:     auditmodels.ActionUpdate,
		OldData:    lead,
		NewData: bson.M{
			"status":          models.LeadStatusConverted,
			"customer_id":     custID.Hex(),
			"customer_number": customerNumber,
			"converted_at":    now,
		},
	})

	response.Success(c, "Lead converted to customer successfully", customer)
}
