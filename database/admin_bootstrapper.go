package database

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

// AdminBootstrapper seeds per-tenant admin artifacts at service startup.
type AdminBootstrapper struct {
	db *DBConnection
}

// NewAdminBootstrapper constructs a bootstrapper.
func NewAdminBootstrapper(db *DBConnection) *AdminBootstrapper {
	return &AdminBootstrapper{db: db}
}

// SeedAllTenants ensures every tenant has admin role, permissions, and bindings.
func (b *AdminBootstrapper) SeedAllTenants() error {
	if b == nil || b.db == nil {
		return fmt.Errorf("admin bootstrapper not initialized")
	}

	log.Println("DEBUG: SeedAllTenants - about to query tenants")
	rows, err := b.db.Query(`SELECT COALESCE(tenant_id, id) AS tenant_id FROM tenants`)
	if err != nil {
		return fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	log.Println("DEBUG: SeedAllTenants - query completed, iterating tenants")
	aur := NewAdminUserRepository(b.db)
	count := 0

	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			log.Printf("Skipping tenant scan error: %v", err)
			continue
		}

		if tenantID == uuid.Nil {
			continue
		}
		count++
		log.Printf("DEBUG: SeedAllTenants - processing tenant %d: %s", count, tenantID)

		if err := aur.EnsureTenantAdminRoleAssignment(tenantID); err != nil {
			log.Printf("Warning: failed to seed admin artifacts for tenant %s: %v", tenantID, err)
		}
	}

	log.Printf("DEBUG: SeedAllTenants - done processing %d tenants", count)

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate tenants: %w", err)
	}

	return nil
}
