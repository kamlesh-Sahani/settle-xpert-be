package integration

import (
	"lead-crm/backend/internal/integration/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	// 1. Public Webhook endpoint for Meta Lead Ads
	webhook := router.Group("/api/v1/integration/meta/webhook")
	{
		webhook.GET("", controllers.VerifyWebhook)
		webhook.POST("", controllers.HandleWebhook)
	}

	// 2. Protected integration configuration endpoints
	meta := router.Group("/api/v1/integration/meta")
	meta.Use(middleware.Auth())
	{
		meta.GET("/get", middleware.RequirePermission("meta:read"), controllers.Get)
		meta.POST("/list", middleware.RequirePermission("meta:read"), controllers.List)
		meta.POST("/save", middleware.RequirePermission("meta:update"), controllers.Save)
		meta.POST("/test-connection", middleware.RequirePermission("meta:update"), controllers.TestConnection)
	}
}
