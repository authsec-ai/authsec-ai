package services

import (
	"context"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/database"
	infrarepos "github.com/authsec-ai/authsec/internal/spire/infrastructure/repositories"

	"github.com/sirupsen/logrus"
)

// AgentService handles agent operations
type AgentService struct {
	connManager *database.ConnectionManager
	tenantRepo  repositories.TenantRepository
	logger      *logrus.Entry
}

// NewAgentService creates a new agent service
func NewAgentService(
	connManager *database.ConnectionManager,
	tenantRepo repositories.TenantRepository,
	logger *logrus.Entry,
) *AgentService {
	return &AgentService{
		connManager: connManager,
		tenantRepo:  tenantRepo,
		logger:      logger,
	}
}

// ListAgentsByTenant lists all active agents for a tenant
func (s *AgentService) ListAgentsByTenant(ctx context.Context, tenantID string) ([]*models.Agent, error) {
	s.logger.WithField("tenant_id", tenantID).Info("Listing agents for tenant")

	// Get tenant-specific database connection
	db, err := s.connManager.GetTenantDB(ctx, tenantID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tenant database connection")
		return nil, errors.NewNotFoundError("Tenant not found", err)
	}

	// Create agent repository
	agentRepo := infrarepos.NewPostgresAgentRepository(db, s.logger)

	// List all agents for the tenant
	agents, err := agentRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{"tenant_id": tenantID}).WithError(err).Error("Failed to list agents")
		return nil, errors.NewInternalError("Failed to list agents", err)
	}

	// Filter to only active agents
	activeAgents := make([]*models.Agent, 0, len(agents))
	for _, agent := range agents {
		if agent.Status == models.AgentStatusActive {
			activeAgents = append(activeAgents, agent)
		}
	}

	s.logger.WithFields(logrus.Fields{"tenant_id": tenantID, "count": len(activeAgents)}).Info("Successfully listed agents")

	return activeAgents, nil
}
