package icp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles communication with ICP service
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient creates a new ICP client
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{Timeout: 2 * time.Minute},
	}
}

// ProvisionPKIRequest represents PKI provisioning request to ICP
type ProvisionPKIRequest struct {
	TenantID   string `json:"tenant_id"`
	CommonName string `json:"common_name"`
	Domain     string `json:"domain"`
	TTL        string `json:"ttl"`
	MaxTTL     string `json:"max_ttl"`
}

// ProvisionPKIResponse represents PKI provisioning response from ICP
type ProvisionPKIResponse struct {
	TenantID    string `json:"tenant_id"`
	PKIMount    string `json:"pki_mount"`
	CACert      string `json:"ca_cert"`
	RoleCreated string `json:"role_created"`
	Message     string `json:"message"`
}

// ProvisionPKI provisions PKI infrastructure for a tenant
func (c *Client) ProvisionPKI(ctx context.Context, req *ProvisionPKIRequest) (*ProvisionPKIResponse, error) {
	url := fmt.Sprintf("%s/admin/pki/provision/%s", c.baseURL, req.TenantID)

	body, err := json.Marshal(req)
	fmt.Println(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ICP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("ICP returned %d (failed to parse error)", resp.StatusCode)
		}

		return nil, fmt.Errorf("ICP returned %d: %s - %s",
			resp.StatusCode, errResp.Error.Code, errResp.Error.Message)
	}

	var result ProvisionPKIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UpdateTenantStatusRequest represents tenant status update request
type UpdateTenantStatusRequest struct {
	Status string `json:"status"`
}

// UpdateTenantStatus updates tenant status in ICP
func (c *Client) UpdateTenantStatus(ctx context.Context, tenantID, status string) error {
	url := fmt.Sprintf("%s/admin/tenants/%s/status", c.baseURL, tenantID)

	req := UpdateTenantStatusRequest{Status: status}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call ICP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update tenant status: HTTP %d", resp.StatusCode)
	}

	return nil
}

// HealthCheck checks if ICP service is healthy
func (c *Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call ICP health endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ICP health check failed: HTTP %d", resp.StatusCode)
	}

	return nil
}
