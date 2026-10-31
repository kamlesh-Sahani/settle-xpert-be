package controllers

import (
	"regexp"
	"strconv"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/role/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RoleOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Link  string `json:"link,omitempty"`
}

func Options(c *gin.Context) {
	search := c.Query("search")

	page := 1
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if limit > 100 {
		limit = 100
	}

	// Also support JSON payload if POST request
	if c.Request.Method == "POST" && c.Request.ContentLength > 0 {
		var req struct {
			Search string `json:"search"`
			Q      string `json:"q"`
			Page   int    `json:"page"`
			Limit  int    `json:"limit"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.Search != "" {
				search = req.Search
			} else if req.Q != "" {
				search = req.Q
			}
			if req.Page > 0 {
				page = req.Page
			}
			if req.Limit > 0 {
				limit = req.Limit
			}
		}
	}

	collection := database.GetCollection("roles")
	filter := bson.M{
		"is_deleted": bson.M{"$ne": true},
		"status":     bson.M{"$ne": models.RoleStatusInactive},
	}

	if search != "" {
		filter["name"] = primitive.Regex{Pattern: regexp.QuoteMeta(search), Options: "i"}
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "name", Value: 1}})

	querySkip := int64((page - 1) * limit)
	queryLimit := int64(limit)
	findOptions.SetSkip(querySkip)
	findOptions.SetLimit(queryLimit)

	cursor, err := collection.Find(c.Request.Context(), filter, findOptions)
	if err != nil {
		response.InternalServerError(c, "Unable to retrieve roles. Please try again later.")
		return
	}
	defer cursor.Close(c.Request.Context())

	var roles []models.Role
	if err = cursor.All(c.Request.Context(), &roles); err != nil {
		response.InternalServerError(c, "Unable to load roles. Please try again later.")
		return
	}

	result := make([]RoleOption, 0, len(roles))
	for _, r := range roles {
		result = append(result, RoleOption{
			Key:   r.ID.Hex(),
			Value: r.Name,
			Link:  "/admin/roles/" + r.ID.Hex(),
		})
	}

	response.Success(c, "Roles retrieved successfully", result)
}
