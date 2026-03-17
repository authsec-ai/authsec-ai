// Package oocmgrsvc contains the service layer for the OIDC Configuration Manager.
// Ported from oath_oidc_configuration_manager/src/service.
package oocmgrsvc

import (
	"fmt"

	oocmgrdto "github.com/authsec-ai/authsec/internal/oocmgr/dto"
	oocmgrrepo "github.com/authsec-ai/authsec/internal/oocmgr/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthService struct {
	authRepo *oocmgrrepo.AuthRepository
}

func NewAuthService(authRepo *oocmgrrepo.AuthRepository) *AuthService {
	return &AuthService{authRepo: authRepo}
}

func (as *AuthService) CreateConfig(c *gin.Context, req *oocmgrdto.CreateConfigRequest) (*oocmgrdto.ConfigResponse, error) {
	if err := as.validateCreateConfigRequest(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	cfg := &oocmgrdto.OAuthOIDCConfiguration{
		ID:          uuid.New(),
		Name:        req.Name,
		OrgID:       req.OrgID,
		TenantID:    req.TenantID,
		ConfigType:  req.ConfigType,
		ConfigFiles: oocmgrdto.JSONMap(req.ConfigFiles),
		IsActive:    req.IsActive,
		CreatedBy:   req.CreatedBy,
		UpdatedBy:   req.CreatedBy,
	}

	created, err := as.authRepo.CreateConfig(c, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create configuration: %w", err)
	}
	return created.ToResponse(), nil
}

func (as *AuthService) GetConfigs(c *gin.Context, req *oocmgrdto.GetConfigsRequest) (*oocmgrdto.ConfigListResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 10
	}

	configs, total, err := as.authRepo.GetConfigs(c, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve configurations: %w", err)
	}

	var responses []*oocmgrdto.ConfigResponse
	for _, cfg := range configs {
		responses = append(responses, cfg.ToResponse())
	}
	totalPages := (total + int64(req.Limit) - 1) / int64(req.Limit)

	return &oocmgrdto.ConfigListResponse{
		Configs:    responses,
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (as *AuthService) GetConfigByID(c *gin.Context, req *oocmgrdto.GetConfigByIDRequest) (*oocmgrdto.ConfigResponse, error) {
	cfg, err := as.authRepo.GetConfigByID(c, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}
	return cfg.ToResponse(), nil
}

func (as *AuthService) UpdateConfig(c *gin.Context, req *oocmgrdto.UpdateConfigRequest) (*oocmgrdto.ConfigResponse, error) {
	if err := as.validateUpdateConfigRequest(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	updated, err := as.authRepo.UpdateConfig(c, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update configuration: %w", err)
	}
	return updated.ToResponse(), nil
}

func (as *AuthService) EditConfig(c *gin.Context, req *oocmgrdto.EditConfigRequest) (*oocmgrdto.ConfigResponse, error) {
	if err := as.validateEditConfigRequest(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	getReq := &oocmgrdto.GetConfigByIDRequest{
		ID:       req.ID,
		TenantID: req.TenantID,
		OrgID:    req.OrgID,
	}
	existing, err := as.authRepo.GetConfigByID(c, getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}
	if existing.ConfigType != req.ConfigType {
		return nil, fmt.Errorf("configuration type mismatch: expected %s, got %s", existing.ConfigType, req.ConfigType)
	}

	name := req.Name
	isActive := req.IsActive
	updateReq := &oocmgrdto.UpdateConfigRequest{
		ID:          req.ID,
		OrgID:       req.OrgID,
		TenantID:    req.TenantID,
		Name:        &name,
		ConfigFiles: req.ConfigFiles,
		IsActive:    &isActive,
		UpdatedBy:   req.UpdatedBy,
	}
	return as.UpdateConfig(c, updateReq)
}

func (as *AuthService) DeleteConfig(c *gin.Context, req *oocmgrdto.DeleteConfigRequest) error {
	if err := as.authRepo.DeleteConfig(c, req); err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}
	return nil
}

// ===== VALIDATION =====

func (as *AuthService) validateCreateConfigRequest(req *oocmgrdto.CreateConfigRequest) error {
	if req.Name == "" {
		return fmt.Errorf("configuration name is required")
	}
	if req.OrgID == uuid.Nil.String() {
		return fmt.Errorf("organization ID is required")
	}
	if req.TenantID == uuid.Nil.String() {
		return fmt.Errorf("tenant ID is required")
	}
	if req.ConfigType == "" {
		return fmt.Errorf("configuration type is required")
	}
	validTypes := []string{"local_auth", "oidc", "oauth_server", "webauthn_mfa", "saml2", "entra_sync", "ad_sync"}
	isValid := false
	for _, vt := range validTypes {
		if req.ConfigType == vt {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid configuration type: %s", req.ConfigType)
	}
	return nil
}

func (as *AuthService) validateUpdateConfigRequest(req *oocmgrdto.UpdateConfigRequest) error {
	if req.ID == uuid.Nil {
		return fmt.Errorf("configuration ID is required")
	}
	if req.TenantID == uuid.Nil.String() {
		return fmt.Errorf("tenant ID is required")
	}
	if req.OrgID == uuid.Nil.String() {
		return fmt.Errorf("organization ID is required")
	}
	return nil
}

func (as *AuthService) validateEditConfigRequest(req *oocmgrdto.EditConfigRequest) error {
	if req.ID == uuid.Nil {
		return fmt.Errorf("configuration ID is required")
	}
	if req.TenantID == uuid.Nil.String() {
		return fmt.Errorf("tenant ID is required")
	}
	if req.OrgID == uuid.Nil.String() {
		return fmt.Errorf("organization ID is required")
	}
	if req.Name == "" {
		return fmt.Errorf("configuration name is required")
	}
	if req.ConfigType == "" {
		return fmt.Errorf("configuration type is required")
	}
	if req.ConfigFiles == nil {
		return fmt.Errorf("configuration files are required")
	}
	return nil
}
