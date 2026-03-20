package sdkmgr

import (
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// DashboardService provides analytics queries on oauth_sessions and users tables.
// Translates sdk-manager's dashboard_service.py.
type DashboardService struct{}

// NewDashboardService creates a new service instance.
func NewDashboardService() *DashboardService {
	return &DashboardService{}
}

// HealthCheck returns service health.
func (s *DashboardService) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":  "healthy",
		"service": "dashboard-api",
		"message": "Dashboard API is running",
	}
}

// GetSessionStatistics returns session statistics for a tenant.
func (s *DashboardService) GetSessionStatistics(tenantID string) map[string]interface{} {
	db, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		logrus.WithError(err).WithField("tenant_id", tenantID).Error("failed to get tenant DB for statistics")
		return map[string]interface{}{
			"success":   false,
			"error":     "Tenant database not found",
			"message":   err.Error(),
			"tenant_id": tenantID,
		}
	}

	now := time.Now().Unix()
	stats := map[string]interface{}{}

	// Total active sessions.
	var totalActive int64
	db.Table("oauth_sessions").
		Where("is_active = true AND access_token IS NOT NULL AND token_expires_at > ?", now).
		Count(&totalActive)
	stats["total_active_sessions"] = totalActive

	// Total sessions (including inactive).
	var totalAll int64
	db.Table("oauth_sessions").Count(&totalAll)
	stats["total_sessions_all_time"] = totalAll

	// Sessions by provider.
	type providerCount struct {
		Provider string
		Count    int64
	}
	var providerCounts []providerCount
	db.Table("oauth_sessions").
		Select("provider, count(*) as count").
		Where("is_active = true AND access_token IS NOT NULL AND token_expires_at > ?", now).
		Group("provider").
		Scan(&providerCounts)

	byProvider := map[string]int64{}
	for _, pc := range providerCounts {
		byProvider[pc.Provider] = pc.Count
	}
	stats["sessions_by_provider"] = byProvider

	// Unique active users (by email).
	var uniqueUsers int64
	db.Table("oauth_sessions").
		Where("is_active = true AND access_token IS NOT NULL AND token_expires_at > ?", now).
		Distinct("user_email").
		Count(&uniqueUsers)
	stats["unique_active_users"] = uniqueUsers

	return map[string]interface{}{
		"success":    true,
		"tenant_id":  tenantID,
		"statistics": stats,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}

// GetAdminUsers returns admin users for a tenant from the master database.
func (s *DashboardService) GetAdminUsers(tenantID string) map[string]interface{} {
	db := config.DB
	if db == nil {
		return map[string]interface{}{
			"success":           false,
			"error":             "master database not initialized",
			"tenant_id":         tenantID,
			"admin_users":       []interface{}{},
			"total_admin_users": 0,
		}
	}

	type adminUser struct {
		ClientID  *string    `gorm:"column:client_id"`
		TenantID  *string    `gorm:"column:tenant_id"`
		Name      *string    `gorm:"column:name"`
		Email     *string    `gorm:"column:email"`
		Provider  *string    `gorm:"column:provider"`
		Active    *bool      `gorm:"column:active"`
		MFAMethod *string    `gorm:"column:mfa_method"`
		CreatedAt *time.Time `gorm:"column:created_at"`
		UpdatedAt *time.Time `gorm:"column:updated_at"`
		LastLogin *time.Time `gorm:"column:last_login"`
		InvitedBy *string    `gorm:"column:invited_by"`
		InvitedAt *time.Time `gorm:"column:invited_at"`
	}

	var users []adminUser
	result := db.Table("users").
		Where("tenant_id = ?", tenantID).
		Order("last_login DESC NULLS LAST").
		Find(&users)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		logrus.WithError(result.Error).WithField("tenant_id", tenantID).Error("failed to get admin users")
		return map[string]interface{}{
			"success":           false,
			"error":             "Failed to retrieve admin users",
			"message":           result.Error.Error(),
			"tenant_id":         tenantID,
			"admin_users":       []interface{}{},
			"total_admin_users": 0,
		}
	}

	usersList := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		entry := map[string]interface{}{
			"client_id":  u.ClientID,
			"tenant_id":  u.TenantID,
			"name":       u.Name,
			"email":      u.Email,
			"provider":   u.Provider,
			"active":     u.Active,
			"mfa_method": u.MFAMethod,
			"created_at": formatTimePtr(u.CreatedAt),
			"updated_at": formatTimePtr(u.UpdatedAt),
			"last_login": formatTimePtr(u.LastLogin),
			"invited_by": u.InvitedBy,
			"invited_at": formatTimePtr(u.InvitedAt),
		}
		usersList = append(usersList, entry)
	}

	return map[string]interface{}{
		"success":           true,
		"tenant_id":         tenantID,
		"admin_users":       usersList,
		"total_admin_users": len(usersList),
		"timestamp":         time.Now().Format(time.RFC3339),
	}
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
