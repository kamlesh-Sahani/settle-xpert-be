package controllers

import (
	"log"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/employee/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DeleteEmployeeRequest struct {
	ID string `json:"id" binding:"required"`
}

func Delete(c *gin.Context) {
	var req DeleteEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	currentEmpID := c.GetString("employee_id")
	if currentEmpID != "" && currentEmpID == req.ID {
		response.BadRequest(c, "You cannot delete your own account")
		return
	}

	objID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		response.BadRequest(c, "Invalid employee ID")
		return
	}

	collection := database.GetCollection("employees")

	var existingEmployee models.Employee
	_ = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&existingEmployee)

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
				"status":     models.EmployeeStatusDeleted,
				"updated_at": now,
			},
		},
	)

	if err != nil {
		log.Printf("[ERROR] [EmployeeDelete] Failed to delete employee: %v", err)
		response.InternalServerError(c, "Unable to delete employee. Please try again.")
		return
	}

	if result.MatchedCount == 0 {
		response.NotFound(c, "Employee not found or already deleted")
		return
	}

	RecordAuditLog(c, objID.Hex(), "DELETE", existingEmployee, nil)

	response.Success(c, "Employee deleted successfully", nil)
}
