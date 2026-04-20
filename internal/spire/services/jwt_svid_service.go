package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/infrastructure/vault"
)

// jwtKVMount is the Vault KV v2 mount used for JWT signing keys.
// The actual Vault path becomes: kv/data/secret/spire/jwt-signing-keys/{tenant_id}
const jwtKVMount = "kv"

// JWTSVIDService handles JWT-SVID issuance and validation
type JWTSVIDService struct {
	vaultClient *vault.Client
	logger      *logrus.Entry
	keyCache    map[string]*rsa.PrivateKey
	keyCacheMu  sync.RWMutex
}

// NewJWTSVIDService creates a new JWT-SVID service
func NewJWTSVIDService(vaultClient *vault.Client, logger *logrus.Entry) *JWTSVIDService {
	return &JWTSVIDService{
		vaultClient: vaultClient,
		logger:      logger,
		keyCache:    make(map[string]*rsa.PrivateKey),
	}
}

// IssueJWTSVIDRequest is the request to issue a JWT-SVID
type IssueJWTSVIDRequest struct {
	TenantID     string                 `json:"tenant_id"`
	SpiffeID     string                 `json:"spiffe_id"`
	Audience     []string               `json:"audience"`
	TTL          int                    `json:"ttl"` // seconds
	CustomClaims map[string]interface{} `json:"custom_claims,omitempty"`
}

