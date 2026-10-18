package controllers

import (
	"fmt"
	"strings"

	"lead-crm/backend/database"
	rolemodels "lead-crm/backend/internal/role/models"
	"lead-crm/backend/internal/user/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetConfig(c *gin.Context) {
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

	userCol := database.GetCollection("users")
	var user models.User
	err = userCol.FindOne(c.Request.Context(), bson.M{
		"_id":        userID,
		"status":     bson.M{"$ne": models.UserStatusDeleted},
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&user)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	var permissions []string = []string{}
	var roleName string
	if user.RoleID != nil {
		roleCol := database.GetCollection("roles")
		var role rolemodels.Role
		err = roleCol.FindOne(c.Request.Context(), bson.M{"_id": user.RoleID}).Decode(&role)
		if err == nil {
			permissions = role.Permissions
			roleName = role.Name
		}
	}

	var roleIDStr string
	if user.RoleID != nil {
		roleIDStr = user.RoleID.Hex()
	}

	scheme := "https"
	if c.Request.TLS == nil && !strings.Contains(c.Request.Host, "localhost") {
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		}
	}
	webhookURL := fmt.Sprintf("%s://%s/api/v1/integration/meta/webhook", scheme, c.Request.Host)

	response.Success(c, "Configuration loaded successfully", gin.H{
		"user": gin.H{
			"id":       user.ID.Hex(),
			"name":     user.Name,
			"email":    user.Email,
			"status":   user.Status,
			"roleId":   roleIDStr,
			"roleName": roleName,
		},
		"permissions": permissions,
		"webhooks": gin.H{
			"meta": webhookURL,
		},
	})
}
