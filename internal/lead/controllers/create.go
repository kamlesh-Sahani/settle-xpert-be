package controllers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var digitCleaner = regexp.MustCompile(`\D`)

func parseFlexibleTime(val string) *time.Time {
	s := strings.TrimSpace(val)
	if s == "" {
		return nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02/01/2006 03:04 PM",
		"02/01/2006 15:04",
		"02/01/2006",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

type CreateLeadDebtItem struct {
	Lender            string  `json:"lender"`
	LoanType          string  `json:"loanType"`
	OutstandingAmount float64 `json:"outstandingAmount"`
	MonthlyEMI        float64 `json:"monthlyEmi"`
}

type CreateConversationPayload struct {
	CallStartTime   string `json:"callStartTime"`
	CallEndTime     string `json:"callEndTime"`
	CallDuration    string `json:"callDuration"`
	Outcome         string `json:"outcome"`
	DiscussionNotes string `json:"discussionNotes"`
	RawChat         string `json:"rawChat"`
}

type CreateLeadRequest struct {
	FullName           string                     `json:"fullName" binding:"required"`
	Phone              string                     `json:"phone" binding:"required"`
	AlternatePhone     string                     `json:"alternatePhone"`
	Email              string                     `json:"email"`
	DOB                string                     `json:"dob"`
	PANCard            string                     `json:"panCard"`
	Address            *models.Address            `json:"address"`
	City               string                     `json:"city"`
	State              string                     `json:"state"`
	ServiceRequired    string                     `json:"serviceRequired" binding:"required"`
	Source             models.LeadSource          `json:"source" binding:"required"`
	Priority           models.LeadPriority        `json:"priority" binding:"omitempty,oneof=LOW NORMAL HIGH"`
	Status             models.LeadStatus          `json:"status" binding:"omitempty,oneof=NEW IN_PROGRESS CONVERTED LOST"`
	InitialEnquiry     string                     `json:"initialEnquiry"`
	QualificationNotes string                     `json:"qualificationNotes"`
	TotalDebtAmount    float64                    `json:"totalDebtAmount"`
	OutstandingAmount  float64                    `json:"outstandingAmount"`
	MonthlyIncome      float64                    `json:"monthlyIncome"`
	CurrentEMIAmount   float64                    `json:"currentEmiAmount"`
	Debts              []CreateLeadDebtItem       `json:"debts"`
	AssignedTo         string                     `json:"assignedTo"`
	Conversation       *CreateConversationPayload `json:"conversation"`
}

func Create(c *gin.Context) {
	var req CreateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	col := database.GetCollection("leads")
	ctx := c.Request.Context()

	// Resolve creator employee info
	creatorName := "System"
	var creatorID *primitive.ObjectID
	if empCtx, ok := utils.GetAuthEmployee(c); ok && empCtx != nil {
		creatorName = empCtx.Name
		if oid, err := primitive.ObjectIDFromHex(empCtx.EmployeeID); err == nil {
			creatorID = &oid
		}
	}

	rawPhone := strings.TrimSpace(req.Phone)
	cleanedPhone := digitCleaner.ReplaceAllString(rawPhone, "")
	last10 := cleanedPhone
	if len(cleanedPhone) > 10 {
		last10 = cleanedPhone[len(cleanedPhone)-10:]
	}

	now := time.Now()

	// Prepare Debts & Financial Assessment
	totalDebt := req.TotalDebtAmount
	if totalDebt == 0 && req.OutstandingAmount > 0 {
		totalDebt = req.OutstandingAmount
	}

	debtItems := make([]models.DebtItem, 0, len(req.Debts))
	var debtSum float64
	for _, d := range req.Debts {
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
		debtSum += d.OutstandingAmount
	}
	if len(debtItems) > 0 {
		totalDebt = debtSum
	}

	finAssessment := &models.FinancialAssessment{
		MonthlyIncome:     req.MonthlyIncome,
		TotalMonthlyEMI:   req.CurrentEMIAmount,
		TotalDebtAmount:   totalDebt,
		OutstandingAmount: totalDebt,
		Debts:             debtItems,
	}

	// Prepare unified initial conversation record
	initialConversations := make([]models.ConversationItem, 0)
	if req.Conversation != nil {
		callStart := parseFlexibleTime(req.Conversation.CallStartTime)
		callEnd := parseFlexibleTime(req.Conversation.CallEndTime)
		duration := strings.TrimSpace(req.Conversation.CallDuration)
		outcome := strings.TrimSpace(req.Conversation.Outcome)
		discussionNotes := strings.TrimSpace(req.Conversation.DiscussionNotes)
		whatsappChat := strings.TrimSpace(req.Conversation.RawChat)

		if callStart != nil || callEnd != nil || duration != "" || discussionNotes != "" || whatsappChat != "" {
			if outcome == "" {
				outcome = "Connected"
			}
			var uploadedAt *time.Time
			if whatsappChat != "" {
				uploadedAt = &now
			}
			initialConversations = append(initialConversations, models.ConversationItem{
				ID:              primitive.NewObjectID(),
				CallStartTime:   callStart,
				CallEndTime:     callEnd,
				Duration:        duration,
				Outcome:         outcome,
				WhatsAppChat:    whatsappChat,
				DiscussionNotes: discussionNotes,
				UploadedBy:      creatorName,
				UploadedAt:      uploadedAt,
				CreatedAt:       now,
			})
		}
	}

	// ─── Deduplication Check (Same Client Returning) ─────────────────────────
	var existing models.Lead
	isDuplicateFound := false
	if len(last10) >= 6 {
		err := col.FindOne(ctx, bson.M{
			"phone":      bson.M{"$regex": last10 + "$"},
			"deleted_at": bson.M{"$exists": false},
		}).Decode(&existing)
		if err == nil {
			isDuplicateFound = true
		}
	}

	if isDuplicateFound {
		repeatCount := existing.RepeatEnquiryCount + 1

		newEnquiry := models.EnquiryItem{
			EnquiryNumber:   repeatCount + 1,
			Date:            now,
			Source:          req.Source,
			ServiceRequired: strings.TrimSpace(req.ServiceRequired),
			Message:         strings.TrimSpace(req.InitialEnquiry),
		}

		// Merge newly submitted debts with existing debts (Option A)
		mergedDebts := existing.Debts
		for _, newItem := range debtItems {
			exists := false
			for i, oldItem := range mergedDebts {
				if strings.EqualFold(oldItem.Lender, newItem.Lender) && strings.EqualFold(oldItem.LoanType, newItem.LoanType) {
					mergedDebts[i].OutstandingAmount = newItem.OutstandingAmount
					mergedDebts[i].MonthlyEMI = newItem.MonthlyEMI
					exists = true
					break
				}
			}
			if !exists {
				mergedDebts = append(mergedDebts, newItem)
			}
		}

		var calculatedDebt float64
		for _, d := range mergedDebts {
			calculatedDebt += d.OutstandingAmount
		}
		if calculatedDebt == 0 && totalDebt > 0 {
			calculatedDebt = totalDebt
		}

		activityDesc := fmt.Sprintf("Repeat enquiry #%d received via %s", repeatCount+1, req.Source)
		if len(debtItems) > 0 {
			activityDesc += fmt.Sprintf(" (Added/updated %d loan(s), total debt: ₹%.0f)", len(debtItems), calculatedDebt)
		}
		if req.InitialEnquiry != "" {
			activityDesc += fmt.Sprintf(": \"%s\"", strings.TrimSpace(req.InitialEnquiry))
		}

		activityItems := []models.ActivityItem{
			{
				ID:              primitive.NewObjectID(),
				Type:            "REPEAT_ENQUIRY",
				Description:     activityDesc,
				PerformedByName: creatorName,
				CreatedAt:       now,
			},
		}

		for _, conv := range initialConversations {
			if conv.CallStartTime != nil || conv.DiscussionNotes != "" {
				desc := fmt.Sprintf("Call logged: %s", conv.Outcome)
				if conv.DiscussionNotes != "" {
					desc += fmt.Sprintf(" — %s", conv.DiscussionNotes)
				}
				activityItems = append(activityItems, models.ActivityItem{
					ID:              primitive.NewObjectID(),
					Type:            "CALL_LOGGED",
					Description:     desc,
					PerformedByName: creatorName,
					CreatedAt:       now,
				})
			}
			if conv.WhatsAppChat != "" {
				activityItems = append(activityItems, models.ActivityItem{
					ID:              primitive.NewObjectID(),
					Type:            "WHATSAPP_LOGGED",
					Description:     "WhatsApp chat transcript uploaded on repeat enquiry",
					PerformedByName: creatorName,
					CreatedAt:       now,
				})
			}
		}

		setUpdates := bson.M{
			"is_reengaged":         true,
			"repeat_enquiry_count": repeatCount,
			"last_enquiry_at":      now,
			"updated_at":           now,
		}

		if len(mergedDebts) > 0 || calculatedDebt > 0 {
			finAssessment.Debts = mergedDebts
			finAssessment.TotalDebtAmount = calculatedDebt
			finAssessment.OutstandingAmount = calculatedDebt
			if req.MonthlyIncome == 0 && existing.FinancialAssessment != nil {
				finAssessment.MonthlyIncome = existing.FinancialAssessment.MonthlyIncome
			}
			if req.CurrentEMIAmount == 0 && existing.FinancialAssessment != nil {
				finAssessment.TotalMonthlyEMI = existing.FinancialAssessment.TotalMonthlyEMI
			}
			setUpdates["financial_assessment"] = finAssessment
			setUpdates["debts"] = mergedDebts
			setUpdates["total_debt_amount"] = calculatedDebt
			setUpdates["outstanding_amount"] = calculatedDebt
		}

		if req.Address != nil {
			setUpdates["address"] = req.Address
		}
		if req.DOB != "" {
			setUpdates["dob"] = req.DOB
		}
		if req.PANCard != "" {
			setUpdates["pan_card"] = req.PANCard
		}
		if req.AlternatePhone != "" {
			setUpdates["alternate_phone"] = req.AlternatePhone
		}

		// Status transitions on duplicate inquiry (Option A for QUALIFIED)
		switch existing.Status {
		case models.LeadStatusLost, models.LeadStatusUnreachable, models.LeadStatusUnqualified:
			setUpdates["status"] = models.LeadStatusReengaged
			setUpdates["priority"] = models.PriorityHigh

		case models.LeadStatusQualified:
			// Option A: Retain QUALIFIED stage, elevate priority to HIGH for agent review
			setUpdates["priority"] = models.PriorityHigh

		case models.LeadStatusConverted:
			// Existing converted client returning for new loan/service
			setUpdates["priority"] = models.PriorityHigh

		default:
			// Active pipeline stages (NEW, CONTACTED, FOLLOW_UP, IN_PROGRESS, RE_ENGAGED)
			setUpdates["priority"] = models.PriorityHigh
		}

		pushUpdates := bson.M{
			"enquiries":         newEnquiry,
			"activity_timeline": bson.M{"$each": activityItems},
		}

		if len(initialConversations) > 0 {
			pushUpdates["conversations"] = bson.M{"$each": initialConversations}
		}

		// Reassignment if an explicit new assignee is requested
		if req.AssignedTo != "" {
			if objID, err := primitive.ObjectIDFromHex(req.AssignedTo); err == nil {
				if existing.AssignedTo == nil || *existing.AssignedTo != objID {
					var emp struct {
						Name string `bson:"name"`
					}
					if err := database.GetCollection("employees").FindOne(ctx, bson.M{"_id": objID}).Decode(&emp); err == nil {
						setUpdates["assigned_to"] = &objID
						setUpdates["assigned_to_name"] = emp.Name
						pushUpdates["assignment_history"] = models.AssignmentLog{
							ID:             primitive.NewObjectID(),
							AssignedTo:     &objID,
							AssignedToName: emp.Name,
							AssignedBy:     creatorID,
							AssignedByName: creatorName,
							AssignedAt:     now,
							Reason:         "Reassigned during repeat enquiry",
						}
					}
				}
			}
		}

		updateDoc := bson.M{
			"$push": pushUpdates,
			"$set":  setUpdates,
		}

		if _, err := col.UpdateOne(ctx, bson.M{"_id": existing.ID}, updateDoc); err != nil {
			response.InternalServerError(c, "Failed to update existing lead with repeat enquiry")
			return
		}

		response.Success(c, "Repeat enquiry recorded for existing lead", gin.H{
			"leadNumber":  existing.LeadNumber,
			"isDuplicate": true,
			"message":     fmt.Sprintf("Existing lead %s re-engaged", existing.LeadNumber),
		})
		return
	}

	// ─── Brand New Lead Creation ─────────────────────────────────────────────
	priority := models.PriorityNormal
	if req.Priority != "" {
		priority = req.Priority
	}

	var assignedToObjID *primitive.ObjectID
	assignedToName := ""
	assignmentHistory := []models.AssignmentLog{}

	if req.AssignedTo != "" {
		if objID, err := primitive.ObjectIDFromHex(req.AssignedTo); err == nil {
			assignedToObjID = &objID
			var emp struct {
				Name string `bson:"name"`
			}
			empCol := database.GetCollection("employees")
			if err := empCol.FindOne(ctx, bson.M{"_id": objID}).Decode(&emp); err == nil {
				assignedToName = emp.Name
				assignmentHistory = append(assignmentHistory, models.AssignmentLog{
					ID:             primitive.NewObjectID(),
					AssignedTo:     assignedToObjID,
					AssignedToName: assignedToName,
					AssignedBy:     creatorID,
					AssignedByName: creatorName,
					AssignedAt:     now,
					Reason:         "Initial manual assignment during lead creation",
				})
			}
		}
	}

	count, _ := col.CountDocuments(ctx, bson.M{})
	seq := count + 1
	leadNumber := fmt.Sprintf("LEAD-%04d", seq)
	for {
		existsCount, _ := col.CountDocuments(ctx, bson.M{"lead_number": leadNumber})
		if existsCount == 0 {
			break
		}
		seq++
		leadNumber = fmt.Sprintf("LEAD-%04d", seq)
	}

	activityTimeline := []models.ActivityItem{
		{
			ID:              primitive.NewObjectID(),
			Type:            "LEAD_CREATED",
			Description:     fmt.Sprintf("Lead %s created via %s", leadNumber, req.Source),
			PerformedByName: creatorName,
			CreatedAt:       now,
		},
	}

	if assignedToName != "" {
		activityTimeline = append(activityTimeline, models.ActivityItem{
			ID:              primitive.NewObjectID(),
			Type:            "ASSIGNED",
			Description:     fmt.Sprintf("Assigned to %s", assignedToName),
			PerformedByName: creatorName,
			CreatedAt:       now,
		})
	}

	for _, conv := range initialConversations {
		if conv.CallStartTime != nil || conv.DiscussionNotes != "" {
			desc := fmt.Sprintf("Call logged: %s", conv.Outcome)
			if conv.DiscussionNotes != "" {
				desc += fmt.Sprintf(" — %s", conv.DiscussionNotes)
			}
			activityTimeline = append(activityTimeline, models.ActivityItem{
				ID:              primitive.NewObjectID(),
				Type:            "CALL_LOGGED",
				Description:     desc,
				PerformedByName: creatorName,
				CreatedAt:       now,
			})
		}
		if conv.WhatsAppChat != "" {
			activityTimeline = append(activityTimeline, models.ActivityItem{
				ID:              primitive.NewObjectID(),
				Type:            "WHATSAPP_LOGGED",
				Description:     "Initial WhatsApp chat transcript uploaded",
				PerformedByName: creatorName,
				CreatedAt:       now,
			})
		}
	}

	initialEnquiryItem := models.EnquiryItem{
		EnquiryNumber:   1,
		Date:            now,
		Source:          req.Source,
		ServiceRequired: strings.TrimSpace(req.ServiceRequired),
		Message:         strings.TrimSpace(req.InitialEnquiry),
	}

	city := req.City
	state := req.State
	if req.Address != nil {
		if req.Address.City != "" {
			city = req.Address.City
		}
		if req.Address.State != "" {
			state = req.Address.State
		}
	}

	initialStatus := models.LeadStatusNew
	if req.Status != "" {
		initialStatus = req.Status
	}

	newLead := models.Lead{
		ID:                  primitive.NewObjectID(),
		LeadNumber:          leadNumber,
		FullName:            strings.TrimSpace(req.FullName),
		Phone:               strings.TrimSpace(req.Phone),
		AlternatePhone:      strings.TrimSpace(req.AlternatePhone),
		Email:               strings.TrimSpace(req.Email),
		DOB:                 strings.TrimSpace(req.DOB),
		PANCard:             strings.TrimSpace(req.PANCard),
		Address:             req.Address,
		City:                strings.TrimSpace(city),
		State:               strings.TrimSpace(state),
		ServiceRequired:     strings.TrimSpace(req.ServiceRequired),
		Source:              req.Source,
		Priority:            priority,
		InitialEnquiry:      strings.TrimSpace(req.InitialEnquiry),
		QualificationNotes:  strings.TrimSpace(req.QualificationNotes),
		Status:              initialStatus,
		TotalDebtAmount:     totalDebt,
		OutstandingAmount:   totalDebt,
		AssignedTo:          assignedToObjID,
		AssignedToName:      assignedToName,
		AssignmentHistory:   assignmentHistory,
		Enquiries:           []models.EnquiryItem{initialEnquiryItem},
		Debts:               debtItems,
		Conversations:       initialConversations,
		FollowUps:           []models.FollowUp{},
		ActivityTimeline:    activityTimeline,
		FinancialAssessment: finAssessment,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if _, err := col.InsertOne(ctx, newLead); err != nil {
		response.InternalServerError(c, "Failed to create lead")
		return
	}

	response.Success(c, "Lead created successfully", nil)
}
