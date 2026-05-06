package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery returns a middleware that recovers from panics, logs the stack trace,
// and returns a 500 error response.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				log.Error("panic recovered",
					"error", r,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"stack", stack,
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "an unexpected error occurred",
					},
				})
			}
		}()

		c.Next()
	}
}
