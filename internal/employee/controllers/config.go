package controllers

import (
	"fmt"
	"strings"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/employee/models"
	rolemodels "lead-crm/backend/internal/role/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetConfig(c *gin.Context) {
	empIDStr, exists := c.Get("employee_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	empID, err := primitive.ObjectIDFromHex(empIDStr.(string))
	if err != nil {
		response.Unauthorized(c, "Invalid employee ID")
		return
	}

	empCol := database.GetCollection("employees")
	var employee models.Employee
	err = empCol.FindOne(c.Request.Context(), bson.M{
		"_id":        empID,
		"status":     bson.M{"$ne": models.EmployeeStatusDeleted},
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&employee)
	if err != nil {
		response.NotFound(c, "Employee not found")
		return
	}

	var permissions []string = []string{}
	var roleName string
	if employee.RoleID != nil {
		roleCol := database.GetCollection("roles")
		var role rolemodels.Role
		err = roleCol.FindOne(c.Request.Context(), bson.M{"_id": employee.RoleID}).Decode(&role)
		if err == nil {
			permissions = role.Permissions
			roleName = role.Name
		}
	}

	var departmentName string
	if employee.DepartmentID != nil {
		deptCol := database.GetCollection("departments")
		var dept bson.M
		if err := deptCol.FindOne(c.Request.Context(), bson.M{"_id": employee.DepartmentID}).Decode(&dept); err == nil {
			if name, ok := dept["name"].(string); ok {
				departmentName = name
			}
		}
	}

	var roleIDStr string
	if employee.RoleID != nil {
		roleIDStr = employee.RoleID.Hex()
	}

	scheme := "https"
	if c.Request.TLS == nil && !strings.Contains(c.Request.Host, "localhost") {
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		}
	}
	webhookURL := fmt.Sprintf("%s://%s/api/v1/integration/meta/webhook", scheme, c.Request.Host)

	empData := gin.H{
		"id":            employee.ID.Hex(),
		"empId":         employee.EmpID,
		"name":          employee.Name,
		"email":         employee.Email,
		"phone":         employee.Phone,
		"status":        employee.Status,
		"roleId":        roleIDStr,
		"roleName":      roleName,
		"logo":          employee.Logo,
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
	}

	if employee.DepartmentID != nil {
		empData["departmentId"] = employee.DepartmentID.Hex()
		empData["departmentName"] = departmentName
	}

	response.Success(c, "Configuration loaded successfully", gin.H{
		"employee":    empData,
		"permissions": permissions,
		"webhooks": gin.H{
			"meta": webhookURL,
		},
	})
}
