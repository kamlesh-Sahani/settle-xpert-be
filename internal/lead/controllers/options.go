package controllers

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LeadOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Link  string `json:"link,omitempty"`
}

// Options returns non-deleted leads as lightweight key/label pairs for SelectApiField.
func Options(c *gin.Context) {
	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
	}

	search := strings.TrimSpace(c.Query("search"))
	page := 1
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if limit > 200 {
		limit = 200
	}

	if search != "" {
		escaped := regexp.QuoteMeta(search)
		filter["$or"] = []bson.M{
			{"lead_number": primitive.Regex{Pattern: escaped, Options: "i"}},
			{"personal.full_name": primitive.Regex{Pattern: escaped, Options: "i"}},
			{"personal.phone": primitive.Regex{Pattern: escaped, Options: "i"}},
		}
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetProjection(bson.M{
			"_id":                1,
			"lead_number":        1,
			"personal.full_name": 1,
			"personal.phone":     1,
		})

	cursor, err := col.Find(c.Request.Context(), filter, findOpts)
	if err != nil {
		log.Printf("[ERROR] [LeadOptions] Query failed: %v", err)
		response.InternalServerError(c, "Failed to load leads")
		return
	}
	defer cursor.Close(c.Request.Context())

	var leads []models.Lead
	if err := cursor.All(c.Request.Context(), &leads); err != nil {
		log.Printf("[ERROR] [LeadOptions] Decode failed: %v", err)
		response.InternalServerError(c, "Failed to decode leads")
		return
	}

	items := make([]LeadOption, 0, len(leads))
	for _, l := range leads {
		label := l.Personal.FullName
		if l.LeadNumber != "" && l.Personal.Phone != "" {
			label = fmt.Sprintf("%s (%s · %s)", l.Personal.FullName, l.LeadNumber, l.Personal.Phone)
		} else if l.LeadNumber != "" {
			label = fmt.Sprintf("%s (%s)", l.Personal.FullName, l.LeadNumber)
		}
		items = append(items, LeadOption{
			Key:   l.ID.Hex(),
			Value: label,
			Link:  "/employee/leads/detail/" + l.ID.Hex(),
		})
	}

	response.Success(c, "Leads loaded successfully", items)
}

type LeadInfoPersonal struct {
	FullName    string `json:"fullName"`
	PANNumber   string `json:"panNumber,omitempty"`
	DOB         string `json:"dob,omitempty"`
	Phone       string `json:"phone"`
	Email       string `json:"email,omitempty"`
	FullAddress string `json:"fullAddress,omitempty"`
}

type LeadInfoFinancial struct {
	MonthlyIncome float64         `json:"monthlyIncome,omitempty"`
	Lenders       []models.Lender `json:"lenders"`
}

type LeadInfoResponse struct {
	ID         string               `json:"id"`
	LeadNumber string               `json:"leadNumber"`
	Personal   LeadInfoPersonal     `json:"personal"`
	Financial  LeadInfoFinancial    `json:"financial"`
	Agreement  *models.AgreementInfo `json:"agreement,omitempty"`
}

// GetInfo returns only the limited lead fields needed for Agreement creation.
func GetInfo(c *gin.Context) {
	idStr := strings.TrimSpace(c.Query("id"))
	if idStr == "" {
		response.BadRequest(c, "Lead ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil || objID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	filter := bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}

	opts := options.FindOne().SetProjection(bson.M{
		"_id":                      1,
		"lead_number":              1,
		"personal.full_name":       1,
		"personal.pan_number":      1,
		"personal.dob":             1,
		"personal.phone":           1,
		"personal.email":           1,
		"personal.full_address":    1,
		"financial.monthly_income": 1,
		"financial.lenders":        1,
		"agreement":                1,
	})

	var lead models.Lead
	if err := col.FindOne(c.Request.Context(), filter, opts).Decode(&lead); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [LeadGetInfo] Query failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead info")
		return
	}

	lenders := []models.Lender{}
	var monthlyIncome float64
	if lead.Financial != nil {
		monthlyIncome = lead.Financial.MonthlyIncome
		if lead.Financial.Lenders != nil {
			lenders = lead.Financial.Lenders
		}
	}

	response.Success(c, "Lead info retrieved successfully", LeadInfoResponse{
		ID:         lead.ID.Hex(),
		LeadNumber: lead.LeadNumber,
		Personal: LeadInfoPersonal{
			FullName:    lead.Personal.FullName,
			PANNumber:   lead.Personal.PANNumber,
			DOB:         lead.Personal.DOB,
			Phone:       lead.Personal.Phone,
			Email:       lead.Personal.Email,
			FullAddress: lead.Personal.FullAddress,
		},
		Financial: LeadInfoFinancial{
			MonthlyIncome: monthlyIncome,
			Lenders:       lenders,
		},
		Agreement: lead.Agreement,
	})
}
