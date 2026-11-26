package controllers

import (
	"errors"
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DeleteLeadRequest struct {
	ID string `json:"id" binding:"required,max=24"`
}

func Delete(c *gin.Context) {
	var req DeleteLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	objID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.ID))
	if err != nil || objID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}
	ctx := c.Request.Context()

	filter := bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}

	var existing struct {
		Status models.LeadStatus `bson:"status"`
	}
	if err := col.FindOne(ctx, filter).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [LeadDelete] Lookup failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	if existing.Status == models.LeadStatusConverted {
		response.Conflict(c, "Cannot delete an already converted lead")
		return
	}

	now := time.Now().UTC()
	res, err := col.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
		},
	})
	if err != nil {
		log.Printf("[ERROR] [LeadDelete] Delete failed: %v", err)
		response.InternalServerError(c, "Failed to delete lead")
		return
	}
	if res.MatchedCount == 0 {
		response.NotFound(c, "Lead not found")
		return
	}

	response.Success(c, "Lead deleted successfully", nil)
}
