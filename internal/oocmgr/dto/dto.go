// Package oocmgrdto contains DTOs for the OIDC Configuration Manager (oocmgr).
// Ported from oath_oidc_configuration_manager/src/dto.
package oocmgrdto

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// ===== CUSTOM TYPES =====

// JSONMap is a custom type for JSONB handling.
type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	ba, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(ba), nil
}

func (m *JSONMap) Scan(value interface{}) error {
	var ba []byte
	switch v := value.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	case nil:
		return nil
	default:
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	var t map[string]interface{}
	if err := json.Unmarshal(ba, &t); err != nil {
		return err
	}
	*m = JSONMap(t)
	return nil
}

func (m JSONMap) GormDataType() string {
	return "jsonb"
}

func (JSONMap) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	if db.Dialector.Name() == "postgres" {
		return "jsonb"
	}
	return "json"
}

// ===== MAIN ENTITY =====

// OAuthOIDCConfiguration is the main configuration entity stored in tenant databases.
type OAuthOIDCConfiguration struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"not null;index"`
	OrgID       string         `json:"org_id" gorm:"not null;index"`
	TenantID    string         `json:"tenant_id" gorm:"not null;index"`
	ConfigType  string         `json:"config_type" gorm:"not null"`
	ConfigFiles JSONMap        `json:"config_files" gorm:"type:jsonb"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy   string         `json:"created_by"`
	UpdatedBy   string         `json:"updated_by"`
}

func (OAuthOIDCConfiguration) TableName() string {
	return "oauth_oidc_configurations"
}

func (c *OAuthOIDCConfiguration) ToResponse() *ConfigResponse {
	configFiles := make(map[string]interface{})
	for k, v := range c.ConfigFiles {
		configFiles[k] = v
	}
	return &ConfigResponse{
		ID:          c.ID,
		Name:        c.Name,
		OrgID:       c.OrgID,
		TenantID:    c.TenantID,
		ConfigType:  c.ConfigType,
		ConfigFiles: configFiles,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		CreatedBy:   c.CreatedBy,
		UpdatedBy:   c.UpdatedBy,
	}
}

// ===== CRUD REQUEST DTOs =====

type CreateConfigRequest struct {
	Name        string                 `json:"name" validate:"required"`
	OrgID       string                 `json:"org_id" validate:"required"`
	TenantID    string                 `json:"tenant_id" validate:"required"`
	ConfigType  string                 `json:"config_type" validate:"required,oneof=local_auth oidc oauth_server webauthn_mfa saml2 entra_sync ad_sync"`
	ConfigFiles map[string]interface{} `json:"config_files" validate:"required"`
	IsActive    bool                   `json:"is_active"`
	CreatedBy   string                 `json:"created_by"`
}

type UpdateConfigRequest struct {
	ID          uuid.UUID              `json:"id" validate:"required"`
	OrgID       string                 `json:"org_id" validate:"required"`
	TenantID    string                 `json:"tenant_id" validate:"required"`
	Name        *string                `json:"name,omitempty"`
	ConfigFiles map[string]interface{} `json:"config_files,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
	UpdatedBy   string                 `json:"updated_by"`
}

type EditConfigRequest struct {
	ID          uuid.UUID              `json:"id" validate:"required"`
	OrgID       string                 `json:"org_id" validate:"required"`
	TenantID    string                 `json:"tenant_id" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	ConfigType  string                 `json:"config_type" validate:"required"`
	ConfigFiles map[string]interface{} `json:"config_files" validate:"required"`
	IsActive    bool                   `json:"is_active"`
	UpdatedBy   string                 `json:"updated_by"`
}

type GetConfigsRequest struct {
	OrgID      string `json:"org_id" validate:"required"`
	TenantID   string `json:"tenant_id" validate:"required"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	ConfigType string `json:"config_type,omitempty"`
	ActiveOnly bool   `json:"active_only"`
}

type GetConfigByNameRequest struct {
	Name     string `json:"name" validate:"required"`
	OrgID    string `json:"org_id" validate:"required"`
	TenantID string `json:"tenant_id" validate:"required"`
}

type GetConfigByIDRequest struct {
	ID       uuid.UUID `json:"id" validate:"required"`
	OrgID    string    `json:"org_id" validate:"required"`
	TenantID string    `json:"tenant_id" validate:"required"`
}

type DeleteConfigRequest struct {
	ID       uuid.UUID `json:"id" validate:"required"`
	OrgID    string    `json:"org_id" validate:"required"`
	TenantID string    `json:"tenant_id" validate:"required"`
}

type GetTenantConfigsRequest struct {
	OrgID      uuid.UUID `json:"org_id" validate:"required"`
	TenantID   uuid.UUID `json:"tenant_id" validate:"required"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	ActiveOnly bool      `json:"active_only"`
}

