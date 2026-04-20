package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// PostgresWorkloadEntryRepository implements WorkloadEntryRepository using PostgreSQL
type PostgresWorkloadEntryRepository struct {
	db     *sql.DB
	logger *logrus.Entry
}

// NewPostgresWorkloadEntryRepository creates a new PostgreSQL workload entry repository
func NewPostgresWorkloadEntryRepository(db *sql.DB, logger *logrus.Entry) repositories.WorkloadEntryRepository {
	return &PostgresWorkloadEntryRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new workload entry
func (r *PostgresWorkloadEntryRepository) Create(ctx context.Context, entry *models.WorkloadEntry) error {
	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Validate entry
	if err := entry.Validate(); err != nil {
		return err
	}

	// Convert selectors to JSONB
	selectorsJSON, err := json.Marshal(entry.Selectors)
	if err != nil {
		return fmt.Errorf("failed to marshal selectors: %w", err)
	}

	// Set timestamps
	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	query := `
		INSERT INTO workload_entries (
			id, tenant_id, spiffe_id, parent_id, selectors,
			ttl, admin, downstream, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.ExecContext(ctx, query,
		entry.ID,
		entry.TenantID,
		entry.SpiffeID,
		entry.ParentID,
		selectorsJSON,
		entry.TTL,
		entry.Admin,
		entry.Downstream,
		entry.CreatedAt,
		entry.UpdatedAt,
	)

	if err != nil {
		r.logger.WithError(err).WithField("spiffe_id", entry.SpiffeID).Error("Failed to create workload entry")
		return fmt.Errorf("failed to create workload entry: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"id":        entry.ID,
		"spiffe_id": entry.SpiffeID,
		"parent_id": entry.ParentID,
	}).Info("Workload entry created")

	return nil
}

// GetByID retrieves a workload entry by ID
func (r *PostgresWorkloadEntryRepository) GetByID(ctx context.Context, id string) (*models.WorkloadEntry, error) {
	query := `
		SELECT id, tenant_id, spiffe_id, parent_id, selectors,
		       ttl, admin, downstream, created_at, updated_at
		FROM workload_entries
		WHERE id = $1
	`

	var entry models.WorkloadEntry
	var selectorsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.TenantID,
		&entry.SpiffeID,
		&entry.ParentID,
		&selectorsJSON,
		&entry.TTL,
		&entry.Admin,
		&entry.Downstream,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to get workload entry by ID")
		return nil, fmt.Errorf("failed to get workload entry: %w", err)
	}

	// Unmarshal selectors
	if err := json.Unmarshal(selectorsJSON, &entry.Selectors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
	}

	return &entry, nil
}

// GetBySpiffeID retrieves a workload entry by SPIFFE ID
func (r *PostgresWorkloadEntryRepository) GetBySpiffeID(ctx context.Context, tenantID, spiffeID string) (*models.WorkloadEntry, error) {
	query := `
		SELECT id, tenant_id, spiffe_id, parent_id, selectors,
		       ttl, admin, downstream, created_at, updated_at
		FROM workload_entries
		WHERE tenant_id = $1 AND spiffe_id = $2
	`

	var entry models.WorkloadEntry
	var selectorsJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, spiffeID).Scan(
		&entry.ID,
		&entry.TenantID,
		&entry.SpiffeID,
		&entry.ParentID,
		&selectorsJSON,
		&entry.TTL,
		&entry.Admin,
		&entry.Downstream,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.logger.WithError(err).WithField("spiffe_id", spiffeID).Error("Failed to get workload entry by SPIFFE ID")
		return nil, fmt.Errorf("failed to get workload entry: %w", err)
	}

	// Unmarshal selectors
	if err := json.Unmarshal(selectorsJSON, &entry.Selectors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
	}

	return &entry, nil
}

// List retrieves workload entries based on filter criteria
func (r *PostgresWorkloadEntryRepository) List(ctx context.Context, filter *models.WorkloadEntryFilter) ([]*models.WorkloadEntry, error) {
	query := `
		SELECT id, tenant_id, spiffe_id, parent_id, selectors,
		       ttl, admin, downstream, created_at, updated_at
		FROM workload_entries
		WHERE tenant_id = $1
	`
	args := []interface{}{filter.TenantID}
	argCount := 1

	// Add optional filters
	if filter.ParentID != "" {
		argCount++
		query += fmt.Sprintf(" AND parent_id = $%d", argCount)
		args = append(args, filter.ParentID)
	}

	if filter.SpiffeID != "" {
		argCount++
		if filter.SpiffeIDPartial {
			// Use LIKE for partial matching (case-insensitive)
			query += fmt.Sprintf(" AND spiffe_id ILIKE $%d", argCount)
			args = append(args, "%"+filter.SpiffeID+"%")
		} else {
			// Exact match
			query += fmt.Sprintf(" AND spiffe_id = $%d", argCount)
			args = append(args, filter.SpiffeID)
		}
	}

	if filter.Admin != nil {
		argCount++
		query += fmt.Sprintf(" AND admin = $%d", argCount)
		args = append(args, *filter.Admin)
	}

	// Filter by selector type (unix, kubernetes, docker)
	if filter.SelectorType != "" {
		argCount++
		switch filter.SelectorType {
		case "unix":
			query += " AND (selectors ?? 'unix:uid' OR selectors ?? 'unix:gid' OR selectors ?? 'unix:pid' OR selectors ?? 'unix:user' OR selectors ?? 'unix:group')"
		case "kubernetes":
			query += " AND (selectors ?? 'k8s:ns' OR selectors ?? 'k8s:sa' OR selectors ?? 'k8s:pod-name' OR selectors ?? 'k8s:pod-uid' OR selectors ?? 'k8s:pod-label:app')"
		case "docker":
			query += " AND (selectors ?? 'docker:label' OR selectors ?? 'docker:env' OR selectors ?? 'docker:image_id')"
		}
		argCount-- // We didn't actually use a parameter
	}

	// Add ordering
	query += " ORDER BY created_at DESC"

	// Add pagination
	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).WithField("tenant_id", filter.TenantID).Error("Failed to list workload entries")
		return nil, fmt.Errorf("failed to list workload entries: %w", err)
	}
	defer rows.Close()

	var entries []*models.WorkloadEntry
	for rows.Next() {
		var entry models.WorkloadEntry
		var selectorsJSON []byte

		err := rows.Scan(
			&entry.ID,
			&entry.TenantID,
			&entry.SpiffeID,
			&entry.ParentID,
			&selectorsJSON,
			&entry.TTL,
			&entry.Admin,
			&entry.Downstream,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan workload entry")
			return nil, fmt.Errorf("failed to scan workload entry: %w", err)
		}

		// Unmarshal selectors
		if err := json.Unmarshal(selectorsJSON, &entry.Selectors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
		}

		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

// Count returns the total count of workload entries matching the filter
// This ignores limit and offset for accurate total count in pagination
func (r *PostgresWorkloadEntryRepository) Count(ctx context.Context, filter *models.WorkloadEntryFilter) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM workload_entries
		WHERE tenant_id = $1
	`
	args := []interface{}{filter.TenantID}
	argCount := 1

	// Add optional filters (same as List method)
	if filter.ParentID != "" {
		argCount++
		query += fmt.Sprintf(" AND parent_id = $%d", argCount)
		args = append(args, filter.ParentID)
	}

	if filter.SpiffeID != "" {
		argCount++
		if filter.SpiffeIDPartial {
			// Use LIKE for partial matching (case-insensitive)
			query += fmt.Sprintf(" AND spiffe_id ILIKE $%d", argCount)
			args = append(args, "%"+filter.SpiffeID+"%")
		} else {
			// Exact match
			query += fmt.Sprintf(" AND spiffe_id = $%d", argCount)
			args = append(args, filter.SpiffeID)
		}
	}

	if filter.Admin != nil {
		argCount++
		query += fmt.Sprintf(" AND admin = $%d", argCount)
		args = append(args, *filter.Admin)
	}

	// Filter by selector type (unix, kubernetes, docker)
	if filter.SelectorType != "" {
		switch filter.SelectorType {
		case "unix":
			query += " AND (selectors ?? 'unix:uid' OR selectors ?? 'unix:gid' OR selectors ?? 'unix:pid' OR selectors ?? 'unix:user' OR selectors ?? 'unix:group')"
		case "kubernetes":
			query += " AND (selectors ?? 'k8s:ns' OR selectors ?? 'k8s:sa' OR selectors ?? 'k8s:pod-name' OR selectors ?? 'k8s:pod-uid' OR selectors ?? 'k8s:pod-label:app')"
		case "docker":
			query += " AND (selectors ?? 'docker:label' OR selectors ?? 'docker:env' OR selectors ?? 'docker:image_id')"
		}
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		r.logger.WithError(err).WithField("tenant_id", filter.TenantID).Error("Failed to count workload entries")
		return 0, fmt.Errorf("failed to count workload entries: %w", err)
	}

	return count, nil
}

// ListByParent retrieves all workload entries for a specific parent (agent)
func (r *PostgresWorkloadEntryRepository) ListByParent(ctx context.Context, tenantID, parentID string) ([]*models.WorkloadEntry, error) {
	query := `
		SELECT id, tenant_id, spiffe_id, parent_id, selectors,
		       ttl, admin, downstream, created_at, updated_at
		FROM workload_entries
		WHERE tenant_id = $1 AND (parent_id = $2 OR parent_id = '')
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, parentID)
	if err != nil {
		r.logger.WithError(err).WithField("parent_id", parentID).Error("Failed to list workload entries by parent")
		return nil, fmt.Errorf("failed to list workload entries by parent: %w", err)
	}
	defer rows.Close()

	var entries []*models.WorkloadEntry
	for rows.Next() {
		var entry models.WorkloadEntry
		var selectorsJSON []byte

		err := rows.Scan(
			&entry.ID,
			&entry.TenantID,
			&entry.SpiffeID,
			&entry.ParentID,
			&selectorsJSON,
			&entry.TTL,
			&entry.Admin,
			&entry.Downstream,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan workload entry")
			return nil, fmt.Errorf("failed to scan workload entry: %w", err)
		}

		// Unmarshal selectors
		if err := json.Unmarshal(selectorsJSON, &entry.Selectors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
		}

		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"parent_id": parentID,
		"count":     len(entries),
	}).Info("Retrieved workload entries by parent")

	return entries, nil
}

// Update updates an existing workload entry
func (r *PostgresWorkloadEntryRepository) Update(ctx context.Context, entry *models.WorkloadEntry) error {
	// Validate entry
	if err := entry.Validate(); err != nil {
		return err
	}

	// Convert selectors to JSONB
	selectorsJSON, err := json.Marshal(entry.Selectors)
	if err != nil {
		return fmt.Errorf("failed to marshal selectors: %w", err)
	}

	// Update timestamp
	entry.UpdatedAt = time.Now()

	query := `
		UPDATE workload_entries
		SET spiffe_id = $1,
		    parent_id = $2,
		    selectors = $3,
		    ttl = $4,
		    admin = $5,
		    downstream = $6,
		    updated_at = $7
		WHERE id = $8
	`

	result, err := r.db.ExecContext(ctx, query,
		entry.SpiffeID,
		entry.ParentID,
		selectorsJSON,
		entry.TTL,
		entry.Admin,
		entry.Downstream,
		entry.UpdatedAt,
		entry.ID,
	)

	if err != nil {
		r.logger.WithError(err).WithField("id", entry.ID).Error("Failed to update workload entry")
		return fmt.Errorf("failed to update workload entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workload entry not found: %s", entry.ID)
	}

	r.logger.WithFields(logrus.Fields{
		"id":        entry.ID,
		"spiffe_id": entry.SpiffeID,
	}).Info("Workload entry updated")

	return nil
}

// Delete deletes a workload entry by ID
func (r *PostgresWorkloadEntryRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workload_entries WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to delete workload entry")
		return fmt.Errorf("failed to delete workload entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workload entry not found: %s", id)
	}

	r.logger.WithField("id", id).Info("Workload entry deleted")

	return nil
}

// ClaimUnassignedEntries sets parent_id on entries that have no parent_id assigned.
// This is called when an agent fetches its entries — unassigned entries in the tenant
// get claimed by the requesting agent so the UI shows the actual agent SPIFFE ID.
func (r *PostgresWorkloadEntryRepository) ClaimUnassignedEntries(ctx context.Context, tenantID, agentSpiffeID string) (int64, error) {
	query := `
		UPDATE workload_entries
		SET parent_id = $1, updated_at = $2
		WHERE tenant_id = $3 AND (parent_id = '' OR parent_id IS NULL)
	`

	result, err := r.db.ExecContext(ctx, query, agentSpiffeID, time.Now(), tenantID)
	if err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":       tenantID,
			"agent_spiffe_id": agentSpiffeID,
		}).Error("Failed to claim unassigned entries")
		return 0, fmt.Errorf("failed to claim unassigned entries: %w", err)
	}

	count, _ := result.RowsAffected()
	if count > 0 {
		r.logger.WithFields(logrus.Fields{
			"agent_spiffe_id": agentSpiffeID,
			"count":           count,
		}).Info("Claimed unassigned workload entries")
	}

	return count, nil
}

// FindMatchingEntries finds workload entries that match the given selectors
// Used during workload attestation to determine which SPIFFE ID to issue
func (r *PostgresWorkloadEntryRepository) FindMatchingEntries(ctx context.Context, tenantID string, selectors map[string]string) ([]*models.WorkloadEntry, error) {
	// Convert selectors to JSONB for PostgreSQL query
	selectorsJSON, err := json.Marshal(selectors)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal selectors: %w", err)
	}

	// Query uses @> operator: entry selectors must be a subset of workload selectors
	// This means ALL entry selectors must match (AND logic)
	query := `
		SELECT id, tenant_id, spiffe_id, parent_id, selectors,
		       ttl, admin, downstream, created_at, updated_at
		FROM workload_entries
		WHERE tenant_id = $1
		  AND $2::jsonb @> selectors
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, selectorsJSON)
	if err != nil {
		r.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to find matching workload entries")
		return nil, fmt.Errorf("failed to find matching workload entries: %w", err)
	}
	defer rows.Close()

	var entries []*models.WorkloadEntry
	for rows.Next() {
		var entry models.WorkloadEntry
		var entrySelectorsJSON []byte

		err := rows.Scan(
			&entry.ID,
			&entry.TenantID,
			&entry.SpiffeID,
			&entry.ParentID,
			&entrySelectorsJSON,
			&entry.TTL,
			&entry.Admin,
			&entry.Downstream,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan workload entry")
			return nil, fmt.Errorf("failed to scan workload entry: %w", err)
		}

		// Unmarshal selectors
		if err := json.Unmarshal(entrySelectorsJSON, &entry.Selectors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
		}

		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"count":     len(entries),
	}).Info("Found matching workload entries")

	return entries, nil
}
