package controllers

import (
	"lead-crm/backend/internal/audit"

	"github.com/gin-gonic/gin"
)

func GetAuditHistory(c *gin.Context) {
	audit.GetAuditHistory(c)
}

func GetAuditHistoryDetails(c *gin.Context) {
	audit.GetAuditHistoryDetails(c)
}
