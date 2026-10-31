package controllers

import (
	"fmt"
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

type CreateFollowUpRequest struct {
	LeadID  string `json:"leadId" binding:"required"`
	DueDate string `json:"dueDate" binding:"required"`
	Notes   string `json:"notes"`
}

type UpdateFollowUpStatusRequest struct {
	LeadID     string `json:"leadId" binding:"required"`
	FollowUpID string `json:"followUpId" binding:"required"`
	Status     string `json:"status" binding:"required,oneof=PENDING COMPLETED CANCELLED"`
	Notes      string `json:"notes"`
	DueDate    string `json:"dueDate"`
}

func CreateFollowUp(c *gin.Context) {
	var req CreateFollowUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	leadObjID, err := primitive.ObjectIDFromHex(req.LeadID)
	if err != nil {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	dueDateTime, err := time.Parse(time.RFC3339, req.DueDate)
	if err != nil {
		response.BadRequest(c, "Invalid dueDate format, must be RFC3339")
		return
	}

	userName := "Counselor"
	var defaultAssignedTo *primitive.ObjectID
	if emp, ok := utils.GetAuthEmployee(c); ok && emp.Name != "" {
		userName = emp.Name
		if empObjID, err := primitive.ObjectIDFromHex(emp.EmployeeID); err == nil {
			defaultAssignedTo = &empObjID
		}
	}

	fu := models.FollowUp{
		ID:             primitive.NewObjectID(),
		DueDate:        dueDateTime,
		AssignedTo:     defaultAssignedTo,
		AssignedToName: userName,
		Status:         "PENDING",
		Notes:          req.Notes,
		CreatedAt:      time.Now(),
	}

	col := database.GetCollection("leads")
	filter := bson.M{
		"_id":        leadObjID,
		"deleted_at": bson.M{"$exists": false},
	}
	if tVal, exists := c.Get("tenant_obj_id"); exists {
		if tid, ok := tVal.(primitive.ObjectID); ok {
			filter["tenant_id"] = tid
		}
	}

	updateDoc := bson.M{
		"$push": bson.M{
			"follow_ups": fu,
			"activity_timeline": models.ActivityItem{
				ID:              primitive.NewObjectID(),
				Type:            "FOLLOWUP_ADDED",
				Description:     fmt.Sprintf("Follow-up scheduled for %s", fu.DueDate.Format("02 Jan 2006, 03:04 PM")),
				PerformedByName: userName,
				CreatedAt:       time.Now(),
			},
		},
		"$set": bson.M{
			"status":     models.LeadStatusFollowUp,
			"updated_at": time.Now(),
		},
	}

	res, err := col.UpdateOne(c.Request.Context(), filter, updateDoc)
	if err != nil {
		response.InternalServerError(c, "Failed to schedule follow-up")
		return
	}
	if res.MatchedCount == 0 {
		response.NotFound(c, "Lead not found")
		return
	}

	response.Success(c, "Follow-up scheduled successfully", fu)
}

func UpdateFollowUpStatus(c *gin.Context) {
	var req UpdateFollowUpStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	leadObjID, err := primitive.ObjectIDFromHex(req.LeadID)
	if err != nil {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	followUpObjID, err := primitive.ObjectIDFromHex(req.FollowUpID)
	if err != nil {
		response.BadRequest(c, "Invalid follow-up ID format")
		return
	}

	col := database.GetCollection("leads")
	filter := bson.M{
		"_id":            leadObjID,
		"follow_ups._id": followUpObjID,
		"deleted_at":     bson.M{"$exists": false},
	}
	if tVal, exists := c.Get("tenant_obj_id"); exists {
		if tid, ok := tVal.(primitive.ObjectID); ok {
			filter["tenant_id"] = tid
		}
	}

	userName := "Counselor"
	if emp, ok := utils.GetAuthEmployee(c); ok && emp.Name != "" {
		userName = emp.Name
	}

	now := time.Now()
	setFields := bson.M{
		"follow_ups.$.status": req.Status,
		"updated_at":          now,
	}
	if req.Status == "COMPLETED" {
		setFields["follow_ups.$.completed_at"] = now
	}
	if req.Notes != "" {
		setFields["follow_ups.$.notes"] = req.Notes
	}
	if req.DueDate != "" {
		if newDue, err := time.Parse(time.RFC3339, req.DueDate); err == nil {
			setFields["follow_ups.$.due_date"] = newDue
		}
	}

	updateDoc := bson.M{
		"$set": setFields,
		"$push": bson.M{
			"activity_timeline": models.ActivityItem{
				ID:              primitive.NewObjectID(),
				Type:            "STATUS_CHANGED",
				Description:     fmt.Sprintf("Follow-up marked as %s", req.Status),
				PerformedByName: userName,
				CreatedAt:       now,
			},
		},
	}

	res, err := col.UpdateOne(c.Request.Context(), filter, updateDoc)
	if err != nil {
		response.InternalServerError(c, "Failed to update follow-up status")
		return
	}
	if res.MatchedCount == 0 {
		response.NotFound(c, "Lead or follow-up not found")
		return
	}

	response.Success(c, "Follow-up status updated successfully", nil)
}
