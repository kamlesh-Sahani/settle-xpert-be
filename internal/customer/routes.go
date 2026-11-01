package customer

import (
	"lead-crm/backend/internal/customer/controllers"
	"lead-crm/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	customerGroup := router.Group("/api/v1/customer")
	{
		customerGroup.POST("/list", middleware.Protect("customer:read"), controllers.List)
		customerGroup.GET("/get", middleware.Protect("customer:read"), controllers.GetByID)
		customerGroup.POST("/case-status", middleware.Protect("customer:update", "customer:read"), controllers.UpdateCaseStatus)
		customerGroup.POST("/loan/update", middleware.Protect("customer:update", "customer:read"), controllers.UpdateLoanSettlement)
	}
}
