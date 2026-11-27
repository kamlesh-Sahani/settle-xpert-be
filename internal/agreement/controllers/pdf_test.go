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
			FullAddress: "17/A, Ram Ratan Bose Lane, Kolkata, West Bengal - 700004",
		},
		Lenders: []leadmodels.Lender{
			{LenderName: "HDFC Bank", LoanType: "Personal Loan", OutstandingAmount: 150000},
			{LenderName: "ICICI Bank", LoanType: "Credit Card", OutstandingAmount: 80000},
		},
		Agreement: AgreementDetailsInput{
			Duration:           "6",
			ResolutionDuration: "8",
			ConsultingFees:     7500,
			StartDate:          &execDate,
			ExecutedDate:       &execDate,
			MonthlyIncome:      45000,
		},
	}

	htmlBody := `<h2 style="text-align: center"><u>CONSULTANCY AGREEMENT</u></h2><p>Executed on <strong>{{executedDate}}</strong> for <em>{{fullName}}</em>.</p><ul><li>First bullet</li><li>Second <strong>bold</strong> bullet</li></ul><ol><li>Numbered item</li></ol><hr><blockquote>Important quote</blockquote><p>{{signatureBlock}}</p><p>{{pageBreak}}</p><h2 style="text-align: center"><u>Annexure A</u></h2><p>{{clientInfoTable}}</p><p>{{lendersTable}}</p><p>{{pageBreak}}</p><h2 style="text-align: center"><u>Annexure B</u></h2><p>{{feeScheduleTable}}</p>`
	htmlFooter := `<p style="text-align: center">Email: info@settlexpert.com | Phone: +91 89292 23949 | Website: www.settlexpert.com  •  Page {{page}} of {{totalPages}}</p>`

	pdfBytes, err := buildAgreementPDFBytes(htmlBody, htmlFooter, req, "Dhruv Consultant")
	if err != nil {
		t.Fatalf("buildAgreementPDFBytes failed: %v", err)
	}
	if len(pdfBytes) < 1000 || string(pdfBytes[:5]) != "%PDF-" {
		t.Fatalf("unexpected PDF output: len=%d", len(pdfBytes))
	}
}
