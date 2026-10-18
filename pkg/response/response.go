package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type Meta struct {
	Total       int64 `json:"total,omitempty"`
	Page        int   `json:"page,omitempty"`
	PageSize    int   `json:"page_size,omitempty"`
	TotalPages  int   `json:"total_pages,omitempty"`
	HasNextPage bool  `json:"has_next_page,omitempty"`
}

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func Success(c *gin.Context, httpStatus int, message string, data interface{}) {
	c.JSON(httpStatus, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessWithMeta(c *gin.Context, httpStatus int, message string, data interface{}, meta *Meta) {
	c.JSON(httpStatus, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func Error(c *gin.Context, httpStatus int, code string, message string, details interface{}) {
	c.JSON(httpStatus, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func BadRequest(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", message, details)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", message, nil)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", message, nil)
}

func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message, nil)
}
