package controllers

import (
	"fmt"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func UpdateStatus(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		response.BadRequest(c, "Lead ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	statusStr := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if statusStr == "" {
		response.BadRequest(c, "Status is required")
		return
	}

	newStatus := models.LeadStatus(statusStr)
	switch newStatus {
	case models.LeadStatusNew,
		models.LeadStatusContacted,
		models.LeadStatusQualified,
		models.LeadStatusUnqualified,
		models.LeadStatusFollowUp,
		models.LeadStatusReengaged,
		models.LeadStatusUnreachable,
		models.LeadStatusConverted,
		models.LeadStatusLost,
		models.LeadStatusInProgress:
	default:
		response.BadRequest(c, "Invalid lead status")
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

	userName := "User"
	if emp, ok := utils.GetAuthEmployee(c); ok && emp.Name != "" {
		userName = emp.Name
	}

	now := time.Now()
	activity := models.ActivityItem{
		ID:              primitive.NewObjectID(),
		Type:            "STATUS_CHANGED",
		Description:     fmt.Sprintf("Status changed to %s", string(newStatus)),
		PerformedByName: userName,
		CreatedAt:       now,
	}

	setFields := bson.M{
		"status":     newStatus,
		"updated_at": now,
	}
	if newStatus == models.LeadStatusConverted {
		setFields["converted_at"] = now
	}

	update := bson.M{
		"$set":  setFields,
		"$push": bson.M{"activity_timeline": activity},
	}

	result, err := col.UpdateOne(ctx, filter, update)
	if err != nil {
		response.InternalServerError(c, "Failed to update lead status")
		return
	}
	if result.MatchedCount == 0 {
		response.NotFound(c, "Lead not found")
		return
	}

	response.Success(c, "Lead status updated successfully", gin.H{
		"id":     idStr,
		"status": newStatus,
	})
}
