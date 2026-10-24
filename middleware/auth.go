package middleware

import (
	"strings"
	"time"

	"lead-crm/backend/database"
	employeemodels "lead-crm/backend/internal/employee/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const maxTokenLength = 4096

type authResult struct {
	SessionID  primitive.ObjectID      `bson:"_id"`
	EmployeeID primitive.ObjectID      `bson:"employee_id"`
	Employee   employeemodels.Employee `bson:"employee"`
	Role       *struct {
		Permissions []string `bson:"permissions"`
	} `bson:"role"`
}

// Protect authenticates requests using the HttpOnly access_token cookie and validates RBAC permissions.
func Protect(requiredPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Strict cookie-only token extraction
		cookie, err := c.Cookie("access_token")
		if err != nil || cookie == "" || len(cookie) > maxTokenLength {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		// 2. Cryptographic JWT validation
		claims, err := utils.ParseJWTToken(strings.TrimSpace(cookie))
		if err != nil || claims == nil {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		empObjID, err := primitive.ObjectIDFromHex(claims.EmployeeID)
		if err != nil || empObjID.IsZero() {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		sessionObjID, err := primitive.ObjectIDFromHex(claims.SessionID)
		if err != nil || sessionObjID.IsZero() {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		// 3. Single atomic MongoDB aggregation (session + active employee + role permissions)
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"_id":         sessionObjID,
				"employee_id": empObjID,
				"expires_at":  bson.M{"$gt": time.Now()},
			}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "employees",
				"localField":   "employee_id",
				"foreignField": "_id",
				"as":           "employee_list",
			}}},
			{{Key: "$unwind", Value: "$employee_list"}},
			{{Key: "$match", Value: bson.M{
				"employee_list.deleted_at": bson.M{"$exists": false},
				"employee_list.status":     employeemodels.EmployeeStatusActive,
			}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "roles",
				"localField":   "employee_list.role_id",
				"foreignField": "_id",
				"as":           "role_list",
			}}},
			{{Key: "$project", Value: bson.M{
				"_id":         1,
				"employee_id": 1,
				"employee":    "$employee_list",
				"role":        bson.M{"$arrayElemAt": bson.A{"$role_list", 0}},
			}}},
		}

		cursor, err := database.GetCollection("sessions").Aggregate(c.Request.Context(), pipeline)
		if err != nil {
			response.Unauthorized(c, "Authentication verification failed")
			c.Abort()
			return
		}
		defer cursor.Close(c.Request.Context())

		if !cursor.Next(c.Request.Context()) {
			response.Unauthorized(c, "Session is invalid, expired, or employee account is inactive")
			c.Abort()
			return
		}

		var authState authResult
		if err := cursor.Decode(&authState); err != nil {
			response.Unauthorized(c, "Failed to parse authentication state")
			c.Abort()
			return
		}

		// 4. Granular RBAC permission check
		var permissions []string
		if authState.Role != nil {
			permissions = authState.Role.Permissions
		}

		if len(requiredPermissions) > 0 {
			hasPerm := false
			for _, req := range requiredPermissions {
				for _, p := range permissions {
					if req == p {
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
		}

		// 5. Hydrate request context
		c.Set("employee", &authState.Employee)
		c.Set("employee_context", &utils.EmployeeContext{
			EmployeeID: empObjID.Hex(),
			Email:      authState.Employee.Email,
			Name:       authState.Employee.Name,
			SessionID:  sessionObjID.Hex(),
		})
		c.Set("employee_id", claims.EmployeeID)
		c.Set("emp_obj_id", empObjID)
		c.Set("email", authState.Employee.Email)
		c.Set("name", authState.Employee.Name)
		c.Set("session_id", claims.SessionID)
		c.Set("session_obj_id", sessionObjID)
		c.Set("permissions", permissions)

		c.Next()
	}
}
