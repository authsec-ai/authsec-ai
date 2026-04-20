package dto

// NodeAttestRequest represents a node attestation request
type NodeAttestRequest struct {
	TenantID        string                 `json:"tenant_id"`
	NodeID          string                 `json:"node_id"`
	AttestationType string                 `json:"attestation_type"` // "kubernetes", "tpm", "aws"
	Evidence        map[string]interface{} `json:"evidence"`
	CSR             string                 `json:"csr"`
}

// NodeAttestResponse represents a node attestation response
type NodeAttestResponse struct {
	AgentID     string `json:"agent_id"`
	SpiffeID    string `json:"spiffe_id"`
	Certificate string `json:"certificate"`
	CABundle    string `json:"ca_bundle"` // Joined CA chain as single PEM string
	TTL         int    `json:"ttl"`       // Certificate TTL in seconds
}
