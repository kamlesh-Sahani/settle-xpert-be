package controllers

import (
	"log"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/user/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DeleteUserRequest struct {
	UserID string `json:"userId"`
	ID     string `json:"id"`
}

func Delete(c *gin.Context) {
	var req DeleteUserRequest
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

	currentUserID := c.GetString("user_id")
	if currentUserID != "" && currentUserID == targetID {
		response.BadRequest(c, "You cannot delete your own account")
		return
	}

	objID, err := primitive.ObjectIDFromHex(targetID)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	collection := database.GetCollection("users")

	var existingUser models.User
	_ = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&existingUser)

	now := time.Now()
	result, err := collection.UpdateOne(
		c.Request.Context(),
		bson.M{
			"_id":        objID,
			"deleted_at": bson.M{"$exists": false},
		},
		bson.M{
			"$set": bson.M{
				"deleted_at": now,
				"status":     models.UserStatusDeleted,
				"updated_at": now,
			},
		},
	)

	if err != nil {
		log.Printf("[ERROR] [UserDelete] Failed to delete user: %v", err)
		response.InternalServerError(c, "Unable to delete user. Please try again.")
		return
	}

	if result.MatchedCount == 0 {
		response.NotFound(c, "User not found or already deleted")
		return
	}

	RecordAuditLog(c, objID.Hex(), "DELETE", existingUser, nil)

	response.Success(c, "User deleted successfully", nil)
}
