package controllers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
)

type LeadImportField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}

func GetImportFields(c *gin.Context) {
	fields := []LeadImportField{
		{Key: "fullName", Label: "Full Name", Required: true},
		{Key: "phone", Label: "Phone Number", Required: true},
		{Key: "email", Label: "Email Address", Required: false},
		{Key: "dob", Label: "Date of Birth", Required: false},
		{Key: "panNumber", Label: "PAN Number", Required: false},
		{Key: "maritalStatus", Label: "Marital Status", Required: false},
		{Key: "fullAddress", Label: "Full Address", Required: false},
		{Key: "city", Label: "City", Required: false},
		{Key: "state", Label: "State", Required: false},
		{Key: "employmentStatus", Label: "Employment Status", Required: false},
		{Key: "employmentType", Label: "Employment Type", Required: false},
		{Key: "totalOutstandingAmount", Label: "Total Outstanding Amount", Required: false},
		{Key: "creditCardDue", Label: "Credit Card Due", Required: false},
		{Key: "personalDue", Label: "Personal Due", Required: false},
		{Key: "monthlyIncome", Label: "Monthly Income", Required: false},
		{Key: "monthlyEmi", Label: "Monthly EMI", Required: false},
		{Key: "harassment", Label: "Harassment", Required: false},
		{Key: "serviceType", Label: "Service Type", Required: false},
	}

	search := strings.ToLower(strings.TrimSpace(c.Query("search")))
	if search != "" {
		filtered := make([]LeadImportField, 0, len(fields))
		for _, f := range fields {
			if strings.Contains(strings.ToLower(f.Label), search) || strings.Contains(strings.ToLower(f.Key), search) {
				filtered = append(filtered, f)
			}
		}
		fields = filtered
	}

	response.Success(c, "CRM Lead fields fetched successfully", fields)
}
