package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MetaConfigStatus string

const (
	MetaConfigStatusActive   MetaConfigStatus = "ACTIVE"
	MetaConfigStatusInactive MetaConfigStatus = "IN_ACTIVE"
)

type MetaIntegrationConfig struct {
	ID                 primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Status             MetaConfigStatus    `bson:"status" json:"status"`
	VerifyToken        string              `bson:"verify_token" json:"verifyToken"`
	PageAccessToken    string              `bson:"page_access_token" json:"pageAccessToken"`
	PageID             string              `bson:"page_id" json:"pageId"`
	PageName           string              `bson:"page_name" json:"pageName"`
	AppSecret          string              `bson:"app_secret" json:"appSecret"`
	FieldMapping       map[string]string   `bson:"field_mapping,omitempty" json:"fieldMapping,omitempty"`
	CreatedAt          time.Time           `bson:"created_at" json:"createdAt"`
	UpdatedAt          time.Time           `bson:"updated_at" json:"updatedAt"`
}

type MetaWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Time    int64  `json:"time"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				LeadgenID string `json:"leadgen_id"`
				PageID    string `json:"page_id"`
				FormID    string `json:"form_id"`
				AdID      string `json:"ad_id"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type MetaLeadDetails struct {
	ID          string `json:"id"`
	CreatedTime string `json:"created_time"`
	AdID        string `json:"ad_id"`
	FormID      string `json:"form_id"`
	FieldData   []struct {
		Name   string   `json:"name"`
		Values []string `json:"values"`
	} `json:"field_data"`
}

type MetaWebhookLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	LeadgenID    string             `bson:"leadgen_id" json:"leadgenId"`
	PageID       string             `bson:"page_id" json:"pageId"`
	FormID       string             `bson:"form_id" json:"formId"`
	LeadNumber   string             `bson:"lead_number,omitempty" json:"leadNumber,omitempty"`
	FullName     string             `bson:"full_name,omitempty" json:"fullName,omitempty"`
	Phone        string             `bson:"phone,omitempty" json:"phone,omitempty"`
	Email        string             `bson:"email,omitempty" json:"email,omitempty"`
	Status       string             `bson:"status" json:"status"` // SUCCESS, DUPLICATE, FAILED
	ErrorMessage string             `bson:"error_message,omitempty" json:"errorMessage,omitempty"`
	RawPayload   string             `bson:"raw_payload,omitempty" json:"rawPayload,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"createdAt"`
}
