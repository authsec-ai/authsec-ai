package models

import "time"

// Agent represents an attested agent (node) identity
type Agent struct {
	ID                string            `json:"id"`
	TenantID          string            `json:"tenant_id"`
	NodeID            string            `json:"node_id"`
	SpiffeID          string            `json:"spiffe_id"`
	AttestationType   string            `json:"attestation_type"`
	NodeSelectors     map[string]string `json:"node_selectors"`
	CertificateSerial string            `json:"certificate_serial"`
	Status            string            `json:"status"` // active, expired, revoked
	ClusterName       string            `json:"cluster_name"`
	LastSeen          time.Time         `json:"last_seen"`
	LastHeartbeat     time.Time         `json:"last_heartbeat"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// AgentStatus constants
const (
	AgentStatusActive  = "active"
	AgentStatusExpired = "expired"
	AgentStatusRevoked = "revoked"
)

// AttestationType constants
const (
	AttestationTypeKubernetes = "kubernetes"
	AttestationTypeTPM        = "tpm"
	AttestationTypeAWS        = "aws"
	AttestationTypeGCP        = "gcp"
	AttestationTypeAzure      = "azure"
	AttestationTypeUnix       = "unix"
	AttestationTypeDocker     = "docker"
)
