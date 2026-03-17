package middlewares

import (
	"github.com/gin-gonic/gin"
)

// AdminAuthMiddleware assumes AuthMiddleware has already run and enforces admin role membership.
func AdminAuthMiddleware() gin.HandlerFunc {
	return Require("admin", "access")
}

// AdminPermissionMiddleware creates middleware to check specific admin permissions using auth-manager
func AdminPermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	// Map permission strings to resource/action pairs
	var resource, action string
	switch requiredPermission {
	case "user_management":
		resource = "users"
		action = "manage"
	case "tenant_management":
		resource = "tenants"
		action = "manage"
	case "project_management":
		resource = "projects"
		action = "manage"
	case "admin_read":
		resource = "admin"
		action = "read"
	case "admin_write":
		resource = "admin"
		action = "write"
	default:
		resource = "admin"
		action = "access"
	}

	requireAdmin := Require("admin", "access")
	requirePermission := Require(resource, action)

	return func(c *gin.Context) {
		requireAdmin(c)
		if c.IsAborted() {
			return
		}
		requirePermission(c)
	}
}
