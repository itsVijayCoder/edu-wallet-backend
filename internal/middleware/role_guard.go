package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RoleGuard returns a middleware that restricts access to users with specific roles.
// Users with the "super_admin" role bypass all checks.
// The middleware expects "user_roles" to be set in the gin context by the Auth middleware.
func RoleGuard(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("user_roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "insufficient permissions",
				},
			})
			return
		}

		userRoles, ok := rolesVal.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "insufficient permissions",
				},
			})
			return
		}

		for _, role := range userRoles {
			// Super admin bypasses all role checks.
			if role == "super_admin" {
				c.Next()
				return
			}

			for _, allowed := range allowedRoles {
				if role == allowed {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "insufficient permissions",
			},
		})
	}
}
