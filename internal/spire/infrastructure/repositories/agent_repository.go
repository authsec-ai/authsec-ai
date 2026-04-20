package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/errors"

	"github.com/sirupsen/logrus"
)

// PostgresAgentRepository implements AgentRepository for PostgreSQL
type PostgresAgentRepository struct {
	db     *sql.DB
	logger *logrus.Entry
}

// NewPostgresAgentRepository creates a new PostgreSQL agent repository
func NewPostgresAgentRepository(db *sql.DB, logger *logrus.Entry) *PostgresAgentRepository {
	return &PostgresAgentRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new agent record
func (r *PostgresAgentRepository) Create(ctx context.Context, agent *models.Agent) error {
	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now

	nodeSelectorsJSON, err := json.Marshal(agent.NodeSelectors)
	if err != nil {
		return errors.NewInternalError("Failed to marshal node selectors", err)
	}

	query := `
		INSERT INTO agents (
			id, tenant_id, node_id, spiffe_id, attestation_type,
			node_selectors, certificate_serial, status, cluster_name,
			last_seen, last_heartbeat, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	// Use LastHeartbeat if set, otherwise use LastSeen for backward compatibility
	lastHeartbeat := agent.LastHeartbeat
	if lastHeartbeat.IsZero() {
		lastHeartbeat = agent.LastSeen
	}

	_, err = r.db.ExecContext(ctx, query,
		agent.ID, agent.TenantID, agent.NodeID, agent.SpiffeID,
		agent.AttestationType, nodeSelectorsJSON, agent.CertificateSerial,
		agent.Status, agent.ClusterName, agent.LastSeen, lastHeartbeat,
		agent.CreatedAt, agent.UpdatedAt,
	)

	if err != nil {
		r.logger.WithError(err).Error("Failed to create agent")
		return errors.NewInternalError("Failed to create agent", err)
	}

	r.logger.WithFields(logrus.Fields{
		"agent_id":  agent.ID,
		"tenant_id": agent.TenantID,
		"spiffe_id": agent.SpiffeID,
	}).Info("Agent created")

	return nil
}

// GetByID retrieves an agent by ID
func (r *PostgresAgentRepository) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	query := `
		SELECT id, tenant_id, node_id, spiffe_id, attestation_type,
		       node_selectors, certificate_serial, status, cluster_name,
		       last_seen, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE id = $1
	`

	agent := &models.Agent{}
	var nodeSelectorsJSON []byte
	var clusterName sql.NullString
	var lastHeartbeat sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&agent.ID, &agent.TenantID, &agent.NodeID, &agent.SpiffeID,
		&agent.AttestationType, &nodeSelectorsJSON, &agent.CertificateSerial,
		&agent.Status, &clusterName, &agent.LastSeen, &lastHeartbeat,
		&agent.CreatedAt, &agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("Agent not found", err)
	}
	if err != nil {
		r.logger.WithError(err).Error("Failed to get agent by ID")
		return nil, errors.NewInternalError("Failed to get agent", err)
	}

	// Handle nullable fields
	if clusterName.Valid {
		agent.ClusterName = clusterName.String
	}
	if lastHeartbeat.Valid {
		agent.LastHeartbeat = lastHeartbeat.Time
	}

	if len(nodeSelectorsJSON) > 0 {
		if err := json.Unmarshal(nodeSelectorsJSON, &agent.NodeSelectors); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal node selectors")
			return nil, errors.NewInternalError("Failed to parse node selectors", err)
		}
	}

	return agent, nil
}

// GetBySpiffeID retrieves an agent by SPIFFE ID
func (r *PostgresAgentRepository) GetBySpiffeID(ctx context.Context, spiffeID string) (*models.Agent, error) {
	query := `
		SELECT id, tenant_id, node_id, spiffe_id, attestation_type,
		       node_selectors, certificate_serial, status, cluster_name,
		       last_seen, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE spiffe_id = $1
	`

	agent := &models.Agent{}
	var nodeSelectorsJSON []byte
	var clusterName sql.NullString
	var lastHeartbeat sql.NullTime

	err := r.db.QueryRowContext(ctx, query, spiffeID).Scan(
		&agent.ID, &agent.TenantID, &agent.NodeID, &agent.SpiffeID,
		&agent.AttestationType, &nodeSelectorsJSON, &agent.CertificateSerial,
		&agent.Status, &clusterName, &agent.LastSeen, &lastHeartbeat,
		&agent.CreatedAt, &agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("Agent not found", err)
	}
	if err != nil {
		r.logger.WithError(err).Error("Failed to get agent by SPIFFE ID")
		return nil, errors.NewInternalError("Failed to get agent", err)
	}

	// Handle nullable fields
	if clusterName.Valid {
		agent.ClusterName = clusterName.String
	}
	if lastHeartbeat.Valid {
		agent.LastHeartbeat = lastHeartbeat.Time
	}

	if len(nodeSelectorsJSON) > 0 {
		if err := json.Unmarshal(nodeSelectorsJSON, &agent.NodeSelectors); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal node selectors")
			return nil, errors.NewInternalError("Failed to parse node selectors", err)
		}
	}

	return agent, nil
}

// GetByTenantAndNode retrieves an agent by tenant ID and node ID
func (r *PostgresAgentRepository) GetByTenantAndNode(ctx context.Context, tenantID, nodeID string) (*models.Agent, error) {
	query := `
		SELECT id, tenant_id, node_id, spiffe_id, attestation_type,
		       node_selectors, certificate_serial, status, cluster_name,
		       last_seen, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE tenant_id = $1 AND node_id = $2
	`

	agent := &models.Agent{}
	var nodeSelectorsJSON []byte
	var clusterName sql.NullString
	var lastHeartbeat sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, nodeID).Scan(
		&agent.ID, &agent.TenantID, &agent.NodeID, &agent.SpiffeID,
		&agent.AttestationType, &nodeSelectorsJSON, &agent.CertificateSerial,
		&agent.Status, &clusterName, &agent.LastSeen, &lastHeartbeat,
		&agent.CreatedAt, &agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("Agent not found", err)
	}
	if err != nil {
		r.logger.WithError(err).Error("Failed to get agent by tenant and node")
		return nil, errors.NewInternalError("Failed to get agent", err)
	}

	// Handle nullable fields
	if clusterName.Valid {
		agent.ClusterName = clusterName.String
	}
	if lastHeartbeat.Valid {
		agent.LastHeartbeat = lastHeartbeat.Time
	}

	if len(nodeSelectorsJSON) > 0 {
		if err := json.Unmarshal(nodeSelectorsJSON, &agent.NodeSelectors); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal node selectors")
			return nil, errors.NewInternalError("Failed to parse node selectors", err)
		}
	}

	return agent, nil
}

// Update updates an existing agent record
func (r *PostgresAgentRepository) Update(ctx context.Context, agent *models.Agent) error {
	agent.UpdatedAt = time.Now()

	nodeSelectorsJSON, err := json.Marshal(agent.NodeSelectors)
	if err != nil {
		return errors.NewInternalError("Failed to marshal node selectors", err)
	}

	// Use LastHeartbeat if set, otherwise use LastSeen for backward compatibility
	lastHeartbeat := agent.LastHeartbeat
	if lastHeartbeat.IsZero() {
		lastHeartbeat = agent.LastSeen
	}

	query := `
		UPDATE agents
		SET node_id = $1, spiffe_id = $2, attestation_type = $3,
		    node_selectors = $4, certificate_serial = $5, status = $6,
		    cluster_name = $7, last_seen = $8, last_heartbeat = $9,
		    updated_at = $10
		WHERE id = $11
	`

	result, err := r.db.ExecContext(ctx, query,
		agent.NodeID, agent.SpiffeID, agent.AttestationType,
		nodeSelectorsJSON, agent.CertificateSerial, agent.Status,
		agent.ClusterName, agent.LastSeen, lastHeartbeat,
		agent.UpdatedAt, agent.ID,
	)

	if err != nil {
		r.logger.WithError(err).Error("Failed to update agent")
		return errors.NewInternalError("Failed to update agent", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("Agent not found", nil)
	}

	r.logger.WithFields(logrus.Fields{
		"agent_id": agent.ID,
		"status":   agent.Status,
	}).Info("Agent updated")

	return nil
}

// Delete deletes an agent by ID
func (r *PostgresAgentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM agents WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete agent")
		return errors.NewInternalError("Failed to delete agent", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("Agent not found", nil)
	}

	r.logger.WithField("agent_id", id).Info("Agent deleted")

	return nil
}

// ListByTenant lists all agents for a tenant
func (r *PostgresAgentRepository) ListByTenant(ctx context.Context, tenantID string) ([]*models.Agent, error) {
	query := `
		SELECT id, tenant_id, node_id, spiffe_id, attestation_type,
		       node_selectors, certificate_serial, status, last_seen,
		       created_at, updated_at
		FROM agents
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to list agents by tenant")
		return nil, errors.NewInternalError("Failed to list agents", err)
	}
	defer rows.Close()

	agents := []*models.Agent{}
	for rows.Next() {
		agent := &models.Agent{}
		var nodeSelectorsJSON []byte

		err := rows.Scan(
			&agent.ID, &agent.TenantID, &agent.NodeID, &agent.SpiffeID,
			&agent.AttestationType, &nodeSelectorsJSON, &agent.CertificateSerial,
			&agent.Status, &agent.LastSeen, &agent.CreatedAt, &agent.UpdatedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan agent row")
			return nil, errors.NewInternalError("Failed to scan agent", err)
		}

		if len(nodeSelectorsJSON) > 0 {
			if err := json.Unmarshal(nodeSelectorsJSON, &agent.NodeSelectors); err != nil {
				r.logger.WithError(err).Error("Failed to unmarshal node selectors")
				continue
			}
		}

		agents = append(agents, agent)
	}

	if err := rows.Err(); err != nil {
		r.logger.WithError(err).Error("Error iterating agent rows")
		return nil, errors.NewInternalError("Failed to iterate agents", err)
	}

	return agents, nil
}

// UpdateLastSeen updates the last_seen timestamp for an agent
func (r *PostgresAgentRepository) UpdateLastSeen(ctx context.Context, id string) error {
	query := `UPDATE agents SET last_seen = $1, updated_at = $2 WHERE id = $3`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, now, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to update agent last_seen")
		return errors.NewInternalError("Failed to update last_seen", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("Agent not found", nil)
	}

	return nil
}
