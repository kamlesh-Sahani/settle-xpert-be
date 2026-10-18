package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/integration/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
)

type TestConnectionRequest struct {
	PageAccessToken string `json:"pageAccessToken"`
	PageID          string `json:"pageId"`
}

// MetaVerificationResult stores verified details returned by Meta Graph API
type MetaVerificationResult struct {
	PageID   string
	PageName string
}

// VerifyMetaCredentials validates access token and page against Meta Graph API
func VerifyMetaCredentials(token, pageID string) (*MetaVerificationResult, error) {
	target := "me"
	cleanPageID := strings.TrimSpace(pageID)
	if cleanPageID != "" {
		target = cleanPageID
	}
	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s?fields=id,name&access_token=%s", target, token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("network error connecting to Meta: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response from Meta")
	}

	var metaResp struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &metaResp); err != nil {
		return nil, fmt.Errorf("unable to parse Meta response")
	}

	if metaResp.Error != nil {
		return nil, fmt.Errorf("Meta Graph API error: %s (code %d)", metaResp.Error.Message, metaResp.Error.Code)
	}

	return &MetaVerificationResult{
		PageID:   metaResp.ID,
		PageName: metaResp.Name,
	}, nil
}

// TestConnection verifies access token with Meta Graph API
func TestConnection(c *gin.Context) {
	var req TestConnectionRequest
	_ = c.ShouldBindJSON(&req)

	token := strings.TrimSpace(req.PageAccessToken)
	coll := database.GetCollection("integrations")

	// If token not provided in request, load from DB
	if token == "" || strings.Contains(token, "...") {
		var config models.MetaIntegrationConfig
		if err := coll.FindOne(c.Request.Context(), bson.M{"provider": "META_ADS"}).Decode(&config); err == nil {
			token = config.PageAccessToken
		}
	}

	if token == "" {
		response.BadRequest(c, "No Page Access Token provided or configured")
		return
	}

	verified, err := VerifyMetaCredentials(token, req.PageID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Update PageName in DB if empty
	if verified.PageName != "" {
		_, _ = coll.UpdateOne(c.Request.Context(), bson.M{"provider": "META_ADS"}, bson.M{
			"$set": bson.M{
				"page_name": verified.PageName,
				"page_id":   verified.PageID,
			},
		})
	}

	response.Success(c, "Connection verified successfully!", gin.H{
		"connected": true,
		"pageId":    verified.PageID,
		"pageName":  verified.PageName,
	})
}
