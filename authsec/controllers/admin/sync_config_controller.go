package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SyncConfigController struct{}

// CreateSyncConfig godoc
// @Summary Create a new sync configuration
// @Description Creates a new Active Directory or Entra ID sync configuration with encrypted credentials
// @Tags SyncConfig
// @Accept json
// @Produce json
// @Param request body models.CreateSyncConfigRequest true "Sync configuration details"
// @Success 200 {object} models.SyncConfigResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/sync-configs/create [post]
func (scc *SyncConfigController) CreateSyncConfig(c *gin.Context) {
	var req models.CreateSyncConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate sync type
	if req.SyncType != "active_directory" && req.SyncType != "entra_id" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sync_type. Must be 'active_directory' or 'entra_id'"})
		return
	}

	// Validate that appropriate config is provided
	if req.SyncType == "active_directory" && req.ADConfig == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ad_config is required for active_directory sync type"})
		return
	}
	if req.SyncType == "entra_id" && req.EntraConfig == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entra_config is required for entra_id sync type"})
		return
	}

	// Parse UUIDs
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
		return
	}
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Create sync configuration
	syncConfig := models.SyncConfiguration{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ClientID:    clientID,
		ProjectID:   projectID,
		SyncType:    req.SyncType,
		ConfigName:  req.ConfigName,
		Description: req.Description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Populate and encrypt AD config if applicable
	if req.SyncType == "active_directory" && req.ADConfig != nil {
		syncConfig.ADServer = req.ADConfig.Server
		syncConfig.ADUsername = req.ADConfig.Username
		syncConfig.ADBaseDN = req.ADConfig.BaseDN
		syncConfig.ADFilter = req.ADConfig.Filter
		syncConfig.ADUseSSL = req.ADConfig.UseSSL
		syncConfig.ADSkipVerify = req.ADConfig.SkipVerify

		// Encrypt password
		encryptedPassword, err := utils.Encrypt(req.ADConfig.Password)
		if err != nil {
			log.Printf("Failed to encrypt AD password: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt credentials"})
			return
		}
		syncConfig.ADPassword = encryptedPassword
	}

	// Populate and encrypt Entra ID config if applicable
	if req.SyncType == "entra_id" && req.EntraConfig != nil {
		syncConfig.EntraTenantID = req.EntraConfig.TenantID
		syncConfig.EntraClientID = req.EntraConfig.ClientID
		syncConfig.EntraSkipVerify = req.EntraConfig.SkipVerify

		// Encrypt client secret
		encryptedSecret, err := utils.Encrypt(req.EntraConfig.ClientSecret)
		if err != nil {
			log.Printf("Failed to encrypt Entra client secret: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt credentials"})
			return
		}
		syncConfig.EntraClientSecret = encryptedSecret

		// Store scopes as JSON
		if len(req.EntraConfig.Scopes) > 0 {
			scopesJSON, _ := json.Marshal(req.EntraConfig.Scopes)
			syncConfig.EntraScopes = string(scopesJSON)
		}
	}

	// Save to database
	if err := config.DB.Create(&syncConfig).Error; err != nil {
		log.Printf("Failed to create sync configuration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save configuration", "details": err.Error()})
		return
	}

	// Audit log: Sync configuration created
	middlewares.Audit(c, "sync_config", syncConfig.ID.String(), "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"config_name": syncConfig.ConfigName,
			"sync_type":   syncConfig.SyncType,
			"tenant_id":   syncConfig.TenantID.String(),
			"client_id":   syncConfig.ClientID.String(),
			"is_active":   syncConfig.IsActive,
		},
	})

	// Mask sensitive fields in response
	syncConfig = scc.maskSensitiveFields(syncConfig)

	c.JSON(http.StatusOK, models.SyncConfigResponse{
		Success: true,
		Message: "Sync configuration created successfully",
		Data:    &syncConfig,
	})
}

// ListSyncConfigs godoc
// @Summary List sync configurations
// @Description Lists all sync configurations for a tenant, optionally filtered by sync type
// @Tags SyncConfig
// @Accept json
// @Produce json
// @Param request body models.ListSyncConfigsRequest true "Filter parameters"
// @Success 200 {object} models.SyncConfigResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/sync-configs/list [post]
func (scc *SyncConfigController) ListSyncConfigs(c *gin.Context) {
	var req models.ListSyncConfigsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse UUIDs
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
		return
	}

	// Build query
	query := config.DB.Where("tenant_id = ? AND client_id = ?", tenantID, clientID)

	// Apply sync type filter if provided
	if req.SyncType != nil && *req.SyncType != "" {
		query = query.Where("sync_type = ?", *req.SyncType)
	}

	// Fetch configurations
	var configs []models.SyncConfiguration
	if err := query.Order("created_at DESC").Find(&configs).Error; err != nil {
		log.Printf("Failed to fetch sync configurations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch configurations"})
		return
	}

	// Mask sensitive fields
	for i := range configs {
		configs[i] = scc.maskSensitiveFields(configs[i])
	}

	c.JSON(http.StatusOK, models.SyncConfigResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d sync configuration(s)", len(configs)),
		Configs: configs,
	})
}

