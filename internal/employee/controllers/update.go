package controllers

import (
	"log"
	"strings"
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

type UpdateEmployeeRequest struct {
	ID            string          `json:"id" binding:"required"`
	Name          *string         `json:"name"`
	Email         *string         `json:"email"`
	Password      *string         `json:"password"`
	Status        *string         `json:"status"`
	RoleID        *string         `json:"roleId"`
	Phone         *string         `json:"phone"`
	Logo          interface{}     `json:"logo"`
	DOB           *string         `json:"dob"`
	Gender        *string         `json:"gender"`
	FatherName    *string         `json:"fatherName"`
	MotherName    *string         `json:"motherName"`
	PANCard       *string         `json:"panCard"`
	DateOfJoining *string         `json:"dateOfJoining"`
	AadharNumber  *string         `json:"aadharNumber"`
	AadharFront   interface{}     `json:"aadharFront"`
	AadharBack    interface{}     `json:"aadharBack"`
	Address       *AddressRequest `json:"address"`
}

func Update(c *gin.Context) {
	var req UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	objID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		response.BadRequest(c, "Invalid employee ID")
		return
	}

	updateFields := bson.M{"updated_at": time.Now()}

	if req.Name != nil {
		updateFields["name"] = *req.Name
	}
	if req.Status != nil {
		updateFields["status"] = *req.Status
	}
	if req.Phone != nil {
		updateFields["phone"] = *req.Phone
	}
	if req.Logo != nil {
		if logoStr := parseLogo(req.Logo); logoStr != nil {
			updateFields["logo"] = *logoStr
		}
	}
	if req.DOB != nil {
		updateFields["dob"] = *req.DOB
	}
	if req.Gender != nil {
		updateFields["gender"] = *req.Gender
	}
	if req.FatherName != nil {
		updateFields["father_name"] = *req.FatherName
	}
	if req.MotherName != nil {
		updateFields["mother_name"] = *req.MotherName
	}
	if req.PANCard != nil {
		updateFields["pan_card"] = *req.PANCard
	}
	if req.DateOfJoining != nil {
		updateFields["date_of_joining"] = *req.DateOfJoining
	}
	if req.AadharNumber != nil {
		updateFields["aadhar_number"] = *req.AadharNumber
	}
	if req.AadharFront != nil {
		if frontStr := parseLogo(req.AadharFront); frontStr != nil {
			updateFields["aadhar_front"] = *frontStr
		}
	}
	if req.AadharBack != nil {
		if backStr := parseLogo(req.AadharBack); backStr != nil {
			updateFields["aadhar_back"] = *backStr
		}
	}
	if req.Address != nil {
		updateFields["address"] = models.Address{
			Line1:   req.Address.Line1,
			Country: req.Address.Country,
			State:   req.Address.State,
			City:    req.Address.City,
			Zip:     req.Address.Zip,
		}
	}

	if req.RoleID != nil {
		if *req.RoleID == "" {
			updateFields["role_id"] = nil
		} else {
			roleObjID, err := primitive.ObjectIDFromHex(*req.RoleID)
			if err != nil {
				response.BadRequest(c, "Invalid role ID format")
				return
			}

			roleCol := database.GetCollection("roles")
			countRole, _ := roleCol.CountDocuments(c.Request.Context(), bson.M{"_id": roleObjID, "is_deleted": bson.M{"$ne": true}})
			if countRole == 0 {
				response.NotFound(c, "Role not found")
				return
			}

			updateFields["role_id"] = roleObjID
		}
	}

	collection := database.GetCollection("employees")

	if req.Email != nil {
		cleanEmail := strings.TrimSpace(strings.ToLower(*req.Email))
		if cleanEmail == "" {
			response.BadRequest(c, "Email cannot be empty")
			return
		}
		count, _ := collection.CountDocuments(c.Request.Context(), bson.M{
			"email":      cleanEmail,
			"_id":        bson.M{"$ne": objID},
			"deleted_at": bson.M{"$exists": false},
		})
		if count > 0 {
			response.Conflict(c, "Employee with this email already exists")
			return
		}
		updateFields["email"] = cleanEmail
	}

	if req.Password != nil && *req.Password != "" {
		if len(*req.Password) < 6 {
			response.BadRequest(c, "Password must be at least 6 characters")
			return
		}

		hashedPassword, err := utils.Hash(*req.Password)
		if err != nil {
			log.Printf("[ERROR] [EmployeeUpdate] Failed to hash password: %v", err)
			response.InternalServerError(c, "Unable to update password. Please try again.")
			return
		}

		updateFields["password"] = hashedPassword
	}

	var existingEmployee models.Employee
	_ = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&existingEmployee)

	result, err := collection.UpdateOne(
		c.Request.Context(),
		bson.M{
			"_id":        objID,
			"deleted_at": bson.M{"$exists": false},
		},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		log.Printf("[ERROR] [EmployeeUpdate] Failed to update employee: %v", err)
		response.InternalServerError(c, "Unable to update employee. Please try again.")
		return
	}

	if result.MatchedCount == 0 {
		response.NotFound(c, "Employee not found")
		return
	}

	RecordAuditLog(c, objID.Hex(), "UPDATE", existingEmployee, updateFields)

	response.Success(c, "Employee updated successfully", nil)
}

