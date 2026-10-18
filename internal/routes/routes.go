package routes

import (
	"lead-crm/backend/internal/integration"
	"lead-crm/backend/internal/role"
	"lead-crm/backend/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
)

func SetupRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/", func(c *gin.Context) {
		response.Success(c, "DebtCare CRM API is active", nil)
	})

	// Module routes
	user.RegisterRoutes(router)
	role.RegisterRoutes(router)
	integration.RegisterRoutes(router)
}

