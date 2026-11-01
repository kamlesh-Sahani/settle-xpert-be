package controllers

import (
	"errors"
	"log"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/customer/models"
	leadmodels "lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetByID(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		response.BadRequest(c, "Customer ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid customer ID format")
		return
	}

	col := database.GetCollection("customers")
	ctx := c.Request.Context()

	filter := bson.M{"_id": objID, "deleted_at": bson.M{"$exists": false}}

	var customer models.Customer
	if err := col.FindOne(ctx, filter).Decode(&customer); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Customer not found")
			return
		}
		log.Printf("[ERROR] [CustomerGet] Database query failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve customer")
		return
	}

	if customer.Debts == nil {
		customer.Debts = []leadmodels.DebtItem{}
	}

	if len(customer.Loans) == 0 && len(customer.Debts) > 0 {
		customer.Loans = make([]models.LoanSettlementItem, len(customer.Debts))
		for i, d := range customer.Debts {
			customer.Loans[i] = models.LoanSettlementItem{
				ID:                primitive.NewObjectID(),
				Lender:            d.Lender,
				LoanType:          d.LoanType,
				OriginalAmount:    d.OutstandingAmount,
				OutstandingAmount: d.OutstandingAmount,
				Status:            "PENDING",
			}
		}
	} else if customer.Loans == nil {
		customer.Loans = []models.LoanSettlementItem{}
	}

	if customer.CaseStatus == "" {
		customer.CaseStatus = models.CaseStatusOnboarding
	}
	if customer.CaseNumber == "" && len(customer.CustomerNumber) > 4 {
		customer.CaseNumber = "CASE" + customer.CustomerNumber[4:]
	}

	response.Success(c, "Customer retrieved successfully", customer)
}