type GetConfigsByTypeRequest struct {
	OrgID      uuid.UUID `json:"org_id" validate:"required"`
	TenantID   uuid.UUID `json:"tenant_id" validate:"required"`
	ConfigType string    `json:"config_type" validate:"required"`
	ActiveOnly bool      `json:"active_only"`
}

type CheckTenantConfigRequest struct {
	OrgID      uuid.UUID `json:"org_id" validate:"required"`
	TenantID   uuid.UUID `json:"tenant_id" validate:"required"`
	ConfigType string    `json:"config_type" validate:"required"`
	ActiveOnly bool      `json:"active_only"`
}

type GetConfigStatsRequest struct {
	OrgID    uuid.UUID `json:"org_id" validate:"required"`
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
}

type ActivateConfigRequest struct {
	ID        uuid.UUID `json:"id" validate:"required"`
	OrgID     uuid.UUID `json:"org_id" validate:"required"`
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	UpdatedBy string    `json:"updated_by"`
}

type BatchConfigUpdate struct {
	ID         uuid.UUID              `json:"id" validate:"required"`
	UpdateData map[string]interface{} `json:"update_data" validate:"required"`
}

// ===== RESPONSE DTOs =====

type ConfigResponse struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	OrgID       string                 `json:"org_id"`
	TenantID    string                 `json:"tenant_id"`
	ConfigType  string                 `json:"config_type"`
	ConfigFiles map[string]interface{} `json:"config_files"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
	UpdatedBy   string                 `json:"updated_by"`
}

type ConfigListResponse struct {
	Configs    []*ConfigResponse `json:"configs"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	Total      int64             `json:"total"`
	TotalPages int64             `json:"total_pages"`
}

