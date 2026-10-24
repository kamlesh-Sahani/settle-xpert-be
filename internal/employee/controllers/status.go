package controllers

import (
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/employee/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateStatusRequest struct {
	ID     string `json:"id" binding:"required"`
	Status string `json:"status" binding:"required"`
}

// UpdateStatus handles POST /api/v1/emp/status dedicated to updating an employee's status
func UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	objID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		response.BadRequest(c, "Invalid employee ID format")
		return
	}

	normalizedStatus := strings.ToUpper(strings.TrimSpace(req.Status))
	if normalizedStatus != models.EmployeeStatusActive && normalizedStatus != models.EmployeeStatusSuspended {
		response.BadRequest(c, "Invalid status. Allowed values are ACTIVE or SUSPENDED")
		return
	}

	currentEmpID := c.GetString("employee_id")
	if currentEmpID == req.ID && normalizedStatus == models.EmployeeStatusSuspended {
		response.BadRequest(c, "You cannot suspend your own account")
		return
	}

	collection := database.GetCollection("employees")

	var existingEmployee models.Employee
	_ = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&existingEmployee)

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
		response.InternalServerError(c, "Failed to update employee status")
		return
	}

	if res.MatchedCount == 0 {
		response.NotFound(c, "Employee not found")
		return
	}

	RecordAuditLog(c, req.ID, "UPDATE", map[string]any{"status": existingEmployee.Status}, map[string]any{"status": normalizedStatus})

	// Invalidate active sessions if the employee is suspended
	if normalizedStatus == models.EmployeeStatusSuspended {
		sessionCol := database.GetCollection("sessions")
		if sessionCol != nil {
			_, _ = sessionCol.DeleteMany(c.Request.Context(), bson.M{"employee_id": objID})
		}
	}

	response.Success(c, "Employee status updated successfully", gin.H{
		"id":     req.ID,
		"status": normalizedStatus,
	})
}
