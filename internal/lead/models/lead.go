package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadStatus string

const (
	LeadStatusNew         LeadStatus = "NEW"
	LeadStatusContacted   LeadStatus = "CONTACTED"
	LeadStatusQualified   LeadStatus = "QUALIFIED"
	LeadStatusUnqualified LeadStatus = "UNQUALIFIED"
	LeadStatusFollowUp    LeadStatus = "FOLLOW_UP"
	LeadStatusReengaged   LeadStatus = "RE_ENGAGED"
	LeadStatusUnreachable LeadStatus = "UNREACHABLE"
	LeadStatusConverted   LeadStatus = "CONVERTED"
	LeadStatusLost        LeadStatus = "LOST"

	// Retained for backward-compatibility
	LeadStatusInProgress LeadStatus = "IN_PROGRESS"
)

type LeadPriority string

const (
	PriorityLow    LeadPriority = "LOW"
	PriorityNormal LeadPriority = "NORMAL"
	PriorityHigh   LeadPriority = "HIGH"
)

type LeadSource string

const (
	SourceFacebookAds  LeadSource = "Facebook Ads"
	SourceInstagramAds LeadSource = "Instagram Ads"
	SourceWebsite      LeadSource = "Website Form"
	SourceWhatsApp     LeadSource = "WhatsApp"
	SourcePhoneCall    LeadSource = "Phone Call / Direct"
	SourceReferral     LeadSource = "Referral"
	SourceManualEntry  LeadSource = "Manual Entry"
	SourceExcelImport  LeadSource = "Excel Import"
	SourceOther        LeadSource = "Other"
)

type MetaDetails struct {
	LeadgenID string `bson:"leadgen_id,omitempty" json:"leadgenId,omitempty"`
	FormID    string `bson:"form_id,omitempty" json:"formId,omitempty"`
	PageID    string `bson:"page_id,omitempty" json:"pageId,omitempty"`
	AdID      string `bson:"ad_id,omitempty" json:"adId,omitempty"`
}

type Address struct {
	Line1   string `bson:"line1,omitempty" json:"line1,omitempty"`
	Country string `bson:"country,omitempty" json:"country,omitempty"`
	State   string `bson:"state,omitempty" json:"state,omitempty"`
	City    string `bson:"city,omitempty" json:"city,omitempty"`
	Zip     string `bson:"zip,omitempty" json:"zip,omitempty"`
}

type DebtItem struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	LoanType          string             `bson:"loan_type" json:"loanType"`
	Lender            string             `bson:"lender" json:"lender"`
	OutstandingAmount float64            `bson:"outstanding_amount" json:"outstandingAmount"`
	MonthlyEMI        float64            `bson:"monthly_emi" json:"monthlyEmi"`
	Status            string             `bson:"status,omitempty" json:"status,omitempty"`
}

type FinancialAssessment struct {
	MonthlyIncome       float64    `bson:"monthly_income" json:"monthlyIncome"`
	TotalMonthlyEMI     float64    `bson:"total_monthly_emi,omitempty" json:"totalMonthlyEmi,omitempty"`
	OverdueDuration     string     `bson:"overdue_duration,omitempty" json:"overdueDuration,omitempty"`
	ReasonForDifficulty string     `bson:"reason_for_difficulty,omitempty" json:"reasonForDifficulty,omitempty"`
	DesiredOutcome      string     `bson:"desired_outcome,omitempty" json:"desiredOutcome,omitempty"`
	TotalDebtAmount     float64    `bson:"total_debt_amount,omitempty" json:"totalDebtAmount,omitempty"`
	OutstandingAmount   float64    `bson:"outstanding_amount,omitempty" json:"outstandingAmount,omitempty"`
	Debts               []DebtItem `bson:"debts,omitempty" json:"debts,omitempty"`
}

type ConversationItem struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CallStartTime   *time.Time         `bson:"call_start_time,omitempty" json:"callStartTime,omitempty"`
	CallEndTime     *time.Time         `bson:"call_end_time,omitempty" json:"callEndTime,omitempty"`
	Duration        string             `bson:"duration,omitempty" json:"duration,omitempty"`
	Outcome         string             `bson:"outcome,omitempty" json:"outcome,omitempty"`
	WhatsAppChat    string             `bson:"whatsapp_chat,omitempty" json:"whatsappChat,omitempty"`
	DiscussionNotes string             `bson:"discussion_notes,omitempty" json:"discussionNotes,omitempty"`
	UploadedBy      string             `bson:"uploaded_by,omitempty" json:"uploadedBy,omitempty"`
	UploadedAt      *time.Time         `bson:"uploaded_at,omitempty" json:"uploadedAt,omitempty"`
	CreatedAt       time.Time          `bson:"created_at" json:"createdAt"`
}

