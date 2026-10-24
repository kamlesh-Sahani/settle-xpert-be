package controllers

import (
	"os"

	"lead-crm/backend/database"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Logout(c *gin.Context) {
	sessionIDVal, exists := c.Get("session_obj_id")
	if exists {
		if sessionObjID, ok := sessionIDVal.(primitive.ObjectID); ok && !sessionObjID.IsZero() {
			sessionCol := database.GetCollection("sessions")
			_, _ = sessionCol.DeleteOne(c.Request.Context(), bson.M{"_id": sessionObjID})
		}
	}

	isProd := os.Getenv("GIN_MODE") == "release"
	// Clear all authentication cookies
	c.SetCookie("access_token", "", -1, "/", "", isProd, true)
	c.SetCookie("refresh_token", "", -1, "/", "", isProd, true)

	response.Success(c, "Logged out successfully", nil)
}
