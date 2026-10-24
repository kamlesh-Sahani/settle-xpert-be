package role

import (
	"lead-crm/backend/internal/role/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	roles := router.Group("/api/v1/roles")
	{
		roles.POST("/create", middleware.Protect("role:create"), controllers.Create)
		roles.POST("/list", middleware.Protect("role:read"), controllers.List)
		roles.GET("/get", middleware.Protect("role:read"), controllers.GetByID)
		roles.POST("/update", middleware.Protect("role:update"), controllers.Update)
		roles.POST("/delete", middleware.Protect("role:delete"), controllers.Delete)
		roles.GET("/permissions", middleware.Protect(), controllers.GetPermissions)
		roles.POST("/key-values", middleware.Protect(), controllers.Options)
		roles.GET("/options", middleware.Protect(), controllers.Options)
	}
}
