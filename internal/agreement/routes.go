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
		agreementGroup.POST("/generate-pdf", middleware.Protect("agreement:read"), controllers.GeneratePDF)
		agreementGroup.GET("/template", middleware.Protect("agreement:read"), controllers.GetTemplate)
		agreementGroup.POST("/template/update", middleware.Protect("agreement:template_update"), controllers.UpdateTemplate)
	}
}
