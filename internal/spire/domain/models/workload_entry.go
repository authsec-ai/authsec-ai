package models

import "time"

// WorkloadEntry represents a workload registration entry.
// It maps workload selectors to SPIFFE identities.
type WorkloadEntry struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id"`
	SpiffeID   string            `json:"spiffe_id"`
	ParentID   string            `json:"parent_id"`   // Agent SPIFFE ID; empty = broadcast to all agents in tenant
	Selectors  map[string]string `json:"selectors"`   // Workload selectors (k8s:ns, unix:uid, etc.)
	TTL        *int              `json:"ttl"`          // Certificate TTL override (seconds), nil = use default
	Admin      bool              `json:"admin"`
	Downstream bool              `json:"downstream"`   // Can issue downstream identities
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// WorkloadEntryFilter represents filters for querying workload entries
type WorkloadEntryFilter struct {
	TenantID        string            `json:"tenant_id"`
	ParentID        string            `json:"parent_id"`
	SpiffeID        string            `json:"spiffe_id"`
	SpiffeIDPartial bool              `json:"spiffe_id_partial"`
	Selectors       map[string]string `json:"selectors"`
	SelectorType    string            `json:"selector_type"`
	Admin           *bool             `json:"admin"`
	Limit           int               `json:"limit"`
	Offset          int               `json:"offset"`
}

// MatchesSelectors checks if this workload entry matches the given selectors.
// Returns true if ALL entry selectors are present in the provided selectors (subset match).
func (we *WorkloadEntry) MatchesSelectors(workloadSelectors map[string]string) bool {
	for key, value := range we.Selectors {
		workloadValue, exists := workloadSelectors[key]
		if !exists || workloadValue != value {
			return false
		}
	}
	return true
}

// Validate performs basic validation on the workload entry
func (we *WorkloadEntry) Validate() error {
	if we.TenantID == "" {
		return ErrInvalidInput("tenant_id is required")
	}
	if we.SpiffeID == "" {
		return ErrInvalidInput("spiffe_id is required")
	}
	if len(we.Selectors) == 0 {
		return ErrInvalidInput("at least one selector is required")
	}
	if we.TTL != nil && *we.TTL <= 0 {
		return ErrInvalidInput("ttl must be positive")
	}
	return nil
}

// ErrInvalidInput creates an invalid input error
func ErrInvalidInput(message string) error {
	return &ValidationError{Message: message}
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
