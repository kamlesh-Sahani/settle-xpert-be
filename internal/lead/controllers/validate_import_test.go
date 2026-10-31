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
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setupValidateTestRouter(t *testing.T) (*gin.Engine, func()) {
	gin.SetMode(gin.TestMode)

	client, err := database.ConnectMongoDB("mongodb://localhost:27017/debt_crm_test")
	if err != nil {
		t.Skipf("Skipping test: MongoDB not available: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("employee_context", &utils.EmployeeContext{
			EmployeeID: "670000000000000000000001",
			Name:       "Counselor Sarah",
			Email:      "sarah@settlexpert.com",
		})
		c.Next()
	})

	r.POST("/api/v1/lead/validate-import", ValidateImport)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Database("debt_crm_test").Collection("leads").Drop(ctx)
	}

	return r, cleanup
}

func TestValidateImport_SuccessAndCategorization(t *testing.T) {
	r, cleanup := setupValidateTestRouter(t)
	defer cleanup()

	col := database.GetCollection("leads")
	ctx := context.Background()

	// Seed one existing lead with phone 9876543210
	_, err := col.InsertOne(ctx, bson.M{
		"_id":        primitive.NewObjectID(),
		"phone":      "9876543210",
		"full_name":  "Existing Borrower",
		"created_at": time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to seed existing lead: %v", err)
	}

	payload := ValidateImportRequest{
		DefaultSource: "EXCEL_IMPORT",
		Leads: []BulkLeadItem{
			{
				FullName:        "Rahul Sharma",
				Phone:           "9876543210", // DUPLICATE_CRM
				TotalDebtAmount: 500000,
			},
			{
				FullName:        "Pooja Verma",
				Phone:           "+91 9988776655", // VALID
				TotalDebtAmount: 350000,
			},
			{
				FullName:        "Pooja Verma Duplicate",
				Phone:           "9988776655", // DUPLICATE_FILE
				TotalDebtAmount: 350000,
			},
			{
				FullName:        "", // INVALID (missing name)
				Phone:           "9123456780",
				TotalDebtAmount: 750000,
			},
			{
				FullName:        "Invalid Phone Person",
				Phone:           "12345", // INVALID (short phone)
				TotalDebtAmount: 100000,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead/validate-import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    ValidateImportResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Data.TotalRows != 5 {
		t.Errorf("expected 5 total rows, got %d", resp.Data.TotalRows)
	}
	if resp.Data.ValidCount != 1 {
		t.Errorf("expected 1 valid lead, got %d", resp.Data.ValidCount)
	}
	if resp.Data.InvalidCount != 2 {
		t.Errorf("expected 2 invalid leads, got %d", resp.Data.InvalidCount)
	}
	if resp.Data.DuplicateCount != 2 {
		t.Errorf("expected 2 duplicate leads, got %d", resp.Data.DuplicateCount)
	}
}
