package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	AgreementTemplateCollection = "agreement_templates"
	DefaultAgreementTemplateKey = "default"
)

type AgreementTemplate struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Key       string              `bson:"key" json:"key"`
	Body      string              `bson:"body" json:"body"`
	Footer    string              `bson:"footer" json:"footer"`
	UpdatedBy *primitive.ObjectID `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	CreatedAt time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time           `bson:"updatedAt" json:"updatedAt"`
}
