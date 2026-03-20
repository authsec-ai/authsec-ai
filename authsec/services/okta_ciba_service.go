package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OktaCIBAService handles Okta CIBA operations
type OktaCIBAService struct {
	domain       string
	clientID     string
	clientSecret string
	issuer       string
	apiToken     string
	httpClient   *http.Client
}

// CIBAAuthRequest represents Okta's CIBA authentication request response
type CIBAAuthRequest struct {
	AuthReqID string `json:"auth_req_id"`
	ExpiresIn int    `json:"expires_in"`
	Interval  int    `json:"interval"`
}

// CIBATokenResponse represents Okta's token response
type CIBATokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// CIBAError represents Okta error response
type CIBAError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// NewOktaCIBAService creates a new Okta CIBA service
func NewOktaCIBAService() *OktaCIBAService {
	return &OktaCIBAService{
		domain:       os.Getenv("OKTA_DOMAIN"),
		clientID:     os.Getenv("OKTA_CLIENT_ID"),
		clientSecret: os.Getenv("OKTA_CLIENT_SECRET"),
		issuer:       os.Getenv("OKTA_ISSUER"),
		apiToken:     os.Getenv("OKTA_API_TOKEN"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// InitiateCIBA starts a backchannel authentication request
func (s *OktaCIBAService) InitiateCIBA(ctx context.Context, loginHint, bindingMessage string) (*CIBAAuthRequest, error) {
	// Validate configuration
	if s.issuer == "" || s.clientID == "" || s.clientSecret == "" {
		return nil, fmt.Errorf("okta configuration missing: ensure OKTA_ISSUER, OKTA_CLIENT_ID, and OKTA_CLIENT_SECRET are set")
	}

	endpoint := fmt.Sprintf("%s/v1/bc/authorize", s.issuer)

	data := url.Values{}
	data.Set("login_hint", loginHint)
	data.Set("scope", "openid profile email")
	data.Set("client_id", s.clientID)

	if bindingMessage != "" {
		data.Set("binding_message", bindingMessage)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(s.clientID, s.clientSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to okta: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var cibaErr CIBAError
		if err := json.Unmarshal(body, &cibaErr); err != nil {
			return nil, fmt.Errorf("okta returned status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("okta error: %s - %s", cibaErr.Error, cibaErr.ErrorDescription)
	}

	var authReq CIBAAuthRequest
	if err := json.Unmarshal(body, &authReq); err != nil {
		return nil, fmt.Errorf("failed to parse okta response: %w", err)
	}

	return &authReq, nil
}

// PollCIBAToken polls Okta for token after user approval
func (s *OktaCIBAService) PollCIBAToken(ctx context.Context, authReqID string) (*CIBATokenResponse, string, error) {
	if s.issuer == "" || s.clientID == "" || s.clientSecret == "" {
		return nil, "", fmt.Errorf("okta configuration missing")
	}

	endpoint := fmt.Sprintf("%s/v1/token", s.issuer)

	data := url.Values{}
	data.Set("grant_type", "urn:openid:params:grant-type:ciba")
	data.Set("auth_req_id", authReqID)
	data.Set("client_id", s.clientID)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(s.clientID, s.clientSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to send request to okta: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Handle success case
	if resp.StatusCode == http.StatusOK {
		var tokenResp CIBATokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return nil, "", fmt.Errorf("failed to parse token response: %w", err)
		}
		return &tokenResp, "approved", nil
	}

	// Handle error cases
	var cibaErr CIBAError
	if err := json.Unmarshal(body, &cibaErr); err != nil {
		return nil, "", fmt.Errorf("okta returned status %d: %s", resp.StatusCode, string(body))
	}

	// Map Okta errors to status
	switch cibaErr.Error {
	case "authorization_pending":
		return nil, "pending", nil
	case "slow_down":
		return nil, "slow_down", nil
	case "access_denied":
		return nil, "denied", nil
	case "expired_token":
		return nil, "expired", nil
	default:
		return nil, "", fmt.Errorf("okta error: %s - %s", cibaErr.Error, cibaErr.ErrorDescription)
	}
}

// GenerateOktaVerifyEnrollment creates enrollment QR code for Okta Verify
func (s *OktaCIBAService) GenerateOktaVerifyEnrollment(ctx context.Context, userID uuid.UUID, email string) (string, string, error) {
	if s.domain == "" || s.apiToken == "" {
		return "", "", fmt.Errorf("okta domain or api token not configured: ensure OKTA_DOMAIN and OKTA_API_TOKEN are set")
	}

	// First, find or create Okta user by email
	oktaUserID, err := s.findOktaUserByEmail(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("failed to find okta user: %w", err)
	}

	// Enroll push factor (Okta Verify)
	endpoint := fmt.Sprintf("https://%s/api/v1/users/%s/factors", s.domain, oktaUserID)

	payload := map[string]interface{}{
		"factorType": "push",
		"provider":   "OKTA",
	}

	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("SSWS %s", s.apiToken))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request to okta: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("okta api error (status %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse okta response: %w", err)
	}

	// Extract QR code and activation links
	embedded, ok := result["_embedded"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid okta response: missing _embedded")
	}

	activation, ok := embedded["activation"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid okta response: missing activation")
	}

	qrCode, ok := activation["qrcode"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid okta response: missing qrcode")
	}

	qrCodeHref, ok := qrCode["href"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid okta response: missing qrcode href")
	}

	links, ok := activation["_links"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid okta response: missing _links")
	}

	qrcodeLink, ok := links["qrcode"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid okta response: missing qrcode link")
	}

	activationLink, ok := qrcodeLink["href"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid okta response: missing activation link")
	}

	return qrCodeHref, activationLink, nil
}

// findOktaUserByEmail finds Okta user ID by email
func (s *OktaCIBAService) findOktaUserByEmail(ctx context.Context, email string) (string, error) {
	endpoint := fmt.Sprintf("https://%s/api/v1/users?search=profile.email eq \"%s\"", s.domain, email)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("SSWS %s", s.apiToken))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to okta: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("okta api error (status %d): %s", resp.StatusCode, string(body))
	}

	var users []map[string]interface{}
	if err := json.Unmarshal(body, &users); err != nil {
		return "", fmt.Errorf("failed to parse okta response: %w", err)
	}

	if len(users) == 0 {
		return "", fmt.Errorf("user not found in okta: %s", email)
	}

	userID, ok := users[0]["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid okta response: missing user id")
	}

	return userID, nil
}
