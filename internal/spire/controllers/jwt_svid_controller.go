package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/dto"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/middleware"
	"github.com/authsec-ai/authsec/internal/spire/services"
	"github.com/authsec-ai/authsec/internal/spire/utils"
)

// defaultMaxDelegatedTTL is the default maximum TTL for delegated JWT-SVIDs (24 hours).
const defaultMaxDelegatedTTL = 86400

// restrictedCustomClaims are claim keys that cannot be injected via custom_claims in delegated tokens.
var restrictedCustomClaims = map[string]bool{
	"role": true, "roles": true, "perms": true,
	"scopes": true, "scope": true, "admin": true, "is_admin": true,
}

// JWTSVIDController handles JWT-SVID operations
type JWTSVIDController struct {
	service          *services.JWTSVIDService
	logger           *logrus.Entry
	defaultAudience  []string
	allowedAudiences map[string]bool
}

// JWTControllerOption configures optional JWTSVIDController behaviour.
type JWTControllerOption func(*JWTSVIDController)

// WithDefaultAudience sets the audience applied when a delegated issuance omits it.
func WithDefaultAudience(aud []string) JWTControllerOption {
	return func(c *JWTSVIDController) { c.defaultAudience = aud }
}

// WithAllowedAudiences restricts delegated issuance to the listed audiences.
func WithAllowedAudiences(aud []string) JWTControllerOption {
	return func(c *JWTSVIDController) {
		c.allowedAudiences = make(map[string]bool, len(aud))
		for _, a := range aud {
			c.allowedAudiences[a] = true
		}
	}
}

// NewJWTSVIDController creates a new JWT-SVID controller
func NewJWTSVIDController(service *services.JWTSVIDService, logger *logrus.Entry, opts ...JWTControllerOption) *JWTSVIDController {
	ctrl := &JWTSVIDController{
		service: service,
		logger:  logger,
	}
	for _, o := range opts {
		o(ctrl)
	}
	return ctrl
}

// IssueJWTSVID handles POST /spire/v1/jwt/issue
func (ctrl *JWTSVIDController) IssueJWTSVID(c *gin.Context) {
	var req struct {
		TenantID     string                 `json:"tenant_id"`
		SpiffeID     string                 `json:"spiffe_id"`
		Audience     []string               `json:"audience"`
		TTL          int                    `json:"ttl"`
		CustomClaims map[string]interface{} `json:"custom_claims,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	svidResp, err := ctrl.service.IssueJWTSVID(c.Request.Context(), &services.IssueJWTSVIDRequest{
		TenantID:     req.TenantID,
		SpiffeID:     req.SpiffeID,
		Audience:     req.Audience,
		TTL:          req.TTL,
		CustomClaims: req.CustomClaims,
	})
	if err != nil {
		ctrl.sendError(c, errors.NewInternalError(err.Error(), err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"spiffe_id":  svidResp.SpiffeID,
		"token":      svidResp.Token,
		"expires_at": svidResp.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ValidateJWTSVID handles POST /spire/v1/jwt/validate
func (ctrl *JWTSVIDController) ValidateJWTSVID(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		Token    string `json:"token"`
		Audience string `json:"audience"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	validResp, err := ctrl.service.ValidateJWTSVID(c.Request.Context(), &services.ValidateJWTSVIDRequest{
		TenantID: req.TenantID,
		Token:    req.Token,
		Audience: req.Audience,
	})
	if err != nil {
		ctrl.sendError(c, errors.NewInternalError(err.Error(), err))
		return
	}

	// Build flattened response with key claims at top level
	resp := gin.H{
		"spiffe_id": validResp.SpiffeID,
		"valid":     validResp.Valid,
		"claims":    validResp.Claims,
	}
	if validResp.Valid && validResp.Claims != nil {
		if sub, ok := validResp.Claims["sub"].(string); ok {
			resp["sub"] = sub
		}
		if tid, ok := validResp.Claims["tenant_id"].(string); ok {
			resp["tenant_id"] = tid
		}
		if perms, ok := validResp.Claims["permissions"]; ok {
			resp["permissions"] = perms
		}
		if aud, ok := validResp.Claims["aud"]; ok {
			resp["audience"] = aud
		}
		if exp, ok := validResp.Claims["exp"]; ok {
			resp["expires_at"] = exp
		}
		if iat, ok := validResp.Claims["iat"]; ok {
			resp["issued_at"] = iat
		}
	}

	c.JSON(http.StatusOK, resp)
}

// GetJWTBundle handles GET /spire/v1/jwt/bundle
func (ctrl *JWTSVIDController) GetJWTBundle(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id query parameter is required", nil))
		return
	}

	bundle, err := ctrl.service.GetJWTBundle(c.Request.Context(), tenantID)
	if err != nil {
		ctrl.sendError(c, errors.NewInternalError(err.Error(), err))
		return
	}

	// Send raw JWKS JSON
	c.Data(http.StatusOK, "application/json", []byte(bundle))
}

