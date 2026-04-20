package repositories

import (
	"context"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
)

// AgentRepository defines the interface for agent data access
type AgentRepository interface {
	Create(ctx context.Context, agent *models.Agent) error
	GetByID(ctx context.Context, id string) (*models.Agent, error)
	GetBySpiffeID(ctx context.Context, spiffeID string) (*models.Agent, error)
	GetByTenantAndNode(ctx context.Context, tenantID, nodeID string) (*models.Agent, error)
	Update(ctx context.Context, agent *models.Agent) error
	Delete(ctx context.Context, id string) error
	ListByTenant(ctx context.Context, tenantID string) ([]*models.Agent, error)
	UpdateLastSeen(ctx context.Context, id string) error
}
