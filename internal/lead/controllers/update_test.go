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

func TestUpdateLead(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client, err := database.ConnectMongoDB("mongodb://localhost:27017/debt_crm_test")
	if err != nil {
		t.Skipf("Skipping test: MongoDB not available: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Database("debt_crm_test").Collection("leads").Drop(ctx)
	}()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("employee_context", &utils.EmployeeContext{
			EmployeeID: "670000000000000000000001",
			Name:       "Test Counselor",
		})
		c.Next()
	})
	r.POST("/api/v1/lead/update", Update)

	col := database.GetCollection("leads")
	testID := primitive.NewObjectID()
	now := time.Now()
	testLead := models.Lead{
		ID:              testID,
		LeadNumber:      "LEAD-0100",
		FullName:        "Deepak Verma",
		Phone:           "9811223344",
		ServiceRequired: "Debt Settlement",
		Source:          models.SourceManualEntry,
		Status:          models.LeadStatusNew,
		TotalDebtAmount: 100000,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_, _ = col.InsertOne(context.Background(), testLead)

	t.Run("Missing ID in request", func(t *testing.T) {
		body := map[string]any{"fullName": "Updated Name"}
		jsonBytes, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/update", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Invalid Hex ID format", func(t *testing.T) {
		body := map[string]any{"id": "invalid-hex", "fullName": "Updated Name"}
		jsonBytes, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/update", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Non-existent Lead", func(t *testing.T) {
		nonExistentID := primitive.NewObjectID().Hex()
		body := map[string]any{"id": nonExistentID, "fullName": "Updated Name"}
		jsonBytes, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/update", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("Successful Lead Update", func(t *testing.T) {
		newName := "Deepak V. Updated"
		newPriority := "HIGH"
		newStatus := "QUALIFIED"
		body := map[string]any{
			"id":       testID.Hex(),
			"fullName": newName,
			"priority": newPriority,
			"status":   newStatus,
			"debts": []map[string]any{
				{
					"lender":            "HDFC Bank",
					"loanType":          "Personal Loan",
					"outstandingAmount": 150000,
					"monthlyEmi":        4500,
				},
			},
		}
		jsonBytes, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/update", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d, body: %s", w.Code, w.Body.String())
		}

		var updated models.Lead
		err := col.FindOne(context.Background(), bson.M{"_id": testID}).Decode(&updated)
		if err != nil {
			t.Fatalf("Failed to fetch updated lead: %v", err)
		}

		if updated.FullName != newName {
			t.Errorf("Expected FullName '%s', got '%s'", newName, updated.FullName)
		}
		if updated.Priority != models.PriorityHigh {
			t.Errorf("Expected Priority '%s', got '%s'", models.PriorityHigh, updated.Priority)
		}
		if updated.Status != models.LeadStatusQualified {
			t.Errorf("Expected Status '%s', got '%s'", models.LeadStatusQualified, updated.Status)
		}
		if updated.TotalDebtAmount != 150000 {
			t.Errorf("Expected TotalDebtAmount 150000, got %f", updated.TotalDebtAmount)
		}
		if len(updated.ActivityTimeline) == 0 {
			t.Errorf("Expected activity timeline to be populated")
		}
	})
}
