package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadStatus string

const (
	LeadStatusNew          LeadStatus = "NEW"
	LeadStatusContacted    LeadStatus = "CONTACTED"
	LeadStatusInReview     LeadStatus = "IN_REVIEW"
	LeadStatusQualified    LeadStatus = "QUALIFIED"
	LeadStatusDisqualified LeadStatus = "DISQUALIFIED"
	LeadStatusConverted    LeadStatus = "CONVERTED"
)

type LeadSource string

const (
	SourceMetaAds    LeadSource = "META_ADS"
	SourceWebsite    LeadSource = "WEBSITE"
	SourceGoogleAds  LeadSource = "GOOGLE_ADS"
	SourceReferral   LeadSource = "REFERRAL"
	SourceDirectCall LeadSource = "DIRECT_CALL"
	SourcePartner    LeadSource = "PARTNER"
)

type MetaDetails struct {
	LeadgenID string `bson:"leadgen_id,omitempty" json:"leadgenId,omitempty"`
	FormID    string `bson:"form_id,omitempty" json:"formId,omitempty"`
	PageID    string `bson:"page_id,omitempty" json:"pageId,omitempty"`
	AdID      string `bson:"ad_id,omitempty" json:"adId,omitempty"`
}

type Lead struct {
	ID                     primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	LeadNumber             string              `bson:"lead_number" json:"leadNumber"`
	FullName               string              `bson:"full_name" json:"fullName"`
	Email                  string              `bson:"email" json:"email"`
	Phone                  string              `bson:"phone" json:"phone"`
	AlternatePhone         string              `bson:"alternate_phone,omitempty" json:"alternatePhone,omitempty"`
	City                   string              `bson:"city,omitempty" json:"city,omitempty"`
	State                  string              `bson:"state,omitempty" json:"state,omitempty"`
	Source                 LeadSource          `bson:"source" json:"source"`
	LoanType               string              `bson:"loan_type,omitempty" json:"loanType,omitempty"`
	TotalDebtAmount        float64             `bson:"total_debt_amount,omitempty" json:"totalDebtAmount,omitempty"`
	MonthlyIncome          float64             `bson:"monthly_income,omitempty" json:"monthlyIncome,omitempty"`
	CurrentEMIAmount       float64             `bson:"current_emi_amount,omitempty" json:"currentEmiAmount,omitempty"`
	LenderNames            []string            `bson:"lender_names,omitempty" json:"lenderNames,omitempty"`
	Status                 LeadStatus          `bson:"status" json:"status"`
	QualificationNotes     string              `bson:"qualification_notes,omitempty" json:"qualificationNotes,omitempty"`
	DisqualificationReason string              `bson:"disqualification_reason,omitempty" json:"disqualificationReason,omitempty"`
	AssignedTo             *primitive.ObjectID `bson:"assigned_to,omitempty" json:"assignedTo,omitempty"`
	AssignedToName         string              `bson:"assigned_to_name,omitempty" json:"assignedToName,omitempty"`
	MetaDetails            *MetaDetails        `bson:"meta_details,omitempty" json:"metaDetails,omitempty"`
	RawFields              map[string]string   `bson:"raw_fields,omitempty" json:"rawFields,omitempty"`
	CreatedAt              time.Time           `bson:"created_at" json:"createdAt"`
	UpdatedAt              time.Time           `bson:"updated_at" json:"updatedAt"`
}
