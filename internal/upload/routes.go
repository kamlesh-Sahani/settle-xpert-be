package upload

import (
	"lead-crm/backend/internal/upload/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	uploadGroup := router.Group("/api/v1/upload", middleware.Protect())
	{
		uploadGroup.POST("/presigned-url", controllers.GetPresignedURL)
		uploadGroup.POST("/verify", controllers.VerifyUpload)
		uploadGroup.POST("/delete", controllers.DeleteFiles)
	}
}
