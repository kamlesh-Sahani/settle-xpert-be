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
		{Key: "totalDebtAmount", Label: "Total Debt Amount", Required: false},
		{Key: "monthlyIncome", Label: "Monthly Income", Required: false},
		{Key: "currentEmiAmount", Label: "Total Monthly EMI", Required: false},
		{Key: "city", Label: "City", Required: false},
		{Key: "state", Label: "State", Required: false},
		{Key: "serviceRequired", Label: "Service Required", Required: false},
		{Key: "qualificationNotes", Label: "Qualification Notes", Required: false},
	}

	search := strings.ToLower(c.Query("search"))
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
