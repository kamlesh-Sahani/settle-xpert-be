package utils

import (
	"github.com/gin-gonic/gin"
)

type EmployeeContext struct {
	EmployeeID string `json:"employee_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	SessionID  string `json:"session_id"`
}

func GetAuthEmployee(c *gin.Context) (*EmployeeContext, bool) {
	val, exists := c.Get("employee_context")
	if !exists {
		return nil, false
	}

	emp, ok := val.(*EmployeeContext)
	return emp, ok
}
