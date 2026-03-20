package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CheckResourceMethodAccess checks if a user has access to a specific HTTP method and path
func CheckResourceMethodAccess(db *gorm.DB, userRoles []string, method, path string, tenantID *uuid.UUID) (bool, error) {
	// First check if the path requires admin role
	if strings.HasPrefix(path, "/admin") && !containsString(userRoles, "admin") {
		return false, nil
	}

	// Check against resource_methods table for specific path patterns
	var resourceMethod struct {
		RequiresAdmin bool   `gorm:"column:requires_admin"`
		ResourceName  string `gorm:"column:resource_name"`
	}

	// Find the most specific matching path pattern
	query := `
		SELECT rm.requires_admin, r.name as resource_name
		FROM resource_methods rm
		JOIN resources r ON rm.resource_id = r.id
		WHERE rm.method = ?
		AND (? LIKE rm.path_pattern OR rm.path_pattern = ?)
		AND (r.tenant_id IS NULL OR r.tenant_id = ?)
		ORDER BY LENGTH(rm.path_pattern) DESC
		LIMIT 1
	`

	pathPattern := path
	if tenantID != nil {
		err := db.Raw(query, method, path, pathPattern, tenantID).Scan(&resourceMethod).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return false, err
		}
	} else {
		err := db.Raw(query, method, path, pathPattern, nil).Scan(&resourceMethod).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return false, err
		}
	}

	// If we found a matching resource method
	if resourceMethod.ResourceName != "" {
		if resourceMethod.RequiresAdmin {
			// This endpoint requires admin role
			return containsString(userRoles, "admin"), nil
		} else {
			// This endpoint allows user role
			return containsString(userRoles, "user") || containsString(userRoles, "admin"), nil
		}
	}

	// If no specific rule found, check general permissions
	// For user-flow service, most protected endpoints require admin role
	if strings.HasPrefix(path, "/uflow/") && (strings.Contains(path, "/projects") ||
		strings.Contains(path, "/scopes") ||
		strings.Contains(path, "/resources") ||
		strings.Contains(path, "/roles") ||
		strings.Contains(path, "/groups") ||
		strings.Contains(path, "/clients") ||
		strings.Contains(path, "/enduser") ||
		strings.Contains(path, "/admin/")) {
		// These are admin-only endpoints
		return containsString(userRoles, "admin"), nil
	}

	// Default: allow access if user has any role
	return len(userRoles) > 0, nil
}

// containsString checks if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
