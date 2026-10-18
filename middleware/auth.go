package middleware

import (
	"strings"

	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Auth is a middleware that extracts and validates a JWT access token from the
// "access_token" HttpOnly cookie or Authorization header.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		claims, err := utils.ParseJWTToken(tokenString)
		if err != nil || claims == nil {
			response.Unauthorized(c, "You are not authorized")
			c.Abort()
			return
		}

		userObjID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil || userObjID.IsZero() {
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

		userCtx := &utils.UserContext{
			UserID:    userObjID.Hex(),
			Email:     claims.Email,
			Name:      claims.Name,
			SessionID: sessionObjID.Hex(),
		}

		c.Set("user_context", userCtx)
		c.Set("user_id", claims.UserID)
		c.Set("user_obj_id", userObjID)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Set("session_id", claims.SessionID)
		c.Set("session_obj_id", sessionObjID)
		c.Set("claims", claims)

		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
		return strings.TrimSpace(cookie)
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	return ""
}
