package controllers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lead-crm/backend/database"
	metaModels "lead-crm/backend/internal/meta/models"
	leadModels "lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VerifyWebhook handles the GET verification handshake from Meta
func VerifyWebhook(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode != "subscribe" || token == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var config metaModels.MetaIntegrationConfig
	coll := database.GetCollection("integrations")
	if coll == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if err := coll.FindOne(c.Request.Context(), bson.M{}).Decode(&config); err != nil || config.VerifyToken == "" {
		log.Printf("[META WEBHOOK] Verification rejected: Meta integration not configured")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if token != config.VerifyToken {
		log.Printf("[META WEBHOOK] Verification failed: received token does not match configured token")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	log.Printf("[META WEBHOOK] Verification challenge succeeded!")
	c.String(http.StatusOK, challenge)
}

// verifySignature checks the X-Hub-Signature-256 header against the request body using AppSecret
func verifySignature(payload []byte, secret, signature string) bool {
	if secret == "" {
		return true
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature[7:]), []byte(expectedMAC))
}

// HandleWebhook handles incoming lead notifications from Meta
func HandleWebhook(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	// Validate HMAC SHA256 signature if AppSecret is configured
	coll := database.GetCollection("integrations")
	var config metaModels.MetaIntegrationConfig
	if coll != nil {
		_ = coll.FindOne(c.Request.Context(), bson.M{}).Decode(&config)
	}

	if config.AppSecret != "" {
		sig := c.GetHeader("X-Hub-Signature-256")
		if !verifySignature(bodyBytes, config.AppSecret, sig) {
			log.Printf("[META WEBHOOK] Signature verification failed")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}

	var payload metaModels.MetaWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format"})
		return
	}

	// Always acknowledge Meta immediately with 200 OK so it doesn't retry
	c.JSON(http.StatusOK, gin.H{"status": "EVENT_RECEIVED"})

	// Process lead asynchronously
	go func(p metaModels.MetaWebhookPayload, raw string) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for _, entry := range p.Entry {
			for _, change := range entry.Changes {
				if change.Field == "leadgen" {
					leadgenID := change.Value.LeadgenID
					formID := change.Value.FormID
					pageID := change.Value.PageID
					adID := change.Value.AdID

					processLead(ctx, leadgenID, formID, pageID, adID, raw)
				}
			}
		}
	}(payload, string(bodyBytes))
}

