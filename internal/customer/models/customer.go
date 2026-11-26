package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

type CustomerStatus string

const (
	CustomerStatusActive   CustomerStatus = "ACTIVE"
	CustomerStatusInactive CustomerStatus = "INACTIVE"
)

type CaseStatus string

const (
	CaseStatusOnboarding       CaseStatus = "ONBOARDING"
	CaseStatusDocsVerification CaseStatus = "DOCS_VERIFICATION"
	CaseStatusBankNegotiation  CaseStatus = "BANK_NEGOTIATION"
	CaseStatusPartiallySettled CaseStatus = "PARTIALLY_SETTLED"
	CaseStatusResolved         CaseStatus = "RESOLVED"
)

type LoanSettlementItem struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Lender             string             `bson:"lender" json:"lender"`
	LoanType           string             `bson:"loan_type" json:"loanType"`
	AccountNumber      string             `bson:"account_number,omitempty" json:"accountNumber,omitempty"`
	OriginalAmount     float64            `bson:"original_amount" json:"originalAmount"`
	OutstandingAmount  float64            `bson:"outstanding_amount" json:"outstandingAmount"`
	SettlementOffered  float64            `bson:"settlement_offered,omitempty" json:"settlementOffered,omitempty"`
	SettlementApproved float64            `bson:"settlement_approved,omitempty" json:"settlementApproved,omitempty"`
	SavingsAmount      float64            `bson:"savings_amount,omitempty" json:"savingsAmount,omitempty"`
	SavingsPercent     float64            `bson:"savings_percent,omitempty" json:"savingsPercent,omitempty"`
	Status             string             `bson:"status" json:"status"` // PENDING, IN_NEGOTIATION, OTS_APPROVED, SETTLED, CLOSED
	Remarks            string             `bson:"remarks,omitempty" json:"remarks,omitempty"`
	UpdatedAt          *time.Time         `bson:"updated_at,omitempty" json:"updatedAt,omitempty"`
}

type Customer struct {
	ID                  primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	CustomerNumber      string               `bson:"customer_number" json:"customerNumber"`
	CaseNumber          string               `bson:"case_number,omitempty" json:"caseNumber,omitempty"`
	CaseStatus          CaseStatus           `bson:"case_status,omitempty" json:"caseStatus,omitempty"`
	LeadID              *primitive.ObjectID  `bson:"lead_id,omitempty" json:"leadId,omitempty"`
	LeadNumber          string               `bson:"lead_number,omitempty" json:"leadNumber,omitempty"`
	FullName            string               `bson:"full_name" json:"fullName"`
	Email               string               `bson:"email,omitempty" json:"email,omitempty"`
	Phone               string               `bson:"phone" json:"phone"`
	AlternatePhone      string               `bson:"alternate_phone,omitempty" json:"alternatePhone,omitempty"`
	DOB                 string               `bson:"dob,omitempty" json:"dob,omitempty"`
	PANCard             string               `bson:"pan_card,omitempty" json:"panCard,omitempty"`
	Address             *Address             `bson:"address,omitempty" json:"address,omitempty"`
	City                string               `bson:"city,omitempty" json:"city,omitempty"`
	State               string               `bson:"state,omitempty" json:"state,omitempty"`
	Status              CustomerStatus       `bson:"status" json:"status"`
	TotalDebtAmount     float64              `bson:"total_debt_amount,omitempty" json:"totalDebtAmount,omitempty"`
	OutstandingAmount   float64              `bson:"outstanding_amount,omitempty" json:"outstandingAmount,omitempty"`
	FinancialAssessment *FinancialAssessment `bson:"financial_assessment,omitempty" json:"financialAssessment,omitempty"`
	Debts               []DebtItem           `bson:"debts,omitempty" json:"debts,omitempty"`
	Loans               []LoanSettlementItem `bson:"loans,omitempty" json:"loans,omitempty"`
	AssignedTo          *primitive.ObjectID  `bson:"assigned_to,omitempty" json:"assignedTo,omitempty"`
	AssignedToName      string               `bson:"assigned_to_name,omitempty" json:"assignedToName,omitempty"`
	AssignedTeam        string               `bson:"assigned_team,omitempty" json:"assignedTeam,omitempty"`
	ConvertedBy         *primitive.ObjectID  `bson:"converted_by,omitempty" json:"convertedBy,omitempty"`
	ConvertedByName     string               `bson:"converted_by_name,omitempty" json:"convertedByName,omitempty"`
	ConvertedAt         time.Time            `bson:"converted_at" json:"convertedAt"`
	CreatedAt           time.Time            `bson:"created_at" json:"createdAt"`
	UpdatedAt           time.Time            `bson:"updated_at" json:"updatedAt"`
	DeletedAt           *time.Time           `bson:"deleted_at,omitempty" json:"deletedAt,omitempty"`
}
