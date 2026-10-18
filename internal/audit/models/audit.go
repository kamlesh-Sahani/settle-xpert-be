package auditmodels

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Standard audit actions (strictly CREATE, UPDATE, or DELETE)
const (
	ActionCreate = "CREATE"
	ActionUpdate = "UPDATE"
	ActionDelete = "DELETE"
)

// AuditLog represents the audit document schema in MongoDB.
type AuditLog struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	TenantID    *primitive.ObjectID `bson:"tenant_id,omitempty" json:"tenantId,omitempty"`
	UserID      *primitive.ObjectID `bson:"user_id,omitempty" json:"userId,omitempty"`
	EntityID    string              `bson:"entity_id,omitempty" json:"entityId,omitempty"`
	EntityType  string              `bson:"entity_type,omitempty" json:"entityType,omitempty"`
	ActionBy    string              `bson:"action_by" json:"actionBy"`
	ActionOn    time.Time           `bson:"action_on" json:"actionOn"`
	UserAgent   string              `bson:"user_agent,omitempty" json:"userAgent,omitempty"`
	Action      string              `bson:"action" json:"action"` // "CREATE", "UPDATE", or "DELETE"
	NoOfChanges int                 `bson:"no_of_changes" json:"noOfChanges"`
	OldData     any                 `bson:"old_data,omitempty" json:"oldData,omitempty"`
	NewData     any                 `bson:"new_data,omitempty" json:"newData,omitempty"`
}
