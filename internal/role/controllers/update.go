package controllers

import (
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/role/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateRoleRequest struct {
	ID          string    `json:"id" binding:"required"`
	Module      *string   `json:"module"`
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	Status      any       `json:"status"`
	Permissions *[]string `json:"permissions"`
}

func Update(c *gin.Context) {
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	objID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		response.BadRequest(c, "Invalid role ID format")
		return
	}

	collection := database.GetCollection("roles")

	updateFields := bson.M{"updated_at": time.Now()}
	if req.Name != nil {
		// Check for duplicate name
		var existingRole models.Role
		err = collection.FindOne(c.Request.Context(), bson.M{
			"name":       *req.Name,
			"_id":        bson.M{"$ne": objID},
			"is_deleted": bson.M{"$ne": true},
		}).Decode(&existingRole)

		if err == nil {
			response.Conflict(c, "Role with this name already exists in the organization")
			return
		}
		updateFields["name"] = *req.Name
	}
	if req.Description != nil {
		updateFields["description"] = *req.Description
	}
	if req.Status != nil {
		statusStr := models.RoleStatusActive
		switch v := req.Status.(type) {
		case string:
			if strings.EqualFold(v, models.RoleStatusInactive) {
				statusStr = models.RoleStatusInactive
			} else if strings.EqualFold(v, models.RoleStatusActive) {
				statusStr = models.RoleStatusActive
			}
		case bool:
			if !v {
				statusStr = models.RoleStatusInactive
			}
		}
		updateFields["status"] = statusStr
	}
	if req.Permissions != nil {
		if req.Module != nil && strings.TrimSpace(*req.Module) != "" {
			// Modular permission update: only replace permissions for the specified module
			moduleName := strings.TrimSpace(*req.Module)
			var targetGroup *models.PermissionGroup
			for _, group := range models.AvailablePermissions {
				if strings.EqualFold(group.Module, moduleName) {
					targetGroup = &group
					break
				}
			}

			if targetGroup == nil {
				response.BadRequest(c, "Module not found: "+moduleName)
				return
			}

			modulePerms := make(map[string]bool)
			for _, p := range targetGroup.Permissions {
				modulePerms[p.ID] = true
			}

			for _, perm := range *req.Permissions {
				if !modulePerms[perm] {
					response.BadRequest(c, "Invalid permission ID for module "+moduleName+": "+perm)
					return
				}
			}

			var currentRole models.Role
			err = collection.FindOne(c.Request.Context(), bson.M{
				"_id":        objID,
				"is_deleted": bson.M{"$ne": true},
			}).Decode(&currentRole)
			if err != nil {
				response.NotFound(c, "Role not found")
				return
			}

			// Keep existing permissions from other modules
			var newPerms []string
			for _, p := range currentRole.Permissions {
				if !modulePerms[p] {
					newPerms = append(newPerms, p)
				}
			}

			// Add updated permissions for this module
			newPerms = append(newPerms, *req.Permissions...)
			updateFields["permissions"] = newPerms
		} else {
			// Global permission update
			validPerms := make(map[string]bool)
			for _, group := range models.AvailablePermissions {
				for _, perm := range group.Permissions {
					validPerms[perm.ID] = true
				}
			}

			for _, perm := range *req.Permissions {
				if !validPerms[perm] {
					response.BadRequest(c, "Invalid permission ID provided: "+perm)
					return
				}
			}
			updateFields["permissions"] = *req.Permissions
		}
	}

	result, err := collection.UpdateOne(
		c.Request.Context(),
		bson.M{"_id": objID, "is_deleted": bson.M{"$ne": true}},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		log.Printf("[ERROR] [RoleUpdate] Failed to update role: %v", err)
		response.InternalServerError(c, "Unable to update role. Please try again.")
		return
	}

	if result.MatchedCount == 0 {
		response.NotFound(c, "Role not found")
		return
	}

	response.Success(c, "Role updated successfully", nil)
}
