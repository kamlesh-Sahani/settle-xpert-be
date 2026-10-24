package controllers

import (
	"strings"
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

type UpdateDepartmentRequest struct {
	ID          string                  `json:"id" binding:"required"`
	Name        string                  `json:"name" binding:"required"`
	Code        string                  `json:"code" binding:"required"`
	Description string                  `json:"description"`
	Status      models.DepartmentStatus `json:"status" binding:"omitempty,oneof=ACTIVE IN_ACTIVE"`
}

func Update(c *gin.Context) {
	var req UpdateDepartmentRequest
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

	name := strings.TrimSpace(req.Name)
	code := strings.ToUpper(strings.TrimSpace(req.Code))

	var duplicate models.Department
	err = col.FindOne(ctx, bson.M{
		"_id":        bson.M{"$ne": objID},
		"deleted_at": bson.M{"$exists": false},
		"$or": []bson.M{
			{"name": primitive.Regex{Pattern: "^" + name + "$", Options: "i"}},
			{"code": code},
		},
	}).Decode(&duplicate)
	if err == nil {
		if strings.EqualFold(duplicate.Name, name) {
			response.Conflict(c, "Department with this name already exists")
			return
		}
		response.Conflict(c, "Department with this code already exists")
		return
	}

	status := existing.Status
	if req.Status != "" {
		status = req.Status
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"name":        name,
			"code":        code,
			"description": strings.TrimSpace(req.Description),
			"status":      status,
			"updated_at":  now,
		},
	}

	_, err = col.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		response.InternalServerError(c, "Failed to update department")
		return
	}

	updatedDept := existing
	updatedDept.Name = name
	updatedDept.Code = code
	updatedDept.Description = strings.TrimSpace(req.Description)
	updatedDept.Status = status
	updatedDept.UpdatedAt = now

	audit.Record(c, audit.RecordParams{
		EntityID:   objID.Hex(),
		EntityType: "department",
		Action:     "UPDATE",
		OldData:    existing,
		NewData:    updatedDept,
	})

	response.Success(c, "Department updated successfully", updatedDept)
}
