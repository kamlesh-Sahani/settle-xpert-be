package routes

import (
	"lead-crm/backend/internal/department"
	"lead-crm/backend/internal/employee"
	"lead-crm/backend/internal/meta"
	"lead-crm/backend/internal/role"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
)

func SetupRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/", func(c *gin.Context) {
		response.Success(c, "DebtCare CRM API is active", nil)
	})

	// Module routes
	employee.RegisterRoutes(router)
	department.RegisterRoutes(router)
	role.RegisterRoutes(router)
	meta.RegisterRoutes(router)
}
