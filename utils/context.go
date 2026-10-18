package utils

import (
	"github.com/gin-gonic/gin"
)

type UserContext struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	SessionID string `json:"session_id"`
}

func GetAuthUser(c *gin.Context) (*UserContext, bool) {
	val, exists := c.Get("user_context")
	if !exists {
		return nil, false
	}

	user, ok := val.(*UserContext)
	return user, ok
}
