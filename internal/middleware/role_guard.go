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

// RequireTenant rejects requests whose access token has not selected a tenant.
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tenantID, exists := c.Get("tenant_id"); exists && tenantID != "" {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "TENANT_REQUIRED",
				"message": "tenant context is required",
			},
		})
	}
}

// PermissionGuard restricts access by tenant-scoped permissions.
// Users with the "super_admin" role bypass permission checks.
func PermissionGuard(requiredPermissions ...string) gin.HandlerFunc {
	required := make(map[string]struct{}, len(requiredPermissions))
	for _, permission := range requiredPermissions {
		required[permission] = struct{}{}
	}

	return func(c *gin.Context) {
		if hasRole(c, "super_admin") {
			c.Next()
			return
		}

		permissionsVal, exists := c.Get("user_permissions")
		if !exists {
			abortForbidden(c)
			return
		}

		userPermissions, ok := permissionsVal.([]string)
		if !ok {
			abortForbidden(c)
			return
		}

		for _, permission := range userPermissions {
			if _, ok := required[permission]; ok {
				c.Next()
				return
			}
		}

		abortForbidden(c)
	}
}

func hasRole(c *gin.Context, expected string) bool {
	rolesVal, exists := c.Get("user_roles")
	if !exists {
		return false
	}
	userRoles, ok := rolesVal.([]string)
	if !ok {
		return false
	}
	for _, role := range userRoles {
		if role == expected {
			return true
		}
	}
	return false
}

func abortForbidden(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "FORBIDDEN",
			"message": "insufficient permissions",
		},
	})
}
