package repositories

import (
	"context"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
)

// PolicyRepository defines the interface for policy data operations
type PolicyRepository interface {
	GetByID(ctx context.Context, tenantID, id string) (*models.AttestationPolicy, error)
	Create(ctx context.Context, policy *models.AttestationPolicy) error
	Update(ctx context.Context, policy *models.AttestationPolicy) error
	Delete(ctx context.Context, tenantID, id string) error
	ListByTenant(ctx context.Context, tenantID string) ([]*models.AttestationPolicy, error)
	FindMatchingPolicy(ctx context.Context, tenantID, attestationType string, selectors map[string]string) (*models.AttestationPolicy, error)
}
