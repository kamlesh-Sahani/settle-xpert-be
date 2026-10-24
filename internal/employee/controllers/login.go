package controllers

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"lead-crm/backend/database"
	authmodels "lead-crm/backend/internal/auth/models"
	"lead-crm/backend/internal/employee/models"
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

	empCol := database.GetCollection("employees")
	var employee models.Employee

	err := empCol.FindOne(c.Request.Context(), bson.M{
		"email":      req.Email,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&employee)

	if err != nil {
		response.Unauthorized(c, "Invalid email or password")
		return
	}

	if strings.ToUpper(employee.Status) != models.EmployeeStatusActive {
		response.Unauthorized(c, "Account is not active")
		return
	}

	if err := utils.VerifyHash(req.Password, employee.Password); err != nil {
		response.Unauthorized(c, "Invalid email or password")
		return
	}

	sessionCol := database.GetCollection("sessions")
	// Clear any previous active sessions for this employee
	_, _ = sessionCol.DeleteMany(c.Request.Context(), bson.M{"employee_id": employee.ID})

	sessionID := primitive.NewObjectID()

	accessToken, err := utils.GenerateAccessToken(employee.ID.Hex(), employee.Email, employee.Name, sessionID.Hex())
	if err != nil {
		log.Printf("[ERROR] [Login] Failed to generate access token: %v", err)
		response.InternalServerError(c, "Unable to sign in. Please try again.")
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(employee.ID.Hex(), sessionID.Hex())
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
		EmployeeID:   employee.ID,
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
	if employee.RoleID != nil {
		roleIDStr = employee.RoleID.Hex()
	}

	response.Success(c, "Login successful", gin.H{
		"token": accessToken,
		"employee": gin.H{
			"id":     employee.ID.Hex(),
			"name":   employee.Name,
			"email":  employee.Email,
			"status": employee.Status,
			"roleId": roleIDStr,
		},
	})
}
