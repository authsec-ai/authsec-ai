package models

import "time"

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	EventAttest AuditEventType = "attest"
	EventRenew  AuditEventType = "renew"
	EventRevoke AuditEventType = "revoke"
	EventBundle AuditEventType = "bundle"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID            string                 `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	EventType     AuditEventType         `json:"event_type" db:"event_type"`
	WorkloadID    string                 `json:"workload_id,omitempty" db:"workload_id"`
	CertificateID string                 `json:"certificate_id,omitempty" db:"certificate_id"`
	SpiffeID      string                 `json:"spiffe_id,omitempty" db:"spiffe_id"`
	Success       bool                   `json:"success" db:"success"`
	ErrorMessage  string                 `json:"error_message,omitempty" db:"error_message"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	IPAddress     string                 `json:"ip_address" db:"ip_address"`
	UserAgent     string                 `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}
