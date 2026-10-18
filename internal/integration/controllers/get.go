package controllers

import (
	"lead-crm/backend/database"
	"lead-crm/backend/internal/integration/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
)

func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// Get returns the Meta Ads configuration
func Get(c *gin.Context) {
	coll := database.GetCollection("integrations")
	var config models.MetaIntegrationConfig

	if err := coll.FindOne(c.Request.Context(), bson.M{"provider": "META_ADS"}).Decode(&config); err != nil {
		config = models.MetaIntegrationConfig{
			Provider: "META_ADS",
			IsActive: false,
		}
	}

	res := gin.H{
		"id":                config.ID.Hex(),
		"provider":          config.Provider,
		"isActive":          config.IsActive,
		"verifyToken":       config.VerifyToken,
		"pageId":            config.PageID,
		"pageName":          config.PageName,
		"hasAccessToken":    config.PageAccessToken != "",
		"maskedAccessToken": maskToken(config.PageAccessToken),
		"hasAppSecret":      config.AppSecret != "",
		"fieldMapping":      config.FieldMapping,
		"updatedAt":         config.UpdatedAt,
	}

	response.Success(c, "Meta Ads integration configuration fetched", res)
}
