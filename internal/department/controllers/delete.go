package controllers

import (
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/audit"
	"lead-crm/backend/internal/department/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DeleteDepartmentRequest struct {
	ID string `json:"id" binding:"required"`
}

func Delete(c *gin.Context) {
	var req DeleteDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	objID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		response.BadRequest(c, "Invalid department ID format")
		return
	}

	col := database.GetCollection("departments")
	ctx := c.Request.Context()

	var existing models.Department
	err = col.FindOne(ctx, bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&existing)
	if err != nil {
		response.NotFound(c, "Department not found")
		return
	}

	empCol := database.GetCollection("employees")
	empCount, err := empCol.CountDocuments(ctx, bson.M{
		"department_id": objID,
		"deleted_at":    bson.M{"$exists": false},
	})
	if err == nil && empCount > 0 {
		response.BadRequest(c, "Cannot delete department because it is currently assigned to one or more employees")
		return
	}

	now := time.Now()
	_, err = col.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{
		"$set": bson.M{
			"status":     models.DepartmentStatusInactive,
			"deleted_at": now,
			"updated_at": now,
		},
	})
	if err != nil {
		response.InternalServerError(c, "Failed to delete department")
		return
	}

	audit.Record(c, audit.RecordParams{
		EntityID:   objID.Hex(),
		EntityType: "department",
		Action:     "DELETE",
		OldData:    existing,
	})

	response.Success(c, "Department deleted successfully", nil)
}
