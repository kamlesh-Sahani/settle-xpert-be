package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	RoleStatusActive   = "ACTIVE"
	RoleStatusInactive = "IN_ACTIVE"
)

type Role struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Status      string             `bson:"status" json:"status"`
	Permissions []string           `bson:"permissions" json:"permissions"`
	IsDeleted   bool               `bson:"is_deleted" json:"isDeleted"`
	CreatedAt   time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updatedAt"`
}
