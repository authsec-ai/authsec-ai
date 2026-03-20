// Package oocmgrrepo contains repository layer for the OIDC Configuration Manager.
// Ported from oath_oidc_configuration_manager/src/repository.
package oocmgrrepo

import (
	"fmt"

	"github.com/authsec-ai/authsec/config"
	oocmgrdto "github.com/authsec-ai/authsec/internal/oocmgr/dto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthRepository struct {
	masterDB *gorm.DB
}

func NewAuthRepository() *AuthRepository {
	return &AuthRepository{
		masterDB: config.DB,
	}
}

// getTenantDB extracts the tenant database from the Gin context.
func (ar *AuthRepository) getTenantDB(c *gin.Context) (*gorm.DB, error) {
	tenantDB, exists := c.Get("tenant_db")
	if !exists {
		return nil, fmt.Errorf("tenant database not found in context")
	}
	db, ok := tenantDB.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("invalid tenant database type in context")
	}
	return db, nil
}

// CreateConfig creates a new authentication configuration in the tenant database.
func (ar *AuthRepository) CreateConfig(c *gin.Context, cfg *oocmgrdto.OAuthOIDCConfiguration) (*oocmgrdto.OAuthOIDCConfiguration, error) {
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	var existing oocmgrdto.OAuthOIDCConfiguration
	result := tenantDB.Where("name = ? AND org_id = ? AND tenant_id = ? AND deleted_at IS NULL",
		cfg.Name, cfg.OrgID, cfg.TenantID).First(&existing)
	if result.Error == nil {
		return nil, fmt.Errorf("configuration with name '%s' already exists for this organization", cfg.Name)
	}
	if result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("error checking existing configuration: %w", result.Error)
	}

	if err := tenantDB.Create(cfg).Error; err != nil {
		return nil, fmt.Errorf("failed to create configuration: %w", err)
	}
	return cfg, nil
}

// GetConfigs retrieves configurations with mandatory tenant/org filtering and pagination.
func (ar *AuthRepository) GetConfigs(c *gin.Context, req *oocmgrdto.GetConfigsRequest) ([]*oocmgrdto.OAuthOIDCConfiguration, int64, error) {
	var configs []*oocmgrdto.OAuthOIDCConfiguration
	var total int64

	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tenant database: %w", err)
	}

	query := tenantDB.Model(&oocmgrdto.OAuthOIDCConfiguration{}).
		Where("tenant_id = ? AND org_id = ?", req.TenantID, req.OrgID)

	if req.ConfigType != "" {
		query = query.Where("config_type = ?", req.ConfigType)
	}
	if req.ActiveOnly {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count configurations: %w", err)
	}

	offset := (req.Page - 1) * req.Limit
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(req.Limit).
		Find(&configs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve configurations: %w", err)
	}
	return configs, total, nil
}

// GetConfigByID retrieves a configuration by ID from the tenant database.
func (ar *AuthRepository) GetConfigByID(c *gin.Context, req *oocmgrdto.GetConfigByIDRequest) (*oocmgrdto.OAuthOIDCConfiguration, error) {
	var cfg oocmgrdto.OAuthOIDCConfiguration
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}
	if err := tenantDB.Where("id = ? AND tenant_id = ? AND org_id = ?",
		req.ID, req.TenantID, req.OrgID).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("configuration not found")
		}
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}
	return &cfg, nil
}

// GetConfigByName retrieves a configuration by name from the tenant database.
func (ar *AuthRepository) GetConfigByName(c *gin.Context, req *oocmgrdto.GetConfigByNameRequest) (*oocmgrdto.OAuthOIDCConfiguration, error) {
	var cfg oocmgrdto.OAuthOIDCConfiguration
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}
	if err := tenantDB.Where("name = ? AND tenant_id = ? AND org_id = ?",
		req.Name, req.TenantID, req.OrgID).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("configuration not found")
		}
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}
	return &cfg, nil
}

