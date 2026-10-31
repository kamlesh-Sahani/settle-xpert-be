package controllers

import (
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

func TestUpdateStatus(t *testing.T) {
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
	r.GET("/api/v1/lead/status", UpdateStatus)

	col := database.GetCollection("leads")
	testID := primitive.NewObjectID()
	now := time.Now()
	testLead := models.Lead{
		ID:         testID,
		LeadNumber: "LEAD-0099",
		FullName:   "Alok Nath",
		Phone:      "9899001122",
		Status:     models.LeadStatusNew,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, _ = col.InsertOne(context.Background(), testLead)

	t.Run("Missing ID param", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lead/status?status=CONTACTED", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Missing Status param", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lead/status?id="+testID.Hex(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Invalid Status param", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lead/status?id="+testID.Hex()+"&status=INVALID_STATUS", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Success Update Status", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lead/status?id="+testID.Hex()+"&status=QUALIFIED", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d (body: %s)", w.Code, w.Body.String())
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to parse json response: %v", err)
		}
		if res.Data.Status != "QUALIFIED" {
			t.Errorf("expected status QUALIFIED, got %s", res.Data.Status)
		}

		// Verify database state
		var updated models.Lead
		_ = col.FindOne(context.Background(), bson.M{"_id": testID}).Decode(&updated)
		if updated.Status != models.LeadStatusQualified {
			t.Errorf("expected db status QUALIFIED, got %s", updated.Status)
		}
		if len(updated.ActivityTimeline) == 0 {
			t.Errorf("expected activity timeline entry to be added")
		}
	})
}