type UpdateProfileRequest struct {
	Name            *string         `json:"name"`
	Phone           *string         `json:"phone"`
	Logo            interface{}     `json:"logo"`
	DOB             *string         `json:"dob"`
	Gender          *string         `json:"gender"`
	FatherName      *string         `json:"fatherName"`
	MotherName      *string         `json:"motherName"`
	PANCard         *string         `json:"panCard"`
	DateOfJoining   *string         `json:"dateOfJoining"`
	AadharNumber    *string         `json:"aadharNumber"`
	AadharFront     interface{}     `json:"aadharFront"`
	AadharBack      interface{}     `json:"aadharBack"`
	Address         *AddressRequest `json:"address"`
	CurrentPassword *string         `json:"currentPassword"`
	NewPassword     *string         `json:"newPassword"`
}

func UpdateProfile(c *gin.Context) {
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

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	collection := database.GetCollection("employees")
	var employee models.Employee
	err = collection.FindOne(c.Request.Context(), bson.M{"_id": empID, "deleted_at": bson.M{"$exists": false}}).Decode(&employee)
	if err != nil {
		response.NotFound(c, "Employee not found")
		return
	}

	permissionsMap := make(map[string]bool)
	if employee.RoleID != nil {
		var roleDoc bson.M
		if err := database.GetCollection("roles").FindOne(c.Request.Context(), bson.M{"_id": employee.RoleID}).Decode(&roleDoc); err == nil {
			if perms, ok := roleDoc["permissions"].(primitive.A); ok {
				for _, p := range perms {
					if pStr, ok := p.(string); ok {
						permissionsMap[pStr] = true
					}
				}
			}
		}
	}

	canUpdate := func(permID string) bool {
		return permissionsMap[permID]
	}

	updateFields := bson.M{"updated_at": time.Now()}

	if req.Name != nil && canUpdate("profile:update_name") {
		if trimmed := strings.TrimSpace(*req.Name); trimmed != "" {
			updateFields["name"] = trimmed
		}
	}
	if req.Phone != nil && canUpdate("profile:update_phone") {
		updateFields["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.Logo != nil && canUpdate("profile:update_photo") {
		if logoStr := parseLogo(req.Logo); logoStr != nil {
			updateFields["logo"] = *logoStr
		}
	}
	if req.DOB != nil && canUpdate("profile:update_dob") {
		updateFields["dob"] = *req.DOB
	}
	if req.Gender != nil && canUpdate("profile:update_gender") {
		updateFields["gender"] = *req.Gender
	}
	if req.FatherName != nil && canUpdate("profile:update_father_name") {
		updateFields["father_name"] = *req.FatherName
	}
	if req.MotherName != nil && canUpdate("profile:update_mother_name") {
		updateFields["mother_name"] = *req.MotherName
	}
	if req.PANCard != nil && canUpdate("profile:update_pan_card") {
		updateFields["pan_card"] = *req.PANCard
	}
	if req.DateOfJoining != nil && canUpdate("profile:update_date_of_joining") {
		updateFields["date_of_joining"] = *req.DateOfJoining
	}
	if req.AadharNumber != nil && canUpdate("profile:update_aadhar_number") {
		updateFields["aadhar_number"] = *req.AadharNumber
	}
	if req.AadharFront != nil && canUpdate("profile:update_aadhar_front") {
		if frontStr := parseLogo(req.AadharFront); frontStr != nil {
			updateFields["aadhar_front"] = *frontStr
		}
	}
	if req.AadharBack != nil && canUpdate("profile:update_aadhar_back") {
		if backStr := parseLogo(req.AadharBack); backStr != nil {
			updateFields["aadhar_back"] = *backStr
		}
	}
	if req.Address != nil && canUpdate("profile:update_address") {
		updateFields["address"] = models.Address{
			Line1:   req.Address.Line1,
			Country: req.Address.Country,
			State:   req.Address.State,
			City:    req.Address.City,
			Zip:     req.Address.Zip,
		}
	}

	if req.NewPassword != nil && *req.NewPassword != "" && canUpdate("profile:update_password") {
		if len(*req.NewPassword) < 6 {
			response.BadRequest(c, "New password must be at least 6 characters")
			return
		}
		if req.CurrentPassword != nil && *req.CurrentPassword != "" {
			if err := utils.VerifyHash(*req.CurrentPassword, employee.Password); err != nil {
				response.BadRequest(c, "Current password does not match")
				return
			}
		}
		hashedPassword, err := utils.Hash(*req.NewPassword)
		if err != nil {
			response.InternalServerError(c, "Unable to update password. Please try again.")
			return
		}
		updateFields["password"] = hashedPassword
	}

	_, err = collection.UpdateOne(c.Request.Context(), bson.M{"_id": empID}, bson.M{"$set": updateFields})
	if err != nil {
		response.InternalServerError(c, "Failed to update profile")
		return
	}

	RecordAuditLog(c, empID.Hex(), "UPDATE_PROFILE", employee, updateFields)

	response.Success(c, "Profile updated successfully", nil)
}
