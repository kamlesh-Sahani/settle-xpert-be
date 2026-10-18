package controllers

import (
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/user/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateStatusRequest struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
	Status string `json:"status" binding:"required"`
}

// UpdateStatus handles POST /api/v1/team/status dedicated to updating a team member's status
func UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	targetID := req.UserID
	if targetID == "" {
		targetID = req.ID
	}
	if targetID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(targetID)
	if err != nil {
		response.BadRequest(c, "Invalid user ID format")
		return
	}

	normalizedStatus := strings.ToUpper(strings.TrimSpace(req.Status))
	if normalizedStatus != models.UserStatusActive && normalizedStatus != models.UserStatusSuspended {
		response.BadRequest(c, "Invalid status. Allowed values are ACTIVE or SUSPENDED")
		return
	}

	currentUserID := c.GetString("user_id")
	if currentUserID == targetID && normalizedStatus == models.UserStatusSuspended {
		response.BadRequest(c, "You cannot suspend your own account")
		return
	}

	collection := database.GetCollection("users")

	var existingUser models.User
	_ = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&existingUser)

	res, err := collection.UpdateOne(
		c.Request.Context(),
		bson.M{
			"_id":        objID,
			"deleted_at": bson.M{"$exists": false},
		},
		bson.M{
			"$set": bson.M{
				"status":     normalizedStatus,
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		response.InternalServerError(c, "Failed to update team member status")
		return
	}

	if res.MatchedCount == 0 {
		response.NotFound(c, "Team member not found")
		return
	}

	RecordAuditLog(c, targetID, "UPDATE", map[string]any{"status": existingUser.Status}, map[string]any{"status": normalizedStatus})

	// Invalidate active sessions if the user is suspended
	if normalizedStatus == models.UserStatusSuspended {
		sessionCol := database.GetCollection("sessions")
		if sessionCol != nil {
			_, _ = sessionCol.DeleteMany(c.Request.Context(), bson.M{"user_id": objID})
		}
	}

	response.Success(c, "Team member status updated successfully", gin.H{
		"id":     targetID,
		"status": normalizedStatus,
	})
}
