package controllers

import (
	"fmt"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/audit"
	"lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateLeadRequest struct {
	ID                  string                     `json:"id" binding:"required"`
	FullName            *string                    `json:"fullName"`
	Phone               *string                    `json:"phone"`
	AlternatePhone      *string                    `json:"alternatePhone"`
	Email               *string                    `json:"email"`
	DOB                 *string                    `json:"dob"`
	PANCard             *string                    `json:"panCard"`
	Address             *models.Address            `json:"address"`
	City                *string                    `json:"city"`
	State               *string                    `json:"state"`
	LoanType            *string                    `json:"loanType"`
	ServiceRequired     *string                    `json:"serviceRequired"`
	Source              *models.LeadSource         `json:"source"`
	Priority            *models.LeadPriority       `json:"priority" binding:"omitempty,oneof=LOW NORMAL HIGH"`
	Status              *models.LeadStatus         `json:"status" binding:"omitempty,oneof=NEW CONTACTED QUALIFIED UNQUALIFIED FOLLOW_UP RE_ENGAGED UNREACHABLE CONVERTED LOST IN_PROGRESS"`
	InitialEnquiry      *string                    `json:"initialEnquiry"`
	QualificationNotes  *string                    `json:"qualificationNotes"`
	TotalDebtAmount     *float64                   `json:"totalDebtAmount"`
	OutstandingAmount   *float64                   `json:"outstandingAmount"`
	MonthlyIncome       *float64                   `json:"monthlyIncome"`
	CurrentEMIAmount    *float64                   `json:"currentEmiAmount"`
	OverdueDuration     *string                    `json:"overdueDuration"`
	ReasonForDifficulty *string                    `json:"reasonForDifficulty"`
	DesiredOutcome      *string                    `json:"desiredOutcome"`
	Debts               *[]CreateLeadDebtItem      `json:"debts"`
	AssignedTo          *string                    `json:"assignedTo"`
	LostReason          *string                    `json:"lostReason"`
	ClosingNotes        *string                    `json:"closingNotes"`
	Conversation        *CreateConversationPayload `json:"conversation"`
}

