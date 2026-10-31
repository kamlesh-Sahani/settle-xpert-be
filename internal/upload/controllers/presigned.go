package controllers

import (
	"log"

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
		log.Printf("[ERROR] [UploadPresigned] Failed to generate S3 post policy: %v", err)
		response.InternalServerError(c, "Failed to generate upload URL. Please try again later.")
		return
	}

	response.Success(c, "Presigned upload URL generated successfully", res)
}
