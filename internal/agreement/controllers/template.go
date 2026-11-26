package controllers

import (
	"errors"
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultAgreementTemplateKey = "default"

type AgreementTemplate struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Key           string              `bson:"key" json:"key"`
	Body          string              `bson:"body" json:"body"`
	UpdatedBy     *primitive.ObjectID `bson:"updated_by,omitempty" json:"updatedBy,omitempty"`
	UpdatedByName string              `bson:"updated_by_name,omitempty" json:"updatedByName,omitempty"`
	UpdatedAt     time.Time           `bson:"updated_at" json:"updatedAt"`
}

type SaveAgreementTemplateRequest struct {
	Body string `json:"body" binding:"required,max=100000"`
}

// GetTemplate returns the saved master agreement body template.
func GetTemplate(c *gin.Context) {
	col := database.GetCollection("agreement_templates")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	var tpl AgreementTemplate
	err := col.FindOne(c.Request.Context(), bson.M{"key": defaultAgreementTemplateKey}).Decode(&tpl)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.Success(c, "Agreement template retrieved", gin.H{
				"key":  defaultAgreementTemplateKey,
				"body": "",
			})
			return
		}
		log.Printf("[ERROR] [AgreementTemplateGet] Lookup failed: %v", err)
		response.InternalServerError(c, "Failed to load agreement template")
		return
	}

	response.Success(c, "Agreement template retrieved", tpl)
}

// SaveTemplate upserts the master agreement body template for future PDF generation.
func SaveTemplate(c *gin.Context) {
	var req SaveAgreementTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	trimmedBody := strings.TrimSpace(req.Body)
	if trimmedBody == "" {
		response.BadRequest(c, "Agreement body cannot be empty")
		return
	}

	col := database.GetCollection("agreement_templates")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	var updatedByID *primitive.ObjectID
	updatedByName := "System"
	if emp, ok := utils.GetAuthEmployee(c); ok && emp != nil {
		updatedByName = emp.Name
		if oid, err := primitive.ObjectIDFromHex(emp.EmployeeID); err == nil && !oid.IsZero() {
			updatedByID = &oid
		}
	}

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"key":             defaultAgreementTemplateKey,
			"body":            req.Body,
			"updated_by":      updatedByID,
			"updated_by_name": updatedByName,
			"updated_at":      now,
		},
	}

	opts := options.Update().SetUpsert(true)
	if _, err := col.UpdateOne(c.Request.Context(), bson.M{"key": defaultAgreementTemplateKey}, update, opts); err != nil {
		log.Printf("[ERROR] [AgreementTemplateSave] Upsert failed: %v", err)
		response.InternalServerError(c, "Failed to save agreement template")
		return
	}

	response.Success(c, "Agreement body template saved successfully", gin.H{
		"key":           defaultAgreementTemplateKey,
		"updatedByName": updatedByName,
		"updatedAt":     now,
	})
}
