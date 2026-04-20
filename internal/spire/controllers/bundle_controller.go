package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/dto"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/services"
)

// BundleController handles CA bundle requests
type BundleController struct {
	service *services.BundleService
	logger  *logrus.Entry
}

// NewBundleController creates a new bundle controller
func NewBundleController(service *services.BundleService, logger *logrus.Entry) *BundleController {
	return &BundleController{
		service: service,
		logger:  logger,
	}
}

// GetBundle handles GET /spire/v1/bundle/:tenant
func (ctrl *BundleController) GetBundle(c *gin.Context) {
	tenantID := c.Param("tenant")

	if tenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("Tenant ID is required", nil))
		return
	}

	bundle, err := ctrl.service.GetBundle(c.Request.Context(), tenantID)
	if err != nil {
		ctrl.sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.BundleResponse{
		CABundle: bundle,
		TenantID: tenantID,
	})
}

// sendError sends an error response
func (ctrl *BundleController) sendError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewInternalError("Internal server error", err)
	}

	ctrl.logger.WithFields(logrus.Fields{
		"code":    appErr.Code,
		"message": appErr.Message,
	}).WithError(appErr.Err).Error("Bundle request failed")

	c.JSON(appErr.Status, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}
