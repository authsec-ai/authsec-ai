package repositories

import (
	"context"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
)

// TenantRepository defines the interface for tenant data operations
type TenantRepository interface {
	GetByID(ctx context.Context, id string) (*models.Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*models.Tenant, error)
	Create(ctx context.Context, tenant *models.Tenant) error
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*models.Tenant, error)
}
