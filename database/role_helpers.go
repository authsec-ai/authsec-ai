package database

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// roleExecutor abstracts *sql.DB and *sql.Tx for role operations.
type roleExecutor interface {
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

// EnsureAdminRoleWithExecutor ensures an admin role exists for the given tenant_id and returns its id.
func EnsureAdminRoleWithExecutor(exec roleExecutor, tenantID uuid.UUID) (uuid.UUID, error) {
	var roleID uuid.UUID
	err := exec.QueryRow(`SELECT id FROM roles WHERE LOWER(name) = 'admin' AND tenant_id = $1 LIMIT 1`, tenantID).Scan(&roleID)
	if err == nil {
		return roleID, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("failed to lookup admin role: %w", err)
	}

	roleID = uuid.New()
	_, err = exec.Exec(`
		INSERT INTO roles (id, tenant_id, name, description, created_at, updated_at)
		VALUES ($1, $2, 'admin', 'Administrator with full access', NOW(), NOW())
		ON CONFLICT ON CONSTRAINT roles_tenant_name_key DO NOTHING
	`, roleID, tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert admin role: %w", err)
	}

	if err := exec.QueryRow(`SELECT id FROM roles WHERE LOWER(name) = 'admin' AND tenant_id = $1 LIMIT 1`, tenantID).Scan(&roleID); err != nil {
		return uuid.Nil, fmt.Errorf("failed to fetch admin role after insert: %w", err)
	}
	return roleID, nil
}
