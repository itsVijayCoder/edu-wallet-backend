package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
// In development mode with no explicit origins, localhost patterns are allowed.
// In non-development mode, allowedOrigins must not be empty.
func CORS(appEnv string, allowedOrigins []string) gin.HandlerFunc {
	isDev := appEnv == "development"
	isProduction := appEnv == "production"

	// Build the origin lookup set.
	originSet := make(map[string]bool, len(allowedOrigins))
	allowWildcard := false
	for _, o := range allowedOrigins {
		origin := strings.TrimSpace(strings.TrimRight(o, "/"))
		if origin == "" {
			continue
		}
		if origin == "*" && !isProduction {
			allowWildcard = true
			continue
		}
		originSet[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		allowed := false
		switch {
		case allowWildcard && origin != "":
			allowed = true
		case len(originSet) > 0 && originSet[origin]:
			allowed = true
		case isDev && len(originSet) == 0 && isLocalhost(origin):
			allowed = true
		}

		// Non-development with no configured origins: reject.
		if !isDev && len(originSet) == 0 {
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-API-Key")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, Retry-After, X-RateLimit-Remaining, X-RateLimit-Reset")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isLocalhost checks whether the origin matches a localhost pattern
// (e.g. http://localhost:3000, http://127.0.0.1:8080).
func isLocalhost(origin string) bool {
	if origin == "" {
		return false
	}
	lower := strings.ToLower(origin)
	return strings.HasPrefix(lower, "http://localhost") ||
		strings.HasPrefix(lower, "https://localhost") ||
		strings.HasPrefix(lower, "http://127.0.0.1") ||
		strings.HasPrefix(lower, "https://127.0.0.1")
}
