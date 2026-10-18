package role

import (
	"lead-crm/backend/internal/role/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	roles := router.Group("/api/v1/roles")
	{
		roles.POST("/create", middleware.Auth(), middleware.RequirePermission("role:create"), controllers.Create)
		roles.POST("/list", middleware.Auth(), middleware.RequirePermission("role:read"), controllers.List)
		roles.GET("/get", middleware.Auth(), middleware.RequirePermission("role:read"), controllers.GetByID)
		roles.POST("/update", middleware.Auth(), middleware.RequirePermission("role:update"), controllers.Update)
		roles.POST("/delete", middleware.Auth(), middleware.RequirePermission("role:delete"), controllers.Delete)
		roles.GET("/permissions", middleware.Auth(), controllers.GetPermissions)
		roles.POST("/key-values", middleware.Auth(), controllers.Options)
		roles.GET("/options", middleware.Auth(), controllers.Options)
	}
}
