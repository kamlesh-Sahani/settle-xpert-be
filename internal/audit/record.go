package audit

import (
	"context"
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	auditmodels "lead-crm/backend/internal/audit/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RecordParams encapsulates all inputs required to log an audit event.
type RecordParams struct {
	EntityID   string
	EntityType string
	Action     string // strictly "CREATE", "UPDATE", or "DELETE"
	OldData    any
	NewData    any
}

// Record extracts user and tenant context and asynchronously stores an audit record.
func Record(c *gin.Context, params RecordParams) {
	if c == nil {
		return
	}

	// 1. Extract Tenant ID (if exists)
	var tenantObjIDPtr *primitive.ObjectID
	if tVal, exists := c.Get("tenant_obj_id"); exists {
		if tid, ok := tVal.(primitive.ObjectID); ok && !tid.IsZero() {
			tenantObjIDPtr = &tid
		}
	}
	if tenantObjIDPtr == nil {
		if tStr := c.GetString("tenant_id"); tStr != "" {
			if tid, err := primitive.ObjectIDFromHex(tStr); err == nil && !tid.IsZero() {
				tenantObjIDPtr = &tid
			}
		}
	}

	// 2. Extract User ID
	var userObjIDPtr *primitive.ObjectID
	if uVal, exists := c.Get("user_obj_id"); exists {
		if uid, ok := uVal.(primitive.ObjectID); ok && !uid.IsZero() {
			userObjIDPtr = &uid
		}
	}
	if userObjIDPtr == nil {
		if uStr := c.GetString("user_id"); uStr != "" {
			if uid, err := primitive.ObjectIDFromHex(uStr); err == nil && !uid.IsZero() {
				userObjIDPtr = &uid
			}
		}
	}

	// 3. Extract Action By (User Name)
	actionBy := c.GetString("name")
	if actionBy == "" {
		if email := c.GetString("email"); email != "" {
			actionBy = email
		} else {
			actionBy = "System"
		}
	}

	// 4. Action normalization: strictly "CREATE", "UPDATE", or "DELETE"
	action := strings.ToUpper(strings.TrimSpace(params.Action))
	if action != auditmodels.ActionCreate && action != auditmodels.ActionDelete {
		action = auditmodels.ActionUpdate
	}

	// 5. Compute number of field changes
	_, noOfChanges := ComputeDiff(params.OldData, params.NewData, action)

	// 6. Construct strongly-typed audit log
	entry := auditmodels.AuditLog{
		TenantID:    tenantObjIDPtr,
		UserID:      userObjIDPtr,
		EntityID:    params.EntityID,
		EntityType:  params.EntityType,
		ActionBy:    actionBy,
		ActionOn:    time.Now(),
		UserAgent:   c.Request.UserAgent(),
		Action:      action,
		NoOfChanges: noOfChanges,
		OldData:     params.OldData,
		NewData:     params.NewData,
	}

	// 7. Asynchronously persist to MongoDB without blocking HTTP response
	go func(logEntry auditmodels.AuditLog) {
		col := database.GetCollection("audit_logs")
		if col == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := col.InsertOne(ctx, logEntry); err != nil {
			log.Printf("[AuditLog] ⚠️ Failed to insert audit log (%s on %s): %v",
				logEntry.Action, logEntry.EntityID, err)
		}
	}(entry)
}
