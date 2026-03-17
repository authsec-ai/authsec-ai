package sdkmgr

import (
	"net/http"

	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
)

// DevServerController handles dev MCP server management endpoints.
type DevServerController struct {
	svc *sdkmgrSvc.DevServerService
}

// NewDevServerController creates a new dev server controller.
func NewDevServerController(svc *sdkmgrSvc.DevServerService) *DevServerController {
	return &DevServerController{svc: svc}
}

type startDevServerRequest struct {
	Code           string `json:"code" binding:"required"`
	ConversationID string `json:"conversation_id" binding:"required"`
	TenantID       string `json:"tenant_id" binding:"required"`
}

type stopDevServerRequest struct {
	ServerID       string `json:"server_id" binding:"required"`
	ConversationID string `json:"conversation_id" binding:"required"`
	TenantID       string `json:"tenant_id" binding:"required"`
}

// Start handles POST /dev-server/start.
func (dc *DevServerController) Start(c *gin.Context) {
	var req startDevServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := dc.svc.StartServer(req.Code, req.ConversationID, req.TenantID)
	if result["success"] != true {
		c.JSON(http.StatusBadRequest, result)
		return
	}
	c.JSON(http.StatusOK, result)
}

// Stop handles POST /dev-server/stop.
func (dc *DevServerController) Stop(c *gin.Context) {
	var req stopDevServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := dc.svc.StopServer(req.ServerID, req.ConversationID, req.TenantID)
	if result["success"] != true {
		c.JSON(http.StatusBadRequest, result)
		return
	}
	c.JSON(http.StatusOK, result)
}

// Status handles GET /dev-server/status.
func (dc *DevServerController) Status(c *gin.Context) {
	conversationID := c.Query("conversation_id")
	tenantID := c.Query("tenant_id")

	if conversationID == "" || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id and tenant_id are required"})
		return
	}

	result := dc.svc.GetServerStatus(conversationID, tenantID)
	c.JSON(http.StatusOK, result)
}
