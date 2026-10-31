package controllers

import (
	"lead-crm/backend/internal/upload/models"
	"lead-crm/backend/internal/upload/services"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
)

func GetPresignedURL(c *gin.Context) {
	var req models.PresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	res, err := services.S3.GeneratePresignedPost(c.Request.Context(), req.Filename, req.ContentType, req.Folder)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, "Presigned upload URL generated successfully", res)
}
