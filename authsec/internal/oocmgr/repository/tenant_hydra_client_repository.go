package oocmgrrepo

import (
	"errors"
	"fmt"

	"github.com/authsec-ai/authsec/config"
	oocmgrdto "github.com/authsec-ai/authsec/internal/oocmgr/dto"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantHydraClientRepository struct {
	masterDB *gorm.DB
}

func NewTenantHydraClientRepository() *TenantHydraClientRepository {
	return &TenantHydraClientRepository{
		masterDB: config.DB,
	}
}

func (r *TenantHydraClientRepository) Create(client *oocmgrdto.TenantHydraClient) error {
	if err := r.masterDB.Create(client).Error; err != nil {
		return fmt.Errorf("failed to create tenant hydra client mapping: %w", err)
	}
	return nil
}

func (r *TenantHydraClientRepository) GetByTenantID(tenantID, orgID string) ([]*oocmgrdto.TenantHydraClient, error) {
	var clients []*oocmgrdto.TenantHydraClient
	if err := r.masterDB.Where("tenant_id = ? AND org_id = ?", tenantID, orgID).Find(&clients).Error; err != nil {
		return nil, fmt.Errorf("failed to get tenant hydra clients: %w", err)
	}
	return clients, nil
}

func (r *TenantHydraClientRepository) GetByHydraClientID(hydraClientID string) (*oocmgrdto.TenantHydraClient, error) {
	var client oocmgrdto.TenantHydraClient
	if err := r.masterDB.Where("hydra_client_id = ?", hydraClientID).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("client not found")
		}
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return &client, nil
}

func (r *TenantHydraClientRepository) GetByTenantAndProvider(tenantID, providerName string) (*oocmgrdto.TenantHydraClient, error) {
	var client oocmgrdto.TenantHydraClient
	if err := r.masterDB.Where("tenant_id = ? AND provider_name = ? AND client_type = ?", tenantID, providerName, "oidc_provider").
		First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get provider client: %w", err)
	}
	return &client, nil
}

func (r *TenantHydraClientRepository) GetMainClient(tenantID, orgID string) (*oocmgrdto.TenantHydraClient, error) {
	var client oocmgrdto.TenantHydraClient
	if err := r.masterDB.Where("tenant_id = ? AND org_id = ? AND client_type = ?",
		tenantID, orgID, "main").First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("main client not found for tenant")
		}
		return nil, fmt.Errorf("failed to get main client: %w", err)
	}
	return &client, nil
}

func (r *TenantHydraClientRepository) GetProviderClients(tenantID, orgID string) ([]*oocmgrdto.TenantHydraClient, error) {
	var clients []*oocmgrdto.TenantHydraClient
	if err := r.masterDB.Where("tenant_id = ? AND org_id = ? AND client_type = ?",
		tenantID, orgID, "oidc_provider").Find(&clients).Error; err != nil {
		return nil, fmt.Errorf("failed to get provider clients: %w", err)
	}
	return clients, nil
}

func (r *TenantHydraClientRepository) Update(id uuid.UUID, updates map[string]interface{}) error {
	if err := r.masterDB.Model(&oocmgrdto.TenantHydraClient{}).
		Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update client mapping: %w", err)
	}
	return nil
}

func (r *TenantHydraClientRepository) Delete(id uuid.UUID) error {
	if err := r.masterDB.Where("id = ?", id).Delete(&oocmgrdto.TenantHydraClient{}).Error; err != nil {
		return fmt.Errorf("failed to delete client mapping: %w", err)
	}
	return nil
}

func (r *TenantHydraClientRepository) DeleteByHydraClientID(hydraClientID string) error {
	if err := r.masterDB.Where("hydra_client_id = ?", hydraClientID).
		Delete(&oocmgrdto.TenantHydraClient{}).Error; err != nil {
		return fmt.Errorf("failed to delete client mapping: %w", err)
	}
	return nil
}

func (r *TenantHydraClientRepository) UpdateByHydraClientID(hydraClientID string, updates map[string]interface{}) error {
	result := r.masterDB.Model(&oocmgrdto.TenantHydraClient{}).
		Where("hydra_client_id = ?", hydraClientID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update tenant hydra client mapping: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *TenantHydraClientRepository) ListAll(req *oocmgrdto.GetTenantHydraClientsRequest) ([]*oocmgrdto.TenantHydraClient, error) {
	var clients []*oocmgrdto.TenantHydraClient
	query := r.masterDB.Model(&oocmgrdto.TenantHydraClient{})

	if req.OrgID != "" {
		query = query.Where("org_id = ?", req.OrgID)
	}
	if req.TenantID != "" {
		query = query.Where("tenant_id = ?", req.TenantID)
	}
	if req.ClientType != "" {
		query = query.Where("client_type = ?", req.ClientType)
	}
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}
	if err := query.Order("created_at DESC").Find(&clients).Error; err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	return clients, nil
}
