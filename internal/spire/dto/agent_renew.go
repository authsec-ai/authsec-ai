package dto

import "time"

// AgentRenewRequest represents an agent SVID renewal request
type AgentRenewRequest struct {
	AgentID  string `json:"agent_id"`
	TenantID string `json:"tenant_id,omitempty"` // Optional: for non-mTLS environments
	CSR      string `json:"csr"`
}

// AgentRenewResponse represents an agent SVID renewal response
type AgentRenewResponse struct {
	SpiffeID    string    `json:"spiffe_id"`
	Certificate string    `json:"certificate"`
	CABundle    string    `json:"ca_bundle"`
	TTL         int       `json:"ttl"`
	CAChain     []string  `json:"ca_chain,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}
