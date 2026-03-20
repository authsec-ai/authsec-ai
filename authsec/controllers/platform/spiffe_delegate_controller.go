package platform

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SpiffeDelegateController exposes two endpoints:
//
//  1. GET  /.well-known/jwks.json      — JWKS document so SPIRE can verify our RS256 JWT-SVIDs
//  2. POST /uflow/auth/enduser/delegate-svid — Exchange a valid user JWT for an RS256 JWT-SVID
//     that an AI agent can use to prove it is acting on behalf of the user.
type SpiffeDelegateController struct {
	keySvc *services.SpiffeKeyService
}

// NewSpiffeDelegateController creates the controller.
// It initialises the singleton RSA key service on first call.
func NewSpiffeDelegateController() (*SpiffeDelegateController, error) {
	svc, err := services.GetSpiffeKeyService()
	if err != nil {
		return nil, fmt.Errorf("init SpiffeKeyService: %w", err)
	}
	return &SpiffeDelegateController{keySvc: svc}, nil
}

// -----------------------------------------------------------------------
// JWKS endpoint
// -----------------------------------------------------------------------

// GetJWKS serves the public RSA key in JWKS format.
//
//	@Summary     JWKS public key endpoint
//	@Description Returns the JSON Web Key Set used to verify RS256 JWT-SVIDs issued by this service.
//	             Configure this URL as the OIDCIssuer JWKS endpoint in your SPIRE server.
//	@Tags        SPIFFE Delegation
//	@Produce     json
//	@Success     200 {object} services.JWKSResponse
//	@Router      /.well-known/jwks.json [get]
func (ctrl *SpiffeDelegateController) GetJWKS(c *gin.Context) {
	data, err := ctrl.keySvc.PublicJWKSJSON()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal JWKS"})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// OIDCDiscovery serves the OIDC discovery document required by SPIRE's OIDC federation.
// SPIRE fetches <issuer>/.well-known/openid-configuration to locate the JWKS URI.
//
//	@Summary     OIDC discovery document
//	@Description OpenID Connect discovery metadata required by SPIRE OIDC federation.
//	@Tags        SPIFFE Delegation
//	@Produce     json
//	@Success     200 {object} map[string]interface{}
//	@Router      /.well-known/openid-configuration [get]
func (ctrl *SpiffeDelegateController) OIDCDiscovery(c *gin.Context) {
	issuer := ctrl.keySvc.Issuer()
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                issuer,
		"jwks_uri":                              issuer + "/.well-known/jwks.json",
		"response_types_supported":              []string{"id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "user_id", "tenant_id", "email", "spiffe_id"},
	})
}

// -----------------------------------------------------------------------
// Delegate-SVID endpoint
// -----------------------------------------------------------------------

// DelegateSVIDRequest is the JSON body for POST /uflow/auth/enduser/delegate-svid
type DelegateSVIDRequest struct {
	// AgentType identifies the AI agent kind.
	// Examples: "ai-assistant", "mcp-agent", "copilot"
	// Becomes part of the SPIFFE ID: spiffe://<trust_domain>/user/<user_id>/agent/<agent_type>
	AgentType string `json:"agent_type" binding:"required"`

	// TTLSeconds is the lifetime of the issued JWT-SVID in seconds.
	// Min: 60 seconds. Max: 28800 seconds (8 hours). Default: 3600 (1 hour).
	TTLSeconds int `json:"ttl_seconds"`
}

// DelegateSVIDResponse is returned on success.
type DelegateSVIDResponse struct {
	// JWTSvid is the signed RS256 JWT the AI agent should use.
	JWTSvid string `json:"jwt_svid"`
	// SpiffeID is the SPIFFE URI encoded in the token's sub claim.
	SpiffeID string `json:"spiffe_id"`
	// Issuer is the OIDC issuer URL (= base URL of this service).
	Issuer string `json:"issuer"`
	// ExpiresIn is the lifetime in seconds.
	ExpiresIn int `json:"expires_in"`
	// JwksURI is where SPIRE (or any verifier) can fetch the public key.
	JwksURI string `json:"jwks_uri"`
}

