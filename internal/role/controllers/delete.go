package controllers

import (
	"log"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/role/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DeleteRoleRequest struct {
	ID     string `json:"id"`
	RoleID string `json:"roleId"`
}

func Delete(c *gin.Context) {
	var req DeleteRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	rawID := req.ID
	if rawID == "" {
		rawID = req.RoleID
	}
	if rawID == "" {
		response.BadRequest(c, "Role ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(rawID)
	if err != nil {
		response.BadRequest(c, "Invalid role ID format")
		return
	}

	// Check if this role is currently assigned to any active employee
	empCol := database.GetCollection("employees")
	empCount, err := empCol.CountDocuments(c.Request.Context(), bson.M{
		"role_id":    bson.M{"$in": []any{objID, rawID}},
		"deleted_at": bson.M{"$exists": false},
	})
	if err != nil {
		log.Printf("[ERROR] [RoleDelete] Failed to check attached employees: %v", err)
		response.InternalServerError(c, "Unable to verify role usage. Please try again.")
		return
	}

	if empCount > 0 {
		response.BadRequest(c, "Cannot delete role because it is currently assigned to one or more employees")
		return
	}

	collection := database.GetCollection("roles")
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
			"status":     models.RoleStatusInactive,
			"updated_at": time.Now(),
		},
	}
	result, err := collection.UpdateOne(c.Request.Context(), bson.M{"_id": objID, "is_deleted": bson.M{"$ne": true}}, update)

	if err != nil {
		log.Printf("[ERROR] [RoleDelete] Failed to delete role: %v", err)
		response.InternalServerError(c, "Unable to delete role. Please try again.")
		return
	}

	if result.MatchedCount == 0 {
		response.NotFound(c, "Role not found")
		return
	}

	response.Success(c, "Role deleted successfully", nil)
}
