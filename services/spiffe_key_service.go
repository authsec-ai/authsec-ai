package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SpiffeKeyService manages the RSA key pair used to sign RS256 JWT-SVIDs
// that SPIRE can validate via the JWKS endpoint.
type SpiffeKeyService struct {
	mu         sync.RWMutex
	privateKey *rsa.PrivateKey
	keyID      string // stable kid so SPIRE JWKS cache works correctly
	issuer     string // must match OIDCIssuer in SPIRE server config
}

var (
	globalSpiffeKeyService *SpiffeKeyService
	spiffeKeyServiceOnce   sync.Once
)

// GetSpiffeKeyService returns the singleton SpiffeKeyService.
// The RSA key is generated once at startup (or loaded from SPIFFE_RSA_PRIVATE_KEY env var).
func GetSpiffeKeyService() (*SpiffeKeyService, error) {
	var initErr error
	spiffeKeyServiceOnce.Do(func() {
		svc, err := newSpiffeKeyService()
		if err != nil {
			initErr = err
			return
		}
		globalSpiffeKeyService = svc
	})
	if initErr != nil {
		return nil, initErr
	}
	return globalSpiffeKeyService, nil
}

func newSpiffeKeyService() (*SpiffeKeyService, error) {
	issuer := os.Getenv("SPIFFE_OIDC_ISSUER")
	if issuer == "" {
		issuer = "https://user-flow.authsec.dev" // fallback; override via env
	}

	keyID := os.Getenv("SPIFFE_JWKS_KEY_ID")
	if keyID == "" {
		keyID = "authsec-spiffe-key-1"
	}

	var privateKey *rsa.PrivateKey

	// Allow injecting a PEM-encoded PKCS8 private key via env for production.
	if pemB64 := os.Getenv("SPIFFE_RSA_PRIVATE_KEY_B64"); pemB64 != "" {
		der, err := base64.StdEncoding.DecodeString(pemB64)
		if err != nil {
			return nil, fmt.Errorf("decode SPIFFE_RSA_PRIVATE_KEY_B64: %w", err)
		}
		key, err := x509.ParsePKCS8PrivateKey(der)
		if err != nil {
			return nil, fmt.Errorf("parse SPIFFE_RSA_PRIVATE_KEY_B64: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("SPIFFE_RSA_PRIVATE_KEY_B64 is not an RSA private key")
		}
	} else {
		// Generate a fresh 2048-bit RSA key in memory.
		var err error
		privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generate RSA key: %w", err)
		}
		fmt.Println("[WARN] SPIFFE JWT signing key is EPHEMERAL — it will rotate on every pod restart. " +
			"Set SPIFFE_RSA_PRIVATE_KEY_B64 env var for persistent keys in production. " +
			"JWKS consumers will see stale keys after restart until they re-fetch.")
	}

	return &SpiffeKeyService{
		privateKey: privateKey,
		keyID:      keyID,
		issuer:     issuer,
	}, nil
}

// JWKSResponse is the JSON structure served at /.well-known/jwks.json
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key (RSA public key, RFC 7517)
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"` // base64url-encoded modulus
	E   string `json:"e"` // base64url-encoded exponent
}

// PublicJWKS returns the JWKS document for the current signing key.
func (s *SpiffeKeyService) PublicJWKS() JWKSResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pub := &s.privateKey.PublicKey
	nBytes := pub.N.Bytes()
	eBytes := big.NewInt(int64(pub.E)).Bytes()

	return JWKSResponse{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Kid: s.keyID,
				Alg: "RS256",
				N:   base64.RawURLEncoding.EncodeToString(nBytes),
				E:   base64.RawURLEncoding.EncodeToString(eBytes),
			},
		},
	}
}

// PublicJWKSJSON returns the JWKS as a JSON byte slice.
func (s *SpiffeKeyService) PublicJWKSJSON() ([]byte, error) {
	return json.Marshal(s.PublicJWKS())
}