// UpdateConfig updates an existing configuration in the tenant database.
func (ar *AuthRepository) UpdateConfig(c *gin.Context, req *oocmgrdto.UpdateConfigRequest) (*oocmgrdto.OAuthOIDCConfiguration, error) {
	var cfg oocmgrdto.OAuthOIDCConfiguration
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	if err := tenantDB.Where("id = ? AND tenant_id = ? AND org_id = ?",
		req.ID, req.TenantID, req.OrgID).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("configuration not found")
		}
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}

	updateData := make(map[string]interface{})
	if req.Name != nil {
		var existing oocmgrdto.OAuthOIDCConfiguration
		result := tenantDB.Where("name = ? AND org_id = ? AND tenant_id = ? AND id != ? AND deleted_at IS NULL",
			*req.Name, req.OrgID, req.TenantID, req.ID).First(&existing)
		if result.Error == nil {
			return nil, fmt.Errorf("configuration with name '%s' already exists for this organization", *req.Name)
		}
		if result.Error != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("error checking existing configuration: %w", result.Error)
		}
		updateData["name"] = *req.Name
	}
	if req.ConfigFiles != nil {
		updateData["config_files"] = oocmgrdto.JSONMap(req.ConfigFiles)
	}
	if req.IsActive != nil {
		updateData["is_active"] = *req.IsActive
	}
	if req.UpdatedBy != "" {
		updateData["updated_by"] = req.UpdatedBy
	}

	if err := tenantDB.Model(&cfg).Updates(updateData).Error; err != nil {
		return nil, fmt.Errorf("failed to update configuration: %w", err)
	}

	if err := tenantDB.Where("id = ? AND tenant_id = ? AND org_id = ?",
		req.ID, req.TenantID, req.OrgID).First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("failed to reload updated configuration: %w", err)
	}
	return &cfg, nil
}

// DeleteConfig soft deletes a configuration from the tenant database.
func (ar *AuthRepository) DeleteConfig(c *gin.Context, req *oocmgrdto.DeleteConfigRequest) error {
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return fmt.Errorf("failed to get tenant database: %w", err)
	}
	result := tenantDB.Where("id = ? AND tenant_id = ? AND org_id = ?",
		req.ID, req.TenantID, req.OrgID).Delete(&oocmgrdto.OAuthOIDCConfiguration{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete configuration: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("configuration not found")
	}
	return nil
}

// GetTenantConfigs retrieves all configurations for a specific tenant.
func (ar *AuthRepository) GetTenantConfigs(c *gin.Context, req *oocmgrdto.GetTenantConfigsRequest) ([]*oocmgrdto.OAuthOIDCConfiguration, int64, error) {
	var configs []*oocmgrdto.OAuthOIDCConfiguration
	var total int64

	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tenant database: %w", err)
	}

	query := tenantDB.Model(&oocmgrdto.OAuthOIDCConfiguration{}).
		Where("tenant_id = ? AND org_id = ?", req.TenantID, req.OrgID)
	if req.ActiveOnly {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tenant configurations: %w", err)
	}

	if req.Page > 0 && req.Limit > 0 {
		offset := (req.Page - 1) * req.Limit
		query = query.Offset(offset).Limit(req.Limit)
	}
	if err := query.Order("config_type ASC, created_at DESC").Find(&configs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve tenant configurations: %w", err)
	}
	return configs, total, nil
}

// GetConfigsByType retrieves configurations by type for a tenant.
func (ar *AuthRepository) GetConfigsByType(c *gin.Context, req *oocmgrdto.GetConfigsByTypeRequest) ([]*oocmgrdto.OAuthOIDCConfiguration, error) {
	var configs []*oocmgrdto.OAuthOIDCConfiguration
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	query := tenantDB.Where("tenant_id = ? AND org_id = ? AND config_type = ?",
		req.TenantID, req.OrgID, req.ConfigType)
	if req.ActiveOnly {
		query = query.Where("is_active = ?", true)
	}
	if err := query.Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve configurations by type: %w", err)
	}
	return configs, nil
}

