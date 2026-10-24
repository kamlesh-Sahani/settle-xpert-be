package controllers

import (
	"lead-crm/backend/database"
	"lead-crm/backend/internal/department/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetByID(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		response.BadRequest(c, "Department ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid department ID format")
		return
	}

	col := database.GetCollection("departments")
	var dept models.Department
	err = col.FindOne(c.Request.Context(), bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&dept)
	if err != nil {
		response.NotFound(c, "Department not found")
		return
	}

	response.Success(c, "Department fetched successfully", dept)
}
