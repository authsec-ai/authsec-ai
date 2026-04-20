package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
)

// PostgresPolicyRepository implements the PolicyRepository interface
type PostgresPolicyRepository struct {
	db *sql.DB
}

// NewPostgresPolicyRepository creates a new policy repository
func NewPostgresPolicyRepository(db *sql.DB) repositories.PolicyRepository {
	return &PostgresPolicyRepository{db: db}
}

// GetByID retrieves a policy by ID
func (r *PostgresPolicyRepository) GetByID(ctx context.Context, tenantID, id string) (*models.AttestationPolicy, error) {
	query := `
		SELECT id, tenant_id, name, description, attestation_type, selector_rules, vault_role,
			ttl, priority, enabled, created_at, updated_at
		FROM attestation_policies
		WHERE id = $1 AND tenant_id = $2
	`

	return r.scanPolicy(ctx, query, id, tenantID)
}

// Create creates a new policy
func (r *PostgresPolicyRepository) Create(ctx context.Context, policy *models.AttestationPolicy) error {
	query := `
		INSERT INTO attestation_policies (id, tenant_id, name, description, attestation_type,
			selector_rules, vault_role, ttl, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	now := time.Now()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	rulesJSON, err := json.Marshal(policy.SelectorRules)
	if err != nil {
		return errors.NewInternalError("Failed to marshal selector rules", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		policy.ID,
		policy.TenantID,
		policy.Name,
		policy.Description,
		policy.AttestationType,
		rulesJSON,
		policy.VaultRole,
		policy.TTL,
		policy.Priority,
		policy.Enabled,
		policy.CreatedAt,
		policy.UpdatedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to create policy", err)
	}

	return nil
}

// Update updates an existing policy
func (r *PostgresPolicyRepository) Update(ctx context.Context, policy *models.AttestationPolicy) error {
	query := `
		UPDATE attestation_policies
		SET name = $3, description = $4, attestation_type = $5, selector_rules = $6,
			vault_role = $7, ttl = $8, priority = $9, enabled = $10, updated_at = $11
		WHERE id = $1 AND tenant_id = $2
	`

	policy.UpdatedAt = time.Now()

	rulesJSON, err := json.Marshal(policy.SelectorRules)
	if err != nil {
		return errors.NewInternalError("Failed to marshal selector rules", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		policy.ID,
		policy.TenantID,
		policy.Name,
		policy.Description,
		policy.AttestationType,
		rulesJSON,
		policy.VaultRole,
		policy.TTL,
		policy.Priority,
		policy.Enabled,
		policy.UpdatedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to update policy", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get affected rows", err)
	}

	if rows == 0 {
		return errors.NewNotFoundError("Policy not found", nil)
	}

	return nil
}

// Delete deletes a policy
func (r *PostgresPolicyRepository) Delete(ctx context.Context, tenantID, id string) error {
	query := `DELETE FROM attestation_policies WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return errors.NewInternalError("Failed to delete policy", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get affected rows", err)
	}

	if rows == 0 {
		return errors.NewNotFoundError("Policy not found", nil)
	}

	return nil
}

// ListByTenant retrieves all policies for a tenant
func (r *PostgresPolicyRepository) ListByTenant(ctx context.Context, tenantID string) ([]*models.AttestationPolicy, error) {
	query := `
		SELECT id, tenant_id, name, description, attestation_type, selector_rules, vault_role,
			ttl, priority, enabled, created_at, updated_at
		FROM attestation_policies
		WHERE tenant_id = $1
		ORDER BY priority DESC, created_at DESC
	`

	return r.queryPolicies(ctx, query, tenantID)
}

// FindMatchingPolicy finds the best matching policy for given selectors
func (r *PostgresPolicyRepository) FindMatchingPolicy(ctx context.Context, tenantID, attestationType string, selectors map[string]string) (*models.AttestationPolicy, error) {
	query := `
		SELECT id, tenant_id, name, description, attestation_type, selector_rules, vault_role,
			ttl, priority, enabled, created_at, updated_at
		FROM attestation_policies
		WHERE tenant_id = $1 AND attestation_type = $2 AND enabled = true
		ORDER BY priority DESC
	`

	policies, err := r.queryPolicies(ctx, query, tenantID, attestationType)
	if err != nil {
		return nil, err
	}

	// Find first matching policy
	for _, policy := range policies {
		if policy.MatchesSelectors(selectors) {
			return policy, nil
		}
	}

	return nil, errors.NewNotFoundError("No matching policy found", nil)
}

// scanPolicy scans a single policy
func (r *PostgresPolicyRepository) scanPolicy(ctx context.Context, query string, args ...interface{}) (*models.AttestationPolicy, error) {
	policy := &models.AttestationPolicy{}
	var rulesJSON []byte

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&policy.ID,
		&policy.TenantID,
		&policy.Name,
		&policy.Description,
		&policy.AttestationType,
		&rulesJSON,
		&policy.VaultRole,
		&policy.TTL,
		&policy.Priority,
		&policy.Enabled,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("Policy not found", err)
	}
	if err != nil {
		return nil, errors.NewInternalError("Failed to get policy", err)
	}

	if err := json.Unmarshal(rulesJSON, &policy.SelectorRules); err != nil {
		return nil, errors.NewInternalError("Failed to unmarshal selector rules", err)
	}

	return policy, nil
}

// queryPolicies queries multiple policies
func (r *PostgresPolicyRepository) queryPolicies(ctx context.Context, query string, args ...interface{}) ([]*models.AttestationPolicy, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewInternalError("Failed to query policies", err)
	}
	defer rows.Close()

	var policies []*models.AttestationPolicy
	for rows.Next() {
		policy := &models.AttestationPolicy{}
		var rulesJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.TenantID,
			&policy.Name,
			&policy.Description,
			&policy.AttestationType,
			&rulesJSON,
			&policy.VaultRole,
			&policy.TTL,
			&policy.Priority,
			&policy.Enabled,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, errors.NewInternalError("Failed to scan policy", err)
		}

		if err := json.Unmarshal(rulesJSON, &policy.SelectorRules); err != nil {
			return nil, errors.NewInternalError("Failed to unmarshal selector rules", err)
		}

		policies = append(policies, policy)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.NewInternalError("Error iterating policies", err)
	}

	return policies, nil
}