// UpdateSyncConfig godoc
// @Summary Update a sync configuration
// @Description Updates an existing sync configuration
// @Tags SyncConfig
// @Accept json
// @Produce json
// @Param request body models.UpdateSyncConfigRequest true "Update details"
// @Success 200 {object} models.SyncConfigResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/sync-configs/update [post]
func (scc *SyncConfigController) UpdateSyncConfig(c *gin.Context) {
	var req models.UpdateSyncConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse UUIDs
	configID, err := uuid.Parse(req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id format"})
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
		return
	}

	// Fetch existing configuration
	var syncConfig models.SyncConfiguration
	if err := config.DB.Where("id = ? AND tenant_id = ? AND client_id = ?", configID, tenantID, clientID).First(&syncConfig).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sync configuration not found"})
		return
	}

	// Update basic fields
	if req.ConfigName != nil {
		syncConfig.ConfigName = *req.ConfigName
	}
	if req.Description != nil {
		syncConfig.Description = *req.Description
	}
	if req.IsActive != nil {
		syncConfig.IsActive = *req.IsActive
	}

	// Update AD config if provided
	if req.ADConfig != nil && syncConfig.SyncType == "active_directory" {
		if req.ADConfig.Server != "" {
			syncConfig.ADServer = req.ADConfig.Server
		}
		if req.ADConfig.Username != "" {
			syncConfig.ADUsername = req.ADConfig.Username
		}
		if req.ADConfig.Password != "" {
			encryptedPassword, err := utils.Encrypt(req.ADConfig.Password)
			if err != nil {
				log.Printf("Failed to encrypt AD password: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt credentials"})
				return
			}
			syncConfig.ADPassword = encryptedPassword
		}
		if req.ADConfig.BaseDN != "" {
			syncConfig.ADBaseDN = req.ADConfig.BaseDN
		}
		if req.ADConfig.Filter != "" {
			syncConfig.ADFilter = req.ADConfig.Filter
		}
		syncConfig.ADUseSSL = req.ADConfig.UseSSL
		syncConfig.ADSkipVerify = req.ADConfig.SkipVerify
	}

	// Update Entra ID config if provided
	if req.EntraConfig != nil && syncConfig.SyncType == "entra_id" {
		if req.EntraConfig.TenantID != "" {
			syncConfig.EntraTenantID = req.EntraConfig.TenantID
		}
		if req.EntraConfig.ClientID != "" {
			syncConfig.EntraClientID = req.EntraConfig.ClientID
		}
		if req.EntraConfig.ClientSecret != "" {
			encryptedSecret, err := utils.Encrypt(req.EntraConfig.ClientSecret)
			if err != nil {
				log.Printf("Failed to encrypt Entra client secret: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt credentials"})
				return
			}
			syncConfig.EntraClientSecret = encryptedSecret
		}
		if len(req.EntraConfig.Scopes) > 0 {
			scopesJSON, _ := json.Marshal(req.EntraConfig.Scopes)
			syncConfig.EntraScopes = string(scopesJSON)
		}
		syncConfig.EntraSkipVerify = req.EntraConfig.SkipVerify
	}

	syncConfig.UpdatedAt = time.Now()

	// Save updates
	if err := config.DB.Save(&syncConfig).Error; err != nil {
		log.Printf("Failed to update sync configuration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration"})
		return
	}

	// Audit log: Sync configuration updated
	middlewares.Audit(c, "sync_config", syncConfig.ID.String(), "update", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"config_name": syncConfig.ConfigName,
			"sync_type":   syncConfig.SyncType,
			"tenant_id":   syncConfig.TenantID.String(),
			"client_id":   syncConfig.ClientID.String(),
			"is_active":   syncConfig.IsActive,
		},
	})

	// Mask sensitive fields
	syncConfig = scc.maskSensitiveFields(syncConfig)

	c.JSON(http.StatusOK, models.SyncConfigResponse{
		Success: true,
		Message: "Sync configuration updated successfully",
		Data:    &syncConfig,
	})
}

// DeleteSyncConfig godoc
// @Summary Delete a sync configuration
// @Description Deletes an existing sync configuration
// @Tags SyncConfig
// @Accept json
// @Produce json
// @Param request body models.DeleteSyncConfigRequest true "Delete parameters"
// @Success 200 {object} models.SyncConfigResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/sync-configs/delete [post]
func (scc *SyncConfigController) DeleteSyncConfig(c *gin.Context) {
	var req models.DeleteSyncConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse UUIDs
	configID, err := uuid.Parse(req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id format"})
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
		return
	}

	// Delete configuration
	result := config.DB.Where("id = ? AND tenant_id = ? AND client_id = ?", configID, tenantID, clientID).Delete(&models.SyncConfiguration{})
	if result.Error != nil {
		log.Printf("Failed to delete sync configuration: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete configuration"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sync configuration not found"})
		return
	}

	// Audit log: Sync configuration deleted
	middlewares.Audit(c, "sync_config", configID.String(), "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"id":        configID.String(),
			"tenant_id": tenantID.String(),
			"client_id": clientID.String(),
		},
	})

	c.JSON(http.StatusOK, models.SyncConfigResponse{
		Success: true,
		Message: "Sync configuration deleted successfully",
	})
}

// maskSensitiveFields masks sensitive credential fields in the response
func (scc *SyncConfigController) maskSensitiveFields(config models.SyncConfiguration) models.SyncConfiguration {
	if config.ADPassword != "" {
		config.ADPassword = "********"
	}
	if config.EntraClientSecret != "" {
		config.EntraClientSecret = "********"
	}
	return config
}