// CheckTenantHasConfig checks if a tenant has a specific configuration type.
func (ar *AuthRepository) CheckTenantHasConfig(c *gin.Context, req *oocmgrdto.CheckTenantConfigRequest) (*oocmgrdto.TenantConfigCheckResponse, error) {
	var count int64
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	query := tenantDB.Model(&oocmgrdto.OAuthOIDCConfiguration{}).
		Where("tenant_id = ? AND org_id = ? AND config_type = ?", req.TenantID, req.OrgID, req.ConfigType)
	if req.ActiveOnly {
		query = query.Where("is_active = ?", true)
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check tenant configuration: %w", err)
	}
	return &oocmgrdto.TenantConfigCheckResponse{
		HasConfig:  count > 0,
		Count:      count,
		ConfigType: req.ConfigType,
		TenantID:   req.TenantID,
		OrgID:      req.OrgID,
		ActiveOnly: req.ActiveOnly,
	}, nil
}

// GetActiveConfigByType retrieves the active configuration by type for a tenant.
func (ar *AuthRepository) GetActiveConfigByType(c *gin.Context, tenantID, orgID uuid.UUID, configType string) (*oocmgrdto.OAuthOIDCConfiguration, error) {
	var cfg oocmgrdto.OAuthOIDCConfiguration
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}
	if err := tenantDB.Where("tenant_id = ? AND org_id = ? AND config_type = ? AND is_active = ?",
		tenantID, orgID, configType, true).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no active %s configuration found for tenant", configType)
		}
		return nil, fmt.Errorf("failed to retrieve %s configuration: %w", configType, err)
	}
	return &cfg, nil
}

// DeactivateOtherConfigs deactivates other configurations of the same type.
func (ar *AuthRepository) DeactivateOtherConfigs(c *gin.Context, tenantID, orgID uuid.UUID, configType string, excludeID uuid.UUID) error {
	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return fmt.Errorf("failed to get tenant database: %w", err)
	}
	result := tenantDB.Model(&oocmgrdto.OAuthOIDCConfiguration{}).
		Where("tenant_id = ? AND org_id = ? AND config_type = ? AND id != ? AND is_active = ?",
			tenantID, orgID, configType, excludeID, true).
		Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to deactivate other %s configurations: %w", configType, result.Error)
	}
	return nil
}

// GetConfigStats returns statistics about configurations for a tenant.
func (ar *AuthRepository) GetConfigStats(c *gin.Context, req *oocmgrdto.GetConfigStatsRequest) (*oocmgrdto.ConfigStatsResponse, error) {
	stats := &oocmgrdto.ConfigStatsResponse{
		TenantID: req.TenantID,
		OrgID:    req.OrgID,
		ByType:   make(map[string]int64),
	}

	tenantDB, err := ar.getTenantDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	if err := tenantDB.Model(&oocmgrdto.OAuthOIDCConfiguration{}).
		Where("tenant_id = ? AND org_id = ?", req.TenantID, req.OrgID).
		Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total configurations: %w", err)
	}

	if err := tenantDB.Model(&oocmgrdto.OAuthOIDCConfiguration{}).
		Where("tenant_id = ? AND org_id = ? AND is_active = ?", req.TenantID, req.OrgID, true).
		Count(&stats.Active).Error; err != nil {
		return nil, fmt.Errorf("failed to count active configurations: %w", err)
	}
	stats.Inactive = stats.Total - stats.Active

	rows, err := tenantDB.Model(&oocmgrdto.OAuthOIDCConfiguration{}).
		Select("config_type, COUNT(*) as count").
		Where("tenant_id = ? AND org_id = ?", req.TenantID, req.OrgID).
		Group("config_type").Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration counts by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var configType string
		var count int64
		if err := rows.Scan(&configType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan configuration type stats: %w", err)
		}
		stats.ByType[configType] = count
	}
	return stats, nil
}
