package controllers

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"slices"
	"strings"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	nonDigitRegex = regexp.MustCompile(`\D`)
	panRegex      = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)
)

func normalizeLeadPhone(raw string) (string, bool) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(raw), ".0")
	digits := nonDigitRegex.ReplaceAllString(trimmed, "")
	if len(digits) == 11 && strings.HasPrefix(digits, "0") {
		digits = digits[1:]
	} else if len(digits) == 12 && strings.HasPrefix(digits, "91") {
		digits = digits[2:]
	}
	return digits, len(digits) == 10
}

func normalizeLeadPAN(raw string) (string, bool) {
	pan := strings.ToUpper(strings.TrimSpace(raw))
	if pan == "" {
		return "", true
	}
	return pan, panRegex.MatchString(pan)
}

func isValidOptionalEmail(email string) bool {
	if email == "" {
		return true
	}
	if len(email) > 160 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func validateBulkLeadRow(item BulkLeadItem) (string, string, string, string) {
	fullName := strings.TrimSpace(item.FullName)
	if len(fullName) < 2 || len(fullName) > 120 {
		return "", "", "", "Borrower full name must be between 2 and 120 characters"
	}
	phone, ok := normalizeLeadPhone(item.Phone)
	if !ok {
		return "", "", "", "Invalid phone number (must be 10 digits)"
	}
	email := strings.ToLower(strings.TrimSpace(item.Email))
	if !isValidOptionalEmail(email) {
		return "", "", "", "Invalid email format"
	}
	pan, panValid := normalizeLeadPAN(item.PANNumber)
	if !panValid {
		return "", "", "", "Invalid PAN number format"
	}
	if item.EmploymentStatus != "" &&
		item.EmploymentStatus != models.EmploymentStatusEmployed &&
		item.EmploymentStatus != models.EmploymentStatusUnemployed {
		return "", "", "", "Employment status must be EMPLOYED or UNEMPLOYED"
	}
	if item.Harassment != "" &&
		item.Harassment != models.HarassmentStatusYes &&
		item.Harassment != models.HarassmentStatusNo {
		return "", "", "", "Harassment must be YES or NO"
	}
	if len(strings.TrimSpace(item.FullAddress)) > 500 ||
		len(strings.TrimSpace(item.City)) > 100 ||
		len(strings.TrimSpace(item.State)) > 100 ||
		len(strings.TrimSpace(item.ServiceType)) > 120 {
		return "", "", "", "One or more text fields exceed maximum allowed length"
	}
	if item.TotalOutstandingAmount < 0 || item.CreditCardDue < 0 || item.PersonalDue < 0 ||
		item.MonthlyIncome < 0 || item.MonthlyEMI < 0 {
		return "", "", "", "Financial amounts cannot be negative"
	}
	return fullName, phone, pan, ""
}

func hasLeadPermission(c *gin.Context, required string) bool {
	perms, _ := c.Get("permissions")
	permList, _ := perms.([]string)
	return slices.Contains(permList, required)
}

// reserveSequenceRange atomically allocates `count` sequential IDs from `counters`,
// seeding from the target collection's existing count if uninitialized.
func reserveSequenceRange(ctx context.Context, counterID string, col *mongo.Collection, count int64) (int64, error) {
	counterCol := database.GetCollection("counters")
	if counterCol == nil {
		return 0, fmt.Errorf("counter collection unavailable")
	}

	counterExists, err := counterCol.CountDocuments(ctx, bson.M{"_id": counterID})
	if err != nil {
		return 0, err
	}
	if counterExists == 0 {
		existingCount, err := col.CountDocuments(ctx, bson.M{})
		if err != nil {
			return 0, err
		}
		_, err = counterCol.InsertOne(ctx, bson.M{
			"_id": counterID,
			"seq": existingCount,
		})
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return 0, err
		}
	}

	var result struct {
		Seq int64 `bson:"seq"`
	}
	err = counterCol.FindOneAndUpdate(
		ctx,
		bson.M{"_id": counterID},
		bson.M{"$inc": bson.M{"seq": count}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&result)
	if err != nil {
		return 0, err
	}

	return result.Seq - count + 1, nil
}
