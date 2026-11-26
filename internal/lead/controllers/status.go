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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateStatus(c *gin.Context) {
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

	statusStr := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if statusStr == "" {
		response.BadRequest(c, "Status is required")
		return
	}

	newStatus := models.LeadStatus(statusStr)
	switch newStatus {
	case models.LeadStatusNew,
		models.LeadStatusCallScheduled,
		models.LeadStatusInterested,
		models.LeadStatusNotInterested,
		models.LeadStatusConverted:
	default:
		response.BadRequest(c, "Invalid lead status")
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

	var existing struct {
		Status models.LeadStatus `bson:"status"`
	}
	if err := col.FindOne(c.Request.Context(), filter).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [LeadStatus] Lookup failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	if existing.Status == models.LeadStatusConverted {
		response.Conflict(c, "Cannot change status of an already converted lead")
		return
	}

	now := time.Now().UTC()
	res, err := col.UpdateOne(c.Request.Context(), filter, bson.M{
		"$set": bson.M{
			"status":     newStatus,
			"updated_at": now,
		},
	})
	if err != nil {
		log.Printf("[ERROR] [LeadStatus] Failed to update status: %v", err)
		response.InternalServerError(c, "Failed to update lead status")
		return
	}
	if res.MatchedCount == 0 {
		response.NotFound(c, "Lead not found")
		return
	}

	response.Success(c, "Lead status updated successfully", gin.H{
		"id":     idStr,
		"status": newStatus,
	})
}
