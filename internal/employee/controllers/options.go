package controllers

import (
	"log"
	"regexp"
	"strconv"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/employee/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EmployeeOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Email string `json:"email"`
	Link  string `json:"link,omitempty"`
}

// Options returns active employees as lightweight key/label pairs for assignee selects.
func Options(c *gin.Context) {
	filter := bson.M{
		"status":     models.EmployeeStatusActive,
		"deleted_at": bson.M{"$exists": false},
	}

	search := c.Query("search")

	page := 1
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if limit > 200 {
		limit = 200
	}

	if c.Request.Method == "POST" && c.Request.ContentLength > 0 {
		var req struct {
			Search string `json:"search"`
			Page   int    `json:"page"`
			Limit  int    `json:"limit"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.Search != "" {
				search = req.Search
			}
			if req.Page > 0 {
				page = req.Page
			}
			if req.Limit > 0 {
				limit = req.Limit
			}
		}
	}

	if search != "" {
		escaped := regexp.QuoteMeta(search)
		filter["$or"] = []bson.M{
			{"name": primitive.Regex{Pattern: escaped, Options: "i"}},
			{"email": primitive.Regex{Pattern: escaped, Options: "i"}},
		}
	}

	collection := database.GetCollection("employees")
	findOptions := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})
	querySkip := int64((page - 1) * limit)
	findOptions.SetSkip(querySkip)
	findOptions.SetLimit(int64(limit))

	cursor, err := collection.Find(c.Request.Context(), filter, findOptions)
	if err != nil {
		log.Printf("[ERROR] [EmployeeOptions] Failed to retrieve employees: %v", err)
		response.InternalServerError(c, "Unable to load employees. Please try again later.")
		return
	}
	defer cursor.Close(c.Request.Context())

	var employees []models.Employee
	if err = cursor.All(c.Request.Context(), &employees); err != nil {
		log.Printf("[ERROR] [EmployeeOptions] Failed to decode employees: %v", err)
		response.InternalServerError(c, "Unable to load employees. Please try again later.")
		return
	}

	result := make([]EmployeeOption, 0, len(employees))
	for _, u := range employees {
		result = append(result, EmployeeOption{
			Key:   u.ID.Hex(),
			Value: u.Name,
			Email: u.Email,
			Link:  "/admin/employee/" + u.ID.Hex(),
		})
	}

	response.Success(c, "Employee options fetched", result)
}
