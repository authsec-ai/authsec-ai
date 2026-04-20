package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/dto"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/middleware"
	"github.com/authsec-ai/authsec/internal/spire/services"
)

// AgentController handles agent operations including listing and renewal
type AgentController struct {
	agentService   *services.AgentService
	renewalService *services.AgentRenewalService
	logger         *logrus.Entry
}

// NewAgentController creates a new agent controller
func NewAgentController(
	agentService *services.AgentService,
	renewalService *services.AgentRenewalService,
	logger *logrus.Entry,
) *AgentController {
	return &AgentController{
		agentService:   agentService,
		renewalService: renewalService,
		logger:         logger,
	}
}

// ListAgents handles GET /spire/v1/agents
// Lists all active agents for the authenticated tenant
func (ctrl *AgentController) ListAgents(c *gin.Context) {
	tenantID, ok := middleware.GetSpireTenantID(c)
	if !ok || tenantID == "" {
		ctrl.sendError(c, errors.NewUnauthorizedError("tenant_id not found in authentication context", nil))
		return
	}

	ctrl.logger.WithField("tenant_id", tenantID).Info("Listing agents")

	agents, err := ctrl.agentService.ListAgentsByTenant(c.Request.Context(), tenantID)
	if err != nil {
		ctrl.sendError(c, err)
		return
	}

	// Convert to response DTOs
	agentResponses := make([]*dto.AgentResponse, len(agents))
	for i, agent := range agents {
		agentResponses[i] = &dto.AgentResponse{
			ID:              agent.ID,
			SpiffeID:        agent.SpiffeID,
			NodeID:          agent.NodeID,
			AttestationType: agent.AttestationType,
			Status:          agent.Status,
			LastSeen:        agent.LastSeen,
			CreatedAt:       agent.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, dto.ListAgentsResponse{
		Agents: agentResponses,
		Count:  len(agentResponses),
	})
}

// RenewAgent handles POST /spire/v1/agent/renew
func (ctrl *AgentController) RenewAgent(c *gin.Context) {
	var req dto.AgentRenewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	// Validate required fields
	if req.AgentID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("agent_id is required", nil))
		return
	}
	if req.CSR == "" {
		ctrl.sendError(c, errors.NewBadRequestError("csr is required", nil))
		return
	}

	// Call service
	serviceReq := &services.AgentRenewRequest{
		AgentID:  req.AgentID,
		TenantID: req.TenantID,
		CSR:      req.CSR,
	}

	resp, err := ctrl.renewalService.Renew(c.Request.Context(), serviceReq)
	if err != nil {
		ctrl.sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AgentRenewResponse{
		SpiffeID:    resp.SpiffeID,
		Certificate: resp.Certificate,
		CABundle:    resp.CABundle,
		TTL:         resp.TTL,
		CAChain:     resp.CAChain,
		ExpiresAt:   resp.ExpiresAt,
	})
}

// sendError sends an error response
func (ctrl *AgentController) sendError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewInternalError("Internal server error", err)
	}

	ctrl.logger.WithFields(logrus.Fields{
		"code":    appErr.Code,
		"message": appErr.Message,
	}).WithError(appErr.Err).Error("Agent request failed")

	c.JSON(appErr.Status, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}