// DelegateSVID validates the caller's user JWT and issues a short-lived RS256 JWT-SVID
// that the AI agent can use to authenticate on behalf of the user.
//
//	@Summary     Delegate user trust to an AI agent via JWT-SVID
//	@Description Validates the caller's Bearer JWT (issued by webauthn-callback) and mints
//	             an RS256 JWT-SVID with a SPIFFE ID of the form:
//	               spiffe://<trust_domain>/user/<user_id>/agent/<agent_type>
//	             The SPIRE server can verify this token using the JWKS endpoint exposed by
//	             this service at /.well-known/jwks.json.
//	@Tags        SPIFFE Delegation
//	@Accept      json
//	@Produce     json
//	@Param       Authorization  header    string                true  "Bearer <user_jwt>"
//	@Param       body           body      DelegateSVIDRequest   true  "Agent type and optional TTL"
//	@Success     200            {object}  DelegateSVIDResponse
//	@Failure     400            {object}  map[string]string
//	@Failure     401            {object}  map[string]string
//	@Failure     500            {object}  map[string]string
//	@Router      /uflow/auth/enduser/delegate-svid [post]
func (ctrl *SpiffeDelegateController) DelegateSVID(c *gin.Context) {
	// 1. Extract the user's Bearer JWT from the Authorization header.
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header, expected: Bearer <token>"})
		return
	}
	rawJWT := parts[1]

	// 2. Validate the HS256 user JWT using the same secrets as AuthMiddleware.
	userClaims, err := validateUserJWT(rawJWT)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired user token", "details": err.Error()})
		return
	}

	// 3. Bind request body.
	var req DelegateSVIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// 4. Resolve TTL.
	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl < 60*time.Second {
		ttl = 1 * time.Hour // sensible default
	}
	if ttl > 8*time.Hour {
		ttl = 8 * time.Hour
	}

	// 5. Extract identity from user claims.
	userID := claimString(userClaims, "user_id", claimString(userClaims, "sub", ""))
	tenantID := claimString(userClaims, "tenant_id", "")
	email := claimString(userClaims, "email_id", claimString(userClaims, "email", ""))

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token missing user_id / sub claim"})
		return
	}

	// 6. Issue the RS256 JWT-SVID.
	svid, err := ctrl.keySvc.IssueJWTSVID(services.DelegateSVIDRequest{
		UserJWT:   rawJWT,
		UserID:    userID,
		TenantID:  tenantID,
		Email:     email,
		AgentType: req.AgentType,
		TTL:       ttl,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue JWT-SVID", "details": err.Error()})
		return
	}

	trustDomain := os.Getenv("SPIFFE_TRUST_DOMAIN")
	if trustDomain == "" {
		trustDomain = "authsec.io"
	}
	spiffeID := fmt.Sprintf("spiffe://%s/user/%s/agent/%s", trustDomain, userID, req.AgentType)
	issuer := ctrl.keySvc.Issuer()

	c.JSON(http.StatusOK, DelegateSVIDResponse{
		JWTSvid:   svid,
		SpiffeID:  spiffeID,
		Issuer:    issuer,
		ExpiresIn: int(ttl.Seconds()),
		JwksURI:   issuer + "/.well-known/jwks.json",
	})
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

// validateUserJWT parses and validates the HS256 user JWT using the same
// environment-based secrets that AuthMiddleware uses.
func validateUserJWT(tokenString string) (jwt.MapClaims, error) {
	secrets := []string{
		os.Getenv("JWT_DEF_SECRET"),
		os.Getenv("JWT_SDK_SECRET"),
		os.Getenv("JWT_SECRET"),
	}

	var lastErr error
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		})
		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				return claims, nil
			}
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no valid signing secret found")
}

// claimString safely extracts a string claim, returning fallback if absent.
func claimString(claims jwt.MapClaims, key, fallback string) string {
	if v, ok := claims[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}
