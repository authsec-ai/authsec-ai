package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/dto"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/services"
)

// NodeAttestationController handles node attestation requests
type NodeAttestationController struct {
	service *services.NodeAttestationService
	logger  *logrus.Entry
}

// NewNodeAttestationController creates a new node attestation controller
func NewNodeAttestationController(service *services.NodeAttestationService, logger *logrus.Entry) *NodeAttestationController {
	return &NodeAttestationController{
		service: service,
		logger:  logger,
	}
}

// Attest handles POST /spire/v1/node/attest
func (ctrl *NodeAttestationController) Attest(c *gin.Context) {
	var req dto.NodeAttestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	// Validate required fields
	if req.TenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}
	if req.NodeID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("node_id is required", nil))
		return
	}
	if req.CSR == "" {
		ctrl.sendError(c, errors.NewBadRequestError("csr is required", nil))
		return
	}
	if req.AttestationType == "" {
		ctrl.sendError(c, errors.NewBadRequestError("attestation_type is required", nil))
		return
	}
	if req.Evidence == nil {
		ctrl.sendError(c, errors.NewBadRequestError("evidence is required", nil))
		return
	}

	// Call service
	serviceReq := &services.NodeAttestRequest{
		TenantID:        req.TenantID,
		NodeID:          req.NodeID,
		AttestationType: req.AttestationType,
		Evidence:        req.Evidence,
		CSR:             req.CSR,
	}

	resp, err := ctrl.service.Attest(c.Request.Context(), serviceReq)
	if err != nil {
		ctrl.sendError(c, err)
		return
	}

	// Convert CA chain array to single PEM bundle
	caBundle := ""
	if len(resp.CAChain) > 0 {
		for i, cert := range resp.CAChain {
			caBundle += cert
			if i < len(resp.CAChain)-1 {
				caBundle += "\n"
			}
		}
	}

	// Calculate TTL in seconds from expiration time
	ttl := int(resp.ExpiresAt.Sub(time.Now()).Seconds())
	if ttl < 0 {
		ttl = 0
	}

	c.JSON(http.StatusOK, dto.NodeAttestResponse{
		AgentID:     resp.AgentID,
		SpiffeID:    resp.SpiffeID,
		Certificate: resp.Certificate,
		CABundle:    caBundle,
		TTL:         ttl,
	})
}

// sendError sends an error response
func (ctrl *NodeAttestationController) sendError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewInternalError("Internal server error", err)
	}

	ctrl.logger.WithFields(logrus.Fields{
		"code":    appErr.Code,
		"message": appErr.Message,
	}).WithError(appErr.Err).Error("Node attestation request failed")

	c.JSON(appErr.Status, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}
