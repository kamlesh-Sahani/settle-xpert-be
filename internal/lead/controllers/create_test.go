package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setupTestRouter(t *testing.T) (*gin.Engine, func()) {
	gin.SetMode(gin.TestMode)

	// Connect to test database
	client, err := database.ConnectMongoDB("mongodb://localhost:27017/debt_crm_test")
	if err != nil {
		t.Skipf("Skipping test: MongoDB not available: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Mock employee auth context
	r.Use(func(c *gin.Context) {
		c.Set("employee_context", &utils.EmployeeContext{
			EmployeeID: "670000000000000000000001",
			Name:       "Counselor Sarah",
			Email:      "sarah@settlexpert.com",
		})
		c.Next()
	})

	r.POST("/api/v1/lead/create", Create)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Database("debt_crm_test").Collection("leads").Drop(ctx)
		_ = client.Database("debt_crm_test").Collection("employees").Drop(ctx)
	}

	return r, cleanup
}

// 1. Validation Failures
func TestCreateLead_Validation_MissingFields(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	testCases := []struct {
		name       string
		payload    map[string]any
		expectCode int
	}{
		{
			name:       "Empty body",
			payload:    map[string]any{},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Missing phone",
			payload: map[string]any{
				"fullName":        "Amit Sharma",
				"serviceRequired": "Credit Card Settlement",
				"source":          "Manual Entry",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Missing fullName",
			payload: map[string]any{
				"phone":           "9876543210",
				"serviceRequired": "Credit Card Settlement",
				"source":          "Manual Entry",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Missing serviceRequired",
			payload: map[string]any{
				"fullName": "Amit Sharma",
				"phone":    "9876543210",
				"source":   "Manual Entry",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Invalid Priority Enum",
			payload: map[string]any{
				"fullName":        "Amit Sharma",
				"phone":           "9876543210",
				"serviceRequired": "Credit Card Settlement",
				"source":          "Manual Entry",
				"priority":        "SUPER_URGENT",
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectCode {
				t.Fatalf("expected status %d, got %d: %s", tc.expectCode, w.Code, w.Body.String())
			}
		})
	}
}

// 2. Brand New Lead Creation (Success)
func TestCreateLead_Success_Basic(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	payload := map[string]any{
		"fullName":        "Rahul Verma",
		"phone":           "9811223344",
		"alternatePhone":  "9811223355",
		"email":           "rahul.verma@example.com",
		"dob":             "1989-05-15",
		"panCard":         "ABCDE1234F",
		"serviceRequired": "Personal Loan Settlement",
		"source":          "Website Form",
		"priority":        "HIGH",
		"initialEnquiry":  "Need urgent help with 2 personal loans",
		"address": map[string]any{
			"line1":   "Flat 402, Lotus Towers",
			"country": "IN",
			"state":   "Maharashtra",
			"city":    "Pune",
			"zip":     "411001",
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify in database
	col := database.GetCollection("leads")
	var lead models.Lead
	err := col.FindOne(context.Background(), bson.M{"phone": "9811223344"}).Decode(&lead)
	if err != nil {
		t.Fatalf("lead was not found in DB: %v", err)
	}

	if lead.LeadNumber == "" {
		t.Error("leadNumber was not generated")
	}
	if lead.FullName != "Rahul Verma" {
		t.Errorf("expected FullName 'Rahul Verma', got '%s'", lead.FullName)
	}
	if lead.Status != models.LeadStatusNew {
		t.Errorf("expected default Status NEW, got '%s'", lead.Status)
	}
	if lead.Priority != models.PriorityHigh {
		t.Errorf("expected Priority HIGH, got '%s'", lead.Priority)
	}
	if lead.Address == nil || lead.Address.City != "Pune" {
		t.Errorf("expected Address.City 'Pune', got '%+v'", lead.Address)
	}
	if len(lead.ActivityTimeline) == 0 || lead.ActivityTimeline[0].Type != "LEAD_CREATED" {
		t.Errorf("expected LEAD_CREATED activity timeline, got %+v", lead.ActivityTimeline)
	}
	if len(lead.Enquiries) == 0 || lead.Enquiries[0].EnquiryNumber != 1 {
		t.Errorf("expected 1 initial enquiry item, got %+v", lead.Enquiries)
	}
}

// 3. Lead Creation with Debts and Financial Assessment Calculation
func TestCreateLead_Success_WithDebts(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	payload := map[string]any{
		"fullName":        "Vikram Patel",
		"phone":           "9822334455",
		"serviceRequired": "Credit Card Settlement",
		"source":          "Facebook Ads",
		"monthlyIncome":   75000,
		"currentEmiAmount": 38000,
		"debts": []map[string]any{
			{
				"lender":            "HDFC Bank",
				"loanType":          "Credit Card",
				"outstandingAmount": 150000,
				"monthlyEmi":        12000,
			},
			{
				"lender":            "ICICI Bank",
				"loanType":          "Personal Loan",
				"outstandingAmount": 250000,
				"monthlyEmi":        26000,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	col := database.GetCollection("leads")
	var lead models.Lead
	_ = col.FindOne(context.Background(), bson.M{"phone": "9822334455"}).Decode(&lead)

	expectedTotalDebt := 400000.0 // 150k + 250k
	if lead.TotalDebtAmount != expectedTotalDebt {
		t.Errorf("expected TotalDebtAmount %.0f, got %.0f", expectedTotalDebt, lead.TotalDebtAmount)
	}
	if lead.OutstandingAmount != expectedTotalDebt {
		t.Errorf("expected OutstandingAmount %.0f, got %.0f", expectedTotalDebt, lead.OutstandingAmount)
	}
	if len(lead.Debts) != 2 {
		t.Fatalf("expected 2 debt items, got %d", len(lead.Debts))
	}
	if lead.FinancialAssessment == nil || lead.FinancialAssessment.TotalDebtAmount != expectedTotalDebt {
		t.Errorf("financial assessment debt amount mismatch: %+v", lead.FinancialAssessment)
	}
}

// 4. Lead Creation with Unified Conversations (Call and WhatsApp Transcript)
func TestCreateLead_Success_WithConversation(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	payload := map[string]any{
		"fullName":        "Pooja Mehta",
		"phone":           "9833445566",
		"serviceRequired": "Anti-Harassment Protection",
		"source":          "Direct Call",
		"conversation": map[string]any{
			"callStartTime":   "2026-10-27T10:00:00Z",
			"callEndTime":     "2026-10-27T10:15:00Z",
			"callDuration":    "15 mins",
			"discussionNotes": "Customer is receiving harassment calls from recovery agents. Advised anti-harassment shield.",
			"rawChat":         "27/10/26, 10:16 AM - Customer: Please help stop recovery calls",
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	col := database.GetCollection("leads")
	var lead models.Lead
	_ = col.FindOne(context.Background(), bson.M{"phone": "9833445566"}).Decode(&lead)

	if len(lead.Conversations) != 1 {
		t.Fatalf("expected 1 unified conversation item, got %d", len(lead.Conversations))
	}
	conv := lead.Conversations[0]
	if conv.Duration != "15 mins" {
		t.Errorf("expected conversation duration '15 mins', got '%s'", conv.Duration)
	}
	if conv.DiscussionNotes == "" {
		t.Errorf("expected discussion notes to be present")
	}
	if conv.WhatsAppChat == "" {
		t.Errorf("expected whatsapp chat to be present")
	}
	if conv.UploadedBy != "Counselor Sarah" {
		t.Errorf("expected UploadedBy 'Counselor Sarah', got '%s'", conv.UploadedBy)
	}

	// Verify timeline recorded conversation
	hasCallActivity := false
	for _, act := range lead.ActivityTimeline {
		if act.Type == "CALL_LOGGED" {
			hasCallActivity = true
			break
		}
	}
	if !hasCallActivity {
		t.Errorf("expected CALL_LOGGED in activity timeline: %+v", lead.ActivityTimeline)
	}
}

// 5. Lead Creation with Assigned Employee
func TestCreateLead_Success_WithAssignment(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	// Seed employee in DB
	empCol := database.GetCollection("employees")
	empID := primitive.NewObjectID()
	_, _ = empCol.InsertOne(context.Background(), bson.M{
		"_id":  empID,
		"name": "Rajesh Kumar",
		"email": "rajesh@settlexpert.com",
	})

	payload := map[string]any{
		"fullName":        "Kavita Nair",
		"phone":           "9844556677",
		"serviceRequired": "Credit Card Settlement",
		"source":          "Manual Entry",
		"assignedTo":      empID.Hex(),
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	col := database.GetCollection("leads")
	var lead models.Lead
	_ = col.FindOne(context.Background(), bson.M{"phone": "9844556677"}).Decode(&lead)

	if lead.AssignedTo == nil || *lead.AssignedTo != empID {
		t.Errorf("expected assignedTo %s, got %+v", empID.Hex(), lead.AssignedTo)
	}
	if lead.AssignedToName != "Rajesh Kumar" {
		t.Errorf("expected assignedToName 'Rajesh Kumar', got '%s'", lead.AssignedToName)
	}
	if len(lead.AssignmentHistory) != 1 {
		t.Errorf("expected 1 assignment history item, got %d", len(lead.AssignmentHistory))
	}
}

// 6. Deduplication - Basic Returning Lead Re-engagement
func TestCreateLead_Deduplication_Basic(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	col := database.GetCollection("leads")
	now := time.Now().Add(-24 * time.Hour)
	existingLead := models.Lead{
		ID:                 primitive.NewObjectID(),
		LeadNumber:         "LEAD-0010",
		FullName:           "Suresh Raina",
		Phone:              "9855667788",
		Status:             models.LeadStatusContacted,
		RepeatEnquiryCount: 0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, _ = col.InsertOne(context.Background(), existingLead)

	// Submit duplicate inquiry with +91 country code formatting
	payload := map[string]any{
		"fullName":        "Suresh Raina",
		"phone":           "+91-9855667788",
		"serviceRequired": "Personal Loan Settlement",
		"source":          "Website Form",
		"initialEnquiry":  "Following up again on my previous request",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify duplicate response payload
	var resp struct {
		Data struct {
			IsDuplicate bool   `json:"isDuplicate"`
			LeadNumber  string `json:"leadNumber"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.Data.IsDuplicate {
		t.Errorf("expected isDuplicate true, got false")
	}
	if resp.Data.LeadNumber != "LEAD-0010" {
		t.Errorf("expected LeadNumber 'LEAD-0010', got '%s'", resp.Data.LeadNumber)
	}

	// Verify updated lead document
	var updated models.Lead
	_ = col.FindOne(context.Background(), bson.M{"_id": existingLead.ID}).Decode(&updated)

	if updated.RepeatEnquiryCount != 1 {
		t.Errorf("expected RepeatEnquiryCount 1, got %d", updated.RepeatEnquiryCount)
	}
	if !updated.IsReengaged {
		t.Errorf("expected IsReengaged true")
	}
	if len(updated.Enquiries) != 1 {
		t.Errorf("expected new inquiry item appended to enquiries: %+v", updated.Enquiries)
	}
}

// 7. Deduplication - Option A: QUALIFIED Lead Adding Another Loan
func TestCreateLead_Deduplication_OptionA_QualifiedLead(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	col := database.GetCollection("leads")
	now := time.Now().Add(-48 * time.Hour)
	existingLead := models.Lead{
		ID:                 primitive.NewObjectID(),
		LeadNumber:         "LEAD-0020",
		FullName:           "Deepak Chahar",
		Phone:              "9866778899",
		Status:             models.LeadStatusQualified,
		Priority:           models.PriorityNormal,
		TotalDebtAmount:    200000,
		OutstandingAmount:  200000,
		Debts: []models.DebtItem{
			{
				ID:                primitive.NewObjectID(),
				Lender:            "HDFC Bank",
				LoanType:          "Credit Card",
				OutstandingAmount: 200000,
				MonthlyEMI:        15000,
			},
		},
		RepeatEnquiryCount: 0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, _ = col.InsertOne(context.Background(), existingLead)

	// Lead fills out another loan for Axis Bank
	payload := map[string]any{
		"fullName":        "Deepak Chahar",
		"phone":           "9866778899",
		"serviceRequired": "Personal Loan Settlement",
		"source":          "Website Form",
		"debts": []map[string]any{
			{
				"lender":            "Axis Bank",
				"loanType":          "Personal Loan",
				"outstandingAmount": 300000,
				"monthlyEmi":        25000,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated models.Lead
	_ = col.FindOne(context.Background(), bson.M{"_id": existingLead.ID}).Decode(&updated)

	// Option A asserts:
	// 1. Status MUST remain QUALIFIED
	if updated.Status != models.LeadStatusQualified {
		t.Errorf("Option A assertion failed: expected Status QUALIFIED, got '%s'", updated.Status)
	}
	// 2. Priority MUST escalate to HIGH
	if updated.Priority != models.PriorityHigh {
		t.Errorf("Option A assertion failed: expected Priority HIGH, got '%s'", updated.Priority)
	}
	// 3. Debts MUST be merged (200k + 300k = 500k total across 2 lenders)
	if len(updated.Debts) != 2 {
		t.Errorf("Option A assertion failed: expected 2 merged debts, got %d", len(updated.Debts))
	}
	if updated.TotalDebtAmount != 500000 {
		t.Errorf("Option A assertion failed: expected TotalDebtAmount 500000, got %.0f", updated.TotalDebtAmount)
	}
	if updated.OutstandingAmount != 500000 {
		t.Errorf("Option A assertion failed: expected OutstandingAmount 500000, got %.0f", updated.OutstandingAmount)
	}
}

// 8. Deduplication - Reactivating LOST / UNREACHABLE Leads
func TestCreateLead_Deduplication_ReactivateLost(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	col := database.GetCollection("leads")
	now := time.Now().Add(-100 * time.Hour)
	existingLead := models.Lead{
		ID:                 primitive.NewObjectID(),
		LeadNumber:         "LEAD-0030",
		FullName:           "Sunil Narine",
		Phone:              "9877889900",
		Status:             models.LeadStatusLost,
		Priority:           models.PriorityLow,
		RepeatEnquiryCount: 0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, _ = col.InsertOne(context.Background(), existingLead)

	payload := map[string]any{
		"fullName":        "Sunil Narine",
		"phone":           "9877889900",
		"serviceRequired": "Credit Card Settlement",
		"source":          "WhatsApp",
		"initialEnquiry":  "I am ready to settle now, please contact me.",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated models.Lead
	_ = col.FindOne(context.Background(), bson.M{"_id": existingLead.ID}).Decode(&updated)

	if updated.Status != models.LeadStatusReengaged {
		t.Errorf("expected LOST lead to transition to RE_ENGAGED, got '%s'", updated.Status)
	}
	if updated.Priority != models.PriorityHigh {
		t.Errorf("expected Priority HIGH, got '%s'", updated.Priority)
	}
}

// 9. Short Phone Protection (No False Duplicate Match)
func TestCreateLead_ShortPhone_NoFalseDuplicateMatch(t *testing.T) {
	r, cleanup := setupTestRouter(t)
	defer cleanup()

	col := database.GetCollection("leads")
	now := time.Now()
	// Pre-seed an existing lead with a normal phone ending in 123
	_, _ = col.InsertOne(context.Background(), models.Lead{
		ID:         primitive.NewObjectID(),
		LeadNumber: "LEAD-0040",
		FullName:   "Existing Borrower",
		Phone:      "9811111123",
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// Submitting a lead with short phone digits (e.g. 12345) should NOT match LEAD-0040
	payload := map[string]any{
		"fullName":        "New Person",
		"phone":           "12345",
		"serviceRequired": "Personal Loan Settlement",
		"source":          "Manual Entry",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should create a NEW lead, NOT re-engage LEAD-0040
	count, _ := col.CountDocuments(context.Background(), bson.M{})
	if count != 2 {
		t.Errorf("expected 2 separate leads in DB, got %d", count)
	}
}
