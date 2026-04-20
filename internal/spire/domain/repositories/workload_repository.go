package repositories

import (
	"context"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
)

// WorkloadRepository defines the interface for workload data operations
type WorkloadRepository interface {
	GetByID(ctx context.Context, tenantID, id string) (*models.Workload, error)
	GetBySpiffeID(ctx context.Context, tenantID, spiffeID string) (*models.Workload, error)
	Create(ctx context.Context, workload *models.Workload) error
	Update(ctx context.Context, workload *models.Workload) error
	Delete(ctx context.Context, tenantID, id string) error
	ListByTenant(ctx context.Context, tenantID string) ([]*models.Workload, error)
	FindBySelectors(ctx context.Context, tenantID string, selectors map[string]string) ([]*models.Workload, error)
}
