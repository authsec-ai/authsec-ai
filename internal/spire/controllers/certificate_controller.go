package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/dto"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/services"
)

// CertificateController handles certificate renewal and revocation
type CertificateController struct {
	renewalService    *services.RenewalService
	revocationService *services.RevocationService
	logger            *logrus.Entry
}

// NewCertificateController creates a new certificate controller
func NewCertificateController(
	renewalService *services.RenewalService,
	revocationService *services.RevocationService,
	logger *logrus.Entry,
) *CertificateController {
	return &CertificateController{
		renewalService:    renewalService,
		revocationService: revocationService,
		logger:            logger,
	}
}

// Renew handles POST /spire/v1/renew
func (ctrl *CertificateController) Renew(c *gin.Context) {
	var req dto.RenewRequest
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
	if req.WorkloadID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("workload_id is required", nil))
		return
	}

	serviceReq := &services.RenewRequest{
		TenantID:       req.TenantID,
		WorkloadID:     req.WorkloadID,
		CSR:            req.CSR,
		OldCertificate: req.OldCertificate,
		IPAddress:      c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
	}

	resp, err := ctrl.renewalService.Renew(c.Request.Context(), serviceReq)
	if err != nil {
		ctrl.sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.RenewResponse{
		Certificate:  resp.Certificate,
		CAChain:      resp.CAChain,
		ExpiresAt:    resp.ExpiresAt,
		SerialNumber: resp.SerialNumber,
	})
}

// Revoke handles POST /spire/v1/revoke
func (ctrl *CertificateController) Revoke(c *gin.Context) {
	var req dto.RevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	// Validate request
	if req.TenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}
	if req.SerialNumber == "" {
		ctrl.sendError(c, errors.NewBadRequestError("serial_number is required", nil))
		return
	}

	serviceReq := &services.RevokeRequest{
		TenantID:     req.TenantID,
		SerialNumber: req.SerialNumber,
		Reason:       req.Reason,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	}

	if err := ctrl.revocationService.Revoke(c.Request.Context(), serviceReq); err != nil {
		ctrl.sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Certificate revoked successfully",
		"serial_number": req.SerialNumber,
	})
}

// sendError sends an error response
func (ctrl *CertificateController) sendError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewInternalError("Internal server error", err)
	}

	ctrl.logger.WithFields(logrus.Fields{
		"code":    appErr.Code,
		"message": appErr.Message,
	}).WithError(appErr.Err).Error("Certificate request failed")

	c.JSON(appErr.Status, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}
