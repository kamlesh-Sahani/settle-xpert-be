package controllers

import (
	"fmt"
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

type CallScheduleInput struct {
	Date       time.Time                 `json:"date" binding:"required"`
	Status     models.CallScheduleStatus `json:"status" binding:"omitempty,oneof=PENDING COMPLETED"`
	CallStatus models.CallStatus         `json:"callStatus" binding:"omitempty,oneof=CONNECTED BUSY RINGING SWITCHED_OFF DISCONNECTED_BY_CLIENT"`
	Remark     string                    `json:"remark" binding:"omitempty,max=1000"`
}

type CreateLeadRequest struct {
	Personal        models.PersonalInfo   `json:"personal" binding:"required"`
	Financial       *models.FinancialInfo `json:"financial" binding:"required"`
	Agreement     *models.AgreementInfo `json:"agreement"`
	ServiceType   string                `json:"serviceType" binding:"omitempty,max=120"`
	Status        models.LeadStatus     `json:"status" binding:"omitempty,oneof=NEW CALL_SCHEDULED INTERESTED NOT_INTERESTED"`
	AssignedTo      string                `json:"assignedTo" binding:"omitempty,max=24"`
	CallSchedules   []CallScheduleInput   `json:"callSchedules" binding:"omitempty,max=20,dive"`
}

func Create(c *gin.Context) {
	// 1. Validate JSON payload using struct binding rules
	var req CreateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	// 2. Trim & re-check required text fields (blocks whitespace-only strings)
	req.Personal.FullName = strings.TrimSpace(req.Personal.FullName)
	if len(req.Personal.FullName) < 2 {
		response.BadRequest(c, "Full name must be at least 2 characters")
		return
	}
	req.Personal.Email = strings.ToLower(strings.TrimSpace(req.Personal.Email))
	req.Personal.DOB = strings.TrimSpace(req.Personal.DOB)
	req.Personal.MaritalStatus = strings.TrimSpace(req.Personal.MaritalStatus)
	req.Personal.FullAddress = strings.TrimSpace(req.Personal.FullAddress)
	req.Personal.City = strings.TrimSpace(req.Personal.City)
	req.Personal.State = strings.TrimSpace(req.Personal.State)
	req.Personal.EmploymentType = strings.TrimSpace(req.Personal.EmploymentType)
	req.ServiceType = strings.TrimSpace(req.ServiceType)

	if req.Financial != nil {
		for i := range req.Financial.Lenders {
			req.Financial.Lenders[i].LenderName = strings.TrimSpace(req.Financial.Lenders[i].LenderName)
			req.Financial.Lenders[i].LoanType = strings.TrimSpace(req.Financial.Lenders[i].LoanType)
			if req.Financial.Lenders[i].LenderName == "" || req.Financial.Lenders[i].LoanType == "" {
				response.BadRequest(c, "Lender name and loan type cannot be empty")
				return
			}
		}
		if req.Financial.TotalOutstandingAmount == 0 && len(req.Financial.Lenders) > 0 {
			for _, l := range req.Financial.Lenders {
				req.Financial.TotalOutstandingAmount += l.OutstandingAmount
			}
		}
		if req.Financial.TotalOutstandingAmount <= 0 {
			response.BadRequest(c, "Total outstanding amount is required and must be greater than 0")
			return
		}
		if req.Financial.MonthlyIncome <= 0 {
			response.BadRequest(c, "Monthly income is required and must be greater than 0")
			return
		}
	}

	// 3. Clean & validate phone number (10 digits)
	phone, ok := normalizeLeadPhone(req.Personal.Phone)
	if !ok {
		response.BadRequest(c, "Invalid phone number: must be 10 digits")
		return
	}
	req.Personal.Phone = phone

	// 4. Clean & validate PAN (optional)
	pan, ok := normalizeLeadPAN(req.Personal.PANNumber)
	if !ok {
		response.BadRequest(c, "Invalid PAN number format")
		return
	}
	req.Personal.PANNumber = pan

	leadCol := database.GetCollection("leads")
	if leadCol == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}
	ctx := c.Request.Context()

	// 5. Verify assignedTo employee (only allowed with "lead:assign" permission)
	var assignedToID *primitive.ObjectID
	if idStr := strings.TrimSpace(req.AssignedTo); idStr != "" {
		if !hasLeadPermission(c, "lead:assign") {
			response.Forbidden(c, "You do not have permission to assign leads")
			return
		}

		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil || objID.IsZero() {
			response.BadRequest(c, "Invalid assignedTo employee ID format")
			return
		}

		empCol := database.GetCollection("employees")
		if empCol == nil {
			response.InternalServerError(c, "Employee service unavailable")
			return
		}
		empCount, err := empCol.CountDocuments(ctx, bson.M{
			"_id":        objID,
			"status":     employeemodels.EmployeeStatusActive,
			"deleted_at": bson.M{"$exists": false},
		})
		if err != nil {
			log.Printf("[ERROR] [LeadCreate] Employee lookup failed: %v", err)
			response.InternalServerError(c, "Failed to verify assigned employee")
			return
		}
		if empCount == 0 {
			response.BadRequest(c, "Assigned employee not found or inactive")
			return
		}
		assignedToID = &objID
	}

	// 6. Check duplicate active lead by phone number
	dupCount, err := leadCol.CountDocuments(ctx, bson.M{
		"personal.phone": req.Personal.Phone,
		"deleted_at":     bson.M{"$exists": false},
	})
	if err != nil {
		log.Printf("[ERROR] [LeadCreate] Duplicate check failed: %v", err)
		response.InternalServerError(c, "Failed to validate lead uniqueness")
		return
	}
	if dupCount > 0 {
		response.Conflict(c, "A lead with this phone number already exists")
		return
	}

	// 7. Generate next sequential lead number (LEAD-0001, LEAD-0002, ...)
	seq, err := reserveSequenceRange(ctx, "lead_number", leadCol, 1)
	if err != nil {
		log.Printf("[ERROR] [LeadCreate] Lead number generation failed: %v", err)
		response.InternalServerError(c, "Failed to generate lead number")
		return
	}
	leadNumber := fmt.Sprintf("LEAD-%04d", seq)

	// 8. Build lead document
	now := time.Now().UTC()
	status := models.LeadStatusNew
	if req.Status != "" {
		status = req.Status
	}

	var schedules []models.CallSchedule
	for _, cs := range req.CallSchedules {
		csStatus := models.CallScheduleStatusPending
		if cs.Status != "" {
			csStatus = cs.Status
		}
		var completedAt *time.Time
		if csStatus == models.CallScheduleStatusCompleted {
			completedAt = &now
		}
		schedules = append(schedules, models.CallSchedule{
			ID:          primitive.NewObjectID(),
			Date:        cs.Date,
			Status:      csStatus,
			CallStatus:  cs.CallStatus,
			Remark:      strings.TrimSpace(cs.Remark),
			CreatedAt:   now,
			CompletedAt: completedAt,
		})
	}

	lead := models.Lead{
		ID:              primitive.NewObjectID(),
		LeadNumber:      leadNumber,
		Status:          status,
		Personal:        req.Personal,
		Financial:       req.Financial,
		Agreement:     req.Agreement,
		ServiceType:   req.ServiceType,
		AssignedTo:    assignedToID,
		CallSchedules:   schedules,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 9. Save to MongoDB
	if _, err := leadCol.InsertOne(ctx, lead); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			if strings.Contains(err.Error(), "personal.phone") {
				response.Conflict(c, "A lead with this phone number already exists")
				return
			}
			log.Printf("[ERROR] [LeadCreate] Duplicate key error: %v", err)
			response.Conflict(c, "A lead with these details already exists")
			return
		}
		log.Printf("[ERROR] [LeadCreate] Insert failed: %v", err)
		response.InternalServerError(c, "Failed to create lead")
		return
	}

	response.Success(c, "Lead created successfully", gin.H{
		"id":         lead.ID.Hex(),
		"leadNumber": lead.LeadNumber,
	})
}
