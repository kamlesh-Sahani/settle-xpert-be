package middleware

import (
	"lead-crm/backend/database"
	rolemodels "lead-crm/backend/internal/role/models"
	usermodels "lead-crm/backend/internal/user/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RequirePermission verifies that the authenticated user possesses the specified exact permission.
// It never checks role names, always checking the granular permission list assigned to the user's role.
// On failure, it returns 403 Forbidden (preventing false refresh loops in the client).
func RequirePermission(requiredPerm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userObjIDVal, exists := c.Get("user_obj_id")
		if !exists {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		userObjID, ok := userObjIDVal.(primitive.ObjectID)
		if !ok || userObjID.IsZero() {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		// Fetch user to retrieve assigned role ID
		userCol := database.GetCollection("users")
		var user usermodels.User
		err := userCol.FindOne(c.Request.Context(), bson.M{
			"_id":        userObjID,
			"deleted_at": bson.M{"$exists": false},
		}).Decode(&user)

		if err != nil || user.RoleID == nil {
			response.Forbidden(c, "You do not have permission to perform this action")
			c.Abort()
			return
		}

		// Fetch Role to verify exact permission match
		roleCol := database.GetCollection("roles")
		var role rolemodels.Role
		err = roleCol.FindOne(c.Request.Context(), bson.M{
			"_id":        user.RoleID,
			"is_deleted": bson.M{"$ne": true},
			"status":     bson.M{"$in": []any{rolemodels.RoleStatusActive, true, "ACTIVE"}},
		}).Decode(&role)

		if err != nil {
			response.Forbidden(c, "You do not have permission to perform this action")
			c.Abort()
			return
		}

		hasPerm := false
		for _, p := range role.Permissions {
			if p == requiredPerm {
				hasPerm = true
				break
			}
		}

		if !hasPerm {
			response.Forbidden(c, "You do not have permission to perform this action")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission verifies that the authenticated user possesses at least one of the specified permissions.
func RequireAnyPermission(requiredPerms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userObjIDVal, exists := c.Get("user_obj_id")
		if !exists {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		userObjID, ok := userObjIDVal.(primitive.ObjectID)
		if !ok || userObjID.IsZero() {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		userCol := database.GetCollection("users")
		var user usermodels.User
		err := userCol.FindOne(c.Request.Context(), bson.M{
			"_id":        userObjID,
			"deleted_at": bson.M{"$exists": false},
		}).Decode(&user)

		if err != nil || user.RoleID == nil {
			response.Forbidden(c, "You do not have permission to perform this action")
			c.Abort()
			return
		}

		roleCol := database.GetCollection("roles")
		var role rolemodels.Role
		err = roleCol.FindOne(c.Request.Context(), bson.M{
			"_id":        user.RoleID,
			"is_deleted": bson.M{"$ne": true},
			"status":     bson.M{"$in": []any{rolemodels.RoleStatusActive, true, "ACTIVE"}},
		}).Decode(&role)

		if err != nil {
			response.Forbidden(c, "You do not have permission to perform this action")
			c.Abort()
			return
		}

		hasPerm := false
		for _, p := range role.Permissions {
			for _, req := range requiredPerms {
				if p == req {
					hasPerm = true
					break
				}
			}
			if hasPerm {
				break
			}
		}

		if !hasPerm {
			response.Forbidden(c, "You do not have permission to perform this action")
			c.Abort()
			return
		}

		c.Next()
	}
}

