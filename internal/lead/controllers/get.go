package controllers

import (
	"errors"
	"log"
	"strings"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetByID(c *gin.Context) {
	idStr := strings.TrimSpace(c.Query("id"))
	if idStr == "" {
		response.BadRequest(c, "Lead ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil || objID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	filter := bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}

	var lead models.Lead
	if err := col.FindOne(c.Request.Context(), filter).Decode(&lead); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [LeadGet] Database query failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	if lead.CallSchedules == nil {
		lead.CallSchedules = []models.CallSchedule{}
	}
	if lead.Financial != nil && lead.Financial.Lenders == nil {
		lead.Financial.Lenders = []models.Lender{}
	}
	if !hasLeadPermission(c, "lead:assign") {
		lead.AssignedTo = nil
	} else if lead.AssignedTo != nil && !lead.AssignedTo.IsZero() {
		employeeNames := fetchEmployeeNameMap(c.Request.Context())
		if name := employeeNames[lead.AssignedTo.Hex()]; name != "" {
			lead.AssignedToName = name
		}
	}

	response.Success(c, "Lead retrieved successfully", lead)
}
