package controllers

import (
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/meta/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SaveRequest struct {
	VerifyToken     string            `json:"verifyToken"`
	PageAccessToken string            `json:"pageAccessToken"`
	PageID          string            `json:"pageId"`
	PageName        string            `json:"pageName"`
	AppSecret       string            `json:"appSecret"`
	FieldMapping    map[string]string `json:"fieldMapping"`
}

// Save updates or creates Meta Ads configuration
func Save(c *gin.Context) {
	var req SaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	coll := database.GetCollection("integrations")

	var existing models.MetaIntegrationConfig
	_ = coll.FindOne(c.Request.Context(), bson.M{}).Decode(&existing)

	pageAccessToken := strings.TrimSpace(req.PageAccessToken)
	if pageAccessToken == "" || strings.Contains(pageAccessToken, "...") {
		pageAccessToken = existing.PageAccessToken
	}

	if pageAccessToken == "" {
		response.BadRequest(c, "Page Access Token is required")
		return
	}

	// Live Meta Verification: verify credentials against Meta Graph API
	status := models.MetaConfigStatusActive
	verified, err := VerifyMetaCredentials(c.Request.Context(), pageAccessToken, req.PageID)
	if err != nil {
		status = models.MetaConfigStatusInactive
		log.Printf("[WARN] [MetaIntegration] Verification failed: %v", err)
		response.BadRequest(c, "Meta connection verification failed. Please check your credentials.")
		return
	}

	pageName := strings.TrimSpace(req.PageName)
	if verified.PageName != "" {
		pageName = verified.PageName
	}

	pageID := strings.TrimSpace(req.PageID)
	if verified.PageID != "" && pageID == "" {
		pageID = verified.PageID
	}

	appSecret := strings.TrimSpace(req.AppSecret)
	if appSecret == "" || strings.Contains(appSecret, "...") {
		appSecret = existing.AppSecret
	}

	verifyToken := strings.TrimSpace(req.VerifyToken)
	if verifyToken == "" {
		verifyToken = existing.VerifyToken
	}
	if verifyToken == "" {
		response.BadRequest(c, "Verify Token is required")
		return
	}

	fieldMapping := req.FieldMapping

	updateDoc := bson.M{
		"status":            status,
		"verify_token":      verifyToken,
		"page_access_token": pageAccessToken,
		"page_id":           pageID,
		"page_name":         pageName,
		"app_secret":        appSecret,
		"field_mapping":     fieldMapping,
		"updated_at":        time.Now(),
	}

	filter := bson.M{}
	if !existing.ID.IsZero() {
		filter = bson.M{"_id": existing.ID}
	}

	opts := options.Update().SetUpsert(true)
	_, err = coll.UpdateOne(c.Request.Context(), filter, bson.M{
		"$set": updateDoc,
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}, opts)

	if err != nil {
		response.InternalServerError(c, "Failed to save configuration")
		return
	}

	response.Success(c, "Meta Ads integration configuration saved successfully", nil)
}
