package models

import "time"

// Workload represents a workload that can request certificates
type Workload struct {
	ID              string            `json:"id" db:"id"`
	TenantID        string            `json:"tenant_id" db:"tenant_id"`
	SpiffeID        string            `json:"spiffe_id" db:"spiffe_id"`
	Selectors       map[string]string `json:"selectors" db:"selectors"`
	VaultRole       string            `json:"vault_role" db:"vault_role"`
	Status          string            `json:"status" db:"status"` // active, revoked, expired
	AttestationType string            `json:"attestation_type" db:"attestation_type"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}
