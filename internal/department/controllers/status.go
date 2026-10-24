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

type UpdateDepartmentStatusRequest struct {
	ID     string                  `json:"id" binding:"required"`
	Status models.DepartmentStatus `json:"status" binding:"required,oneof=ACTIVE IN_ACTIVE"`
}

func UpdateStatus(c *gin.Context) {
	var req UpdateDepartmentStatusRequest
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

	now := time.Now()
	_, err = col.UpdateOne(ctx, bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}, bson.M{
		"$set": bson.M{
			"status":     req.Status,
			"updated_at": now,
		},
	})
	if err != nil {
		response.InternalServerError(c, "Failed to update department status")
		return
	}

	audit.Record(c, audit.RecordParams{
		EntityID:   req.ID,
		EntityType: "department",
		Action:     "UPDATE",
		OldData:    map[string]any{"status": existing.Status},
		NewData:    map[string]any{"status": req.Status},
	})

	response.Success(c, "Department status updated successfully", gin.H{
		"id":     req.ID,
		"status": req.Status,
	})
}
