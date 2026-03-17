package hydramodels

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/services"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// OAuthLoginService manages OAuth operations
type OAuthLoginService struct {
	cfg config.Config
}

// NewOAuthLoginService creates a new OAuthLoginService
func NewOAuthLoginService(cfg config.Config) *OAuthLoginService {
	return &OAuthLoginService{cfg: cfg}
}

// GetConfig returns the configuration
func (s *OAuthLoginService) GetConfig() config.Config {
	return s.cfg
}

// JWT Claims structure
type JWTClaims struct {
	Audience  []string `json:"aud"`
	ClientID  string   `json:"client_id"`
	ExpiresAt int64    `json:"exp"`
	Ext       struct {
		Email      string `json:"email"`
		Name       string `json:"name"`
		OrgID      string `json:"org_id"`
		Provider   string `json:"provider"`
		ProviderID string `json:"provider_id"`
		TenantID   string `json:"tenant_id"`
		UserID     string `json:"user_id"`
	} `json:"ext"`
	IssuedAt  int64    `json:"iat"`
	Issuer    string   `json:"iss"`
	JWTID     string   `json:"jti"`
	NotBefore int64    `json:"nbf"`
	Scopes    []string `json:"scp"`
	Subject   string   `json:"sub"`
	jwt.RegisteredClaims
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string   `json:"kty"`
	Use string   `json:"use"`
	Kid string   `json:"kid"`
	X5t string   `json:"x5t"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

var jwksCache = make(map[string]*rsa.PublicKey)
var jwksCacheMutex sync.RWMutex

func fetchJWKS(issuer string) (*JWKS, error) {
	jwksURL := strings.TrimSuffix(issuer, "/") + "/.well-known/jwks.json"
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status: %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}
	return &jwks, nil
}

func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E: %w", err)
	}
	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func getPublicKey(issuer, kid string) (*rsa.PublicKey, error) {
	jwksCacheMutex.RLock()
	if key, exists := jwksCache[kid]; exists {
		jwksCacheMutex.RUnlock()
		return key, nil
	}
	jwksCacheMutex.RUnlock()

	jwks, err := fetchJWKS(issuer)
	if err != nil {
		return nil, err
	}

	for _, jwk := range jwks.Keys {
		if jwk.Kid == kid {
			publicKey, err := jwkToRSAPublicKey(jwk)
			if err != nil {
				return nil, err
			}
			jwksCacheMutex.Lock()
			jwksCache[kid] = publicKey
			jwksCacheMutex.Unlock()
			return publicKey, nil
		}
	}
	return nil, fmt.Errorf("key with kid %s not found", kid)
}

func DecodeJWTToken(tokenString string) (*JWTClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token header: %w", err)
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("no kid in token header")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	issuer := claims.Issuer
	if issuer == "" {
		return nil, fmt.Errorf("no issuer in token")
	}

	publicKey, err := getPublicKey(issuer, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	verifiedToken, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if verifiedClaims, ok := verifiedToken.Claims.(*JWTClaims); ok && verifiedToken.Valid {
		return verifiedClaims, nil
	}
	return nil, fmt.Errorf("invalid token claims")
}

func (s *OAuthLoginService) CreateOrUpdateUser(accessToken string, users *User) (*User, error) {
	tenantID := users.TenantID
	clientID := users.ClientID
	tenantIDStr := tenantID.String()
	clientIDStr := strings.TrimSuffix(clientID.String(), "-main-client")

	if tenantIDStr == "" || clientIDStr == "" {
		return nil, fmt.Errorf("missing tenant_id or client_id in JWT token")
	}

	db, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	var client Client
	if err := db.Table("clients").Where("client_id = ? and tenant_id = ?", clientIDStr, tenantIDStr).First(&client).Error; err != nil {
		return nil, fmt.Errorf("failed to get client details: %w", err)
	}

	var existingUser User
	err = db.Table("users").Where(
		"provider = ? AND provider_id = ? AND tenant_id = ? AND client_id = ?",
		users.Provider, users.ProviderID, tenantID, clientID,
	).First(&existingUser).Error

	now := time.Now()
	if err == nil {
		existingUser.ProjectID = client.ProjectID
		existingUser.Name = *users.Username
		existingUser.Email = users.Email
		existingUser.ProviderData = datatypes.JSON(users.ProviderData)
		existingUser.Active = true
		existingUser.UpdatedAt = now

		if err := db.Table("users").Save(&existingUser).Error; err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
		return &existingUser, nil
	}

	user := &User{
		ID:           uuid.New(),
		ClientID:     clientID,
		TenantID:     tenantID,
		ProjectID:    client.ProjectID,
		Name:         *users.Username,
		Username:     nil,
		Email:        users.Email,
		Provider:     users.Provider,
		ProviderID:   users.ProviderID,
		ProviderData: datatypes.JSON(users.ProviderData),
		AvatarURL:    nil,
		Active:       true,
		MFAVerified:  false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := db.Table("users").Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func (s *OAuthLoginService) GetHydraLoginRequest(loginChallenge string) (*HydraLoginRequest, error) {
	reqURL := fmt.Sprintf("%s/admin/oauth2/auth/requests/login?login_challenge=%s",
		s.cfg.HydraAdminURL, loginChallenge)

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch login request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("Hydra login request response for challenge %s: %s", loginChallenge, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login request not found, status code: %d", resp.StatusCode)
	}

	var loginRequest HydraLoginRequest
	if err := json.Unmarshal(bodyBytes, &loginRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal login request: %w", err)
	}
	return &loginRequest, nil
}

func (s *OAuthLoginService) AcceptHydraLoginRequestWithContext(loginChallenge, subject string, ctx map[string]interface{}) (*HydraAcceptLoginResponse, error) {
	acceptRequest := HydraAcceptLoginRequest{
		Subject:     subject,
		Remember:    true,
		RememberFor: 3600,
		Context:     ctx,
	}

	jsonData, err := json.Marshal(acceptRequest)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/admin/oauth2/auth/requests/login/accept?login_challenge=%s",
		s.cfg.HydraAdminURL, loginChallenge)

	req, err := http.NewRequest("PUT", reqURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var acceptResponse HydraAcceptLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&acceptResponse); err != nil {
		return nil, err
	}
	return &acceptResponse, nil
}

func (s *OAuthLoginService) GetHydraConsentRequest(consentChallenge string) (*HydraConsentRequest, error) {
	reqURL := fmt.Sprintf("%s/admin/oauth2/auth/requests/consent?consent_challenge=%s",
		s.cfg.HydraAdminURL, consentChallenge)

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var consentRequest HydraConsentRequest
	if err := json.NewDecoder(resp.Body).Decode(&consentRequest); err != nil {
		return nil, err
	}
	return &consentRequest, nil
}

func (s *OAuthLoginService) AcceptHydraConsentRequest(consentChallenge string, consentRequest *HydraConsentRequest) (*HydraAcceptConsentResponse, error) {
	userContext := make(map[string]interface{})
	if consentRequest.Context != nil {
		userContext = consentRequest.Context
	}

	acceptRequest := HydraAcceptConsentRequest{
		GrantScope:               consentRequest.RequestedScope,
		GrantAccessTokenAudience: consentRequest.RequestedAccessTokenAudience,
		Remember:                 true,
		RememberFor:              3600,
		Session: map[string]interface{}{
			"access_token": map[string]interface{}{
				"user_id":     consentRequest.Subject,
				"email":       userContext["email"],
				"name":        userContext["name"],
				"provider":    userContext["provider"],
				"provider_id": userContext["provider_id"],
				"tenant_id":   userContext["tenant_id"],
				"org_id":      userContext["org_id"],
			},
			"id_token": map[string]interface{}{
				"user_id":     consentRequest.Subject,
				"email":       userContext["email"],
				"name":        userContext["name"],
				"provider":    userContext["provider"],
				"provider_id": userContext["provider_id"],
				"tenant_id":   userContext["tenant_id"],
				"org_id":      userContext["org_id"],
			},
		},
	}

	jsonData, err := json.Marshal(acceptRequest)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/admin/oauth2/auth/requests/consent/accept?consent_challenge=%s",
		s.cfg.HydraAdminURL, consentChallenge)

	req, err := http.NewRequest("PUT", reqURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var acceptResponse HydraAcceptConsentResponse
	if err := json.NewDecoder(resp.Body).Decode(&acceptResponse); err != nil {
		return nil, err
	}
	return &acceptResponse, nil
}

func (s *OAuthLoginService) GetHydraClient(clientID string) (*HydraClient, string, error) {
	reqURL := fmt.Sprintf("%s/admin/clients/%s", s.cfg.HydraAdminURL, clientID)

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch client: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, string(bodyBytes), fmt.Errorf("client not found, status code: %d", resp.StatusCode)
	}

	var client HydraClient
	if err := json.Unmarshal(bodyBytes, &client); err != nil {
		return nil, string(bodyBytes), fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &client, string(bodyBytes), nil
}

func (s *OAuthLoginService) GetAllHydraClients() ([]HydraClient, error) {
	reqURL := fmt.Sprintf("%s/admin/clients", s.cfg.HydraAdminURL)

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var clients []HydraClient
	if err := json.NewDecoder(resp.Body).Decode(&clients); err != nil {
		return nil, err
	}
	return clients, nil
}

func (s *OAuthLoginService) GetOIDCProvidersForTenant(tenantID string) ([]OIDCProvider, error) {
	clients, err := s.GetAllHydraClients()
	if err != nil {
		return nil, err
	}

	var providers []OIDCProvider
	for _, client := range clients {
		if clientTenantID, ok := client.Metadata["tenant_id"].(string); ok && clientTenantID == tenantID {
			if clientType, ok := client.Metadata["type"].(string); ok && clientType == "oidc_provider" {
				providerName, _ := client.Metadata["provider_name"].(string)
				displayName, _ := client.Metadata["display_name"].(string)
				isActive, _ := client.Metadata["is_active"].(bool)
				sortOrder, _ := client.Metadata["sort_order"].(float64)
				callbackURL, _ := client.Metadata["callback_url"].(string)
				providerConfig, _ := client.Metadata["provider_config"].(map[string]interface{})

				if providerName != "" && isActive {
					providers = append(providers, OIDCProvider{
						ProviderName: providerName,
						DisplayName:  displayName,
						IsActive:     isActive,
						SortOrder:    int(sortOrder),
						CallbackURL:  callbackURL,
						Config:       providerConfig,
					})
				}
			}
		}
	}

	for i := 0; i < len(providers)-1; i++ {
		for j := i + 1; j < len(providers); j++ {
			if providers[i].SortOrder > providers[j].SortOrder {
				providers[i], providers[j] = providers[j], providers[i]
			}
		}
	}
	return providers, nil
}

func (s *OAuthLoginService) ExchangeCodeForTokens(ctx context.Context, provider *OIDCProvider, code, redirectURI string) (map[string]interface{}, error) {
	providerConfig := provider.Config
	clientID, _ := providerConfig["client_id"].(string)
	clientSecret, _ := providerConfig["client_secret"].(string)
	tokenURL, _ := providerConfig["token_url"].(string)

	if clientID == "" || clientSecret == "" || tokenURL == "" {
		return nil, fmt.Errorf("incomplete provider configuration: missing clientID, clientSecret, or tokenURL")
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "OAuth-Login-Service/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	if accessToken, ok := tokenResponse["access_token"].(string); !ok || accessToken == "" {
		return nil, fmt.Errorf("no access_token in response")
	}
	return tokenResponse, nil
}

func (s *OAuthLoginService) ExchangeCodeForTokensWithProvider(ctx context.Context, provider *OIDCProvider, code, redirectURI string) (*TokenResponse, error) {
	providerConfig := provider.Config
	clientID, _ := providerConfig["client_id"].(string)
	clientSecret, _ := providerConfig["client_secret"].(string)
	tokenURL, _ := providerConfig["token_url"].(string)

	if clientID == "" || clientSecret == "" || tokenURL == "" {
		return nil, fmt.Errorf("incomplete provider configuration")
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	accessToken, _ := tokenResponse["access_token"].(string)
	tokenType, _ := tokenResponse["token_type"].(string)

	var expiresIn int
	if exp, ok := tokenResponse["expires_in"].(float64); ok {
		expiresIn = int(exp)
	} else if exp, ok := tokenResponse["expires_in"].(int); ok {
		expiresIn = exp
	} else {
		expiresIn = 3600
	}

	if accessToken == "" {
		return nil, fmt.Errorf("no access_token in response")
	}
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return &TokenResponse{AccessToken: accessToken, TokenType: tokenType, ExpiresIn: expiresIn}, nil
}

func (s *OAuthLoginService) GetUserInfo(ctx context.Context, provider *OIDCProvider, accessToken string) (map[string]interface{}, error) {
	providerConfig := provider.Config
	userInfoURL, _ := providerConfig["user_info_url"].(string)

	if userInfoURL == "" {
		return nil, fmt.Errorf("missing user_info_url in provider configuration")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "OAuth-Login-Service/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make user info request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info response: %w", err)
	}
	return userInfo, nil
}

func (s *OAuthLoginService) GetUIAccessToken(ctx context.Context, req string) (*TokenResponse, error) {
	resp, err := services.IssueOIDCJWT(ctx, req)
	if err != nil {
		return nil, err
	}
	return &TokenResponse{
		AccessToken: resp.AccessToken,
		TokenType:   resp.TokenType,
		ExpiresIn:   int(resp.ExpiresIn),
	}, nil
}