// IssueJWTSVIDResponse is the response containing the JWT-SVID
type IssueJWTSVIDResponse struct {
	SpiffeID  string    `json:"spiffe_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ValidateJWTSVIDRequest is the request to validate a JWT-SVID
type ValidateJWTSVIDRequest struct {
	TenantID string `json:"tenant_id"`
	Token    string `json:"token"`
	Audience string `json:"audience"`
}

// ValidateJWTSVIDResponse is the response from JWT validation
type ValidateJWTSVIDResponse struct {
	SpiffeID string                 `json:"spiffe_id"`
	Valid    bool                   `json:"valid"`
	Claims   map[string]interface{} `json:"claims"`
}

// IssueJWTSVID issues a new JWT-SVID for a workload
func (s *JWTSVIDService) IssueJWTSVID(
	ctx context.Context,
	req *IssueJWTSVIDRequest,
) (*IssueJWTSVIDResponse, error) {
	// Validate request
	if req.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if req.SpiffeID == "" {
		return nil, fmt.Errorf("spiffe_id is required")
	}
	if len(req.Audience) == 0 {
		return nil, fmt.Errorf("audience is required")
	}

	// Default TTL: 1 hour
	ttl := req.TTL
	if ttl == 0 {
		ttl = 3600
	}

	// Calculate expiration
	now := time.Now()
	expiresAt := now.Add(time.Duration(ttl) * time.Second)

	signingKey, err := s.getOrCreateSigningKey(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get signing key: %w", err)
	}

	// Build claims as MapClaims to support arbitrary custom fields
	mapClaims := jwt.MapClaims{
		"iss": fmt.Sprintf("spiffe://%s", req.TenantID),
		"sub": req.SpiffeID,
		"aud": req.Audience,
		"exp": jwt.NewNumericDate(expiresAt),
		"nbf": jwt.NewNumericDate(now),
		"iat": jwt.NewNumericDate(now),
		"jti": generateJTI(),
	}

	// Merge custom claims as top-level JWT fields
	for k, v := range req.CustomClaims {
		// Protect standard claims from being overwritten
		switch k {
		case "iss", "sub", "aud", "exp", "nbf", "iat", "jti":
			continue
		default:
			mapClaims[k] = v
		}
	}

	// Create and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, mapClaims)
	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT: %w", err)
	}

	return &IssueJWTSVIDResponse{
		SpiffeID:  req.SpiffeID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateJWTSVID validates a JWT-SVID token
func (s *JWTSVIDService) ValidateJWTSVID(
	ctx context.Context,
	req *ValidateJWTSVIDRequest,
) (*ValidateJWTSVIDResponse, error) {
	// Get public key for validation
	publicKey, err := s.getPublicKey(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Parse and validate token using MapClaims (supports custom claims)
	token, err := jwt.Parse(
		req.Token,
		func(token *jwt.Token) (interface{}, error) {
			// Verify signing method
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return publicKey, nil
		},
	)

	if err != nil {
		return &ValidateJWTSVIDResponse{
			Valid: false,
		}, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return &ValidateJWTSVIDResponse{
			Valid: false,
		}, nil
	}

	// Extract SPIFFE ID from sub claim
	spiffeID, _ := claims["sub"].(string)

	// Verify audience if specified
	if req.Audience != "" {
		audienceMatch := false
		if audClaim, ok := claims["aud"]; ok {
			switch aud := audClaim.(type) {
			case []interface{}:
				for _, a := range aud {
					if aStr, ok := a.(string); ok && aStr == req.Audience {
						audienceMatch = true
						break
					}
				}
			case string:
				audienceMatch = aud == req.Audience
			}
		}
		if !audienceMatch {
			return &ValidateJWTSVIDResponse{
				Valid:    false,
				SpiffeID: spiffeID,
			}, nil
		}
	}

	// Return all claims from the token (preserves arrays like permissions)
	claimsMap := make(map[string]interface{})
	for k, v := range claims {
		claimsMap[k] = v
	}

	return &ValidateJWTSVIDResponse{
		SpiffeID: spiffeID,
		Valid:    true,
		Claims:   claimsMap,
	}, nil
}

// GetJWTBundle returns the JWT bundle (JWKS) for a tenant
func (s *JWTSVIDService) GetJWTBundle(
	ctx context.Context,
	tenantID string,
) (string, error) {
	// Get public key
	publicKey, err := s.getPublicKey(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}

	// Encode modulus and exponent as base64url (no padding) per RFC 7517
	nEncoded := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	eBig := big.NewInt(int64(publicKey.E))
	eEncoded := base64.RawURLEncoding.EncodeToString(eBig.Bytes())

	// Convert to JWKS format
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "default",
				"alg": "RS256",
				"n":   nEncoded,
				"e":   eEncoded,
			},
		},
	}

	jwksJSON, err := json.Marshal(jwks)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	return string(jwksJSON), nil
}

// getOrCreateSigningKey returns the RSA private key for the tenant.
// Lookup order: in-memory cache -> Vault KV -> generate new & persist to Vault.
func (s *JWTSVIDService) getOrCreateSigningKey(
	ctx context.Context,
	tenantID string,
) (*rsa.PrivateKey, error) {
	// Fast path: read lock
	s.keyCacheMu.RLock()
	if key, ok := s.keyCache[tenantID]; ok {
		s.keyCacheMu.RUnlock()
		return key, nil
	}
	s.keyCacheMu.RUnlock()

	// Slow path: write lock
	s.keyCacheMu.Lock()
	defer s.keyCacheMu.Unlock()

	// Double-check after acquiring write lock
	if key, ok := s.keyCache[tenantID]; ok {
		return key, nil
	}

	// Try loading from Vault (stored under kv/data/secret/spire/jwt-signing-keys/{tenant_id})
	vaultPath := fmt.Sprintf("secret/spire/jwt-signing-keys/%s", tenantID)
	data, err := s.vaultClient.ReadKVSecret(ctx, jwtKVMount, vaultPath)
	if err != nil {
		s.logger.WithField("tenant_id", tenantID).WithError(err).Warn("Failed to read JWT signing key from Vault, will generate new key")
	}

	if data != nil {
		if pemStr, ok := data["private_key_pem"].(string); ok && pemStr != "" {
			key, parseErr := parseRSAPrivateKeyPEM(pemStr)
			if parseErr != nil {
				s.logger.WithField("tenant_id", tenantID).WithError(parseErr).Warn("Failed to parse stored JWT signing key, will regenerate")
			} else {
				s.logger.WithField("tenant_id", tenantID).Info("Loaded JWT signing key from Vault")
				s.keyCache[tenantID] = key
				return key, nil
			}
		}
	}

	// Generate new key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Persist to Vault
	pemStr := marshalRSAPrivateKeyPEM(privateKey)
	writeErr := s.vaultClient.WriteKVSecret(ctx, jwtKVMount, vaultPath, map[string]interface{}{
		"private_key_pem": pemStr,
		"created_at":      time.Now().UTC().Format(time.RFC3339),
	})
	if writeErr != nil {
		// Log but don't fail — key works in-memory, next restart will retry
		s.logger.WithField("tenant_id", tenantID).WithError(writeErr).Error("Failed to persist JWT signing key to Vault — key is ephemeral until next restart")
	} else {
		s.logger.WithField("tenant_id", tenantID).Info("JWT signing key generated and persisted to Vault")
	}

	s.keyCache[tenantID] = privateKey
	return privateKey, nil
}

// marshalRSAPrivateKeyPEM encodes an RSA private key as PKCS#8 PEM.
func marshalRSAPrivateKeyPEM(key *rsa.PrivateKey) string {
	derBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		// PKCS#8 marshalling of RSA keys should never fail
		panic(fmt.Sprintf("failed to marshal RSA private key: %v", err))
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	}
	return string(pem.EncodeToMemory(block))
}

// parseRSAPrivateKeyPEM decodes a PEM-encoded RSA private key (PKCS#8 or PKCS#1).
func parseRSAPrivateKeyPEM(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	// Try PKCS#8 first
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not RSA")
		}
		return rsaKey, nil
	}

	// Fall back to PKCS#1
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// getPublicKey retrieves the public key for JWT verification using the same cache.
func (s *JWTSVIDService) getPublicKey(
	ctx context.Context,
	tenantID string,
) (*rsa.PublicKey, error) {
	privateKey, err := s.getOrCreateSigningKey(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &privateKey.PublicKey, nil
}

// generateJTI generates a unique JWT ID
func generateJTI() string {
	return uuid.New().String()
}
