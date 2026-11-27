package controllers

import (
	"errors"
	"log"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreateCallScheduleRequest struct {
	LeadID     string              `json:"leadId" binding:"required,max=24"`
	Date       time.Time           `json:"date" binding:"required"`
	Priority   models.CallPriority `json:"priority" binding:"omitempty,oneof=LOW MEDIUM HIGH"`
	CallStatus models.CallStatus   `json:"callStatus" binding:"omitempty,oneof=CONNECTED BUSY RINGING SWITCHED_OFF DISCONNECTED_BY_CLIENT"`
	Remark     string              `json:"remark" binding:"omitempty,max=1000"`
}

type UpdateCallScheduleRequest struct {
	LeadID     string                    `json:"leadId" binding:"required,max=24"`
	ScheduleID string                    `json:"scheduleId" binding:"required,max=24"`
	Status     models.CallScheduleStatus `json:"status" binding:"required,oneof=PENDING COMPLETED"`
	CallStatus models.CallStatus         `json:"callStatus" binding:"omitempty,oneof=CONNECTED BUSY RINGING SWITCHED_OFF DISCONNECTED_BY_CLIENT"`
	LeadStatus models.LeadStatus         `json:"leadStatus" binding:"omitempty,oneof=NEW CALL_SCHEDULED INTERESTED NOT_INTERESTED CONVERTED"`
	Remark     string                    `json:"remark" binding:"omitempty,max=1000"`
}

func CreateCallSchedule(c *gin.Context) {
	var req CreateCallScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	leadObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.LeadID))
	if err != nil || leadObjID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	now := time.Now().UTC()
	priority := req.Priority
	if priority == "" {
		priority = models.CallPriorityMedium
	}
	cs := models.CallSchedule{
		ID:         primitive.NewObjectID(),
		Date:       req.Date,
		Status:     models.CallScheduleStatusPending,
		Priority:   priority,
		CallStatus: req.CallStatus,
		Remark:     strings.TrimSpace(req.Remark),
		CreatedAt:  now,
	}

	filter := bson.M{
		"_id":        leadObjID,
		"deleted_at": bson.M{"$exists": false},
	}

	var existing struct {
		Status        models.LeadStatus     `bson:"status"`
		CallSchedules []models.CallSchedule `bson:"call_schedules"`
	}
	if err := col.FindOne(c.Request.Context(), filter).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [CallScheduleCreate] Lookup failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	for _, item := range existing.CallSchedules {
		if item.Status == models.CallScheduleStatusPending {
			response.BadRequest(c, "A call is already scheduled for this lead. Complete it before scheduling a new one.")
			return
		}
	}

	setFields := bson.M{"updated_at": now}
	if existing.Status != models.LeadStatusConverted {
		setFields["status"] = models.LeadStatusCallScheduled
	}

	updateFilter := bson.M{
		"_id":        leadObjID,
		"deleted_at": bson.M{"$exists": false},
		"call_schedules": bson.M{
			"$not": bson.M{
				"$elemMatch": bson.M{"status": models.CallScheduleStatusPending},
			},
		},
	}

	updateDoc := bson.M{
		"$push": bson.M{"call_schedules": cs},
		"$set":  setFields,
	}

	res, err := col.UpdateOne(c.Request.Context(), updateFilter, updateDoc)
	if err != nil {
		log.Printf("[ERROR] [CallScheduleCreate] Update failed: %v", err)
		response.InternalServerError(c, "Failed to schedule call")
		return
	}
	if res.MatchedCount == 0 {
		response.BadRequest(c, "A call is already scheduled for this lead. Complete it before scheduling a new one.")
		return
	}

	response.Success(c, "Call scheduled successfully", cs)
}

func UpdateCallScheduleStatus(c *gin.Context) {
	var req UpdateCallScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	leadObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.LeadID))
	if err != nil || leadObjID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	scheduleObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.ScheduleID))
	if err != nil || scheduleObjID.IsZero() {
		response.BadRequest(c, "Invalid call schedule ID format")
		return
	}

	col := database.GetCollection("leads")
	if col == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	now := time.Now().UTC()
	setFields := bson.M{
		"call_schedules.$.status": req.Status,
		"updated_at":              now,
	}
	updateDoc := bson.M{"$set": setFields}

	if req.Status == models.CallScheduleStatusCompleted {
		setFields["call_schedules.$.completed_at"] = now
	} else {
		updateDoc["$unset"] = bson.M{"call_schedules.$.completed_at": ""}
	}

	if req.CallStatus != "" {
		setFields["call_schedules.$.call_status"] = req.CallStatus
	}
	if remark := strings.TrimSpace(req.Remark); remark != "" {
		setFields["call_schedules.$.remark"] = remark
	}

	if req.LeadStatus != "" {
		setFields["status"] = req.LeadStatus
	}

	filter := bson.M{
		"_id":                leadObjID,
		"call_schedules._id": scheduleObjID,
		"deleted_at":         bson.M{"$exists": false},
	}

	res, err := col.UpdateOne(c.Request.Context(), filter, updateDoc)
	if err != nil {
		log.Printf("[ERROR] [CallScheduleUpdate] Update failed: %v", err)
		response.InternalServerError(c, "Failed to update call schedule")
		return
	}
	if res.MatchedCount == 0 {
		response.NotFound(c, "Lead or call schedule not found")
		return
	}

	response.Success(c, "Call schedule updated successfully", nil)
}
