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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGetByID(t *testing.T) {
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
	r.GET("/api/v1/lead/get", GetByID)

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
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lead/get", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Invalid ID param", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lead/get?id=invalid-hex", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Valid ID - Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lead/get?id="+testID.Hex(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data models.Lead `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.FullName != "Alok Nath" {
			t.Errorf("expected FullName 'Alok Nath', got '%s'", resp.Data.FullName)
		}
		if resp.Data.LeadNumber != "LEAD-0099" {
			t.Errorf("expected LeadNumber 'LEAD-0099', got '%s'", resp.Data.LeadNumber)
		}
	})
}