func Update(c *gin.Context) {
	var req UpdateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	objID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	col := database.GetCollection("leads")
	ctx := c.Request.Context()

	filter := bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}
	if tVal, exists := c.Get("tenant_obj_id"); exists {
		if tid, ok := tVal.(primitive.ObjectID); ok {
			filter["tenant_id"] = tid
		}
	}

	var existing models.Lead
	if err := col.FindOne(ctx, filter).Decode(&existing); err != nil {
		response.NotFound(c, "Lead not found")
		return
	}

	if req.Phone != nil {
		newPhone := strings.TrimSpace(*req.Phone)
		if newPhone != "" && newPhone != existing.Phone {
			cleanedPhone := digitCleaner.ReplaceAllString(newPhone, "")
			last10 := cleanedPhone
			if len(cleanedPhone) > 10 {
				last10 = cleanedPhone[len(cleanedPhone)-10:]
			}
			if len(last10) >= 6 {
				dupFilter := bson.M{
					"_id":        bson.M{"$ne": objID},
					"phone":      bson.M{"$regex": last10 + "$"},
					"deleted_at": bson.M{"$exists": false},
				}
				if tid, ok := filter["tenant_id"]; ok {
					dupFilter["tenant_id"] = tid
				}
				if count, _ := col.CountDocuments(ctx, dupFilter); count > 0 {
					response.Conflict(c, "Another lead with this phone number already exists")
					return
				}
			}
		}
	}

	now := time.Now()
	setFields := bson.M{"updated_at": now}
	pushFields := bson.M{}
	var activities []models.ActivityItem

	userName := "User"
	var creatorID *primitive.ObjectID
	if emp, ok := utils.GetAuthEmployee(c); ok && emp != nil {
		userName = emp.Name
		if oid, err := primitive.ObjectIDFromHex(emp.EmployeeID); err == nil {
			creatorID = &oid
		}
	}

	if req.FullName != nil {
		setFields["full_name"] = strings.TrimSpace(*req.FullName)
	}
	if req.Phone != nil {
		setFields["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.AlternatePhone != nil {
		setFields["alternate_phone"] = strings.TrimSpace(*req.AlternatePhone)
	}
	if req.Email != nil {
		setFields["email"] = strings.TrimSpace(*req.Email)
	}
	if req.DOB != nil {
		setFields["dob"] = strings.TrimSpace(*req.DOB)
	}
	if req.PANCard != nil {
		setFields["pan_card"] = strings.ToUpper(strings.TrimSpace(*req.PANCard))
	}
	if req.Address != nil {
		setFields["address"] = req.Address
		if req.Address.City != "" {
			setFields["city"] = req.Address.City
		}
		if req.Address.State != "" {
			setFields["state"] = req.Address.State
		}
	}
	if req.City != nil {
		setFields["city"] = strings.TrimSpace(*req.City)
	}
	if req.State != nil {
		setFields["state"] = strings.TrimSpace(*req.State)
	}
	if req.LoanType != nil {
		setFields["loan_type"] = strings.TrimSpace(*req.LoanType)
	}
	if req.ServiceRequired != nil {
		setFields["service_required"] = strings.TrimSpace(*req.ServiceRequired)
	}
	if req.Source != nil {
		setFields["source"] = *req.Source
	}
	if req.Priority != nil {
		setFields["priority"] = *req.Priority
	}
	if req.Status != nil {
		newStatus := *req.Status
		setFields["status"] = newStatus
		if newStatus == models.LeadStatusConverted && existing.ConvertedAt == nil {
			setFields["converted_at"] = now
		}
		if newStatus != existing.Status {
			activities = append(activities, models.ActivityItem{
				ID:              primitive.NewObjectID(),
				Type:            "STATUS_CHANGED",
				Description:     fmt.Sprintf("Status changed from %s to %s", existing.Status, newStatus),
				PerformedByName: userName,
				CreatedAt:       now,
			})
		}
	}
	if req.InitialEnquiry != nil {
		setFields["initial_enquiry"] = strings.TrimSpace(*req.InitialEnquiry)
	}
	if req.QualificationNotes != nil {
		setFields["qualification_notes"] = strings.TrimSpace(*req.QualificationNotes)
	}
	if req.LostReason != nil {
		setFields["lost_reason"] = strings.TrimSpace(*req.LostReason)
	}
	if req.ClosingNotes != nil {
		setFields["closing_notes"] = strings.TrimSpace(*req.ClosingNotes)
	}

	finAssessment := existing.FinancialAssessment
	if finAssessment == nil {
		finAssessment = &models.FinancialAssessment{}
	}

	hasFinUpdate := false
	if req.MonthlyIncome != nil {
		finAssessment.MonthlyIncome = *req.MonthlyIncome
		hasFinUpdate = true
	}
	if req.CurrentEMIAmount != nil {
		finAssessment.TotalMonthlyEMI = *req.CurrentEMIAmount
		hasFinUpdate = true
	}
	if req.OverdueDuration != nil {
		finAssessment.OverdueDuration = strings.TrimSpace(*req.OverdueDuration)
		hasFinUpdate = true
	}
	if req.ReasonForDifficulty != nil {
		finAssessment.ReasonForDifficulty = strings.TrimSpace(*req.ReasonForDifficulty)
		hasFinUpdate = true
	}
	if req.DesiredOutcome != nil {
		finAssessment.DesiredOutcome = strings.TrimSpace(*req.DesiredOutcome)
		hasFinUpdate = true
	}

	if req.Debts != nil {
		debtItems := make([]models.DebtItem, 0, len(*req.Debts))
		var totalDebt float64
		for _, d := range *req.Debts {
			lender := strings.TrimSpace(d.Lender)
			if lender == "" {
				continue
			}
			item := models.DebtItem{
				ID:                primitive.NewObjectID(),
				Lender:            lender,
				LoanType:          strings.TrimSpace(d.LoanType),
				OutstandingAmount: d.OutstandingAmount,
				MonthlyEMI:        d.MonthlyEMI,
				Status:            "Active",
			}
			debtItems = append(debtItems, item)
			totalDebt += d.OutstandingAmount
		}
		setFields["debts"] = debtItems
		setFields["total_debt_amount"] = totalDebt
		setFields["outstanding_amount"] = totalDebt
		finAssessment.Debts = debtItems
		finAssessment.TotalDebtAmount = totalDebt
		finAssessment.OutstandingAmount = totalDebt
		hasFinUpdate = true
	} else {
		if req.TotalDebtAmount != nil {
			setFields["total_debt_amount"] = *req.TotalDebtAmount
			finAssessment.TotalDebtAmount = *req.TotalDebtAmount
			hasFinUpdate = true
		}
		if req.OutstandingAmount != nil {
			setFields["outstanding_amount"] = *req.OutstandingAmount
			finAssessment.OutstandingAmount = *req.OutstandingAmount
			hasFinUpdate = true
		}
	}

	if hasFinUpdate {
		setFields["financial_assessment"] = finAssessment
	}

	if req.AssignedTo != nil {
		assignedToStr := strings.TrimSpace(*req.AssignedTo)
		if assignedToStr == "" {
			setFields["assigned_to"] = nil
			setFields["assigned_to_name"] = ""
		} else if empObjID, err := primitive.ObjectIDFromHex(assignedToStr); err == nil {
			if existing.AssignedTo == nil || *existing.AssignedTo != empObjID {
				var emp struct {
					Name string `bson:"name"`
				}
				if err := database.GetCollection("employees").FindOne(ctx, bson.M{"_id": empObjID}).Decode(&emp); err == nil {
					setFields["assigned_to"] = &empObjID
					setFields["assigned_to_name"] = emp.Name

					pushFields["assignment_history"] = models.AssignmentLog{
						ID:             primitive.NewObjectID(),
						AssignedTo:     &empObjID,
						AssignedToName: emp.Name,
						AssignedBy:     creatorID,
						AssignedByName: userName,
						AssignedAt:     now,
						Reason:         "Reassigned via lead update",
					}

					activities = append(activities, models.ActivityItem{
						ID:              primitive.NewObjectID(),
						Type:            "ASSIGNED",
						Description:     fmt.Sprintf("Assigned to counselor %s", emp.Name),
						PerformedByName: userName,
						CreatedAt:       now,
					})
				}
			}
		}
	}

	if req.Conversation != nil {
		convStart := parseFlexibleTime(req.Conversation.CallStartTime)
		convEnd := parseFlexibleTime(req.Conversation.CallEndTime)
		duration := strings.TrimSpace(req.Conversation.CallDuration)
		outcome := strings.TrimSpace(req.Conversation.Outcome)
		discussionNotes := strings.TrimSpace(req.Conversation.DiscussionNotes)
		rawChat := strings.TrimSpace(req.Conversation.RawChat)

		if convStart != nil || convEnd != nil || duration != "" || discussionNotes != "" || rawChat != "" {
			if outcome == "" {
				outcome = "Connected"
			}
			var uploadedAt *time.Time
			if rawChat != "" {
				uploadedAt = &now
			}
			convItem := models.ConversationItem{
				ID:              primitive.NewObjectID(),
				CallStartTime:   convStart,
				CallEndTime:     convEnd,
				Duration:        duration,
				Outcome:         outcome,
				WhatsAppChat:    rawChat,
				DiscussionNotes: discussionNotes,
				UploadedBy:      userName,
				UploadedAt:      uploadedAt,
				CreatedAt:       now,
			}
			pushFields["conversations"] = convItem

			activities = append(activities, models.ActivityItem{
				ID:              primitive.NewObjectID(),
				Type:            "CALL_LOGGED",
				Description:     fmt.Sprintf("Call record logged (%s)", outcome),
				PerformedByName: userName,
				CreatedAt:       now,
			})
		}
	}

	activities = append(activities, models.ActivityItem{
		ID:              primitive.NewObjectID(),
		Type:            "LEAD_UPDATED",
		Description:     "Lead details updated",
		PerformedByName: userName,
		CreatedAt:       now,
	})
	pushFields["activity_timeline"] = bson.M{"$each": activities}

	updateDoc := bson.M{"$set": setFields, "$push": pushFields}

	if _, err := col.UpdateOne(ctx, filter, updateDoc); err != nil {
		response.InternalServerError(c, "Failed to update lead")
		return
	}

	audit.Record(c, audit.RecordParams{
		EntityID:   objID.Hex(),
		EntityType: "lead",
		Action:     "UPDATE",
		OldData:    existing,
		NewData:    setFields,
	})

	response.Success(c, "Lead updated successfully", gin.H{
		"id":         req.ID,
		"leadNumber": existing.LeadNumber,
	})
}
