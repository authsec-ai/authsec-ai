package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/authsec-ai/authsec/config"
)

// SpireService handles communication with authsec-spire for workload entry
// and JWT-SVID operations.
type SpireService struct {
	baseURL    string
	httpClient *http.Client
}

// NewSpireService creates a new SpireService using the ICP_SERVICE_URL from config.
func NewSpireService() *SpireService {
	return &SpireService{
		baseURL: config.AppConfig.ICPServiceURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// --- Request/Response types for authsec-spire ---

// SpireCreateAgentEntryRequest is sent to POST /v1/entries/agent
type SpireCreateAgentEntryRequest struct {
	TenantID  string            `json:"tenant_id"`
	ClientID  string            `json:"client_id"`
	AgentType string            `json:"agent_type"`
	ParentID  string            `json:"parent_id"`
	Selectors map[string]string `json:"selectors,omitempty"`
	TTL       *int              `json:"ttl,omitempty"`
}

// SpireCreateAgentEntryResponse is returned by POST /v1/entries/agent
type SpireCreateAgentEntryResponse struct {
	EntryID   string            `json:"entry_id"`
	SpiffeID  string            `json:"spiffe_id"`
	TenantID  string            `json:"tenant_id"`
	ClientID  string            `json:"client_id"`
	ParentID  string            `json:"parent_id"`
	Selectors map[string]string `json:"selectors"`
	TTL       *int              `json:"ttl"`
	CreatedAt time.Time         `json:"created_at"`
}

// SpireIssueJWTSVIDRequest is sent to POST /v1/jwt/issue-delegated
type SpireIssueJWTSVIDRequest struct {
	TenantID     string                 `json:"tenant_id"`
	SpiffeID     string                 `json:"spiffe_id"`
	Audience     []string               `json:"audience"`
	TTL          int                    `json:"ttl"`
	CustomClaims map[string]interface{} `json:"custom_claims,omitempty"`
}

// SpireIssueJWTSVIDResponse is returned by POST /v1/jwt/issue-delegated
type SpireIssueJWTSVIDResponse struct {
	SpiffeID  string `json:"spiffe_id"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// SpireAgent represents a SPIRE agent returned by GET /v1/agents
type SpireAgent struct {
	ID              string `json:"id"`
	SpiffeID        string `json:"spiffe_id"`
	NodeID          string `json:"node_id"`
	AttestationType string `json:"attestation_type"`
	Status          string `json:"status"`
	LastSeen        string `json:"last_seen"`
	CreatedAt       string `json:"created_at"`
}

// --- Service methods ---

// ListAgents fetches the list of SPIRE agents.
func (s *SpireService) ListAgents(authToken string) ([]SpireAgent, error) {
	url := s.baseURL + "/v1/agents"
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call authsec-spire: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("authsec-spire returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Agents []SpireAgent `json:"agents"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Agents, nil
}

// CreateAgentEntry creates a SPIRE workload entry for an AI agent.
func (s *SpireService) CreateAgentEntry(req *SpireCreateAgentEntryRequest, authToken string) (*SpireCreateAgentEntryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := s.baseURL + "/v1/entries/agent"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+authToken)
	}

	log.Printf("[SpireService] Creating agent entry: tenant=%s client=%s agent_type=%s", req.TenantID, req.ClientID, req.AgentType)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call authsec-spire: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("authsec-spire returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result SpireCreateAgentEntryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	log.Printf("[SpireService] Agent entry created: entry_id=%s spiffe_id=%s", result.EntryID, result.SpiffeID)
	return &result, nil
}

// IssueDelegatedJWTSVID requests a JWT-SVID with custom claims for a delegated AI agent.
func (s *SpireService) IssueDelegatedJWTSVID(req *SpireIssueJWTSVIDRequest, authToken string) (*SpireIssueJWTSVIDResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := s.baseURL + "/v1/jwt/issue-delegated"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+authToken)
	}

	log.Printf("[SpireService] Issuing delegated JWT-SVID: tenant=%s spiffe_id=%s audience=%v", req.TenantID, req.SpiffeID, req.Audience)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call authsec-spire: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("authsec-spire returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result SpireIssueJWTSVIDResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	log.Printf("[SpireService] JWT-SVID issued: spiffe_id=%s expires_at=%s", result.SpiffeID, result.ExpiresAt)
	return &result, nil
}

// DeleteSpireEntry deletes a SPIRE workload entry by ID.
func (s *SpireService) DeleteSpireEntry(tenantID, entryID, authToken string) error {
	url := fmt.Sprintf("%s/v1/entries/%s?tenant_id=%s", s.baseURL, entryID, tenantID)
	httpReq, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call authsec-spire: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authsec-spire returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[SpireService] Entry deleted: entry_id=%s tenant=%s", entryID, tenantID)
	return nil
}