type TenantConfigListResponse struct {
	Configs    []*ConfigResponse `json:"configs"`
	TenantID   uuid.UUID         `json:"tenant_id"`
	OrgID      uuid.UUID         `json:"org_id"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	Total      int64             `json:"total"`
	TotalPages int64             `json:"total_pages"`
	ActiveOnly bool              `json:"active_only"`
}

type ConfigsByTypeResponse struct {
	Configs    []*ConfigResponse `json:"configs"`
	ConfigType string            `json:"config_type"`
	TenantID   uuid.UUID         `json:"tenant_id"`
	OrgID      uuid.UUID         `json:"org_id"`
	Count      int64             `json:"count"`
	ActiveOnly bool              `json:"active_only"`
}

type TenantConfigCheckResponse struct {
	HasConfig  bool      `json:"has_config"`
	Count      int64     `json:"count"`
	ConfigType string    `json:"config_type"`
	TenantID   uuid.UUID `json:"tenant_id"`
	OrgID      uuid.UUID `json:"org_id"`
	ActiveOnly bool      `json:"active_only"`
}

type ConfigStatsResponse struct {
	TenantID uuid.UUID        `json:"tenant_id"`
	OrgID    uuid.UUID        `json:"org_id"`
	Total    int64            `json:"total"`
	Active   int64            `json:"active"`
	Inactive int64            `json:"inactive"`
	ByType   map[string]int64 `json:"by_type"`
}

type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Code      int       `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

type MessageResponse struct {
	Message   string      `json:"message"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ===== SAML DTOs =====

type AddSAMLProviderRequest struct {
	TenantID         string                 `json:"tenant_id" binding:"required"`
	ClientID         string                 `json:"client_id" binding:"required"`
	ProviderName     string                 `json:"provider_name" binding:"required"`
	DisplayName      string                 `json:"display_name" binding:"required"`
	EntityID         string                 `json:"entity_id" binding:"required"`
	SSOURL           string                 `json:"sso_url" binding:"required"`
	SLOURL           string                 `json:"slo_url"`
	Certificate      string                 `json:"certificate" binding:"required"`
	MetadataURL      string                 `json:"metadata_url"`
	NameIDFormat     string                 `json:"name_id_format"`
	AttributeMapping map[string]interface{} `json:"attribute_mapping"`
	IsActive         *bool                  `json:"is_active"`
	SortOrder        int                    `json:"sort_order"`
}

type UpdateSAMLProviderRequest struct {
	TenantID         string                 `json:"tenant_id" binding:"required"`
	ProviderID       string                 `json:"provider_id" binding:"required"`
	ClientID         string                 `json:"client_id"`
	ProviderName     string                 `json:"provider_name"`
	DisplayName      string                 `json:"display_name"`
	EntityID         string                 `json:"entity_id"`
	SSOURL           string                 `json:"sso_url"`
	SLOURL           string                 `json:"slo_url"`
	Certificate      string                 `json:"certificate"`
	MetadataURL      string                 `json:"metadata_url"`
	NameIDFormat     string                 `json:"name_id_format"`
	AttributeMapping map[string]interface{} `json:"attribute_mapping"`
	IsActive         *bool                  `json:"is_active"`
	SortOrder        *int                   `json:"sort_order"`
}

type DeleteSAMLProviderRequest struct {
	TenantID   string `json:"tenant_id" binding:"required"`
	ProviderID string `json:"provider_id" binding:"required"`
	ClientID   string `json:"client_id"`
}

type GetSAMLProviderRequest struct {
	TenantID   string `json:"tenant_id" binding:"required"`
	ProviderID string `json:"provider_id" binding:"required"`
}

type ListSAMLProvidersRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	ClientID string `json:"client_id"`
}

type SAMLProviderResponse struct {
	ID               uuid.UUID              `json:"id"`
	TenantID         uuid.UUID              `json:"tenant_id"`
	ProviderName     string                 `json:"provider_name"`
	DisplayName      string                 `json:"display_name"`
	EntityID         string                 `json:"entity_id"`
	SSOURL           string                 `json:"sso_url"`
	SLOURL           string                 `json:"slo_url"`
	MetadataURL      string                 `json:"metadata_url"`
	NameIDFormat     string                 `json:"name_id_format"`
	AttributeMapping map[string]interface{} `json:"attribute_mapping"`
	IsActive         bool                   `json:"is_active"`
	SortOrder        int                    `json:"sort_order"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

type SAMLProviderTemplatesResponse struct {
	Success   bool                       `json:"success"`
	Templates map[string]SAMLTemplateDTO `json:"templates"`
}

type SAMLTemplateDTO struct {
	ProviderName     string                 `json:"provider_name"`
	DisplayName      string                 `json:"display_name"`
	NameIDFormat     string                 `json:"name_id_format"`
	AttributeMapping map[string]interface{} `json:"attribute_mapping"`
	Instructions     string                 `json:"instructions"`
	DocumentationURL string                 `json:"documentation_url"`
	ConfigFields     []string               `json:"config_fields"`
}

// ===== TENANT HYDRA CLIENT =====

type TenantHydraClient struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrgID             string         `json:"org_id" gorm:"not null;index"`
	TenantID          string         `json:"tenant_id" gorm:"not null;index"`
	TenantName        string         `json:"tenant_name" gorm:"not null"`
	HydraClientID     string         `json:"hydra_client_id" gorm:"not null;unique"`
	HydraClientSecret string         `json:"hydra_client_secret" gorm:"not null"`
	ClientName        string         `json:"client_name" gorm:"not null"`
	RedirectURIs      pq.StringArray `json:"redirect_uris" gorm:"type:text[];default:'{}'"`
	Scopes            pq.StringArray `json:"scopes" gorm:"type:text[];default:'{openid,profile,email}'"`
	ClientType        string         `json:"client_type" gorm:"not null"`
	ProviderName      string         `json:"provider_name,omitempty"`
	IsActive          bool           `json:"is_active" gorm:"default:true"`
	CreatedAt         time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
	CreatedBy         string         `json:"created_by" gorm:"default:'system'"`
	UpdatedBy         string         `json:"updated_by" gorm:"default:'system'"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (TenantHydraClient) TableName() string {
	return "tenant_hydra_clients"
}

type GetTenantHydraClientsRequest struct {
	OrgID      string `json:"org_id,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	ClientType string `json:"client_type,omitempty"`
	IsActive   *bool  `json:"is_active,omitempty"`
}

// TenantHydraClientResponse for API responses
type TenantHydraClientResponse struct {
	ID                uuid.UUID `json:"id"`
	OrgID             string    `json:"org_id"`
	TenantID          string    `json:"tenant_id"`
	TenantName        string    `json:"tenant_name"`
	HydraClientID     string    `json:"hydra_client_id"`
	HydraClientSecret string    `json:"hydra_client_secret,omitempty"`
	ClientName        string    `json:"client_name"`
	RedirectURIs      []string  `json:"redirect_uris"`
	Scopes            []string  `json:"scopes"`
	ClientType        string    `json:"client_type"`
	ProviderName      string    `json:"provider_name,omitempty"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	CreatedBy         string    `json:"created_by"`
	UpdatedBy         string    `json:"updated_by"`
}

// Client represents a client record in tenant DB (used by GetClientsByTenant)
type Client struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	ClientID  string    `json:"client_id" gorm:"uniqueIndex;not null"`
	TenantID  string    `json:"tenant_id" gorm:"not null;index"`
	ProjectID string    `json:"project_id" gorm:"not null;index"`
	Name      string    `json:"name"`
	Active    bool      `json:"active" gorm:"default:true;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (Client) TableName() string {
	return "clients"
}

// SAMLConfigResponse wraps a list of SAML providers for a tenant.
type SAMLConfigResponse struct {
	Success   bool                   `json:"success"`
	TenantID  string                 `json:"tenant_id"`
	Providers []SAMLProviderResponse `json:"providers"`
}
