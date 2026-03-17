package sdkmgr

import (
	"net/http"

	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SPIREController handles the 5 SPIRE proxy endpoints.
type SPIREController struct {
	Service *sdkmgrSvc.SPIREProxyService
}

// NewSPIREController creates a new controller wired to the service.
func NewSPIREController(svc *sdkmgrSvc.SPIREProxyService) *SPIREController {
	return &SPIREController{Service: svc}
}

// ---------- Request types ----------

type initializeWorkloadRequest struct {
	ClientID            string            `json:"client_id" binding:"required"`
	SocketPath          string            `json:"socket_path"`
	EnvironmentMetadata map[string]string `json:"environment_metadata"`
}

type renewSVIDRequest struct {
	ClientID            string            `json:"client_id" binding:"required"`
	SocketPath          string            `json:"socket_path"`
	EnvironmentMetadata map[string]string `json:"environment_metadata"`
}

type getSVIDStatusRequest struct {
	ClientID string  `json:"client_id" binding:"required"`
	SpiffeID *string `json:"spiffe_id"`
}

// ---------- Handlers ----------

const defaultSocketPath = "/run/spire/sockets/agent.sock"

// Health handles GET /health
func (c *SPIREController) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.Service.HealthCheck())
}

// Initialize handles POST /workload/initialize
func (c *SPIREController) Initialize(ctx *gin.Context) {
	var req initializeWorkloadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.SocketPath == "" {
		req.SocketPath = defaultSocketPath
	}
	if req.EnvironmentMetadata == nil {
		req.EnvironmentMetadata = map[string]string{}
	}

	result, err := c.Service.FetchSVIDForWorkload(req.ClientID, req.SocketPath, req.EnvironmentMetadata)
	if err != nil {
		logrus.WithError(err).Error("workload initialization failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// Renew handles POST /workload/renew
func (c *SPIREController) Renew(ctx *gin.Context) {
	var req renewSVIDRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.SocketPath == "" {
		req.SocketPath = defaultSocketPath
	}
	if req.EnvironmentMetadata == nil {
		req.EnvironmentMetadata = map[string]string{}
	}

	result, err := c.Service.FetchSVIDForWorkload(req.ClientID, req.SocketPath, req.EnvironmentMetadata)
	if err != nil {
		logrus.WithError(err).Error("SVID renewal failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// Status handles POST /workload/status
func (c *SPIREController) Status(ctx *gin.Context) {
	var req getSVIDStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.Service.GetSVIDStatus(req.ClientID, req.SpiffeID)
	if err != nil {
		logrus.WithError(err).Error("get SVID status failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// ValidateConnection handles GET /validate-agent-connection
func (c *SPIREController) ValidateConnection(ctx *gin.Context) {
	socketPath := ctx.DefaultQuery("socket_path", defaultSocketPath)
	result := c.Service.ValidateAgentConnection(socketPath)
	ctx.JSON(http.StatusOK, result)
}