type FollowUp struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	DueDate        time.Time           `bson:"due_date" json:"dueDate"`
	AssignedTo     *primitive.ObjectID `bson:"assigned_to,omitempty" json:"assignedTo,omitempty"`
	AssignedToName string              `bson:"assigned_to_name,omitempty" json:"assignedToName,omitempty"`
	Status         string              `bson:"status" json:"status"` // PENDING, COMPLETED, CANCELLED
	Notes          string              `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt      time.Time           `bson:"created_at" json:"createdAt"`
	CompletedAt    *time.Time          `bson:"completed_at,omitempty" json:"completedAt,omitempty"`
}

type AssignmentLog struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	AssignedTo     *primitive.ObjectID `bson:"assigned_to,omitempty" json:"assignedTo,omitempty"`
	AssignedToName string              `bson:"assigned_to_name,omitempty" json:"assignedToName,omitempty"`
	AssignedBy     *primitive.ObjectID `bson:"assigned_by,omitempty" json:"assignedBy,omitempty"`
	AssignedByName string              `bson:"assigned_by_name,omitempty" json:"assignedByName,omitempty"`
	AssignedAt     time.Time           `bson:"assigned_at" json:"assignedAt"`
	Reason         string              `bson:"reason,omitempty" json:"reason,omitempty"`
}

type EnquiryItem struct {
	EnquiryNumber   int        `bson:"enquiry_number" json:"enquiryNumber"`
	Date            time.Time  `bson:"date" json:"date"`
	Source          LeadSource `bson:"source" json:"source"`
	ServiceRequired string     `bson:"service_required,omitempty" json:"serviceRequired,omitempty"`
	Message         string     `bson:"message,omitempty" json:"message,omitempty"`
}

type ActivityItem struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type            string             `bson:"type" json:"type"` // LEAD_CREATED, REPEAT_ENQUIRY, CALL_LOGGED, STATUS_CHANGED, ASSIGNED, FOLLOWUP_ADDED, FINANCIAL_UPDATED
	Description     string             `bson:"description" json:"description"`
	PerformedByName string             `bson:"performed_by_name" json:"performedByName"`
	CreatedAt       time.Time          `bson:"created_at" json:"createdAt"`
}

type Lead struct {
	ID                  primitive.ObjectID       `bson:"_id,omitempty" json:"id"`
	TenantID            *primitive.ObjectID      `bson:"tenant_id,omitempty" json:"tenantId,omitempty"`
	LeadNumber          string                   `bson:"lead_number" json:"leadNumber"`
	FullName            string                   `bson:"full_name" json:"fullName"`
	Email               string                   `bson:"email,omitempty" json:"email,omitempty"`
	Phone               string                   `bson:"phone" json:"phone"`
	AlternatePhone      string                   `bson:"alternate_phone,omitempty" json:"alternatePhone,omitempty"`
	DOB                 string                   `bson:"dob,omitempty" json:"dob,omitempty"`
	PANCard             string                   `bson:"pan_card,omitempty" json:"panCard,omitempty"`
	Address             *Address                 `bson:"address,omitempty" json:"address,omitempty"`
	City                string                   `bson:"city,omitempty" json:"city,omitempty"`
	State               string                   `bson:"state,omitempty" json:"state,omitempty"`
	LoanType            string                   `bson:"loan_type,omitempty" json:"loanType,omitempty"`
	ServiceRequired     string                   `bson:"service_required,omitempty" json:"serviceRequired,omitempty"`
	Source              LeadSource               `bson:"source" json:"source"`
	Priority            LeadPriority             `bson:"priority,omitempty" json:"priority,omitempty"`
	InitialEnquiry      string                   `bson:"initial_enquiry,omitempty" json:"initialEnquiry,omitempty"`
	QualificationNotes  string                   `bson:"qualification_notes,omitempty" json:"qualificationNotes,omitempty"`
	Status              LeadStatus               `bson:"status" json:"status"`
	IsReengaged         bool                     `bson:"is_reengaged,omitempty" json:"isReengaged,omitempty"`
	RepeatEnquiryCount  int                      `bson:"repeat_enquiry_count,omitempty" json:"repeatEnquiryCount,omitempty"`
	LastEnquiryAt       *time.Time               `bson:"last_enquiry_at,omitempty" json:"lastEnquiryAt,omitempty"`
	AssignedTo          *primitive.ObjectID      `bson:"assigned_to,omitempty" json:"assignedTo,omitempty"`
	AssignedToName      string                   `bson:"assigned_to_name,omitempty" json:"assignedToName,omitempty"`
	AssignedTeam        string                   `bson:"assigned_team,omitempty" json:"assignedTeam,omitempty"`
	AssignmentHistory   []AssignmentLog          `bson:"assignment_history,omitempty" json:"assignmentHistory,omitempty"`
	TotalDebtAmount     float64                  `bson:"total_debt_amount,omitempty" json:"totalDebtAmount,omitempty"`
	OutstandingAmount   float64                  `bson:"outstanding_amount,omitempty" json:"outstandingAmount,omitempty"`
	FinancialAssessment *FinancialAssessment     `bson:"financial_assessment,omitempty" json:"financialAssessment,omitempty"`
	Debts               []DebtItem               `bson:"debts,omitempty" json:"debts,omitempty"`
	Enquiries           []EnquiryItem            `bson:"enquiries,omitempty" json:"enquiries,omitempty"`
	Conversations       []ConversationItem       `bson:"conversations,omitempty" json:"conversations,omitempty"`
	FollowUps           []FollowUp               `bson:"follow_ups,omitempty" json:"followUps,omitempty"`
	ActivityTimeline    []ActivityItem           `bson:"activity_timeline,omitempty" json:"activityTimeline,omitempty"`
	LostReason          string                   `bson:"lost_reason,omitempty" json:"lostReason,omitempty"`
	ClosingNotes        string                   `bson:"closing_notes,omitempty" json:"closingNotes,omitempty"`
	ConvertedAt         *time.Time               `bson:"converted_at,omitempty" json:"convertedAt,omitempty"`
	CustomerID          *primitive.ObjectID      `bson:"customer_id,omitempty" json:"customerId,omitempty"`
	MetaDetails         *MetaDetails             `bson:"meta_details,omitempty" json:"metaDetails,omitempty"`
	RawFields           map[string]string        `bson:"raw_fields,omitempty" json:"rawFields,omitempty"`
	CreatedAt           time.Time                `bson:"created_at" json:"createdAt"`
	UpdatedAt           time.Time                `bson:"updated_at" json:"updatedAt"`
	DeletedAt           *time.Time               `bson:"deleted_at,omitempty" json:"deletedAt,omitempty"`
}
