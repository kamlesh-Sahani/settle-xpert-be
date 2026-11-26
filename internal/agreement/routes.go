package agreement

import (
	"lead-crm/backend/internal/agreement/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	agreementGroup := router.Group("/api/v1/agreement")
	{
		agreementGroup.POST("/create", middleware.Protect("agreement:create"), controllers.Create)
		agreementGroup.GET("/template", middleware.Protect("agreement:read"), controllers.GetTemplate)
		agreementGroup.POST("/template", middleware.Protect("agreement:template"), controllers.SaveTemplate)
		agreementGroup.POST("/preview-pdf", middleware.Protect("agreement:template"), controllers.PreviewTemplatePDF)
		agreementGroup.POST("/generate-pdf", middleware.Protect("agreement:read"), controllers.GeneratePDF)
	}
}
