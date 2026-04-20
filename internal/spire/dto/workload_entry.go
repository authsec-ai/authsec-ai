package dto

import "time"

// CreateWorkloadEntryRequest represents a request to create a workload entry
type CreateWorkloadEntryRequest struct {
	TenantID   string            `json:"tenant_id" binding:"required"`
	SpiffeID   string            `json:"spiffe_id" binding:"required"`
	ParentID   string            `json:"parent_id"`                    // Optional: populated when agent starts running
	Selectors  map[string]string `json:"selectors" binding:"required"`
	TTL        *int              `json:"ttl"`        // Optional TTL override in seconds
	Admin      bool              `json:"admin"`      // Default: false
	Downstream bool              `json:"downstream"` // Default: false
}

// UpdateWorkloadEntryRequest represents a request to update a workload entry
type UpdateWorkloadEntryRequest struct {
	SpiffeID   string            `json:"spiffe_id" binding:"required"`
	ParentID   string            `json:"parent_id" binding:"required"`
	Selectors  map[string]string `json:"selectors" binding:"required"`
	TTL        *int              `json:"ttl"`
	Admin      bool              `json:"admin"`
	Downstream bool              `json:"downstream"`
}

// WorkloadEntryResponse represents a workload entry response
type WorkloadEntryResponse struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id"`
	SpiffeID   string            `json:"spiffe_id"`
	ParentID   string            `json:"parent_id"`
	Selectors  map[string]string `json:"selectors"`
	TTL        *int              `json:"ttl"`
	Admin      bool              `json:"admin"`
	Downstream bool              `json:"downstream"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// CreateAgentEntryRequest creates a workload entry for an AI agent.
// authsec-spire generates the SPIFFE ID from tenant_id, client_id, and agent_type.
type CreateAgentEntryRequest struct {
	TenantID  string            `json:"tenant_id" binding:"required"`
	ClientID  string            `json:"client_id" binding:"required"`
	AgentType string            `json:"agent_type" binding:"required"`
	ParentID  string            `json:"parent_id"`                   // Optional: auto-populated when agent starts running
	Selectors map[string]string `json:"selectors"` // Optional extra selectors
	TTL       *int              `json:"ttl"`        // Optional TTL override in seconds
}

// CreateAgentEntryResponse returns the generated SPIFFE ID and entry details
type CreateAgentEntryResponse struct {
	EntryID  string            `json:"entry_id"`
	SpiffeID string            `json:"spiffe_id"`
	TenantID string            `json:"tenant_id"`
	ClientID string            `json:"client_id"`
	ParentID string            `json:"parent_id"`
	Selectors map[string]string `json:"selectors"`
	TTL       *int              `json:"ttl"`
	CreatedAt time.Time         `json:"created_at"`
}

// ListWorkloadEntriesRequest represents query parameters for listing workload entries
type ListWorkloadEntriesRequest struct {
	TenantID       string `form:"tenant_id" binding:"required"`
	ParentID       string `form:"parent_id"`        // Optional: filter by parent agent
	SpiffeID       string `form:"spiffe_id"`        // Optional: exact SPIFFE ID match
	SpiffeIDSearch string `form:"spiffe_id_search"` // Optional: partial SPIFFE ID search (use instead of spiffe_id)
	SelectorType   string `form:"selector_type"`    // Optional: filter by type (unix, kubernetes, docker)
	Admin          *bool  `form:"admin"`            // Optional: filter by admin flag
	Limit          int    `form:"limit"`            // Optional: pagination limit (default: 100, max: 1000)
	Offset         int    `form:"offset"`           // Optional: pagination offset (default: 0)
}

// ListWorkloadEntriesResponse represents a list of workload entries
type ListWorkloadEntriesResponse struct {
	Entries []*WorkloadEntryResponse `json:"entries"`
	Total   int                      `json:"total"`
}
