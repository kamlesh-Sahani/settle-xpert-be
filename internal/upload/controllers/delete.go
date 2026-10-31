package controllers

import (
	"lead-crm/backend/internal/upload/models"
	"lead-crm/backend/internal/upload/services"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
)

func DeleteFiles(c *gin.Context) {
	var req models.DeleteFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	if err := services.S3.DeleteObjects(c.Request.Context(), req.Keys); err != nil {
		response.InternalServerError(c, "Failed to delete files from storage")
		return
	}

	response.Success(c, "Files deleted successfully", nil)
}
