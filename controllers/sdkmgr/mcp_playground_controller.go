package sdkmgr

import (
	"fmt"
	"net/http"
	"strconv"

	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MCPPlaygroundController handles the 16 playground endpoints.
type MCPPlaygroundController struct {
	Service *sdkmgrSvc.MCPPlaygroundService
}

// NewMCPPlaygroundController creates a new controller.
func NewMCPPlaygroundController(svc *sdkmgrSvc.MCPPlaygroundService) *MCPPlaygroundController {
	return &MCPPlaygroundController{Service: svc}
}

// ---------- Request types ----------

type createConversationRequest struct {
	TenantID     string   `json:"tenant_id" binding:"required"`
	Title        *string  `json:"title"`
	Model        *string  `json:"model"`
	SystemPrompt *string  `json:"system_prompt"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int     `json:"max_tokens"`
}

type updateConversationRequest struct {
	Title        *string  `json:"title"`
	SystemPrompt *string  `json:"system_prompt"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int     `json:"max_tokens"`
}

type chatRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	Message  string `json:"message" binding:"required"`
}

type chatStreamRequest struct {
	TenantID       string `json:"tenant_id" binding:"required"`
	ConversationID string `json:"conversation_id" binding:"required"`
	Message        string `json:"message" binding:"required"`
}

type addMCPServerRequest struct {
	TenantID          string                 `json:"tenant_id" binding:"required"`
	Name              string                 `json:"name" binding:"required"`
	Protocol          string                 `json:"protocol" binding:"required"`
	ServerURL         string                 `json:"server_url" binding:"required"`
	Config            map[string]interface{} `json:"config"`
	OAuthAccessToken  *string                `json:"oauth_access_token"`
	OAuthRefreshToken *string                `json:"oauth_refresh_token"`
	OAuthTokenExpiry  *string                `json:"oauth_token_expiry"`
	OAuthConfig       map[string]interface{} `json:"oauth_config"`
}

// ---------- Handlers ----------

// Health handles GET /health
func (c *MCPPlaygroundController) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.Service.HealthCheck())
}

// CreateConversation handles POST /conversations
func (c *MCPPlaygroundController) CreateConversation(ctx *gin.Context) {
	var req createConversationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	title := "New Conversation"
	if req.Title != nil {
		title = *req.Title
	}

	conv, err := c.Service.CreateConversation(req.TenantID, title, req.Model, req.SystemPrompt, req.Temperature, req.MaxTokens)
	if err != nil {
		logrus.WithError(err).Error("error creating conversation")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "conversation": conv})
}

// ListConversations handles GET /conversations
func (c *MCPPlaygroundController) ListConversations(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))

	convs, err := c.Service.ListConversations(tenantID, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "conversations": convs, "count": len(convs)})
}

// GetConversation handles GET /conversations/:id
func (c *MCPPlaygroundController) GetConversation(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")

	conv, err := c.Service.GetConversation(tenantID, convID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if conv == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "conversation": conv})
}

// UpdateConversation handles PATCH /conversations/:id
func (c *MCPPlaygroundController) UpdateConversation(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")

	var req updateConversationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conv, err := c.Service.UpdateConversation(tenantID, convID, req.Title, req.SystemPrompt, req.Temperature, req.MaxTokens)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if conv == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "conversation": conv})
}

// DeleteConversation handles DELETE /conversations/:id
func (c *MCPPlaygroundController) DeleteConversation(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")

	deleted, err := c.Service.DeleteConversation(tenantID, convID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !deleted {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "message": "Conversation deleted successfully"})
}

// GetMessages handles GET /conversations/:id/messages
func (c *MCPPlaygroundController) GetMessages(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")

	msgs, err := c.Service.GetConversationMessages(tenantID, convID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "messages": msgs, "count": len(msgs)})
}

// Chat handles POST /conversations/:id/chat (non-streaming)
func (c *MCPPlaygroundController) Chat(ctx *gin.Context) {
	convID := ctx.Param("id")

	var req chatRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.Service.ChatCompletion(req.TenantID, convID, req.Message)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "Azure OpenAI is not configured" {
			status = http.StatusServiceUnavailable
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "response": result})
}

// ChatStream handles POST /chat/stream (SSE streaming)
func (c *MCPPlaygroundController) ChatStream(ctx *gin.Context) {
	var req chatStreamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")

	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	err := c.Service.ChatCompletionStreamWriter(req.TenantID, req.ConversationID, req.Message, ctx.Writer, flusher)
	if err != nil {
		logrus.WithError(err).Error("error in chat stream")
		fmt.Fprintf(ctx.Writer, "data: Sorry, I couldn't process your request. Please try again.\n\n")
		flusher.Flush()
	}
}

// AddMCPServer handles POST /conversations/:id/mcp-servers
func (c *MCPPlaygroundController) AddMCPServer(ctx *gin.Context) {
	convID := ctx.Param("id")

	var req addMCPServerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := c.Service.AddMCPServer(
		req.TenantID, convID, req.Name, req.Protocol, req.ServerURL,
		req.Config, req.OAuthAccessToken, req.OAuthRefreshToken,
		req.OAuthTokenExpiry, req.OAuthConfig,
	)
	if err != nil {
		logrus.WithError(err).Error("error adding MCP server")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "server": server})
}

// ListMCPServers handles GET /conversations/:id/mcp-servers
func (c *MCPPlaygroundController) ListMCPServers(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")

	servers, err := c.Service.ListMCPServers(tenantID, convID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "servers": servers, "count": len(servers)})
}

// DisconnectMCPServer handles POST /conversations/:id/mcp-servers/:sid/disconnect
func (c *MCPPlaygroundController) DisconnectMCPServer(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")
	srvID := ctx.Param("sid")

	ok, err := c.Service.DisconnectMCPServer(tenantID, convID, srvID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "message": "MCP server disconnected successfully"})
}

// ReconnectMCPServer handles POST /conversations/:id/mcp-servers/:sid/reconnect
func (c *MCPPlaygroundController) ReconnectMCPServer(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")
	srvID := ctx.Param("sid")

	ok, err := c.Service.ReconnectMCPServer(tenantID, convID, srvID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found or failed to reconnect"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "message": "MCP server reconnected successfully"})
}

// RemoveMCPServer handles DELETE /conversations/:id/mcp-servers/:sid
func (c *MCPPlaygroundController) RemoveMCPServer(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")
	srvID := ctx.Param("sid")

	ok, err := c.Service.RemoveMCPServer(tenantID, convID, srvID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "message": "MCP server removed successfully"})
}

// GetMCPTools handles GET /conversations/:id/mcp-servers/:sid/tools
func (c *MCPPlaygroundController) GetMCPTools(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")
	srvID := ctx.Param("sid")

	tools, err := c.Service.GetMCPTools(tenantID, convID, srvID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "tools": tools, "count": len(tools)})
}

// GetAllConversationTools handles GET /conversations/:id/tools
func (c *MCPPlaygroundController) GetAllConversationTools(ctx *gin.Context) {
	tenantID := ctx.Query("tenant_id")
	convID := ctx.Param("id")

	allTools, err := c.Service.GetAllConversationTools(tenantID, convID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalTools := 0
	for _, tools := range allTools {
		totalTools += len(tools)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":        true,
		"tools_by_server": allTools,
		"total_tools":    totalTools,
		"server_count":   len(allTools),
	})
}
