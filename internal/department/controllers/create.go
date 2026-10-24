package controllers

import (
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/audit"
	"lead-crm/backend/internal/department/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreateDepartmentRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Code        string                  `json:"code" binding:"required"`
	Description string                  `json:"description"`
	Status      models.DepartmentStatus `json:"status" binding:"omitempty,oneof=ACTIVE IN_ACTIVE"`
}

func Create(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	col := database.GetCollection("departments")
	ctx := c.Request.Context()

	name := strings.TrimSpace(req.Name)
	code := strings.ToUpper(strings.TrimSpace(req.Code))

	var existing models.Department
	err := col.FindOne(ctx, bson.M{
		"deleted_at": bson.M{"$exists": false},
		"$or": []bson.M{
			{"name": primitive.Regex{Pattern: "^" + name + "$", Options: "i"}},
			{"code": code},
		},
	}).Decode(&existing)
	if err == nil {
		if strings.EqualFold(existing.Name, name) {
			response.Conflict(c, "Department with this name already exists")
			return
		}
		response.Conflict(c, "Department with this code already exists")
		return
	}

	status := models.DepartmentStatusActive
	if req.Status != "" {
		status = req.Status
	}

	now := time.Now()
	dept := models.Department{
		Name:        name,
		Code:        code,
		Description: strings.TrimSpace(req.Description),
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	res, err := col.InsertOne(ctx, dept)
	if err != nil {
		response.InternalServerError(c, "Failed to create department")
		return
	}

	dept.ID = res.InsertedID.(primitive.ObjectID)

	audit.Record(c, audit.RecordParams{
		EntityID:   dept.ID.Hex(),
		EntityType: "department",
		Action:     "CREATE",
		NewData:    dept,
	})

	response.Created(c, "Department created successfully", dept)
}
