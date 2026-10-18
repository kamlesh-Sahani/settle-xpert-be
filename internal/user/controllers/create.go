package controllers

import (
	"log"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/user/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AddressRequest struct {
	Line1   string `json:"line1"`
	Country string `json:"country"`
	State   string `json:"state"`
	City    string `json:"city"`
	Zip     string `json:"zip"`
}

type RegisterUserRequest struct {
	Email         string          `json:"email" binding:"required,email"`
	Name          string          `json:"name" binding:"required"`
	Password      string          `json:"password" binding:"required,min=6"`
	RoleID        *string         `json:"roleId"`
	RoleIDSnake   *string         `json:"role_id"`
	Phone         string          `json:"phone"`
	Status        *string         `json:"status"`
	Logo          interface{}     `json:"logo"`
	TeamID        *string         `json:"teamId"`
	DOB           string          `json:"dob"`
	Gender        string          `json:"gender"`
	FatherName    string          `json:"fatherName"`
	MotherName    string          `json:"motherName"`
	PANCard       string          `json:"panCard"`
	DateOfJoining string          `json:"dateOfJoining"`
	AadharNumber  string          `json:"aadharNumber"`
	AadharFront   interface{}     `json:"aadharFront"`
	AadharBack    interface{}     `json:"aadharBack"`
	Address       *AddressRequest `json:"address"`
}

func parseLogo(logo interface{}) *string {
	if logo == nil {
		return nil
	}
	switch v := logo.(type) {
	case string:
		return &v
	case []interface{}:
		if len(v) > 0 {
			if firstMap, ok := v[0].(map[string]interface{}); ok {
				if url, ok := firstMap["url"].(string); ok {
					return &url
				}
				if resp, ok := firstMap["response"].(map[string]interface{}); ok {
					if url, ok := resp["url"].(string); ok {
						return &url
					}
				}
			} else if firstStr, ok := v[0].(string); ok {
				return &firstStr
			}
		}
		empty := ""
		return &empty
	default:
		return nil
	}
}

func Create(c *gin.Context) {
	var req RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	effectiveRoleID := req.RoleID
	if (effectiveRoleID == nil || *effectiveRoleID == "") && req.RoleIDSnake != nil {
		effectiveRoleID = req.RoleIDSnake
	}

	if effectiveRoleID == nil || *effectiveRoleID == "" {
		response.BadRequest(c, "Role is required")
		return
	}

	if req.DOB == "" {
		response.BadRequest(c, "Date of Birth is required")
		return
	}

	if req.Gender == "" {
		response.BadRequest(c, "Gender is required")
		return
	}

	if req.DateOfJoining == "" {
		response.BadRequest(c, "Date of Joining is required")
		return
	}

	var roleObjID *primitive.ObjectID
	id, err := primitive.ObjectIDFromHex(*effectiveRoleID)
	if err != nil {
		response.BadRequest(c, "Invalid role ID format")
		return
	}
	roleObjID = &id

	collection := database.GetCollection("users")

	count, _ := collection.CountDocuments(c.Request.Context(), bson.M{
		"email":      req.Email,
		"deleted_at": bson.M{"$exists": false},
	})
	if count > 0 {
		response.Conflict(c, "User with this email already exists")
		return
	}

	hashedPassword, err := utils.Hash(req.Password)
	if err != nil {
		log.Printf("[ERROR] [UserCreate] Failed to hash password: %v", err)
		response.InternalServerError(c, "Unable to create user. Please try again.")
		return
	}

	status := models.UserStatusActive
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	var logo string
	if req.Logo != nil {
		if logoStr := parseLogo(req.Logo); logoStr != nil {
			logo = *logoStr
		}
	}

	var teamID string
	if req.TeamID != nil && *req.TeamID != "" {
		teamID = *req.TeamID
	} else {
		generated, err := NextTeamIden(c.Request.Context())
		if err == nil {
			teamID = generated
		}
	}

	var aadharFront string
	if req.AadharFront != nil {
		if frontStr := parseLogo(req.AadharFront); frontStr != nil {
			aadharFront = *frontStr
		}
	}

	var aadharBack string
	if req.AadharBack != nil {
		if backStr := parseLogo(req.AadharBack); backStr != nil {
			aadharBack = *backStr
		}
	}

	var addr models.Address
	if req.Address != nil {
		addr = models.Address{
			Line1:   req.Address.Line1,
			Country: req.Address.Country,
			State:   req.Address.State,
			City:    req.Address.City,
			Zip:     req.Address.Zip,
		}
	}

	user := models.User{
		TeamID:        teamID,
		RoleID:        roleObjID,
		Email:         req.Email,
		Name:          req.Name,
		Password:      hashedPassword,
		Phone:         req.Phone,
		Status:        status,
		Logo:          logo,
		DOB:           req.DOB,
		Gender:        req.Gender,
		FatherName:    req.FatherName,
		MotherName:    req.MotherName,
		PANCard:       req.PANCard,
		DateOfJoining: req.DateOfJoining,
		AadharNumber:  req.AadharNumber,
		AadharFront:   aadharFront,
		AadharBack:    aadharBack,
		Address:       addr,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	result, err := collection.InsertOne(c.Request.Context(), user)
	if err != nil {
		log.Printf("[ERROR] [UserCreate] Failed to insert user: %v", err)
		response.InternalServerError(c, "Unable to create user. Please try again.")
		return
	}

	user.ID = result.InsertedID.(primitive.ObjectID)

	RecordAuditLog(c, user.ID.Hex(), "CREATE", nil, user)

	response.Created(c, "User created successfully", gin.H{
		"id":     user.ID.Hex(),
		"teamId": user.TeamID,
		"name":   user.Name,
		"email":  user.Email,
		"roleId": effectiveRoleID,
		"status": user.Status,
	})
}
