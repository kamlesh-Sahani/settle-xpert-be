package lead

import (
	"lead-crm/backend/internal/lead/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	leadGroup := router.Group("/api/v1/lead")
	{
		leadGroup.POST("/list", middleware.Protect("lead:read"), controllers.List)
		leadGroup.GET("/get", middleware.Protect("lead:read"), controllers.GetByID)
		leadGroup.GET("/status", middleware.Protect("lead:update", "lead:read"), controllers.UpdateStatus)
		leadGroup.POST("/create", middleware.Protect("lead:create"), controllers.Create)
		leadGroup.POST("/bulk-create", middleware.Protect("lead:create"), controllers.BulkCreate)
		leadGroup.POST("/update", middleware.Protect("lead:update"), controllers.Update)
		leadGroup.POST("/follow-up/create", middleware.Protect("lead:update"), controllers.CreateFollowUp)
		leadGroup.POST("/follow-up/status", middleware.Protect("lead:update"), controllers.UpdateFollowUpStatus)
	}
}
