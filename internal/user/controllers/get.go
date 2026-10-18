package controllers

import (
	"lead-crm/backend/database"
	"lead-crm/backend/internal/user/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetByID(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		id = c.Query("userId")
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	collection := database.GetCollection("users")
	var user models.User

	err = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&user)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	roleName := ""
	if user.RoleID != nil {
		var roleDoc bson.M
		if err := database.GetCollection("roles").FindOne(c.Request.Context(), bson.M{"_id": user.RoleID}).Decode(&roleDoc); err == nil {
			if name, ok := roleDoc["name"].(string); ok {
				roleName = name
			}
		}
	}

	userData := gin.H{
		"id":            user.ID.Hex(),
		"teamId":        user.TeamID,
		"name":          user.Name,
		"email":         user.Email,
		"phone":         user.Phone,
		"status":        user.Status,
		"logo":          user.Logo,
		"roleName":      roleName,
		"dob":           user.DOB,
		"gender":        user.Gender,
		"fatherName":    user.FatherName,
		"motherName":    user.MotherName,
		"panCard":       user.PANCard,
		"dateOfJoining": user.DateOfJoining,
		"aadharNumber":  user.AadharNumber,
		"aadharFront":   user.AadharFront,
		"aadharBack":    user.AadharBack,
		"address":       user.Address,
		"createdAt":     user.CreatedAt,
		"updatedAt":     user.UpdatedAt,
	}
	if user.RoleID != nil {
		userData["roleId"] = user.RoleID.Hex()
		userData["role_id"] = user.RoleID.Hex()
	}

	response.Success(c, "User retrieved successfully", userData)
}
