package models

import "time"

// Certificate represents an issued certificate
type Certificate struct {
	ID                string     `json:"id" db:"id"`
	TenantID          string     `json:"tenant_id" db:"tenant_id"`
	WorkloadID        string     `json:"workload_id" db:"workload_id"`
	SerialNumber      string     `json:"serial_number" db:"serial_number"`
	SHA256Fingerprint string     `json:"sha256_fingerprint" db:"sha256_fingerprint"`
	SpiffeID          string     `json:"spiffe_id" db:"spiffe_id"`
	CertPEM           string     `json:"-" db:"cert_pem"`
	CAChain           []string   `json:"-" db:"ca_chain"`
	IssuedAt          time.Time  `json:"issued_at" db:"issued_at"`
	ExpiresAt         time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	Status            string     `json:"status" db:"status"`         // active, expired, revoked
	IssueType         string     `json:"issue_type" db:"issue_type"` // attest, renew
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// IsValid checks if the certificate is currently valid
func (c *Certificate) IsValid() bool {
	now := time.Now()
	return c.Status == "active" && now.Before(c.ExpiresAt) && c.RevokedAt == nil
}

// IsExpiringSoon checks if the certificate is expiring within the given duration
func (c *Certificate) IsExpiringSoon(threshold time.Duration) bool {
	return time.Until(c.ExpiresAt) < threshold
}
