package dto

import "time"

// AttestResponse represents the HTTP response for attestation
type AttestResponse struct {
	Certificate  string    `json:"certificate"`
	CAChain      []string  `json:"ca_chain"`
	SpiffeID     string    `json:"spiffe_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	WorkloadID   string    `json:"workload_id"`
	SerialNumber string    `json:"serial_number"`
}

// RenewResponse represents the HTTP response for renewal
type RenewResponse struct {
	Certificate  string    `json:"certificate"`
	CAChain      []string  `json:"ca_chain"`
	ExpiresAt    time.Time `json:"expires_at"`
	SerialNumber string    `json:"serial_number"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BundleResponse represents the CA bundle response
type BundleResponse struct {
	CABundle string `json:"ca_bundle"`
	TenantID string `json:"tenant_id"`
}

// ListWorkloadsResponse represents the HTTP response for listing workloads
type ListWorkloadsResponse struct {
	Workloads interface{} `json:"workloads"`
	Count     int         `json:"count"`
}

// DeleteWorkloadResponse represents the HTTP response for deleting a workload
type DeleteWorkloadResponse struct {
	Message    string `json:"message"`
	WorkloadID string `json:"workload_id"`
}

// AgentResponse represents an agent in the response
type AgentResponse struct {
	ID              string    `json:"id"`
	SpiffeID        string    `json:"spiffe_id"`
	NodeID          string    `json:"node_id"`
	AttestationType string    `json:"attestation_type"`
	Status          string    `json:"status"`
	LastSeen        time.Time `json:"last_seen"`
	CreatedAt       time.Time `json:"created_at"`
}

// ListAgentsResponse represents the HTTP response for listing agents
type ListAgentsResponse struct {
	Agents []*AgentResponse `json:"agents"`
	Count  int              `json:"count"`
}
