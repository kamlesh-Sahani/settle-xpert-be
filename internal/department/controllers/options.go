package controllers

import (
	"regexp"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/department/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DepartmentOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Code  string `json:"code,omitempty"`
}

func Options(c *gin.Context) {
	search := c.Query("search")

	col := database.GetCollection("departments")
	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
		"status":     models.DepartmentStatusActive,
	}

	if search != "" {
		escaped := regexp.QuoteMeta(search)
		filter["$or"] = []bson.M{
			{"name": primitive.Regex{Pattern: escaped, Options: "i"}},
			{"code": primitive.Regex{Pattern: escaped, Options: "i"}},
		}
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetLimit(100)
	cursor, err := col.Find(c.Request.Context(), filter, findOptions)
	if err != nil {
		response.InternalServerError(c, "Unable to load departments")
		return
	}
	defer cursor.Close(c.Request.Context())

	var depts []models.Department
	if err = cursor.All(c.Request.Context(), &depts); err != nil {
		response.InternalServerError(c, "Unable to parse departments")
		return
	}

	res := make([]DepartmentOption, 0, len(depts))
	for _, d := range depts {
		res = append(res, DepartmentOption{
			Key:   d.ID.Hex(),
			Value: d.Name,
			Code:  d.Code,
		})
	}

	response.Success(c, "Departments loaded successfully", res)
}
