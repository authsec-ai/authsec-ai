package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/dto"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/services"
)

// AttestationController handles attestation requests
type AttestationController struct {
	service *services.AttestationService
	logger  *logrus.Entry
}

// NewAttestationController creates a new attestation controller
func NewAttestationController(service *services.AttestationService, logger *logrus.Entry) *AttestationController {
	return &AttestationController{
		service: service,
		logger:  logger,
	}
}

// Attest handles POST /spire/v1/attest
func (ctrl *AttestationController) Attest(c *gin.Context) {
	var req dto.AttestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	// Validate request
	if req.TenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
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

	// Call service
	serviceReq := &services.AttestRequest{
		TenantID:        req.TenantID,
		CSR:             req.CSR,
		AttestationType: req.AttestationType,
		Selectors:       req.Selectors,
		VaultMount:      req.VaultMount,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
	}

	resp, err := ctrl.service.Attest(c.Request.Context(), serviceReq)
	if err != nil {
		ctrl.sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AttestResponse{
		Certificate:  resp.Certificate,
		CAChain:      resp.CAChain,
		SpiffeID:     resp.SpiffeID,
		ExpiresAt:    resp.ExpiresAt,
		WorkloadID:   resp.WorkloadID,
		SerialNumber: resp.SerialNumber,
	})
}

// sendError sends an error response
func (ctrl *AttestationController) sendError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewInternalError("Internal server error", err)
	}

	ctrl.logger.WithFields(logrus.Fields{
		"code":    appErr.Code,
		"message": appErr.Message,
	}).WithError(appErr.Err).Error("Attestation request failed")

	c.JSON(appErr.Status, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}
