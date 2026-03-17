package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	repositories "github.com/authsec-ai/authsec/repository"
	"github.com/authsec-ai/authsec/vault"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ExternalServiceManager handles business logic for external services.
type ExternalServiceManager interface {
	Create(input *repositories.ExternalService, clientID, tenantID string, secretData map[string]interface{}) (*repositories.ExternalService, error)
	Get(id, clientID string) (*repositories.ExternalService, error)
	List(clientID string) ([]repositories.ExternalService, error)
	Update(id, clientID string, in ExternalServiceUpdateInput) (*repositories.ExternalService, error)
	Delete(id, clientID string) error
}

// ExternalServiceUpdateInput captures the fields that can be patched on a service.
type ExternalServiceUpdateInput struct {
	Name            *string
	Type            *string
	URL             *string
	Description     *string
	Tags            []string
	ResourceID      *string
	AuthType        *string
	AgentAccessible *bool
	SecretData      map[string]interface{}
}

type externalServiceManager struct {
	repo  repositories.ExternalServiceRepository
	vault vault.VaultClient
}

// NewExternalServiceManager constructs an ExternalServiceManager.
func NewExternalServiceManager(repo repositories.ExternalServiceRepository, vaultClient vault.VaultClient) ExternalServiceManager {
	return &externalServiceManager{repo: repo, vault: vaultClient}
}

func (m *externalServiceManager) Create(in *repositories.ExternalService, clientID, tenantID string, secretData map[string]interface{}) (*repositories.ExternalService, error) {
	if in.Name == "" || in.AuthType == "" {
		return nil, errors.New("name and auth_type are required")
	}
	if in.ResourceID == "" {
		return nil, errors.New("resource_id is required")
	}

	serviceID := uuid.NewString()
	vaultPath := fmt.Sprintf("kv/data/secret/tenants/%s/services/%s", tenantID, serviceID)

	in.ID = serviceID
	in.CreatedBy = clientID
	in.VaultPath = vaultPath
	in.CreatedAt = time.Now()
	in.UpdatedAt = time.Now()

	if m.vault != nil && len(secretData) > 0 {
		if err := m.vault.WriteSecret(vaultPath, secretData); err != nil {
			return nil, fmt.Errorf("failed to store credentials in vault: %w", err)
		}
	} else if len(secretData) > 0 && m.vault == nil {
		return nil, errors.New("vault client not initialized, cannot store secrets")
	}

	if err := m.repo.Create(in); err != nil {
		if m.vault != nil && len(secretData) > 0 {
			m.vault.DeleteSecret(vaultPath) // best-effort rollback
		}
		return nil, fmt.Errorf("failed to create service in database: %w", err)
	}
	return in, nil
}

func (m *externalServiceManager) Get(id, clientID string) (*repositories.ExternalService, error) {
	svc, err := m.repo.GetByID(id)
	if err != nil || svc.CreatedBy != clientID {
		return nil, errors.New("not found or forbidden")
	}
	return svc, nil
}

func (m *externalServiceManager) List(clientID string) ([]repositories.ExternalService, error) {
	return m.repo.ListByClient(clientID)
}

func (m *externalServiceManager) Update(id, clientID string, in ExternalServiceUpdateInput) (*repositories.ExternalService, error) {
	svc, err := m.repo.GetByID(id)
	if err != nil || svc.CreatedBy != clientID {
		return nil, errors.New("not found or forbidden")
	}

	touch := false
	if in.Name != nil {
		svc.Name = *in.Name
		touch = true
	}
	if in.Type != nil {
		svc.Type = *in.Type
		touch = true
	}
	if in.URL != nil {
		svc.URL = *in.URL
		touch = true
	}
	if in.Description != nil {
		svc.Description = *in.Description
		touch = true
	}
	if in.Tags != nil {
		svc.Tags = pq.StringArray(in.Tags)
		touch = true
	}
	if in.ResourceID != nil {
		svc.ResourceID = *in.ResourceID
		touch = true
	}
	if in.AuthType != nil {
		svc.AuthType = *in.AuthType
		touch = true
	}
	if in.AgentAccessible != nil {
		svc.AgentAccessible = *in.AgentAccessible
		touch = true
	}

	if touch || len(in.SecretData) > 0 {
		svc.UpdatedAt = time.Now()
		if err = m.repo.Update(svc); err != nil {
			return nil, err
		}
	}

	if len(in.SecretData) > 0 {
		if m.vault == nil {
			return nil, fmt.Errorf("vault client not configured")
		}
		if svc.VaultPath == "" {
			return nil, fmt.Errorf("service %s has no vault path configured", id)
		}
		if err := m.vault.WriteSecret(svc.VaultPath, in.SecretData); err != nil {
			return nil, fmt.Errorf("failed to update credentials in vault: %w", err)
		}
	}
	return svc, nil
}

func (m *externalServiceManager) Delete(id, clientID string) error {
	svc, err := m.repo.GetByID(id)
	if err != nil || svc.CreatedBy != clientID {
		return errors.New("not found or forbidden")
	}
	if err := m.repo.Delete(id); err != nil {
		return err
	}
	if m.vault != nil && svc.VaultPath != "" {
		if err := m.vault.DeleteSecret(svc.VaultPath); err != nil {
			log.Printf("Failed to delete vault secret for service %s: %v", id, err)
		}
	}
	return nil
}
