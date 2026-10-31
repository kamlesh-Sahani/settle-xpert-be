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

func setupBulkTestRouter(t *testing.T) (*gin.Engine, func()) {
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

	r.POST("/api/v1/lead/bulk-create", BulkCreate)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Database("debt_crm_test").Collection("leads").Drop(ctx)
	}

	return r, cleanup
}

func TestBulkCreate_EmptyList(t *testing.T) {
	r, cleanup := setupBulkTestRouter(t)
	defer cleanup()

	payload := map[string]interface{}{
		"leads": []map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead/bulk-create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty list, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBulkCreate_SuccessAndDeduplication(t *testing.T) {
	r, cleanup := setupBulkTestRouter(t)
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

	payload := BulkCreateLeadsRequest{
		DefaultSource: "EXCEL_IMPORT",
		Leads: []BulkLeadItem{
			{
				FullName:        "Rahul Sharma",
				Phone:           "9876543210", // Should be skipped (already in DB)
				TotalDebtAmount: 500000,
			},
			{
				FullName:        "Pooja Verma",
				Phone:           "+91 9988776655", // Cleaned to 9988776655, Should succeed
				TotalDebtAmount: 350000,
			},
			{
				FullName:        "Pooja Verma Duplicate",
				Phone:           "9988776655", // Duplicate in same batch, Should be skipped
				TotalDebtAmount: 350000,
			},
			{
				FullName:        "Anand Patel",
				Phone:           "9123456780", // Should succeed
				TotalDebtAmount: 750000,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead/bulk-create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			TotalSubmitted int               `json:"totalSubmitted"`
			CreatedCount   int               `json:"createdCount"`
			SkippedCount   int               `json:"skippedCount"`
			SkippedLeads   []SkippedLeadInfo `json:"skippedLeads"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Data.CreatedCount != 2 {
		t.Errorf("expected 2 created leads, got %d", resp.Data.CreatedCount)
	}
	if resp.Data.SkippedCount != 2 {
		t.Errorf("expected 2 skipped leads, got %d", resp.Data.SkippedCount)
	}
}
