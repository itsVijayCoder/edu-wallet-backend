package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// BodySizeLimit rejects oversized requests early and wraps the body reader so
// handlers that stream payloads cannot accidentally read past the route limit.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 || c.Request.Body == nil {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			abortPayloadTooLarge(c, maxBytes)
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func abortPayloadTooLarge(c *gin.Context, maxBytes int64) {
	c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
		"success": false,
		"error": gin.H{
			"code":      "REQUEST_BODY_TOO_LARGE",
			"message":   "request body is too large",
			"max_bytes": strconv.FormatInt(maxBytes, 10),
		},
	})
}
