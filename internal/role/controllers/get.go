package controllers

import (
	"strings"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/role/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModulePermissionItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Enabled     bool   `json:"enabled"`
}

func GetByID(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		id = c.Query("roleId")
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		response.BadRequest(c, "Invalid role ID format")
		return
	}

	collection := database.GetCollection("roles")
	var role models.Role

	err = collection.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"is_deleted": bson.M{"$ne": true},
	}).Decode(&role)
	if err != nil {
		response.NotFound(c, "Role not found")
		return
	}

	module := strings.TrimSpace(c.Query("module"))

	// If no module or detail requested, return basic role information
	if module == "" || strings.EqualFold(module, "detail") {
		response.Success(c, "Role details retrieved successfully", gin.H{
			"id":          role.ID.Hex(),
			"name":        role.Name,
			"description": role.Description,
			"status":      role.Status,
			"isDeleted":   role.IsDeleted,
		})
		return
	}

	// Find the requested module in available permissions
	var targetGroup *models.PermissionGroup
	for _, group := range models.AvailablePermissions {
		if strings.EqualFold(group.Module, module) {
			targetGroup = &group
			break
		}
	}

	if targetGroup == nil {
		response.BadRequest(c, "Module not found: "+module)
		return
	}

	// Check which permissions in this module are enabled for the role
	rolePermSet := make(map[string]bool)
	for _, p := range role.Permissions {
		rolePermSet[p] = true
	}

	var modulePermissions []ModulePermissionItem
	for _, p := range targetGroup.Permissions {
		modulePermissions = append(modulePermissions, ModulePermissionItem{
			ID:          p.ID,
			Title:       p.Title,
			Description: p.Description,
			Category:    p.Category,
			Enabled:     rolePermSet[p.ID],
		})
	}

	response.Success(c, "Module permissions retrieved successfully", gin.H{
		"module":      targetGroup.Module,
		"permissions": modulePermissions,
	})
}
