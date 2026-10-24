package employee

import (
	"lead-crm/backend/internal/employee/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	emp := router.Group("/api/v1/emp")
	{
		// Auth & Session Endpoints
		emp.POST("/login", controllers.Login)
		emp.POST("/refresh", controllers.Refresh)
		emp.POST("/logout", controllers.Logout)
		emp.GET("/config", middleware.Protect(), controllers.GetConfig)
		emp.POST("/update-profile", middleware.Protect(), controllers.UpdateProfile)

		// Employee Management Endpoints
		emp.GET("/next-iden", middleware.Protect(), controllers.NextIden)
		emp.POST("/create", middleware.Protect("employee:create"), controllers.Create)
		emp.GET("/get", middleware.Protect("employee:read"), controllers.GetByID)
		emp.POST("/list", middleware.Protect("employee:read"), controllers.GetAll)
		emp.POST("/update", middleware.Protect("employee:update"), controllers.Update)
		emp.POST("/status", middleware.Protect("employee:update"), controllers.UpdateStatus)
		emp.POST("/delete", middleware.Protect("employee:update"), controllers.Delete)
		emp.GET("/options", middleware.Protect(), controllers.Options)
		emp.POST("/audit-history", middleware.Protect("employee:audit"), controllers.GetAuditHistory)
		emp.POST("/audit-history/details", middleware.Protect("employee:audit"), controllers.GetAuditHistoryDetails)
	}
}
