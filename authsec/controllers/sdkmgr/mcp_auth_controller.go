package sdkmgr

import (
	"fmt"
	"net/http"
	"strings"

	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MCPAuthController handles the 12 MCP Auth endpoints.
type MCPAuthController struct {
	Service *sdkmgrSvc.MCPAuthService
}

// NewMCPAuthController creates a new controller wired to the service.
func NewMCPAuthController(svc *sdkmgrSvc.MCPAuthService) *MCPAuthController {
	return &MCPAuthController{Service: svc}
}

// ---------- Request types ----------

type oauthStartRequest struct {
	ClientID string `json:"client_id" binding:"required"`
	AppName  string `json:"app_name" binding:"required"`
}

type authenticateRequest struct {
	JWTToken  string `json:"jwt_token" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
	ExpiresIn int64  `json:"expires_in"`
}

type protectToolRequest struct {
	SessionID *string `json:"session_id"`
	ToolName  string  `json:"tool_name" binding:"required"`
	ClientID  string  `json:"client_id" binding:"required"`
	AppName   string  `json:"app_name" binding:"required"`
}

type toolsListRequest struct {
	ClientID  string        `json:"client_id" binding:"required"`
	AppName   string        `json:"app_name" binding:"required"`
	UserTools []interface{} `json:"user_tools" binding:"required"`
}

type toolCallRequest struct {
	ClientID  string                 `json:"client_id" binding:"required"`
	AppName   string                 `json:"app_name" binding:"required"`
	Arguments map[string]interface{} `json:"arguments" binding:"required"`
}

type cleanupSessionsRequest struct {
	ClientID string `json:"client_id" binding:"required"`
	AppName  string `json:"app_name" binding:"required"`
	Reason   string `json:"reason"`
}

type callbackRequest struct {
	Code      string  `json:"code" binding:"required"`
	State     string  `json:"state" binding:"required"`
	SessionID *string `json:"session_id"`
	ClientID  *string `json:"client_id"`
}

// ---------- Handlers ----------

// Health handles GET /health
func (c *MCPAuthController) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.Service.HealthCheck())
}

// ToolsList handles POST /tools/list
func (c *MCPAuthController) ToolsList(ctx *gin.Context) {
	var req toolsListRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.WithFields(logrus.Fields{
		"client_id":  req.ClientID,
		"app_name":   req.AppName,
		"tool_count": len(req.UserTools),
	}).Info("tools list request")

	result := c.Service.GetToolsList(req.ClientID, req.AppName, req.UserTools)
	ctx.JSON(http.StatusOK, result)
}

// ToolCall handles POST /tools/call/:tool_name
func (c *MCPAuthController) ToolCall(ctx *gin.Context) {
	toolName := ctx.Param("tool_name")
	if !strings.HasPrefix(toolName, "oauth_") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid OAuth tool: %s", toolName)})
		return
	}

	var req toolCallRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := c.Service.ExecuteOAuthTool(toolName, req.ClientID, req.AppName, req.Arguments)
	ctx.JSON(http.StatusOK, result)
}

// Start handles POST /start
func (c *MCPAuthController) Start(ctx *gin.Context) {
	var req oauthStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.Service.StartOAuthFlow(req.ClientID, req.AppName)
	if err != nil {
		logrus.WithError(err).Error("OAuth start failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// Authenticate handles POST /authenticate
func (c *MCPAuthController) Authenticate(ctx *gin.Context) {
	var req authenticateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 3600
	}

	result, err := c.Service.AuthenticateWithJWT(req.JWTToken, req.SessionID, req.ExpiresIn)
	if err != nil {
		logrus.WithError(err).Error("authentication failed")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// CallbackJSON handles POST /callback (JSON response)
func (c *MCPAuthController) CallbackJSON(ctx *gin.Context) {
	var req callbackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sid := ""
	if req.SessionID != nil {
		sid = *req.SessionID
	}
	cid := ""
	if req.ClientID != nil {
		cid = *req.ClientID
	}

	result, err := c.Service.HandleOAuthCallback(req.Code, req.State, sid, cid)
	if err != nil {
		logrus.WithError(err).Error("OAuth callback failed")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// CallbackHTML handles GET /callback (browser redirect, HTML response)
func (c *MCPAuthController) CallbackHTML(ctx *gin.Context) {
	code := ctx.Query("code")
	state := ctx.Query("state")
	sessionID := ctx.Query("session_id")
	clientID := ctx.Query("client_id")

	result, err := c.Service.HandleOAuthCallback(code, state, sessionID, clientID)
	if err != nil {
		logrus.WithError(err).Error("OAuth callback page failed")
		ctx.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(fmt.Sprintf(`
			<html>
			  <body style="font-family: sans-serif; padding: 24px;">
			    <h2>Authentication Failed</h2>
			    <p>%s</p>
			    <p>Return to MCP Inspector and retry <code>oauth_start</code>.</p>
			  </body>
			</html>`, err.Error())))
		return
	}

	sid, _ := result["session_id"].(string)
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`
		<html>
		  <body style="font-family: sans-serif; padding: 24px;">
		    <h2>Authentication Successful</h2>
		    <p>Your OAuth session is authenticated.</p>
		    <p><b>Session ID:</b> <code>%s</code></p>
		    <p>You can return to MCP Inspector and call <code>oauth_status</code> or your protected tool.</p>
		  </body>
		</html>`, sid)))
}

// Status handles GET /status/:session_id
func (c *MCPAuthController) Status(ctx *gin.Context) {
	sessionID := ctx.Param("session_id")
	result := c.Service.GetSessionStatus(sessionID)
	ctx.JSON(http.StatusOK, result)
}

// SessionsStatus handles GET /sessions/status
func (c *MCPAuthController) SessionsStatus(ctx *gin.Context) {
	result := c.Service.GetActiveSessionsCount()
	ctx.JSON(http.StatusOK, result)
}

// Logout handles POST /logout
func (c *MCPAuthController) Logout(ctx *gin.Context) {
	sessionID := ctx.Query("session_id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id query parameter is required"})
		return
	}
	result := c.Service.LogoutSession(sessionID)
	ctx.JSON(http.StatusOK, result)
}

// CleanupSessions handles POST /cleanup-sessions
func (c *MCPAuthController) CleanupSessions(ctx *gin.Context) {
	var req cleanupSessionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Reason == "" {
		req.Reason = "server_shutdown"
	}

	result := c.Service.CleanupSessions(req.ClientID, req.AppName, req.Reason)
	ctx.JSON(http.StatusOK, result)
}

// ProtectTool handles POST /protect-tool
func (c *MCPAuthController) ProtectTool(ctx *gin.Context) {
	var req protectToolRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := c.Service.ProtectTool(req.SessionID, req.ToolName, req.ClientID, req.AppName)
	ctx.JSON(http.StatusOK, result)
}