func processLead(ctx context.Context, leadgenID, formID, pageID, adID, raw string) {
	logColl := database.GetCollection("meta_webhook_logs")
	leadColl := database.GetCollection("leads")
	intColl := database.GetCollection("integrations")

	syncLog := metaModels.MetaWebhookLog{
		LeadgenID:  leadgenID,
		PageID:     pageID,
		FormID:     formID,
		RawPayload: raw,
		CreatedAt:  time.Now(),
	}

	// 1. Fetch integration configuration
	var config metaModels.MetaIntegrationConfig
	if err := intColl.FindOne(ctx, bson.M{}).Decode(&config); err != nil {
		syncLog.Status = "FAILED"
		syncLog.ErrorMessage = "Meta integration is not configured in CRM"
		if logColl != nil {
			_, _ = logColl.InsertOne(ctx, syncLog)
		}
		return
	}

	token := config.PageAccessToken
	if token == "" {
		syncLog.Status = "FAILED"
		syncLog.ErrorMessage = "No Page Access Token configured in CRM"
		if logColl != nil {
			_, _ = logColl.InsertOne(ctx, syncLog)
		}
		return
	}

	// 2. Fetch lead data from Meta Graph API
	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s?access_token=%s", leadgenID, token)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		syncLog.Status = "FAILED"
		syncLog.ErrorMessage = fmt.Sprintf("Failed to query Meta Graph API: %v", err)
		if logColl != nil {
			_, _ = logColl.InsertOne(ctx, syncLog)
		}
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var metaLead metaModels.MetaLeadDetails
	if err := json.Unmarshal(body, &metaLead); err != nil {
		syncLog.Status = "FAILED"
		syncLog.ErrorMessage = "Failed to parse Meta lead response"
		if logColl != nil {
			_, _ = logColl.InsertOne(ctx, syncLog)
		}
		return
	}

	// 3. Map field data
	lead := leadModels.Lead{
		Source: leadModels.SourceFacebookAds,
		Status: leadModels.LeadStatusNew,
		MetaDetails: &leadModels.MetaDetails{
			LeadgenID: leadgenID,
			FormID:    formID,
			PageID:    pageID,
			AdID:      adID,
		},
		RawFields: make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var firstName, lastName string

	for _, field := range metaLead.FieldData {
		val := ""
		if len(field.Values) > 0 {
			val = strings.TrimSpace(field.Values[0])
		}
		key := strings.ToLower(field.Name)
		lead.RawFields[field.Name] = val

		// Map to CRM fields
		switch key {
		case "full_name", "name":
			lead.FullName = val
		case "first_name":
			firstName = val
		case "last_name":
			lastName = val
		case "email":
			lead.Email = strings.ToLower(val)
		case "phone_number", "phone":
			lead.Phone = val
		case "city":
			lead.City = val
		case "state":
			lead.State = val
		case "loan_type", "type_of_loan":
			lead.LoanType = val
		case "total_debt", "total_debt_amount", "debt_amount":
			if parsed, err := strconv.ParseFloat(val, 64); err == nil {
				lead.TotalDebtAmount = parsed
			}
		}
	}

	if lead.FullName == "" && (firstName != "" || lastName != "") {
		lead.FullName = strings.TrimSpace(firstName + " " + lastName)
	}

	syncLog.FullName = lead.FullName
	syncLog.Phone = lead.Phone
	syncLog.Email = lead.Email

	// 4. Duplicate Check (by Phone or Email)
	if lead.Phone != "" || lead.Email != "" {
		dupFilter := bson.M{}
		var orConditions []bson.M
		if lead.Phone != "" {
			orConditions = append(orConditions, bson.M{"phone": lead.Phone})
		}
		if lead.Email != "" {
			orConditions = append(orConditions, bson.M{"email": lead.Email})
		}
		dupFilter["$or"] = orConditions

		var existingLead leadModels.Lead
		if err := leadColl.FindOne(ctx, dupFilter).Decode(&existingLead); err == nil {
			syncLog.Status = "DUPLICATE"
			syncLog.LeadNumber = existingLead.LeadNumber
			syncLog.ErrorMessage = fmt.Sprintf("Lead already exists with number %s", existingLead.LeadNumber)
			if logColl != nil {
				_, _ = logColl.InsertOne(ctx, syncLog)
			}
			return
		}
	}

	// 5. Generate Lead Number & Assign
	count, _ := leadColl.CountDocuments(ctx, bson.M{})
	lead.LeadNumber = fmt.Sprintf("LD-%d-%04d", time.Now().Year(), count+1)
	syncLog.LeadNumber = lead.LeadNumber

	// 6. Insert lead into DB
	insertRes, err := leadColl.InsertOne(ctx, lead)
	if err != nil {
		syncLog.Status = "FAILED"
		syncLog.ErrorMessage = fmt.Sprintf("Failed to insert lead into database: %v", err)
		if logColl != nil {
			_, _ = logColl.InsertOne(ctx, syncLog)
		}
		return
	}

	if oid, ok := insertRes.InsertedID.(primitive.ObjectID); ok {
		lead.ID = oid
	}

	syncLog.Status = "SUCCESS"
	if logColl != nil {
		_, _ = logColl.InsertOne(ctx, syncLog)
	}
	log.Printf("[META WEBHOOK] Successfully synced new lead: %s (%s)", lead.FullName, lead.LeadNumber)
}