// Issuer returns the configured OIDC issuer URL.
func (s *SpiffeKeyService) Issuer() string {
	return s.issuer
}

// DelegateSVIDRequest carries the data needed to mint a JWT-SVID.
type DelegateSVIDRequest struct {
	// UserJWT is the existing HS256 Bearer token from webauthn-callback.
	UserJWT string
	// UserID extracted from the user JWT (sub / user_id claim).
	UserID string
	// TenantID extracted from the user JWT.
	TenantID string
	// Email extracted from the user JWT.
	Email string
	// AgentType is a label for the AI agent kind (e.g. "ai-assistant", "mcp-agent").
	AgentType string
	// TTL for the issued JWT-SVID (default 1 hour, max 8 hours).
	TTL time.Duration
}

// JWTSVIDClaims are the claims inside the RS256 JWT-SVID.
type JWTSVIDClaims struct {
	// SPIFFE ID as the subject — SPIRE uses this for policy decisions.
	// Pattern: spiffe://<trust-domain>/user/<user_id>/agent/<agent_type>
	Sub string `json:"sub"`

	// Standard OIDC/JWT fields
	Iss string   `json:"iss"`
	Aud []string `json:"aud"`
	Iat int64    `json:"iat"`
	Exp int64    `json:"exp"`
	Nbf int64    `json:"nbf"`

	// AuthSec identity claims (carry user context to the agent)
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`

	// SPIFFE delegation metadata
	AgentType  string `json:"agent_type"`
	SpiffeID   string `json:"spiffe_id"` // same as sub for clarity
	DelegatedBy string `json:"delegated_by"` // always "authsec-ai/user-flow"
}

// IssueJWTSVID mints an RS256 JWT-SVID that encodes the SPIFFE ID of the agent
// acting on behalf of the authenticated user.
//
// The resulting token can be presented to SPIRE's OIDC-backed workload authorizer
// or directly to services that trust the JWKS endpoint.
func (s *SpiffeKeyService) IssueJWTSVID(req DelegateSVIDRequest) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if req.TTL <= 0 || req.TTL > 8*time.Hour {
		req.TTL = 1 * time.Hour
	}

	trustDomain := os.Getenv("SPIFFE_TRUST_DOMAIN")
	if trustDomain == "" {
		trustDomain = "authsec.io"
	}

	agentType := req.AgentType
	if agentType == "" {
		agentType = "ai-agent"
	}

	spiffeID := fmt.Sprintf("spiffe://%s/user/%s/agent/%s", trustDomain, req.UserID, agentType)

	now := time.Now()
	claims := JWTSVIDClaims{
		Sub:         spiffeID,
		Iss:         s.issuer,
		Aud:         []string{"spire-server"},
		Iat:         now.Unix(),
		Nbf:         now.Unix(),
		Exp:         now.Add(req.TTL).Unix(),
		UserID:      req.UserID,
		TenantID:    req.TenantID,
		Email:       req.Email,
		AgentType:   agentType,
		SpiffeID:    spiffeID,
		DelegatedBy: "authsec-ai/user-flow",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, toMapClaims(claims))
	token.Header["kid"] = s.keyID

	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT-SVID: %w", err)
	}
	return signed, nil
}

// toMapClaims converts JWTSVIDClaims to jwt.MapClaims so the library can marshal it.
func toMapClaims(c JWTSVIDClaims) jwt.MapClaims {
	return jwt.MapClaims{
		"sub":          c.Sub,
		"iss":          c.Iss,
		"aud":          c.Aud,
		"iat":          c.Iat,
		"nbf":          c.Nbf,
		"exp":          c.Exp,
		"user_id":      c.UserID,
		"tenant_id":    c.TenantID,
		"email":        c.Email,
		"agent_type":   c.AgentType,
		"spiffe_id":    c.SpiffeID,
		"delegated_by": c.DelegatedBy,
	}
}
