package repositories

import (
	"context"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
)

// WorkloadEntryRepository defines the interface for workload entry persistence
type WorkloadEntryRepository interface {
	Create(ctx context.Context, entry *models.WorkloadEntry) error
	GetByID(ctx context.Context, id string) (*models.WorkloadEntry, error)
	GetBySpiffeID(ctx context.Context, tenantID, spiffeID string) (*models.WorkloadEntry, error)
	List(ctx context.Context, filter *models.WorkloadEntryFilter) ([]*models.WorkloadEntry, error)
	Count(ctx context.Context, filter *models.WorkloadEntryFilter) (int, error)
	ListByParent(ctx context.Context, tenantID, parentID string) ([]*models.WorkloadEntry, error)
	ClaimUnassignedEntries(ctx context.Context, tenantID, agentSpiffeID string) (int64, error)
	Update(ctx context.Context, entry *models.WorkloadEntry) error
	Delete(ctx context.Context, id string) error
	FindMatchingEntries(ctx context.Context, tenantID string, selectors map[string]string) ([]*models.WorkloadEntry, error)
}
