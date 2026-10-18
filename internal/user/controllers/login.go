package controllers

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"lead-crm/backend/database"
	authmodels "lead-crm/backend/internal/auth/models"
	"lead-crm/backend/internal/user/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	userCol := database.GetCollection("users")
	var user models.User

	err := userCol.FindOne(c.Request.Context(), bson.M{
		"email":      req.Email,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&user)

	if err != nil {
		response.Unauthorized(c, "Invalid email or password")
		return
	}

	if strings.ToUpper(user.Status) != models.UserStatusActive {
		response.Unauthorized(c, "Account is not active")
		return
	}

	if err := utils.VerifyHash(req.Password, user.Password); err != nil {
		response.Unauthorized(c, "Invalid email or password")
		return
	}

	sessionCol := database.GetCollection("sessions")
	// Clear any previous active sessions for this user
	_, _ = sessionCol.DeleteMany(c.Request.Context(), bson.M{"user_id": user.ID})

	sessionID := primitive.NewObjectID()

	accessToken, err := utils.GenerateAccessToken(user.ID.Hex(), user.Email, user.Name, sessionID.Hex())
	if err != nil {
		log.Printf("[ERROR] [Login] Failed to generate access token: %v", err)
		response.InternalServerError(c, "Unable to sign in. Please try again.")
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID.Hex(), sessionID.Hex())
	if err != nil {
		log.Printf("[ERROR] [Login] Failed to generate refresh token: %v", err)
		response.InternalServerError(c, "Unable to sign in. Please try again.")
		return
	}

	refreshExpiryMinutes, _ := strconv.Atoi(os.Getenv("JWT_REFRESH_EXPIRY_MINUTES"))
	if refreshExpiryMinutes <= 0 {
		refreshExpiryMinutes = 43200 // 30 days
	}
	expiresAt := time.Now().Add(time.Duration(refreshExpiryMinutes) * time.Minute)

	newSession := authmodels.Session{
		ID:           sessionID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    c.Request.UserAgent(),
		IPAddress:    c.ClientIP(),
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}

	_, err = sessionCol.InsertOne(c.Request.Context(), newSession)
	if err != nil {
		log.Printf("[ERROR] [Login] Failed to create session: %v", err)
		response.InternalServerError(c, "Unable to sign in. Please try again.")
		return
	}

	isProd := os.Getenv("GIN_MODE") == "release"
	accessExpiryMinutes, _ := strconv.Atoi(os.Getenv("JWT_EXPIRY_MINUTES"))
	if accessExpiryMinutes <= 0 {
		accessExpiryMinutes = 120
	}

	c.SetCookie("access_token", accessToken, accessExpiryMinutes*60, "/", "", isProd, true)
	c.SetCookie("refresh_token", refreshToken, refreshExpiryMinutes*60, "/", "", isProd, true)

	var roleIDStr string
	if user.RoleID != nil {
		roleIDStr = user.RoleID.Hex()
	}

	response.Success(c, "Login successful", gin.H{
		"token": accessToken,
		"user": gin.H{
			"id":     user.ID.Hex(),
			"name":   user.Name,
			"email":  user.Email,
			"status": user.Status,
			"roleId": roleIDStr,
		},
	})
}
