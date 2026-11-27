package controllers

import (
	"errors"
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/agreement/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UpdateAgreementTemplateRequest struct {
	Body   *string `json:"body"`
	Footer *string `json:"footer"`
}

// GetTemplate returns the stored agreement body and footer templates, seeding defaults on first access.
func GetTemplate(c *gin.Context) {
	col := database.GetCollection(models.AgreementTemplateCollection)
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}
	ctx := c.Request.Context()
	filter := bson.M{"key": models.DefaultAgreementTemplateKey}

	var tpl models.AgreementTemplate
	err := col.FindOne(ctx, filter).Decode(&tpl)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.Success(c, "Agreement template fetched successfully", gin.H{
				"body":   "",
				"footer": "",
			})
			return
		}
		log.Printf("[ERROR] [AgreementGetTemplate] FindOne failed: %v", err)
		response.InternalServerError(c, "Failed to fetch agreement template")
		return
	}

	response.Success(c, "Agreement template fetched successfully", gin.H{
		"body":   tpl.Body,
		"footer": tpl.Footer,
	})
}

// UpdateTemplate updates the agreement body, footer, or both in the database.
func UpdateTemplate(c *gin.Context) {
	emp, ok := utils.GetAuthEmployee(c)
	if !ok || emp == nil {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	updatedByID, err := primitive.ObjectIDFromHex(strings.TrimSpace(emp.EmployeeID))
	if err != nil || updatedByID.IsZero() {
		response.Unauthorized(c, "Invalid authenticated employee")
		return
	}

	var req UpdateAgreementTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload")
		return
	}
	if req.Body == nil && req.Footer == nil {
		response.BadRequest(c, "At least one of body or footer must be provided")
		return
	}

	now := time.Now().UTC()
	setFields := bson.M{
		"updatedBy": &updatedByID,
		"updatedAt": now,
	}
	setOnInsert := bson.M{
		"key":       models.DefaultAgreementTemplateKey,
		"createdAt": now,
	}

	if req.Body != nil {
		setFields["body"] = *req.Body
	} else {
		setOnInsert["body"] = ""
	}

	if req.Footer != nil {
		setFields["footer"] = *req.Footer
	} else {
		setOnInsert["footer"] = ""
	}

	col := database.GetCollection(models.AgreementTemplateCollection)
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}
	ctx := c.Request.Context()
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var updated models.AgreementTemplate
	err = col.FindOneAndUpdate(
		ctx,
		bson.M{"key": models.DefaultAgreementTemplateKey},
		bson.M{
			"$set":         setFields,
			"$setOnInsert": setOnInsert,
		},
		opts,
	).Decode(&updated)
	if err != nil {
		log.Printf("[ERROR] [AgreementUpdateTemplate] FindOneAndUpdate failed: %v", err)
		response.InternalServerError(c, "Failed to update agreement template")
		return
	}

	response.Success(c, "Agreement template updated successfully", nil)
}
