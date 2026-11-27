package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadStatus string

const (
	LeadStatusNew           LeadStatus = "NEW"
	LeadStatusCallScheduled LeadStatus = "CALL_SCHEDULED"
	LeadStatusInterested    LeadStatus = "INTERESTED"
	LeadStatusNotInterested LeadStatus = "NOT_INTERESTED"
	LeadStatusConverted     LeadStatus = "CONVERTED"
)

type CallScheduleStatus string

const (
	CallScheduleStatusPending   CallScheduleStatus = "PENDING"
	CallScheduleStatusCompleted CallScheduleStatus = "COMPLETED"
)

type CallStatus string

const (
	CallStatusConnected            CallStatus = "CONNECTED"
	CallStatusBusy                 CallStatus = "BUSY"
	CallStatusRinging              CallStatus = "RINGING"
	CallStatusSwitchedOff          CallStatus = "SWITCHED_OFF"
	CallStatusDisconnectedByClient CallStatus = "DISCONNECTED_BY_CLIENT"
)

type CallPriority string

const (
	CallPriorityLow    CallPriority = "LOW"
	CallPriorityMedium CallPriority = "MEDIUM"
	CallPriorityHigh   CallPriority = "HIGH"
)

type EmploymentStatus string

const (
	EmploymentStatusEmployed   EmploymentStatus = "EMPLOYED"
	EmploymentStatusUnemployed EmploymentStatus = "UNEMPLOYED"
)

type HarassmentStatus string

const (
	HarassmentStatusYes HarassmentStatus = "YES"
	HarassmentStatusNo  HarassmentStatus = "NO"
)

type CallSchedule struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Date        time.Time          `bson:"date" json:"date"`
	Status      CallScheduleStatus `bson:"status" json:"status"`
	Priority    CallPriority       `bson:"priority,omitempty" json:"priority,omitempty"`
	CallStatus  CallStatus         `bson:"call_status,omitempty" json:"callStatus,omitempty"`
	Remark      string             `bson:"remark,omitempty" json:"remark,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"createdAt"`
	CompletedAt *time.Time         `bson:"completed_at,omitempty" json:"completedAt,omitempty"`
}

type PersonalInfo struct {
	FullName         string           `bson:"full_name" json:"fullName" binding:"required,min=2,max=120"`
	Phone            string           `bson:"phone" json:"phone" binding:"required,max=20"`
	Email            string           `bson:"email,omitempty" json:"email,omitempty" binding:"omitempty,email,max=160"`
	DOB              string           `bson:"dob,omitempty" json:"dob,omitempty" binding:"omitempty,max=32"`
	PANNumber        string           `bson:"pan_number,omitempty" json:"panNumber,omitempty" binding:"omitempty,max=10"`
	MaritalStatus    string           `bson:"marital_status,omitempty" json:"maritalStatus,omitempty" binding:"omitempty,max=32"`
	FullAddress      string           `bson:"full_address,omitempty" json:"fullAddress,omitempty" binding:"omitempty,max=500"`
	City             string           `bson:"city,omitempty" json:"city,omitempty" binding:"omitempty,max=100"`
	State            string           `bson:"state,omitempty" json:"state,omitempty" binding:"omitempty,max=100"`
	EmploymentStatus EmploymentStatus `bson:"employment_status,omitempty" json:"employmentStatus,omitempty" binding:"omitempty,oneof=EMPLOYED UNEMPLOYED"`
	EmploymentType   string           `bson:"employment_type,omitempty" json:"employmentType,omitempty" binding:"omitempty,max=100"`
}

type Lender struct {
	LenderName        string  `bson:"lender_name" json:"lenderName" binding:"required,max=120"`
	LoanType          string  `bson:"loan_type" json:"loanType" binding:"required,max=80"`
	OutstandingAmount float64 `bson:"outstanding_amount" json:"outstandingAmount" binding:"gte=0"`
}

type FinancialInfo struct {
	TotalOutstandingAmount float64          `bson:"total_outstanding_amount,omitempty" json:"totalOutstandingAmount,omitempty" binding:"omitempty,gte=0"`
	CreditCardDue          float64          `bson:"credit_card_due,omitempty" json:"creditCardDue,omitempty" binding:"omitempty,gte=0"`
	PersonalDue            float64          `bson:"personal_due,omitempty" json:"personalDue,omitempty" binding:"omitempty,gte=0"`
	MonthlyIncome          float64          `bson:"monthly_income,omitempty" json:"monthlyIncome,omitempty" binding:"omitempty,gte=0"`
	MonthlyEMI             float64          `bson:"monthly_emi,omitempty" json:"monthlyEmi,omitempty" binding:"omitempty,gte=0"`
	CurrentlyPayingEMIs    string           `bson:"currently_paying_emis,omitempty" json:"currentlyPayingEmis,omitempty" binding:"omitempty,oneof=YES NO"`
	Harassment             HarassmentStatus `bson:"harassment,omitempty" json:"harassment,omitempty" binding:"omitempty,oneof=YES NO"`
	Lenders                []Lender         `bson:"lenders,omitempty" json:"lenders,omitempty" binding:"omitempty,max=50,dive"`
}

type AgreementInfo struct {
	Duration           string              `bson:"duration,omitempty" json:"duration,omitempty" binding:"omitempty,max=80"`
	ResolutionDuration string              `bson:"resolution_duration,omitempty" json:"resolutionDuration,omitempty" binding:"omitempty,max=80"`
	ConsultingFees     float64             `bson:"consulting_fees,omitempty" json:"consultingFees,omitempty" binding:"omitempty,gte=0"`
	StartDate      *time.Time          `bson:"start_date,omitempty" json:"startDate,omitempty"`
	ExecutedDate   *time.Time          `bson:"executed_date,omitempty" json:"executedDate,omitempty"`
	PreparedBy     *primitive.ObjectID `bson:"prepared_by,omitempty" json:"preparedBy,omitempty"`
	PreparedByName string              `bson:"prepared_by_name,omitempty" json:"preparedByName,omitempty"`
}

type Lead struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	LeadNumber string             `bson:"lead_number" json:"leadNumber"`
	Status     LeadStatus         `bson:"status" json:"status"`

	// Grouped Details
	Personal  PersonalInfo   `bson:"personal" json:"personal"`
	Financial *FinancialInfo `bson:"financial,omitempty" json:"financial,omitempty"`
	Agreement *AgreementInfo `bson:"agreement,omitempty" json:"agreement,omitempty"`

	// Qualification & Service
	ServiceType string `bson:"service_type,omitempty" json:"serviceType,omitempty"`

	// Ownership & Conversion
	AssignedTo     *primitive.ObjectID `bson:"assigned_to,omitempty" json:"assignedTo,omitempty"`
	AssignedToName string              `bson:"-" json:"assignedToName,omitempty"`
	ConvertedAt    *time.Time          `bson:"converted_at,omitempty" json:"convertedAt,omitempty"`

	// Call Scheduling
	CallSchedules []CallSchedule `bson:"call_schedules,omitempty" json:"callSchedules,omitempty"`

	// Lifecycle
	CreatedAt time.Time  `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deletedAt,omitempty"`
}
