package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DepartmentStatus string

const (
	DepartmentStatusActive   DepartmentStatus = "ACTIVE"
	DepartmentStatusInactive DepartmentStatus = "IN_ACTIVE"
)

type Department struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Code        string             `bson:"code" json:"code"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Status      DepartmentStatus   `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updatedAt"`
	DeletedAt   *time.Time         `bson:"deleted_at,omitempty" json:"deletedAt,omitempty"`
}
