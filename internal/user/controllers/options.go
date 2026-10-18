package controllers

import (
	"log"
	"strconv"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/user/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Email string `json:"email"`
	Link  string `json:"link,omitempty"`
}

// Options returns active users as lightweight key/label pairs for assignee selects.
func Options(c *gin.Context) {
	filter := bson.M{
		"status":     models.UserStatusActive,
		"deleted_at": bson.M{"$exists": false},
	}

	search := c.Query("search")
	if search == "" {
		search = c.Query("q")
	}

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

	if search != "" {
		filter["$or"] = []bson.M{
			{"name": primitive.Regex{Pattern: search, Options: "i"}},
			{"email": primitive.Regex{Pattern: search, Options: "i"}},
		}
	}

	collection := database.GetCollection("users")
	findOptions := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})
	querySkip := int64((page - 1) * limit)
	findOptions.SetSkip(querySkip)
	findOptions.SetLimit(int64(limit))

	cursor, err := collection.Find(c.Request.Context(), filter, findOptions)
	if err != nil {
		log.Printf("[ERROR] [UserOptions] Failed to retrieve users: %v", err)
		response.InternalServerError(c, "Unable to load team members. Please try again later.")
		return
	}
	defer cursor.Close(c.Request.Context())

	var users []models.User
	if err = cursor.All(c.Request.Context(), &users); err != nil {
		log.Printf("[ERROR] [UserOptions] Failed to decode users: %v", err)
		response.InternalServerError(c, "Unable to load team members. Please try again later.")
		return
	}

	result := make([]UserOption, 0, len(users))
	for _, u := range users {
		result = append(result, UserOption{
			Key:   u.ID.Hex(),
			Value: u.Name,
			Email: u.Email,
			Link:  "/admin/team/" + u.ID.Hex(),
		})
	}

	response.Success(c, "User options fetched", result)
}
