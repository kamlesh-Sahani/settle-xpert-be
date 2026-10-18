package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Session struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"user_id" json:"user_id"`
	RefreshToken string             `bson:"refresh_token" json:"refresh_token"`
	UserAgent    string             `bson:"user_agent" json:"user_agent"`
	IPAddress    string             `bson:"ip_address" json:"ip_address"`
	ExpiresAt    time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}
