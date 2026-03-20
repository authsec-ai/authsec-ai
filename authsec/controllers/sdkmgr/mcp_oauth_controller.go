package sdkmgr

import (
	"net/http"

	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MCPOAuthController handles the 4 MCP OAuth endpoints for the playground.
type MCPOAuthController struct {
	Service *sdkmgrSvc.MCPOAuthService
}

// NewMCPOAuthController creates a new controller.
func NewMCPOAuthController(svc *sdkmgrSvc.MCPOAuthService) *MCPOAuthController {
	return &MCPOAuthController{Service: svc}
}

// ---------- Request types ----------

type refreshTokenRequest struct {
	ServerID       string `json:"server_id" binding:"required"`
	ConversationID string `json:"conversation_id" binding:"required"`
	TenantID       string `json:"tenant_id" binding:"required"`
	RefreshToken   string `json:"refresh_token" binding:"required"`
	TokenURL       string `json:"token_url" binding:"required"`
	ClientID       string `json:"client_id" binding:"required"`
}

// ---------- Handlers ----------

// CheckRequirements handles GET /check-requirements
func (c *MCPOAuthController) CheckRequirements(ctx *gin.Context) {
	serverURL := ctx.Query("server_url")
	if serverURL == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "server_url query parameter is required"})
		return
	}
	result := c.Service.CheckOAuthRequirements(serverURL)
	ctx.JSON(http.StatusOK, result)
}

// Authorize handles GET /authorize
func (c *MCPOAuthController) Authorize(ctx *gin.Context) {
	serverID := ctx.Query("server_id")
	conversationID := ctx.Query("conversation_id")
	tenantID := ctx.Query("tenant_id")
	authURL := ctx.Query("auth_url")
	tokenURL := ctx.Query("token_url")

	if serverID == "" || conversationID == "" || tenantID == "" || authURL == "" || tokenURL == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "server_id, conversation_id, tenant_id, auth_url, and token_url are required"})
		return
	}

	clientID := queryPtr(ctx, "client_id")
	registrationURL := queryPtr(ctx, "registration_url")
	scope := queryPtr(ctx, "scope")
	redirectURI := queryPtr(ctx, "redirect_uri")

	result, err := c.Service.Authorize(
		serverID, conversationID, tenantID, authURL, tokenURL,
		clientID, registrationURL, scope, redirectURI,
	)
	if err != nil {
		logrus.WithError(err).Error("OAuth authorization error")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// Callback handles GET /callback (HTML response with postMessage)
func (c *MCPOAuthController) Callback(ctx *gin.Context) {
	code := ctx.DefaultQuery("code", "")
	state := ctx.DefaultQuery("state", "")
	oauthError := ctx.DefaultQuery("error", "")
	errorDesc := ctx.DefaultQuery("error_description", "")

	html, status := c.Service.HandleCallback(code, state, oauthError, errorDesc)
	ctx.Data(status, "text/html; charset=utf-8", []byte(html))
}

// Refresh handles POST /refresh
func (c *MCPOAuthController) Refresh(ctx *gin.Context) {
	var req refreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.Service.RefreshToken(req.RefreshToken, req.TokenURL, req.ClientID)
	if err != nil {
		logrus.WithError(err).Error("token refresh error")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// queryPtr returns a *string for a query parameter, or nil if absent.
func queryPtr(ctx *gin.Context, key string) *string {
	v := ctx.Query(key)
	if v == "" {
		return nil
	}
	return &v
}
