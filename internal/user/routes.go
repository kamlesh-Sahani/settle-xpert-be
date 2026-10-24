package user

import (
	"lead-crm/backend/internal/user/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	team := router.Group("/api/v1/team")
	{
		// Auth & Session Endpoints
		team.POST("/login", controllers.Login)
		team.POST("/refresh", controllers.Refresh)
		team.POST("/logout", controllers.Logout)
		team.GET("/config", middleware.Auth(), controllers.GetConfig)
		team.POST("/update-profile", middleware.Auth(), controllers.UpdateProfile)

		// Member Management Endpoints
		team.GET("/next-iden", middleware.Auth(), controllers.NextIden)
		team.POST("/create", middleware.Auth(), middleware.RequirePermission("member:create"), controllers.Create)
		team.GET("/get", middleware.Auth(), middleware.RequirePermission("member:read"), controllers.GetByID)
		team.POST("/list", middleware.Auth(), middleware.RequirePermission("member:read"), controllers.GetAll)
		team.POST("/update", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.Update)
		team.POST("/status", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.UpdateStatus)
		team.POST("/delete", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.Delete)
		team.GET("/options", middleware.Auth(), controllers.Options)
		team.POST("/audit-history", middleware.Auth(), middleware.RequireAnyPermission("member:audit", "member:read"), controllers.GetAuditHistory)
		team.POST("/audit-history/details", middleware.Auth(), middleware.RequireAnyPermission("member:audit", "member:read"), controllers.GetAuditHistoryDetails)
	}
}
