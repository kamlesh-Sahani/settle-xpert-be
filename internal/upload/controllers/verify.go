package controllers

import (
	"lead-crm/backend/internal/upload/models"
	"lead-crm/backend/internal/upload/services"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
)

func VerifyUpload(c *gin.Context) {
	var req models.VerifyUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	res, err := services.S3.VerifyUpload(c.Request.Context(), req.Key)
	if err != nil {
		response.BadRequest(c, "Uploaded file not found or invalid in storage")
		return
	}

	response.Success(c, "File verified successfully", res)
}
