package controllers

import (
	"log"
	"os"
	"strconv"
	"time"

	"lead-crm/backend/database"
	authmodels "lead-crm/backend/internal/auth/models"
	"lead-crm/backend/internal/user/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = c.ShouldBindJSON(&req)
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		response.Unauthorized(c, "Refresh token required")
		return
	}

	claims, err := utils.ParseJWTToken(refreshToken)
	if err != nil {
		response.Unauthorized(c, "Invalid refresh token")
		return
	}

	sessionCol := database.GetCollection("sessions")
	sessionID, _ := primitive.ObjectIDFromHex(claims.SessionID)

	var session authmodels.Session
	err = sessionCol.FindOne(c.Request.Context(), bson.M{
		"_id":           sessionID,
		"refresh_token": refreshToken,
		"expires_at":    bson.M{"$gt": time.Now()},
	}).Decode(&session)

	if err != nil {
		response.Unauthorized(c, "Session expired or invalid")
		return
	}

	userCol := database.GetCollection("users")
	var user models.User
	err = userCol.FindOne(c.Request.Context(), bson.M{
		"_id":        session.UserID,
		"status":     models.UserStatusActive,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&user)

	if err != nil {
		response.Unauthorized(c, "User account not active")
		return
	}

	newAccessToken, err := utils.GenerateAccessToken(user.ID.Hex(), user.Email, user.Name, session.ID.Hex())
	if err != nil {
		log.Printf("[ERROR] [Refresh] Failed to generate access token: %v", err)
		response.InternalServerError(c, "Failed to refresh session")
		return
	}

	isProd := os.Getenv("GIN_MODE") == "release"
	accessExpiryMinutes, _ := strconv.Atoi(os.Getenv("JWT_EXPIRY_MINUTES"))
	if accessExpiryMinutes <= 0 {
		accessExpiryMinutes = 120
	}

	c.SetCookie("access_token", newAccessToken, accessExpiryMinutes*60, "/", "", isProd, true)

	response.Success(c, "Token refreshed successfully", gin.H{
		"token": newAccessToken,
	})
}
