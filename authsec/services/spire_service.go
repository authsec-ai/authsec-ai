package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
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
	cfg := config.GetConfig()
	return &SpireService{
		baseURL: cfg.ICPServiceURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// --- Request/Response types for authsec-spire ---

// CreateAgentEntryRequest is sent to POST /v1/entries/agent
type CreateAgentEntryRequest struct {
	TenantID  string            `json:"tenant_id"`
	ClientID  string            `json:"client_id"`
	AgentType string            `json:"agent_type"`
	ParentID  string            `json:"parent_id"`
	Selectors map[string]string `json:"selectors,omitempty"`
	TTL       *int              `json:"ttl,omitempty"`
}

// CreateAgentEntryResponse is returned by POST /v1/entries/agent
type CreateAgentEntryResponse struct {
	EntryID   string            `json:"entry_id"`
	SpiffeID  string            `json:"spiffe_id"`
	TenantID  string            `json:"tenant_id"`
	ClientID  string            `json:"client_id"`
	ParentID  string            `json:"parent_id"`
	Selectors map[string]string `json:"selectors"`
	TTL       *int              `json:"ttl"`
	CreatedAt time.Time         `json:"created_at"`
}

// IssueJWTSVIDRequest is sent to POST /v1/jwt/issue-delegated
type IssueJWTSVIDRequest struct {
	TenantID     string                 `json:"tenant_id"`
	SpiffeID     string                 `json:"spiffe_id"`
	Audience     []string               `json:"audience"`
	TTL          int                    `json:"ttl"`
	CustomClaims map[string]interface{} `json:"custom_claims,omitempty"`
}

// IssueJWTSVIDResponse is returned by POST /v1/jwt/issue-delegated
type IssueJWTSVIDResponse struct {
	SpiffeID  string `json:"spiffe_id"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// DeleteEntryResponse is returned by DELETE /v1/entries/{id}
type DeleteEntryResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
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
// authsec-spire generates the SPIFFE ID based on tenant_id, client_id, and agent_type.
func (s *SpireService) CreateAgentEntry(req *CreateAgentEntryRequest, authToken string) (*CreateAgentEntryResponse, error) {
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

	var result CreateAgentEntryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	log.Printf("[SpireService] Agent entry created: entry_id=%s spiffe_id=%s", result.EntryID, result.SpiffeID)
	return &result, nil
}

// IssueDelegatedJWTSVID requests a JWT-SVID with custom claims for a delegated AI agent.
func (s *SpireService) IssueDelegatedJWTSVID(req *IssueJWTSVIDRequest, authToken string) (*IssueJWTSVIDResponse, error) {
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

	var result IssueJWTSVIDResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	log.Printf("[SpireService] JWT-SVID issued: spiffe_id=%s expires_at=%s", result.SpiffeID, result.ExpiresAt)
	return &result, nil
}

// DeleteEntry deletes a SPIRE workload entry by ID.
func (s *SpireService) DeleteEntry(tenantID, entryID, authToken string) error {
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

// --- Package-level convenience functions for clients controller ---

var icpHTTPClient = &http.Client{Timeout: 10 * time.Second}

// RegisterAgentWorkloadEntry calls the ICP service to create a workload entry
// for an AI agent and returns the generated SPIFFE ID.
func RegisterAgentWorkloadEntry(tenantID, clientID, agentType, authToken string, platformSelectors map[string]string) (string, error) {
	cfg := config.GetConfig()
	if cfg.ICPServiceURL == "" {
		return "", fmt.Errorf("ICP_SERVICE_URL is not configured")
	}

	parentID := fmt.Sprintf("spiffe://%s/agent", tenantID)

	reqBody := CreateAgentEntryRequest{
		TenantID:  tenantID,
		ClientID:  clientID,
		AgentType: agentType,
		ParentID:  parentID,
		Selectors: platformSelectors,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ICP request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/entries/agent", cfg.ICPServiceURL)
	log.Printf("Registering AI agent workload entry with ICP: %s", url)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create ICP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := icpHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call ICP service: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read ICP response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("ICP service returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result CreateAgentEntryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse ICP response: %w", err)
	}

	log.Printf("AI agent workload entry created: spiffe_id=%s, entry_id=%s", result.SpiffeID, result.EntryID)
	return result.SpiffeID, nil
}

// DeleteAgentWorkloadEntry calls the ICP service to delete all workload entries
// for an AI agent identified by tenant_id and client_id.
func DeleteAgentWorkloadEntry(tenantID, clientID, authToken string) error {
	cfg := config.GetConfig()
	if cfg.ICPServiceURL == "" {
		return fmt.Errorf("ICP_SERVICE_URL is not configured")
	}

	spiffeIDSearch := fmt.Sprintf("spiffe://%s/agent/%s/", tenantID, clientID)
	listURL := fmt.Sprintf("%s/v1/entries?tenant_id=%s&spiffe_id_search=%s",
		cfg.ICPServiceURL,
		neturl.QueryEscape(tenantID),
		neturl.QueryEscape(spiffeIDSearch),
	)

	log.Printf("[ICP-DELETE] Searching for workload entries to delete: tenant=%s, client=%s", tenantID, clientID)

	listReq, err := http.NewRequest(http.MethodGet, listURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create ICP list request: %w", err)
	}
	listReq.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		listReq.Header.Set("Authorization", "Bearer "+authToken)
	}

	listResp, err := icpHTTPClient.Do(listReq)
	if err != nil {
		return fmt.Errorf("failed to call ICP service for listing entries: %w", err)
	}
	defer listResp.Body.Close()

	listBody, err := io.ReadAll(listResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read ICP list response: %w", err)
	}

	if listResp.StatusCode != http.StatusOK {
		return fmt.Errorf("ICP list entries returned %d: %s", listResp.StatusCode, string(listBody))
	}

	var entries struct {
		Entries []struct {
			ID       string `json:"id"`
			SpiffeID string `json:"spiffe_id"`
		} `json:"entries"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(listBody, &entries); err != nil {
		return fmt.Errorf("failed to parse ICP list response: %w", err)
	}

	if entries.Total == 0 {
		log.Printf("[ICP-DELETE] No workload entries found for client %s in tenant %s", clientID, tenantID)
		return nil
	}

	var deleteErrors []error
	for _, entry := range entries.Entries {
		deleteURL := fmt.Sprintf("%s/v1/entries/%s?tenant_id=%s",
			cfg.ICPServiceURL, entry.ID, neturl.QueryEscape(tenantID))

		delReq, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			deleteErrors = append(deleteErrors, fmt.Errorf("failed to create delete request for entry %s: %w", entry.ID, err))
			continue
		}
		delReq.Header.Set("Content-Type", "application/json")
		if authToken != "" {
			delReq.Header.Set("Authorization", "Bearer "+authToken)
		}

		delResp, err := icpHTTPClient.Do(delReq)
		if err != nil {
			deleteErrors = append(deleteErrors, fmt.Errorf("failed to delete entry %s: %w", entry.ID, err))
			continue
		}
		delResp.Body.Close()

		if delResp.StatusCode != http.StatusOK {
			deleteErrors = append(deleteErrors, fmt.Errorf("ICP returned %d when deleting entry %s", delResp.StatusCode, entry.ID))
			continue
		}

		log.Printf("[ICP-DELETE] Deleted workload entry: id=%s, spiffe_id=%s", entry.ID, entry.SpiffeID)
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("partial ICP cleanup: %d/%d entries failed to delete: %v", len(deleteErrors), len(entries.Entries), deleteErrors[0])
	}

	log.Printf("[ICP-DELETE] All %d workload entries deleted for client %s in tenant %s", entries.Total, clientID, tenantID)
	return nil
}
