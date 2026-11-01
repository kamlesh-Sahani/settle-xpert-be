package controllers

import (
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/customer/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateCaseStatusRequest struct {
	CustomerID string            `json:"customerId" binding:"required"`
	CaseStatus models.CaseStatus `json:"caseStatus" binding:"required,oneof=ONBOARDING DOCS_VERIFICATION BANK_NEGOTIATION PARTIALLY_SETTLED RESOLVED"`
}

func UpdateCaseStatus(c *gin.Context) {
	var req UpdateCaseStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	custObjID, err := primitive.ObjectIDFromHex(req.CustomerID)
	if err != nil {
		response.BadRequest(c, "Invalid customer ID format")
		return
	}

	col := database.GetCollection("customers")
	ctx := c.Request.Context()

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"case_status": req.CaseStatus,
			"updated_at":  now,
		},
	}

	res, err := col.UpdateOne(ctx, bson.M{"_id": custObjID, "deleted_at": bson.M{"$exists": false}}, update)
	if err != nil {
		response.InternalServerError(c, "Failed to update case status")
		return
	}
	if res.MatchedCount == 0 {
		response.NotFound(c, "Customer not found")
		return
	}

	response.Success(c, "Case status updated successfully", gin.H{
		"customerId": req.CustomerID,
		"caseStatus": req.CaseStatus,
	})
}

type UpdateLoanSettlementRequest struct {
	CustomerID         string  `json:"customerId" binding:"required"`
	LoanID             string  `json:"loanId" binding:"required"`
	AccountNumber      string  `json:"accountNumber"`
	SettlementOffered  float64 `json:"settlementOffered"`
	SettlementApproved float64 `json:"settlementApproved"`
	Status             string  `json:"status" binding:"required,oneof=PENDING IN_NEGOTIATION OTS_APPROVED SETTLED CLOSED"`
	Remarks            string  `json:"remarks"`
}

func UpdateLoanSettlement(c *gin.Context) {
	var req UpdateLoanSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	custObjID, err := primitive.ObjectIDFromHex(req.CustomerID)
	if err != nil {
		response.BadRequest(c, "Invalid customer ID format")
		return
	}

	loanObjID, err := primitive.ObjectIDFromHex(req.LoanID)
	if err != nil {
		response.BadRequest(c, "Invalid loan ID format")
		return
	}

	col := database.GetCollection("customers")
	ctx := c.Request.Context()

	var customer models.Customer
	if err := col.FindOne(ctx, bson.M{"_id": custObjID, "deleted_at": bson.M{"$exists": false}}).Decode(&customer); err != nil {
		response.NotFound(c, "Customer not found")
		return
	}

	// If customer.Loans was not initialized yet, initialize from debts
	if len(customer.Loans) == 0 && len(customer.Debts) > 0 {
		customer.Loans = make([]models.LoanSettlementItem, len(customer.Debts))
		for i, d := range customer.Debts {
			customer.Loans[i] = models.LoanSettlementItem{
				ID:                d.ID,
				Lender:            d.Lender,
				LoanType:          d.LoanType,
				OriginalAmount:    d.OutstandingAmount,
				OutstandingAmount: d.OutstandingAmount,
				Status:            "PENDING",
			}
			if customer.Loans[i].ID.IsZero() {
				customer.Loans[i].ID = primitive.NewObjectID()
			}
		}
	}

	now := time.Now()
	loanFound := false
	for i := range customer.Loans {
		if customer.Loans[i].ID == loanObjID {
			loanFound = true
			customer.Loans[i].AccountNumber = req.AccountNumber
			customer.Loans[i].SettlementOffered = req.SettlementOffered
			customer.Loans[i].SettlementApproved = req.SettlementApproved
			customer.Loans[i].Status = req.Status
			customer.Loans[i].Remarks = req.Remarks
			customer.Loans[i].UpdatedAt = &now

			if req.SettlementApproved > 0 && customer.Loans[i].OriginalAmount > 0 {
				customer.Loans[i].SavingsAmount = customer.Loans[i].OriginalAmount - req.SettlementApproved
				customer.Loans[i].SavingsPercent = (customer.Loans[i].SavingsAmount / customer.Loans[i].OriginalAmount) * 100
			}
			break
		}
	}

	if !loanFound {
		response.NotFound(c, "Loan account not found")
		return
	}

	update := bson.M{
		"$set": bson.M{
			"loans":      customer.Loans,
			"updated_at": now,
		},
	}

	if _, err := col.UpdateOne(ctx, bson.M{"_id": custObjID}, update); err != nil {
		response.InternalServerError(c, "Failed to update loan settlement details")
		return
	}

	response.Success(c, "Loan settlement updated successfully", customer.Loans)
}
