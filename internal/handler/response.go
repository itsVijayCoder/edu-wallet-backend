package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
)

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Success   bool      `json:"success"`
	RequestID string    `json:"request_id,omitempty"`
	Data      any       `json:"data,omitempty"`
	Error     *APIError `json:"error,omitempty"`
	Meta      *Meta     `json:"meta,omitempty"`
}

// APIError contains machine-readable error information.
type APIError struct {
	Code    string   `json:"code"`              // Machine-readable: "AUTH_INVALID_CREDENTIALS"
	Message string   `json:"message"`           // Human-readable: "invalid email or password"
	Details []string `json:"details,omitempty"` // Validation field errors
}

// Meta contains pagination metadata.
type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func RespondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, APIResponse{
		Success:   true,
		RequestID: getRequestID(c),
		Data:      data,
	})
}

func RespondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, APIResponse{
		Success:   true,
		RequestID: getRequestID(c),
		Data:      data,
	})
}

func RespondPaginated(c *gin.Context, data any, page, pageSize int, total int64, totalPages int) {
	c.JSON(http.StatusOK, APIResponse{
		Success:   true,
		RequestID: getRequestID(c),
		Data:      data,
		Meta: &Meta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func RespondMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, APIResponse{
		Success:   true,
		RequestID: getRequestID(c),
		Data:      gin.H{"message": message},
	})
}

func RespondValidationError(c *gin.Context, details []string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success:   false,
		RequestID: getRequestID(c),
		Error: &APIError{
			Code:    apperror.ErrValidationFailed.Code,
			Message: apperror.ErrValidationFailed.Message,
			Details: details,
		},
	})
}
