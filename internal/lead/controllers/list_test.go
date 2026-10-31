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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestLeadList_LeadNumberLinkType(t *testing.T) {
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
	r.POST("/api/v1/lead/list", List)

	col := database.GetCollection("leads")
	testID := primitive.NewObjectID()
	now := time.Now()
	testLead := models.Lead{
		ID:         testID,
		LeadNumber: "LEAD-1001",
		FullName:   "Manish Pandey",
		Phone:      "9800112233",
		Status:     models.LeadStatusNew,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, _ = col.InsertOne(context.Background(), testLead)

	payload := map[string]any{
		"page":  1,
		"limit": 10,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/lead/list", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Rows []map[string][]struct {
				Type  string `json:"type"`
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"rows"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data.Rows) == 0 {
		t.Fatalf("expected at least 1 row in list, got 0")
	}

	leadNumCell := resp.Data.Rows[0]["lead_number"]
	if len(leadNumCell) == 0 {
		t.Fatalf("expected lead_number cell, got empty")
	}

	if leadNumCell[0].Type != "LINK" {
		t.Errorf("expected lead_number cell type 'LINK', got '%s'", leadNumCell[0].Type)
	}
	if leadNumCell[0].Title != "LEAD-1001" {
		t.Errorf("expected cell title 'LEAD-1001', got '%s'", leadNumCell[0].Title)
	}
	expectedURL := "/employee/leads/detail/" + testID.Hex()
	if leadNumCell[0].URL != expectedURL {
		t.Errorf("expected cell URL '%s', got '%s'", expectedURL, leadNumCell[0].URL)
	}
}
