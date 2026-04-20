package models

import "time"

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	VaultMount  string    `json:"vault_mount" db:"vault_mount"`
	DatabaseURL string    `json:"-" db:"database_url"`
	Status      string    `json:"status" db:"status"` // active, suspended, deleted
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// IsActive checks if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == "active"
}
