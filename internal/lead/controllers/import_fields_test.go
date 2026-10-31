package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetImportFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/lead/import-fields", GetImportFields)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lead/import-fields", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool              `json:"success"`
		Message string            `json:"message"`
		Data    []LeadImportField `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(resp.Data) == 0 {
		t.Fatal("expected at least one import field")
	}

	hasFullName := false
	hasPhone := false
	for _, f := range resp.Data {
		if f.Key == "fullName" && f.Required {
			hasFullName = true
		}
		if f.Key == "phone" && f.Required {
			hasPhone = true
		}
	}

	if !hasFullName || !hasPhone {
		t.Errorf("expected required fields fullName and phone, got: %+v", resp.Data)
	}

	// Test Search parameter
	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/lead/import-fields?search=income", nil)
	searchW := httptest.NewRecorder()
	r.ServeHTTP(searchW, searchReq)

	var searchResp struct {
		Data []LeadImportField `json:"data"`
	}
	_ = json.Unmarshal(searchW.Body.Bytes(), &searchResp)
	if len(searchResp.Data) != 1 || searchResp.Data[0].Key != "monthlyIncome" {
		t.Errorf("expected 1 result for 'income' search, got: %+v", searchResp.Data)
	}
}
