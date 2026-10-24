package controllers

import (
	"lead-crm/backend/internal/audit"

	"github.com/gin-gonic/gin"
)

// RecordAuditLog asynchronously logs an audit entry to the audit_logs collection using the reusable audit service.
func RecordAuditLog(c *gin.Context, entityID, action string, oldData, newData any) {
	audit.Record(c, audit.RecordParams{
		EntityID:   entityID,
		EntityType: "employee",
		Action:     action,
		OldData:    oldData,
		NewData:    newData,
	})
}

// GetAuditHistory delegates to the centralized audit table listing in internal/audit.
func GetAuditHistory(c *gin.Context) {
	audit.GetAuditHistory(c)
}

// GetAuditHistoryDetails delegates to the centralized audit detail fetcher in internal/audit.
func GetAuditHistoryDetails(c *gin.Context) {
	audit.GetAuditHistoryDetails(c)
}
