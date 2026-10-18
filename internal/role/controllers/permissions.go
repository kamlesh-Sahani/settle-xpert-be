package controllers

import (
	"lead-crm/backend/internal/role/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
)

func GetPermissions(c *gin.Context) {
	response.Success(c, "Permissions retrieved successfully", models.AvailablePermissions)
}
