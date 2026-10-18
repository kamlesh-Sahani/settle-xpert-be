package user

import (
	"lead-crm/backend/internal/user/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	users := router.Group("/api/v1/users")
	{
		// Auth Endpoints
		users.POST("/login", controllers.Login)
		users.POST("/refresh", controllers.Refresh)
		users.POST("/logout", controllers.Logout)
		users.GET("/config", middleware.Auth(), controllers.GetConfig)
		users.GET("/options", middleware.Auth(), controllers.Options)

		// Member Management Endpoints (Permission-protected)
		users.GET("/next-iden", middleware.Auth(), controllers.NextIden)
		users.POST("/create", middleware.Auth(), middleware.RequirePermission("member:create"), controllers.Create)
		users.GET("/get", middleware.Auth(), middleware.RequirePermission("member:read"), controllers.GetByID)
		users.POST("/list", middleware.Auth(), middleware.RequirePermission("member:read"), controllers.GetAll)
		users.POST("/update", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.Update)
		users.POST("/status", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.UpdateStatus)
		users.POST("/delete", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.Delete)
		users.POST("/audit-history", middleware.Auth(), middleware.RequireAnyPermission("member:audit", "member:read"), controllers.GetAuditHistory)
		users.POST("/audit-history/details", middleware.Auth(), middleware.RequireAnyPermission("member:audit", "member:read"), controllers.GetAuditHistoryDetails)
	}

	teams := router.Group("/api/v1/team")
	{
		teams.GET("/next-iden", middleware.Auth(), controllers.NextIden)
		teams.POST("/create", middleware.Auth(), middleware.RequirePermission("member:create"), controllers.Create)
		teams.GET("/get", middleware.Auth(), middleware.RequirePermission("member:read"), controllers.GetByID)
		teams.POST("/list", middleware.Auth(), middleware.RequirePermission("member:read"), controllers.GetAll)
		teams.POST("/update", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.Update)
		teams.POST("/status", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.UpdateStatus)
		teams.POST("/delete", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.Delete)
		teams.GET("/options", middleware.Auth(), controllers.Options)
		teams.POST("/key-values", middleware.Auth(), controllers.Options)
		teams.POST("/audit-history", middleware.Auth(), middleware.RequireAnyPermission("member:audit", "member:read"), controllers.GetAuditHistory)
		teams.POST("/audit-history/details", middleware.Auth(), middleware.RequireAnyPermission("member:audit", "member:read"), controllers.GetAuditHistoryDetails)
	}

	teamsPlural := router.Group("/api/v1/teams")
	{
		teamsPlural.GET("/next-iden", middleware.Auth(), controllers.NextIden)
		teamsPlural.POST("/create", middleware.Auth(), middleware.RequirePermission("member:create"), controllers.Create)
		teamsPlural.GET("/get", middleware.Auth(), middleware.RequirePermission("member:read"), controllers.GetByID)
		teamsPlural.POST("/list", middleware.Auth(), middleware.RequirePermission("member:read"), controllers.GetAll)
		teamsPlural.POST("/update", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.Update)
		teamsPlural.POST("/status", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.UpdateStatus)
		teamsPlural.POST("/delete", middleware.Auth(), middleware.RequirePermission("member:update"), controllers.Delete)
		teamsPlural.POST("/audit-history", middleware.Auth(), middleware.RequireAnyPermission("member:audit", "member:read"), controllers.GetAuditHistory)
		teamsPlural.POST("/audit-history/details", middleware.Auth(), middleware.RequireAnyPermission("member:audit", "member:read"), controllers.GetAuditHistoryDetails)
	}
}
