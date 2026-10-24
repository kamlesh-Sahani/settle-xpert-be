package department

import (
	"lead-crm/backend/internal/department/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	departments := router.Group("/api/v1/departments")
	{
		departments.POST("/create", middleware.Protect("department:create"), controllers.Create)
		departments.POST("/list", middleware.Protect("department:read"), controllers.List)
		departments.GET("/get", middleware.Protect("department:read"), controllers.GetByID)
		departments.POST("/update", middleware.Protect("department:update"), controllers.Update)
		departments.POST("/status", middleware.Protect("department:update"), controllers.UpdateStatus)
		departments.POST("/delete", middleware.Protect("department:delete"), controllers.Delete)
		departments.POST("/audit-history", middleware.Protect("department:read"), controllers.GetAuditHistory)
		departments.GET("/audit-details", middleware.Protect("department:read"), controllers.GetAuditHistoryDetails)
		departments.GET("/options", middleware.Protect(), controllers.Options)
		departments.POST("/key-values", middleware.Protect(), controllers.Options)
	}
}
