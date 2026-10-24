package controllers

import (
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/user/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateUserRequest struct {
	UserID        string          `json:"userId"`
	ID            string          `json:"id"`
	Name          *string         `json:"name"`
	Email         *string         `json:"email"`
	Password      *string         `json:"password"`
	Status        *string         `json:"status"`
	RoleID        *string         `json:"roleId"`
	RoleIDSnake   *string         `json:"role_id"`
	Phone         *string         `json:"phone"`
	Logo          interface{}     `json:"logo"`
	TeamID        *string         `json:"teamId"`
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
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	targetID := req.UserID
	if targetID == "" {
		targetID = req.ID
	}
	if targetID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(targetID)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
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
	if req.TeamID != nil && *req.TeamID != "" {
		updateFields["team_id"] = *req.TeamID
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

	effectiveRoleID := req.RoleID
	if (effectiveRoleID == nil || *effectiveRoleID == "") && req.RoleIDSnake != nil {
		effectiveRoleID = req.RoleIDSnake
	}

	if effectiveRoleID != nil {
		if *effectiveRoleID == "" {
			updateFields["role_id"] = nil
		} else {
			roleObjID, err := primitive.ObjectIDFromHex(*effectiveRoleID)
			if err != nil {
				response.BadRequest(c, "Invalid role ID format")
				return
			}

			// Verify Role exists
			roleCol := database.GetCollection("roles")
			countRole, _ := roleCol.CountDocuments(c.Request.Context(), bson.M{"_id": roleObjID, "is_deleted": bson.M{"$ne": true}})
			if countRole == 0 {
				response.NotFound(c, "Role not found")
				return
			}

			updateFields["role_id"] = roleObjID
		}
	}

	collection := database.GetCollection("users")

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
			response.Conflict(c, "User with this email already exists")
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
			log.Printf("[ERROR] [UserUpdate] Failed to hash password: %v", err)
			response.InternalServerError(c, "Unable to update password. Please try again.")
			return
		}

		updateFields["password"] = hashedPassword
	}

	var existingUser models.User
	_ = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&existingUser)

	result, err := collection.UpdateOne(
		c.Request.Context(),
		bson.M{
			"_id":        objID,
			"deleted_at": bson.M{"$exists": false},
		},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		log.Printf("[ERROR] [UserUpdate] Failed to update user: %v", err)
		response.InternalServerError(c, "Unable to update user. Please try again.")
		return
	}

	if result.MatchedCount == 0 {
		response.NotFound(c, "User not found")
		return
	}

	RecordAuditLog(c, objID.Hex(), "UPDATE", existingUser, updateFields)

	response.Success(c, "User updated successfully", nil)
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
	userIDStr, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		response.Unauthorized(c, "Invalid user ID")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	collection := database.GetCollection("users")
	var user models.User
	err = collection.FindOne(c.Request.Context(), bson.M{"_id": userID, "deleted_at": bson.M{"$exists": false}}).Decode(&user)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	// Permission map check - strictly verify permissions
	permissionsMap := make(map[string]bool)
	if user.RoleID != nil {
		var roleDoc bson.M
		if err := database.GetCollection("roles").FindOne(c.Request.Context(), bson.M{"_id": user.RoleID}).Decode(&roleDoc); err == nil {
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
			if err := utils.VerifyHash(*req.CurrentPassword, user.Password); err != nil {
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

	_, err = collection.UpdateOne(c.Request.Context(), bson.M{"_id": userID}, bson.M{"$set": updateFields})
	if err != nil {
		response.InternalServerError(c, "Failed to update profile")
		return
	}

	RecordAuditLog(c, userID.Hex(), "UPDATE_PROFILE", user, updateFields)

	response.Success(c, "Profile updated successfully", nil)
}
