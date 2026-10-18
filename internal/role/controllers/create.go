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

type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Status      any      `json:"status"`
	Permissions []string `json:"permissions"`
}

func Create(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	collection := database.GetCollection("roles")

	// Validate permissions against available permissions if provided
	if len(req.Permissions) > 0 {
		validPerms := make(map[string]bool)
		for _, group := range models.AvailablePermissions {
			for _, perm := range group.Permissions {
				validPerms[perm.ID] = true
			}
		}

		for _, perm := range req.Permissions {
			if !validPerms[perm] {
				response.BadRequest(c, "Invalid permission ID provided: "+perm)
				return
			}
		}
	}

	var existingRole models.Role
	err := collection.FindOne(c.Request.Context(), bson.M{
		"name":       req.Name,
		"is_deleted": bson.M{"$ne": true},
	}).Decode(&existingRole)
	if err == nil {
		response.Conflict(c, "Role with this name already exists in the organization")
		return
	}

	status := models.RoleStatusActive
	if req.Status != nil {
		switch v := req.Status.(type) {
		case string:
			if strings.EqualFold(v, models.RoleStatusInactive) {
				status = models.RoleStatusInactive
			} else if strings.EqualFold(v, models.RoleStatusActive) {
				status = models.RoleStatusActive
			}
		case bool:
			if !v {
				status = models.RoleStatusInactive
			}
		}
	}

	permissions := req.Permissions
	if permissions == nil {
		permissions = []string{}
	}

	role := models.Role{
		Name:        req.Name,
		Description: req.Description,
		Status:      status,
		IsDeleted:   false,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result, err := collection.InsertOne(c.Request.Context(), role)
	if err != nil {
		log.Printf("[ERROR] [RoleCreate] Failed to create role: %v", err)
		response.InternalServerError(c, "Unable to create role. Please try again.")
		return
	}

	role.ID = result.InsertedID.(primitive.ObjectID)

	response.Created(c, "Role created successfully", gin.H{
		"id": role.ID.Hex(),
	})
}
