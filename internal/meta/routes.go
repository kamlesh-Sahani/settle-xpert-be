package meta

import (
	"lead-crm/backend/internal/meta/controllers"
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
	{
		meta.GET("/get", middleware.Protect("meta:read"), controllers.Get)
		meta.POST("/list", middleware.Protect("meta:read"), controllers.List)
		meta.POST("/save", middleware.Protect("meta:update"), controllers.Save)
		meta.POST("/test-connection", middleware.Protect("meta:update"), controllers.TestConnection)
	}
}
