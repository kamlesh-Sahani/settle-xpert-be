package controllers

import (
	"fmt"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BulkCreateLeadsRequest struct {
	Leads []CreateLeadRequest `json:"leads" binding:"required,min=1"`
}

func BulkCreate(c *gin.Context) {
	var req BulkCreateLeadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	col := database.GetCollection("leads")
	empCol := database.GetCollection("employees")
	ctx := c.Request.Context()

	var tenantID *primitive.ObjectID
	if tVal, exists := c.Get("tenant_obj_id"); exists {
		if tid, ok := tVal.(primitive.ObjectID); ok && !tid.IsZero() {
			tenantID = &tid
		}
	}

	empCache := make(map[string]string)
	now := time.Now()

	documents := make([]interface{}, 0, len(req.Leads))

	for i, item := range req.Leads {
		fullName := strings.TrimSpace(item.FullName)
		phone := strings.TrimSpace(item.Phone)
		if fullName == "" || phone == "" {
			continue
		}

		source := models.SourceExcelImport
		if item.Source != "" {
			source = item.Source
		}

		status := models.LeadStatusNew
		if item.Status != "" {
			status = item.Status
		} else if item.AssignedTo != "" {
			status = models.LeadStatusInProgress
		}

		var assignedToObjID *primitive.ObjectID
		assignedToName := ""

		if item.AssignedTo != "" {
			if objID, err := primitive.ObjectIDFromHex(item.AssignedTo); err == nil {
				assignedToObjID = &objID
				if name, ok := empCache[item.AssignedTo]; ok {
					assignedToName = name
				} else {
					var emp struct {
						Name string `bson:"name"`
					}
					if err := empCol.FindOne(ctx, bson.M{"_id": objID}).Decode(&emp); err == nil {
						assignedToName = emp.Name
						empCache[item.AssignedTo] = emp.Name
					}
				}
			}
		}

		leadNumber := fmt.Sprintf("LD-%d-%02d", (now.UnixNano()/1000)%10000000, i+1)

		lead := models.Lead{
			ID:                 primitive.NewObjectID(),
			TenantID:           tenantID,
			LeadNumber:         leadNumber,
			FullName:           fullName,
			Phone:              phone,
			Email:              strings.TrimSpace(item.Email),
			City:               strings.TrimSpace(item.City),
			State:              strings.TrimSpace(item.State),
			Source:             source,
			TotalDebtAmount:    item.TotalDebtAmount,
			OutstandingAmount:  item.TotalDebtAmount,
			Status:             status,
			QualificationNotes: strings.TrimSpace(item.QualificationNotes),
			FinancialAssessment: &models.FinancialAssessment{
				MonthlyIncome:   item.MonthlyIncome,
				TotalMonthlyEMI: item.CurrentEMIAmount,
				TotalDebtAmount: item.TotalDebtAmount,
				Debts:           []models.DebtItem{},
			},
			AssignedTo:     assignedToObjID,
			AssignedToName: assignedToName,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		documents = append(documents, lead)
	}

	if len(documents) == 0 {
		response.BadRequest(c, "No valid leads found to import")
		return
	}

	if _, err := col.InsertMany(ctx, documents); err != nil {
		response.InternalServerError(c, "Failed to import leads")
		return
	}

	response.Success(c, fmt.Sprintf("Successfully imported %d leads", len(documents)), gin.H{
		"importedCount": len(documents),
	})
}
