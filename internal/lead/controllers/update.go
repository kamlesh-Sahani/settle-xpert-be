package controllers

import (
	"errors"
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	employeemodels "lead-crm/backend/internal/employee/models"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PersonalUpdateInput struct {
	FullName         *string                  `json:"fullName" binding:"omitempty,min=2,max=120"`
	Phone            *string                  `json:"phone" binding:"omitempty,max=20"`
	Email            *string                  `json:"email" binding:"omitempty,email,max=160"`
	DOB              *string                  `json:"dob" binding:"omitempty,max=32"`
	PANNumber        *string                  `json:"panNumber" binding:"omitempty,max=10"`
	MaritalStatus    *string                  `json:"maritalStatus" binding:"omitempty,max=32"`
	FullAddress      *string                  `json:"fullAddress" binding:"omitempty,max=500"`
	City             *string                  `json:"city" binding:"omitempty,max=100"`
	State            *string                  `json:"state" binding:"omitempty,max=100"`
	EmploymentStatus *models.EmploymentStatus `json:"employmentStatus" binding:"omitempty,oneof=EMPLOYED UNEMPLOYED"`
	EmploymentType   *string                  `json:"employmentType" binding:"omitempty,max=100"`
}

type FinancialUpdateInput struct {
	TotalOutstandingAmount *float64                 `json:"totalOutstandingAmount" binding:"omitempty,gte=0"`
	CreditCardDue          *float64                 `json:"creditCardDue" binding:"omitempty,gte=0"`
	PersonalDue            *float64                 `json:"personalDue" binding:"omitempty,gte=0"`
	MonthlyIncome          *float64         `json:"monthlyIncome" binding:"omitempty,gte=0"`
	MonthlyEMI             *float64         `json:"monthlyEmi" binding:"omitempty,gte=0"`
	CurrentlyPayingEMIs    *string          `json:"currentlyPayingEmis"`
	Harassment             *string          `json:"harassment"`
	Lenders                *[]models.Lender `json:"lenders" binding:"omitempty,max=50,dive"`
}

type AgreementUpdateInput struct {
	Duration       *string    `json:"duration" binding:"omitempty,max=80"`
	ConsultingFees *float64   `json:"consultingFees" binding:"omitempty,gte=0"`
	ExecutedDate   *time.Time `json:"executedDate"`
}

type UpdateLeadRequest struct {
	ID              string                `json:"id" binding:"required,max=24"`
	Personal        *PersonalUpdateInput  `json:"personal"`
	Financial       *FinancialUpdateInput `json:"financial"`
	Agreement   *AgreementUpdateInput `json:"agreement"`
	ServiceType *string               `json:"serviceType" binding:"omitempty,max=120"`
	Status      *models.LeadStatus    `json:"status" binding:"omitempty,oneof=NEW CALL_SCHEDULED INTERESTED NOT_INTERESTED CONVERTED"`
	AssignedTo      *string               `json:"assignedTo" binding:"omitempty,max=24"`
}

func Update(c *gin.Context) {
	var req UpdateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	objID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.ID))
	if err != nil || objID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	leadCol := database.GetCollection("leads")
	if leadCol == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}
	ctx := c.Request.Context()

	filter := bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}

	var existing models.Lead
	if err := leadCol.FindOne(ctx, filter).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [LeadUpdate] Lookup failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	now := time.Now().UTC()
	setFields := bson.M{"updated_at": now}

	// 1. Personal updates
	if req.Personal != nil {
		if req.Personal.FullName != nil {
			name := strings.TrimSpace(*req.Personal.FullName)
			if len(name) < 2 {
				response.BadRequest(c, "Full name must be at least 2 characters")
				return
			}
			setFields["personal.full_name"] = name
		}
		if req.Personal.Phone != nil {
			digits, ok := normalizeLeadPhone(*req.Personal.Phone)
			if !ok {
				response.BadRequest(c, "Invalid phone number: must be 10 digits")
				return
			}
			if digits != existing.Personal.Phone {
				dupCount, err := leadCol.CountDocuments(ctx, bson.M{
					"_id":            bson.M{"$ne": objID},
					"personal.phone": digits,
					"deleted_at":     bson.M{"$exists": false},
				})
				if err != nil {
					log.Printf("[ERROR] [LeadUpdate] Duplicate phone check failed: %v", err)
					response.InternalServerError(c, "Failed to validate phone uniqueness")
					return
				}
				if dupCount > 0 {
					response.Conflict(c, "Another lead with this phone number already exists")
					return
				}
			}
			setFields["personal.phone"] = digits
		}
		if req.Personal.Email != nil {
			setFields["personal.email"] = strings.ToLower(strings.TrimSpace(*req.Personal.Email))
		}
		if req.Personal.DOB != nil {
			setFields["personal.dob"] = strings.TrimSpace(*req.Personal.DOB)
		}
		if req.Personal.PANNumber != nil {
			pan, ok := normalizeLeadPAN(*req.Personal.PANNumber)
			if !ok {
				response.BadRequest(c, "Invalid PAN number format")
				return
			}
			setFields["personal.pan_number"] = pan
		}
		if req.Personal.MaritalStatus != nil {
			setFields["personal.marital_status"] = strings.TrimSpace(*req.Personal.MaritalStatus)
		}
		if req.Personal.FullAddress != nil {
			setFields["personal.full_address"] = strings.TrimSpace(*req.Personal.FullAddress)
		}
		if req.Personal.City != nil {
			setFields["personal.city"] = strings.TrimSpace(*req.Personal.City)
		}
		if req.Personal.State != nil {
			setFields["personal.state"] = strings.TrimSpace(*req.Personal.State)
		}
		if req.Personal.EmploymentStatus != nil {
			setFields["personal.employment_status"] = *req.Personal.EmploymentStatus
		}
		if req.Personal.EmploymentType != nil {
			setFields["personal.employment_type"] = strings.TrimSpace(*req.Personal.EmploymentType)
		}
	}

	// 2. Financial updates
	if req.Financial != nil {
		if req.Financial.CreditCardDue != nil {
			setFields["financial.credit_card_due"] = *req.Financial.CreditCardDue
		}
		if req.Financial.PersonalDue != nil {
			setFields["financial.personal_due"] = *req.Financial.PersonalDue
		}
		if req.Financial.MonthlyIncome != nil {
			setFields["financial.monthly_income"] = *req.Financial.MonthlyIncome
		}
		if req.Financial.MonthlyEMI != nil {
			setFields["financial.monthly_emi"] = *req.Financial.MonthlyEMI
		}
		if req.Financial.CurrentlyPayingEMIs != nil {
			val := strings.ToUpper(strings.TrimSpace(*req.Financial.CurrentlyPayingEMIs))
			if val != "" && val != "YES" && val != "NO" {
				response.BadRequest(c, "Currently paying EMIs must be YES or NO")
				return
			}
			setFields["financial.currently_paying_emis"] = val
		}
		if req.Financial.Harassment != nil {
			val := strings.ToUpper(strings.TrimSpace(*req.Financial.Harassment))
			if val != "" && val != string(models.HarassmentStatusYes) && val != string(models.HarassmentStatusNo) {
				response.BadRequest(c, "Harassment must be YES or NO")
				return
			}
			setFields["financial.harassment"] = models.HarassmentStatus(val)
		}
		if req.Financial.Lenders != nil {
			lenders := make([]models.Lender, 0, len(*req.Financial.Lenders))
			var lenderSum float64
			for _, l := range *req.Financial.Lenders {
				name := strings.TrimSpace(l.LenderName)
				loanType := strings.TrimSpace(l.LoanType)
				if name == "" || loanType == "" {
					response.BadRequest(c, "Lender name and loan type cannot be empty")
					return
				}
				lenders = append(lenders, models.Lender{
					LenderName:        name,
					LoanType:          loanType,
					OutstandingAmount: l.OutstandingAmount,
				})
				lenderSum += l.OutstandingAmount
			}
			setFields["financial.lenders"] = lenders
			if req.Financial.TotalOutstandingAmount == nil && len(lenders) > 0 {
				setFields["financial.total_outstanding_amount"] = lenderSum
			}
		}
		if req.Financial.TotalOutstandingAmount != nil {
			setFields["financial.total_outstanding_amount"] = *req.Financial.TotalOutstandingAmount
		}
	}

	// 3. Agreement updates
	if req.Agreement != nil {
		if req.Agreement.Duration != nil {
			setFields["agreement.duration"] = strings.TrimSpace(*req.Agreement.Duration)
		}
		if req.Agreement.ConsultingFees != nil {
			setFields["agreement.consulting_fees"] = *req.Agreement.ConsultingFees
		}
		if req.Agreement.ExecutedDate != nil {
			setFields["agreement.executed_date"] = req.Agreement.ExecutedDate
		}
	}

	// 4. Service & Status updates
	if req.ServiceType != nil {
		setFields["service_type"] = strings.TrimSpace(*req.ServiceType)
	}
	if req.Status != nil {
		setFields["status"] = *req.Status
	}

	// 5. Assignment updates (requires lead:assign)
	if req.AssignedTo != nil {
		idStr := strings.TrimSpace(*req.AssignedTo)
		canAssign := hasLeadPermission(c, "lead:assign")

		if idStr != "" && !canAssign {
			response.Forbidden(c, "You do not have permission to assign leads")
			return
		}

		if canAssign {
			if idStr == "" {
				setFields["assigned_to"] = nil
			} else {
				empObjID, err := primitive.ObjectIDFromHex(idStr)
				if err != nil || empObjID.IsZero() {
					response.BadRequest(c, "Invalid assignedTo employee ID format")
					return
				}
				empCol := database.GetCollection("employees")
				if empCol == nil {
					response.InternalServerError(c, "Employee service unavailable")
					return
				}
				empCount, err := empCol.CountDocuments(ctx, bson.M{
					"_id":        empObjID,
					"status":     employeemodels.EmployeeStatusActive,
					"deleted_at": bson.M{"$exists": false},
				})
				if err != nil {
					log.Printf("[ERROR] [LeadUpdate] Employee lookup failed: %v", err)
					response.InternalServerError(c, "Failed to verify assigned employee")
					return
				}
				if empCount == 0 {
					response.BadRequest(c, "Assigned employee not found or inactive")
					return
				}
				setFields["assigned_to"] = &empObjID
			}
		}
	}

	if _, err := leadCol.UpdateOne(ctx, filter, bson.M{"$set": setFields}); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			response.Conflict(c, "Another lead with this phone number already exists")
			return
		}
		log.Printf("[ERROR] [LeadUpdate] Update failed: %v", err)
		response.InternalServerError(c, "Failed to update lead")
		return
	}

	response.Success(c, "Lead updated successfully", gin.H{
		"id":         req.ID,
		"leadNumber": existing.LeadNumber,
	})
}
