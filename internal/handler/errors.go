package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
)

// HandleError maps an error to the correct HTTP response.
// If the error is an *apperror.AppError, it uses the structured code and status.
// Otherwise, it returns a generic 500 error.
func HandleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.Status, APIResponse{
			Success:   false,
			RequestID: getRequestID(c),
			Error: &APIError{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		})
		return
	}

	// Unknown error: don't leak internal details
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success:   false,
		RequestID: getRequestID(c),
		Error: &APIError{
			Code:    apperror.ErrInternal.Code,
			Message: apperror.ErrInternal.Message,
		},
	})
}

// HandleValidationError sends a 400 with validation details.
func HandleValidationError(c *gin.Context, details []string) {
	RespondValidationError(c, details)
}

func getRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return ""
}
