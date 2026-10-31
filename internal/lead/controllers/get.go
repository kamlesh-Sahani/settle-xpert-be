package controllers

import (
	"errors"
	"log"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetByID(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		response.BadRequest(c, "Lead ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	col := database.GetCollection("leads")
	ctx := c.Request.Context()

	filter := bson.M{"_id": objID, "deleted_at": bson.M{"$exists": false}}
	if tVal, exists := c.Get("tenant_obj_id"); exists {
		if tid, ok := tVal.(primitive.ObjectID); ok {
			filter["tenant_id"] = tid
		}
	}

	var lead models.Lead
	if err := col.FindOne(ctx, filter).Decode(&lead); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [LeadGet] Database query failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	// Ensure slices are initialized for JSON response
	if lead.Conversations == nil {
		lead.Conversations = []models.ConversationItem{}
	}
	if lead.Debts == nil {
		lead.Debts = []models.DebtItem{}
	}
	if lead.FollowUps == nil {
		lead.FollowUps = []models.FollowUp{}
	}
	if lead.ActivityTimeline == nil {
		lead.ActivityTimeline = []models.ActivityItem{}
	}

	response.Success(c, "Lead retrieved successfully", lead)
}
