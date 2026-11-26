package controllers

import (
	"testing"
	"time"

	leadmodels "lead-crm/backend/internal/lead/models"
)

func TestBuildAgreementPDFBytes(t *testing.T) {
	execDate := time.Date(2026, 11, 26, 10, 0, 0, 0, time.Local)
	req := CreateAgreementRequest{
		LeadID: "65f1a2b3c4d5e6f7a8b9c0d1",
		Personal: AgreementPersonalInput{
			FullName:    "SANCHITA CHATTERJEE",
			PANNumber:   "BTWPC6838E",
			DOB:         "1990-01-01",
			Phone:       "+91 70441 18629",
			Email:       "acsanchita@gmail.com",
			FullAddress: "C/O Arka Chatterjee, 17/A, Ram Ratan Bose Lane, Deshbondhu Park, Shyambazar, Kolkata, West Bengal - 700004",
		},
		Lenders: []leadmodels.Lender{
			{LenderName: "HDFC Bank", LoanType: "Personal Loan", OutstandingAmount: 150000},
			{LenderName: "ICICI Bank", LoanType: "Credit Card", OutstandingAmount: 80000},
		},
		Agreement: AgreementDetailsInput{
			Duration:       "6",
			ConsultingFees: 7500,
			StartDate:      &execDate,
			ExecutedDate:   &execDate,
			MonthlyIncome:  45000,
		},
	}

	sampleTemplate := `SETTLEXPERT LLP
CB-201, Naraina Vihar, Ring Road, New Delhi, Delhi, India
================================================================================

CONSULTANCY AGREEMENT

This Consultancy Agreement ("Agreement") is executed on {{executedDate}} between:

M/s. SettleXpert LLP, having its registered/operations office at CB-201, Naraina Vihar, Ring Road, New Delhi, Delhi, India, hereinafter referred to as the "First Party" or the "Company", which expression shall, unless repugnant to the context, include its successors, affiliates and permitted assigns.

AND

{{fullName}}, residing at {{fullAddress}}, Date of Birth: {{dob}}, Mobile No.: {{phone}}, Email ID: {{email}}, hereinafter referred to as the "Second Party" or the "Client".

The Company and the Client are hereinafter collectively referred to as the "Parties" and individually as a "Party".

WHEREAS, the Company is engaged in providing financial consultancy and debt resolution advisory services to individuals experiencing financial hardship.

NOW, THEREFORE, in consideration of the mutual promises and covenants contained herein, the Parties agree as follows:

NOW THIS AGREEMENT WITNESSETH AS FOLLOWS:

1. PURPOSE OF THE AGREEMENT
The purpose of this Agreement is to appoint the Company as the Client's financial consultancy partner for providing advisory, negotiation support and debt resolution consultancy in relation to the debt accounts specifically mentioned in Annexure A.

2. SCOPE OF CONSULTANCY SERVICES
Subject to the terms of this Agreement, the Company shall provide consultancy services including, but not limited to:
a. Reviewing and assessing the Client's financial position and outstanding unsecured liabilities.
b. Advising the Client regarding possible debt resolution options based on the financial information shared by the Client.

9. TERMINATION
The Company may suspend or terminate this Agreement if the Client:
- Fails to pay the agreed consultancy fees within the stipulated time;
- Provides false, misleading or incomplete information; or
- Fails to cooperate or repeatedly breaches the terms of this Agreement.

13. GRIEVANCE REDRESSAL & DISPUTE RESOLUTION
In the event of any grievance, the Client shall first communicate the matter by sending a written email to:
Grievance & Escalation Email:
info@settlexpert.com

14. DECLARATION & ACCEPTANCE
This Agreement may be executed physically or electronically and shall be deemed valid and legally binding.

FIRST PARTY
NAME: SettleXpert LLP`

	pdfBytes, err := buildAgreementPDFBytes(sampleTemplate, req, "Dhruv Consultant")
	if err != nil {
		t.Fatalf("buildAgreementPDFBytes failed: %v", err)
	}
	if len(pdfBytes) < 1000 || string(pdfBytes[:5]) != "%PDF-" {
		t.Fatalf("unexpected PDF output: len=%d", len(pdfBytes))
	}
}
