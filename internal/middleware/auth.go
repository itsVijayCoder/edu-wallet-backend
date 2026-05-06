package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/pkg/jwt"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/logger"
)

// Auth returns a middleware that validates JWT Bearer tokens.
// It extracts user claims and stores them in the gin context for downstream handlers.
func Auth(tokenMgr jwt.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_INVALID_TOKEN",
					"message": "missing authorization header",
				},
			})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_INVALID_TOKEN",
					"message": "authorization header must be Bearer {token}",
				},
			})
			return
		}

		token := parts[1]
		claims, err := tokenMgr.ValidateAccess(token)
		if err != nil {
			if errors.Is(err, jwt.ErrExpiredToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "AUTH_TOKEN_EXPIRED",
						"message": "token has expired",
					},
				})
				return
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_INVALID_TOKEN",
					"message": "invalid or malformed token",
				},
			})
			return
		}

		// Store claims in gin context for handlers.
		userID := claims.UserID.String()
		c.Set("user_id", userID)
		c.Set("user_email", claims.Email)
		c.Set("user_roles", claims.Roles)

		// Propagate user_id into request context for structured logging.
		ctx := logger.ContextWithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
