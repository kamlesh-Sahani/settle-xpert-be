package controllers

import (
	"log"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/employee/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AddressRequest struct {
	Line1   string `json:"line1"`
	Country string `json:"country"`
	State   string `json:"state"`
	City    string `json:"city"`
	Zip     string `json:"zip"`
}

type RegisterEmployeeRequest struct {
	Email         string          `json:"email" binding:"required,email"`
	Name          string          `json:"name" binding:"required"`
	Password      string          `json:"password" binding:"required,min=6"`
	RoleID        string          `json:"roleId" binding:"required"`
	DepartmentID  string          `json:"departmentId" binding:"required"`
	Phone         string          `json:"phone"`
	Status        *string         `json:"status"`
	Logo          interface{}     `json:"logo"`
	EmpID         *string         `json:"empId"`
	DOB           string          `json:"dob" binding:"required"`
	Gender        string          `json:"gender" binding:"required"`
	FatherName    string          `json:"fatherName"`
	MotherName    string          `json:"motherName"`
	PANCard       string          `json:"panCard"`
	DateOfJoining string          `json:"dateOfJoining" binding:"required"`
	AadharNumber  string          `json:"aadharNumber"`
	AadharFront   interface{}     `json:"aadharFront"`
	AadharBack    interface{}     `json:"aadharBack"`
	Address       *AddressRequest `json:"address"`
}

func parseLogo(logo interface{}) *string {
	if logo == nil {
		return nil
	}
	switch v := logo.(type) {
	case string:
		return &v
	case []interface{}:
		if len(v) > 0 {
			if firstMap, ok := v[0].(map[string]interface{}); ok {
				if url, ok := firstMap["url"].(string); ok {
					return &url
				}
				if resp, ok := firstMap["response"].(map[string]interface{}); ok {
					if url, ok := resp["url"].(string); ok {
						return &url
					}
				}
			} else if firstStr, ok := v[0].(string); ok {
				return &firstStr
			}
		}
		empty := ""
		return &empty
	default:
		return nil
	}
}

func Create(c *gin.Context) {
	var req RegisterEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	roleObjID, err := primitive.ObjectIDFromHex(req.RoleID)
	if err != nil {
		response.BadRequest(c, "Invalid role ID format")
		return
	}

	deptObjID, err := primitive.ObjectIDFromHex(req.DepartmentID)
	if err != nil {
		response.BadRequest(c, "Invalid department ID format")
		return
	}

	deptCol := database.GetCollection("departments")
	deptCount, _ := deptCol.CountDocuments(c.Request.Context(), bson.M{"_id": deptObjID, "deleted_at": bson.M{"$exists": false}})
	if deptCount == 0 {
		response.BadRequest(c, "Selected department not found")
		return
	}

	collection := database.GetCollection("employees")

	count, _ := collection.CountDocuments(c.Request.Context(), bson.M{
		"email":      req.Email,
		"deleted_at": bson.M{"$exists": false},
	})
	if count > 0 {
		response.Conflict(c, "Employee with this email already exists")
		return
	}

	hashedPassword, err := utils.Hash(req.Password)
	if err != nil {
		log.Printf("[ERROR] [EmployeeCreate] Failed to hash password: %v", err)
		response.InternalServerError(c, "Unable to create employee. Please try again.")
		return
	}

	status := models.EmployeeStatusActive
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	var logo string
	if req.Logo != nil {
		if logoStr := parseLogo(req.Logo); logoStr != nil {
			logo = *logoStr
		}
	}

	var empID string
	if req.EmpID != nil && *req.EmpID != "" {
		empID = *req.EmpID
	} else {
		generated, err := NextEmployeeIden(c.Request.Context())
		if err == nil {
			empID = generated
		}
	}

	var aadharFront string
	if req.AadharFront != nil {
		if frontStr := parseLogo(req.AadharFront); frontStr != nil {
			aadharFront = *frontStr
		}
	}

	var aadharBack string
	if req.AadharBack != nil {
		if backStr := parseLogo(req.AadharBack); backStr != nil {
			aadharBack = *backStr
		}
	}

	var addr models.Address
	if req.Address != nil {
		addr = models.Address{
			Line1:   req.Address.Line1,
			Country: req.Address.Country,
			State:   req.Address.State,
			City:    req.Address.City,
			Zip:     req.Address.Zip,
		}
	}

	employee := models.Employee{
		EmpID:         empID,
		RoleID:        &roleObjID,
		DepartmentID:  &deptObjID,
		Email:         req.Email,
		Name:          req.Name,
		Password:      hashedPassword,
		Phone:         req.Phone,
		Status:        status,
		Logo:          logo,
		DOB:           req.DOB,
		Gender:        req.Gender,
		FatherName:    req.FatherName,
		MotherName:    req.MotherName,
		PANCard:       req.PANCard,
		DateOfJoining: req.DateOfJoining,
		AadharNumber:  req.AadharNumber,
		AadharFront:   aadharFront,
		AadharBack:    aadharBack,
		Address:       addr,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	result, err := collection.InsertOne(c.Request.Context(), employee)
	if err != nil {
		log.Printf("[ERROR] [EmployeeCreate] Failed to insert employee: %v", err)
		response.InternalServerError(c, "Unable to create employee. Please try again.")
		return
	}

	employee.ID = result.InsertedID.(primitive.ObjectID)

	RecordAuditLog(c, employee.ID.Hex(), "CREATE", nil, employee)

	response.Created(c, "Employee created successfully", gin.H{
		"id":           employee.ID.Hex(),
		"empId":        employee.EmpID,
		"name":         employee.Name,
		"email":        employee.Email,
		"roleId":       req.RoleID,
		"departmentId": req.DepartmentID,
		"status":       employee.Status,
	})
}
