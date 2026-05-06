package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/pkg/logger"
)

// RequestID returns a middleware that assigns a unique request ID to every request.
// The ID is set as the X-Request-ID response header, stored in the gin context,
// and propagated into the request context for structured logging.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.New().String()

		// Set response header.
		c.Header("X-Request-ID", id)

		// Store in gin context for handler access.
		c.Set("request_id", id)

		// Propagate into request context for slog.
		ctx := logger.ContextWithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
