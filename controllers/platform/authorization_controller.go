package platform

import (
	"net/http"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PolicyCheckRequest represents PDP input.
type PolicyCheckRequest struct {
	PrincipalID string `json:"principal_id" binding:"required"`
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
	ScopeID     string `json:"scope_id,omitempty"`
}

// PolicyCheckResponse represents PDP output.
type PolicyCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Trace   string `json:"trace"`
}

// AuthorizationController exposes PDP checks for admin and end-user contexts.
type AuthorizationController struct{}

func NewAuthorizationController() *AuthorizationController {
	return &AuthorizationController{}
}

// PolicyDecisionPointCheckAdmin godoc
// @Summary Admin Authorization - Policy Check (primary DB)
// @Description Uses the primary admin database. Queries role_bindings -> roles -> role_permissions -> permissions. Scope match: specific scope or tenant-wide (NULL). Tenant context from token.
// @Tags Admin Authorization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body PolicyCheckRequest true "Policy check payload"
// @Success 200 {object} PolicyCheckResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/policy/check [post]
func (ac *AuthorizationController) PolicyDecisionPointCheckAdmin(c *gin.Context) {
	var req PolicyCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	principalID, err := uuid.Parse(req.PrincipalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Principal ID format"})
		return
	}

	var scopeID *uuid.UUID
	if req.ScopeID != "" {
		sid, err := uuid.Parse(req.ScopeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Scope ID format"})
			return
		}
		scopeID = &sid
	}

	result, err := services.NewRBACService(config.DB).PolicyDecisionPointCheck(principalID, req.Resource, req.Action, scopeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy check failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, PolicyCheckResponse{
		Allowed: result.Allowed,
		Trace:   result.Trace,
	})
}

// PolicyDecisionPointCheckUser godoc
// @Summary Enduser Authorization - Policy Check (tenant DB)
// @Description Uses the tenant database. Queries role_bindings -> roles -> role_permissions -> permissions. Scope match: specific scope or tenant-wide (NULL). Tenant context from token.
// @Tags Enduser Authorization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body PolicyCheckRequest true "Policy check payload"
// @Success 200 {object} PolicyCheckResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/policy/check [post]
func (ac *AuthorizationController) PolicyDecisionPointCheckUser(c *gin.Context) {
	var req PolicyCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect tenant database: " + err.Error()})
		return
	}

	principalID, err := uuid.Parse(req.PrincipalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Principal ID format"})
		return
	}

	var scopeID *uuid.UUID
	if req.ScopeID != "" {
		sid, err := uuid.Parse(req.ScopeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Scope ID format"})
			return
		}
		scopeID = &sid
	}

	result, err := services.NewRBACService(tenantDB).PolicyDecisionPointCheck(principalID, req.Resource, req.Action, scopeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy check failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, PolicyCheckResponse{
		Allowed: result.Allowed,
		Trace:   result.Trace,
	})
}
