package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/pkg/logger"
)

// Logger returns a middleware that logs every HTTP request using slog.
// Log level is chosen based on the response status code:
//
//	5xx -> Error, 4xx -> Warn, otherwise -> Info.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path

		// Enrich the logger with request context (request_id, user_id).
		l := logger.WithContext(c.Request.Context(), log)

		attrs := []any{
			"method", method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"client_ip", c.ClientIP(),
		}

		switch {
		case status >= 500:
			l.Error("request completed", attrs...)
		case status >= 400:
			l.Warn("request completed", attrs...)
		default:
			l.Info("request completed", attrs...)
		}
	}
}