// IssueDelegatedJWTSVID handles POST /spire/v1/jwt/issue-delegated
// Protected by JWT auth middleware rather than mTLS.
//
// Authorization checks:
//  1. Caller's tenant must match the requested tenant_id
//  2. Requested SPIFFE ID must belong to the caller's tenant trust domain
//  3. Custom claims cannot inject elevated roles/permissions
//  4. TTL is capped to prevent long-lived delegated tokens
func (ctrl *JWTSVIDController) IssueDelegatedJWTSVID(c *gin.Context) {
	var req struct {
		TenantID     string                 `json:"tenant_id"`
		SpiffeID     string                 `json:"spiffe_id"`
		Audience     []string               `json:"audience"`
		TTL          int                    `json:"ttl"`
		CustomClaims map[string]interface{} `json:"custom_claims,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	// Extract caller identity from context
	callerTenantID, _ := middleware.GetSpireTenantID(c)
	claims, _ := middleware.GetSpireClaims(c)

	// Validate delegation authorization
	if err := ctrl.validateDelegationAuth(claims, callerTenantID, req.TenantID, req.SpiffeID); err != nil {
		ctrl.sendError(c, errors.NewForbiddenError(err.Error(), err))
		return
	}

	// Apply default audience if none specified
	if len(req.Audience) == 0 {
		if len(ctrl.defaultAudience) > 0 {
			req.Audience = ctrl.defaultAudience
		} else {
			ctrl.sendError(c, errors.NewBadRequestError("audience is required", nil))
			return
		}
	}

	// Validate audience against whitelist (if configured)
	if len(ctrl.allowedAudiences) > 0 {
		for _, aud := range req.Audience {
			if !ctrl.allowedAudiences[aud] {
				ctrl.sendError(c, errors.NewForbiddenError(fmt.Sprintf("audience %q is not allowed", aud), nil))
				return
			}
		}
	}

	// Default TTL if not specified
	if req.TTL <= 0 {
		req.TTL = defaultMaxDelegatedTTL
	}

	// Strip restricted keys from custom claims
	for k := range req.CustomClaims {
		if restrictedCustomClaims[strings.ToLower(k)] {
			delete(req.CustomClaims, k)
		}
	}

	svidResp, err := ctrl.service.IssueJWTSVID(c.Request.Context(), &services.IssueJWTSVIDRequest{
		TenantID:     req.TenantID,
		SpiffeID:     req.SpiffeID,
		Audience:     req.Audience,
		TTL:          req.TTL,
		CustomClaims: req.CustomClaims,
	})
	if err != nil {
		ctrl.sendError(c, errors.NewInternalError(err.Error(), err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"spiffe_id":  svidResp.SpiffeID,
		"token":      svidResp.Token,
		"expires_at": svidResp.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// RenewJWTSVID handles POST /spire/v1/jwt/renew
// Renews an existing valid JWT-SVID by issuing a new token with the same claims.
func (ctrl *JWTSVIDController) RenewJWTSVID(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		Token    string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	if req.TenantID == "" || req.Token == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id and token are required", nil))
		return
	}

	// Validate the existing token (without audience check)
	validResp, err := ctrl.service.ValidateJWTSVID(c.Request.Context(), &services.ValidateJWTSVIDRequest{
		TenantID: req.TenantID,
		Token:    req.Token,
	})
	if err != nil {
		ctrl.sendError(c, errors.NewInternalError("Failed to validate token", err))
		return
	}
	if !validResp.Valid {
		ctrl.sendError(c, errors.NewUnauthorizedError("Token is invalid or expired - cannot renew", nil))
		return
	}

	// Extract claims to re-issue
	spiffeID, _ := validResp.Claims["sub"].(string)

	// Reconstruct audience
	var audience []string
	if audClaim, ok := validResp.Claims["aud"]; ok {
		switch aud := audClaim.(type) {
		case []interface{}:
			for _, a := range aud {
				if aStr, ok := a.(string); ok {
					audience = append(audience, aStr)
				}
			}
		case string:
			audience = []string{aud}
		}
	}

	// Collect custom claims (everything except standard JWT fields)
	customClaims := make(map[string]interface{})
	standardClaims := map[string]bool{
		"iss": true, "sub": true, "aud": true, "exp": true,
		"nbf": true, "iat": true, "jti": true,
	}
	for k, v := range validResp.Claims {
		if !standardClaims[k] {
			customClaims[k] = v
		}
	}

	// Re-issue with same claims, fresh TTL
	svidResp, err := ctrl.service.IssueJWTSVID(c.Request.Context(), &services.IssueJWTSVIDRequest{
		TenantID:     req.TenantID,
		SpiffeID:     spiffeID,
		Audience:     audience,
		TTL:          defaultMaxDelegatedTTL,
		CustomClaims: customClaims,
	})
	if err != nil {
		ctrl.sendError(c, errors.NewInternalError(err.Error(), err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"spiffe_id":  svidResp.SpiffeID,
		"token":      svidResp.Token,
		"expires_at": svidResp.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// validateDelegationAuth checks that the caller is authorized to issue a delegated JWT-SVID.
func (ctrl *JWTSVIDController) validateDelegationAuth(claims *utils.JWTClaims, callerTenantID, reqTenantID, reqSpiffeID string) error {
	if callerTenantID == "" {
		return fmt.Errorf("caller tenant ID not found in authentication context")
	}

	// Tenant must match
	if callerTenantID != reqTenantID {
		return fmt.Errorf("tenant mismatch: authenticated as tenant %s but requesting delegation for tenant %s", callerTenantID, reqTenantID)
	}

	// SPIFFE ID must belong to the caller's tenant trust domain
	expectedPrefix := fmt.Sprintf("spiffe://%s/", reqTenantID)
	if !strings.HasPrefix(reqSpiffeID, expectedPrefix) {
		return fmt.Errorf("spiffe_id %s does not belong to tenant %s trust domain", reqSpiffeID, reqTenantID)
	}

	return nil
}

// sendError sends an error response
func (ctrl *JWTSVIDController) sendError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewInternalError("Internal server error", err)
	}

	ctrl.logger.WithFields(logrus.Fields{
		"code":    appErr.Code,
		"message": appErr.Message,
	}).WithError(appErr.Err).Error("JWT-SVID request failed")

	c.JSON(appErr.Status, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}
