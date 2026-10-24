package controllers

import (
	"lead-crm/backend/database"
	"lead-crm/backend/internal/employee/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetByID(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		response.BadRequest(c, "Employee ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		response.BadRequest(c, "Invalid employee ID")
		return
	}

	collection := database.GetCollection("employees")
	var employee models.Employee

	err = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&employee)
	if err != nil {
		response.NotFound(c, "Employee not found")
		return
	}

	roleName := ""
	if employee.RoleID != nil {
		var roleDoc bson.M
		if err := database.GetCollection("roles").FindOne(c.Request.Context(), bson.M{"_id": employee.RoleID}).Decode(&roleDoc); err == nil {
			if name, ok := roleDoc["name"].(string); ok {
				roleName = name
			}
		}
	}

	departmentName := ""
	if employee.DepartmentID != nil {
		var deptDoc bson.M
		if err := database.GetCollection("departments").FindOne(c.Request.Context(), bson.M{"_id": employee.DepartmentID}).Decode(&deptDoc); err == nil {
			if name, ok := deptDoc["name"].(string); ok {
				departmentName = name
			}
		}
	}

	empData := gin.H{
		"id":            employee.ID.Hex(),
		"empId":         employee.EmpID,
		"name":          employee.Name,
		"email":         employee.Email,
		"phone":         employee.Phone,
		"status":        employee.Status,
		"logo":          employee.Logo,
		"roleName":      roleName,
		"dob":           employee.DOB,
		"gender":        employee.Gender,
		"fatherName":    employee.FatherName,
		"motherName":    employee.MotherName,
		"panCard":       employee.PANCard,
		"dateOfJoining": employee.DateOfJoining,
		"aadharNumber":  employee.AadharNumber,
		"aadharFront":   employee.AadharFront,
		"aadharBack":    employee.AadharBack,
		"address":       employee.Address,
		"createdAt":     employee.CreatedAt,
		"updatedAt":     employee.UpdatedAt,
	}
	if employee.RoleID != nil {
		empData["roleId"] = employee.RoleID.Hex()
	}
	if employee.DepartmentID != nil {
		empData["departmentId"] = employee.DepartmentID.Hex()
		empData["departmentName"] = departmentName
	}

	response.Success(c, "Employee retrieved successfully", empData)
}
