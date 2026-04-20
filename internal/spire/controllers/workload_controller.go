package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/dto"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/middleware"
	"github.com/authsec-ai/authsec/internal/spire/services"
)

// WorkloadController handles workload attestation and entry management
type WorkloadController struct {
	attestationService *services.WorkloadAttestationService
	entryService       *services.WorkloadEntryService
	logger             *logrus.Entry
}

// NewWorkloadController creates a new workload controller
func NewWorkloadController(
	attestationService *services.WorkloadAttestationService,
	entryService *services.WorkloadEntryService,
	logger *logrus.Entry,
) *WorkloadController {
	return &WorkloadController{
		attestationService: attestationService,
		entryService:       entryService,
		logger:             logger,
	}
}

// --- Workload Attestation ---

// AttestWorkload handles POST /spire/v1/workload/attest
func (ctrl *WorkloadController) AttestWorkload(c *gin.Context) {
	var req struct {
		TenantID  string            `json:"tenant_id"`
		AgentID   string            `json:"agent_id"`
		Selectors map[string]string `json:"selectors"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	if req.TenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}
	if req.AgentID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("agent_id is required", nil))
		return
	}
	if len(req.Selectors) == 0 {
		ctrl.sendError(c, errors.NewBadRequestError("selectors are required", nil))
		return
	}

	// If caller authenticated via mTLS, verify agent identity matches the request
	callerSpiffeID, _ := middleware.GetSpireSpiffeID(c)
	if callerSpiffeID != "" && callerSpiffeID != req.AgentID {
		ctrl.sendError(c, errors.NewForbiddenError("Agent identity mismatch: authenticated SPIFFE ID does not match agent_id in request", nil))
		return
	}

	ctrl.logger.WithFields(logrus.Fields{
		"tenant_id":      req.TenantID,
		"agent_id":       req.AgentID,
		"selector_count": len(req.Selectors),
	}).Info("Workload attestation request received")

	svcReq := &services.AttestWorkloadRequest{
		TenantID:  req.TenantID,
		AgentID:   req.AgentID,
		Selectors: req.Selectors,
	}

	svidResp, err := ctrl.attestationService.AttestWorkload(c.Request.Context(), svcReq)
	if err != nil {
		ctrl.logger.WithError(err).WithField("tenant_id", req.TenantID).Error("Workload attestation failed")
		ctrl.sendError(c, errors.NewBadRequestError("Attestation failed: "+err.Error(), err))
		return
	}

	ctrl.logger.WithFields(logrus.Fields{
		"spiffe_id": svidResp.SpiffeID,
		"tenant_id": req.TenantID,
	}).Info("Workload SVID issued successfully")

	c.JSON(http.StatusOK, gin.H{
		"spiffe_id":    svidResp.SpiffeID,
		"certificate":  svidResp.Certificate,
		"trust_bundle": svidResp.TrustBundle,
		"expires_at":   svidResp.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		"ttl":          svidResp.TTL,
	})
}

// RevokeWorkloadSVID handles POST /spire/v1/workload/revoke
func (ctrl *WorkloadController) RevokeWorkloadSVID(c *gin.Context) {
	var req struct {
		TenantID     string `json:"tenant_id"`
		SerialNumber string `json:"serial_number"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	if req.TenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}
	if req.SerialNumber == "" {
		ctrl.sendError(c, errors.NewBadRequestError("serial_number is required", nil))
		return
	}

	ctrl.logger.WithFields(logrus.Fields{
		"tenant_id":     req.TenantID,
		"serial_number": req.SerialNumber,
	}).Info("Workload SVID revocation request received")

	if err := ctrl.attestationService.RevokeWorkloadSVID(c.Request.Context(), req.TenantID, req.SerialNumber); err != nil {
		ctrl.logger.WithError(err).WithField("serial_number", req.SerialNumber).Error("Workload SVID revocation failed")
		ctrl.sendError(c, errors.NewInternalError("Revocation failed: "+err.Error(), err))
		return
	}

	ctrl.logger.WithField("serial_number", req.SerialNumber).Info("Workload SVID revoked successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":       "Workload SVID revoked successfully",
		"serial_number": req.SerialNumber,
	})
}

// --- Workload Entry Management ---

// CreateEntry handles POST /spire/v1/entries
func (ctrl *WorkloadController) CreateEntry(c *gin.Context) {
	var req dto.CreateWorkloadEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	// Validate tenant ownership
	if err := ctrl.validateTenantOwnership(c, req.TenantID); err != nil {
		ctrl.sendError(c, errors.NewForbiddenError("Tenant ownership validation failed", err))
		return
	}

	// Convert DTO to domain model
	entry := &models.WorkloadEntry{
		TenantID:   req.TenantID,
		SpiffeID:   req.SpiffeID,
		ParentID:   req.ParentID,
		Selectors:  req.Selectors,
		TTL:        req.TTL,
		Admin:      req.Admin,
		Downstream: req.Downstream,
	}

	createdEntry, err := ctrl.entryService.CreateEntry(c.Request.Context(), entry)
	if err != nil {
		ctrl.logger.WithError(err).WithField("spiffe_id", req.SpiffeID).Error("Failed to create workload entry")
		ctrl.sendError(c, errors.NewInternalError("Failed to create workload entry", err))
		return
	}

	c.JSON(http.StatusCreated, ctrl.toEntryResponse(createdEntry))
}

// CreateAgentEntry handles POST /spire/v1/entries/agent
// Generates a SPIFFE ID for an AI agent based on tenant_id, client_id, and agent_type.
func (ctrl *WorkloadController) CreateAgentEntry(c *gin.Context) {
	var req dto.CreateAgentEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	// Validate required fields
	if req.TenantID == "" || req.ClientID == "" || req.AgentType == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id, client_id, and agent_type are required", nil))
		return
	}

	// Validate tenant ownership
	if err := ctrl.validateTenantOwnership(c, req.TenantID); err != nil {
		ctrl.sendError(c, errors.NewForbiddenError("Tenant ownership validation failed", err))
		return
	}

	// Generate SPIFFE ID: spiffe://<tenant_id>/agent/<client_id>/<agent_type>
	spiffeID := "spiffe://" + req.TenantID + "/agent/" + req.ClientID + "/" + req.AgentType

	// Build selectors with authsec-specific selectors
	selectors := map[string]string{
		"authsec:client_id":  req.ClientID,
		"authsec:agent_type": req.AgentType,
		"authsec:tenant_id":  req.TenantID,
	}
	for k, v := range req.Selectors {
		selectors[k] = v
	}

	entry := &models.WorkloadEntry{
		TenantID:  req.TenantID,
		SpiffeID:  spiffeID,
		ParentID:  req.ParentID,
		Selectors: selectors,
		TTL:       req.TTL,
	}

	createdEntry, err := ctrl.entryService.CreateEntry(c.Request.Context(), entry)
	if err != nil {
		ctrl.logger.WithError(err).WithFields(logrus.Fields{
			"spiffe_id": spiffeID,
			"client_id": req.ClientID,
		}).Error("Failed to create agent workload entry")
		ctrl.sendError(c, errors.NewInternalError("Failed to create agent workload entry", err))
		return
	}

	c.JSON(http.StatusCreated, dto.CreateAgentEntryResponse{
		EntryID:   createdEntry.ID,
		SpiffeID:  createdEntry.SpiffeID,
		TenantID:  createdEntry.TenantID,
		ClientID:  req.ClientID,
		ParentID:  createdEntry.ParentID,
		Selectors: createdEntry.Selectors,
		TTL:       createdEntry.TTL,
		CreatedAt: createdEntry.CreatedAt,
	})
}

// GetEntry handles GET /spire/v1/entries/:id
func (ctrl *WorkloadController) GetEntry(c *gin.Context) {
	entryID := c.Param("id")
	tenantID := c.Query("tenant_id")

	if tenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}

	// Validate tenant ownership
	if err := ctrl.validateTenantOwnership(c, tenantID); err != nil {
		ctrl.sendError(c, errors.NewForbiddenError("Tenant ownership validation failed", err))
		return
	}

	entry, err := ctrl.entryService.GetEntry(c.Request.Context(), tenantID, entryID)
	if err != nil {
		ctrl.logger.WithError(err).WithField("id", entryID).Error("Failed to get workload entry")
		ctrl.sendError(c, errors.NewNotFoundError("Workload entry not found", err))
		return
	}

	c.JSON(http.StatusOK, ctrl.toEntryResponse(entry))
}

// ListEntries handles GET /spire/v1/entries
func (ctrl *WorkloadController) ListEntries(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}

	// Validate tenant ownership
	if err := ctrl.validateTenantOwnership(c, tenantID); err != nil {
		ctrl.sendError(c, errors.NewForbiddenError("Tenant ownership validation failed", err))
		return
	}

	parentID := c.Query("parent_id")
	spiffeID := c.Query("spiffe_id")
	spiffeIDSearch := c.Query("spiffe_id_search")
	selectorType := c.Query("selector_type")

	// Determine if using partial SPIFFE ID search
	usePartialSearch := false
	searchValue := spiffeID
	if spiffeIDSearch != "" {
		searchValue = spiffeIDSearch
		usePartialSearch = true
	}

	// Validate selector_type if provided
	if selectorType != "" {
		validTypes := map[string]bool{"unix": true, "kubernetes": true, "docker": true}
		if !validTypes[selectorType] {
			ctrl.sendError(c, errors.NewBadRequestError("selector_type must be one of: unix, kubernetes, docker", nil))
			return
		}
	}

	// Parse pagination parameters
	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Parse admin filter
	var adminFilter *bool
	if adminParam := c.Query("admin"); adminParam != "" {
		if adminParam == "true" {
			trueVal := true
			adminFilter = &trueVal
		} else if adminParam == "false" {
			falseVal := false
			adminFilter = &falseVal
		}
	}

	filter := &models.WorkloadEntryFilter{
		TenantID:        tenantID,
		ParentID:        parentID,
		SpiffeID:        searchValue,
		SpiffeIDPartial: usePartialSearch,
		SelectorType:    selectorType,
		Admin:           adminFilter,
		Limit:           limit,
		Offset:          offset,
	}

	entries, err := ctrl.entryService.ListEntries(c.Request.Context(), filter)
	if err != nil {
		ctrl.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to list workload entries")
		ctrl.sendError(c, errors.NewInternalError("Failed to list workload entries", err))
		return
	}

	// Get total count
	totalCount, err := ctrl.entryService.CountEntries(c.Request.Context(), filter)
	if err != nil {
		ctrl.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to count workload entries")
		totalCount = len(entries)
	}

	var responseEntries []*dto.WorkloadEntryResponse
	for _, entry := range entries {
		responseEntries = append(responseEntries, ctrl.toEntryResponse(entry))
	}

	c.JSON(http.StatusOK, dto.ListWorkloadEntriesResponse{
		Entries: responseEntries,
		Total:   totalCount,
	})
}

// ListEntriesByParent handles GET /spire/v1/entries/by-parent
func (ctrl *WorkloadController) ListEntriesByParent(c *gin.Context) {
	parentID := c.Query("parent_id")
	tenantID := c.Query("tenant_id")

	if tenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}
	if parentID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("parent_id is required", nil))
		return
	}

	// Validate tenant ownership if auth context is available
	callerTenantID, ok := middleware.GetSpireTenantID(c)
	if ok && callerTenantID != "" && callerTenantID != tenantID {
		ctrl.sendError(c, errors.NewForbiddenError("Tenant mismatch", nil))
		return
	}

	entries, err := ctrl.entryService.ListEntriesByParent(c.Request.Context(), tenantID, parentID)
	if err != nil {
		ctrl.logger.WithError(err).WithField("parent_id", parentID).Error("Failed to list workload entries by parent")
		ctrl.sendError(c, errors.NewInternalError("Failed to list workload entries", err))
		return
	}

	var responseEntries []*dto.WorkloadEntryResponse
	for _, entry := range entries {
		responseEntries = append(responseEntries, ctrl.toEntryResponse(entry))
	}

	c.JSON(http.StatusOK, dto.ListWorkloadEntriesResponse{
		Entries: responseEntries,
		Total:   len(responseEntries),
	})
}

// UpdateEntry handles PUT /spire/v1/entries/:id
func (ctrl *WorkloadController) UpdateEntry(c *gin.Context) {
	entryID := c.Param("id")
	tenantID := c.Query("tenant_id")

	if tenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}

	// Validate tenant ownership
	if err := ctrl.validateTenantOwnership(c, tenantID); err != nil {
		ctrl.sendError(c, errors.NewForbiddenError("Tenant ownership validation failed", err))
		return
	}

	var req dto.UpdateWorkloadEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.sendError(c, errors.NewBadRequestError("Invalid request body", err))
		return
	}

	entry := &models.WorkloadEntry{
		ID:         entryID,
		TenantID:   tenantID,
		SpiffeID:   req.SpiffeID,
		ParentID:   req.ParentID,
		Selectors:  req.Selectors,
		TTL:        req.TTL,
		Admin:      req.Admin,
		Downstream: req.Downstream,
	}

	updatedEntry, err := ctrl.entryService.UpdateEntry(c.Request.Context(), entry)
	if err != nil {
		ctrl.logger.WithError(err).WithField("id", entryID).Error("Failed to update workload entry")
		ctrl.sendError(c, errors.NewInternalError("Failed to update workload entry", err))
		return
	}

	c.JSON(http.StatusOK, ctrl.toEntryResponse(updatedEntry))
}

// DeleteEntry handles DELETE /spire/v1/entries/:id
func (ctrl *WorkloadController) DeleteEntry(c *gin.Context) {
	entryID := c.Param("id")
	tenantID := c.Query("tenant_id")

	if tenantID == "" {
		ctrl.sendError(c, errors.NewBadRequestError("tenant_id is required", nil))
		return
	}

	// Validate tenant ownership
	if err := ctrl.validateTenantOwnership(c, tenantID); err != nil {
		ctrl.sendError(c, errors.NewForbiddenError("Tenant ownership validation failed", err))
		return
	}

	if err := ctrl.entryService.DeleteEntry(c.Request.Context(), tenantID, entryID); err != nil {
		ctrl.logger.WithError(err).WithField("id", entryID).Error("Failed to delete workload entry")
		ctrl.sendError(c, errors.NewInternalError("Failed to delete workload entry", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Workload entry deleted successfully",
		"id":      entryID,
	})
}

// --- Helpers ---

// toEntryResponse converts a domain model to response DTO
func (ctrl *WorkloadController) toEntryResponse(entry *models.WorkloadEntry) *dto.WorkloadEntryResponse {
	return &dto.WorkloadEntryResponse{
		ID:         entry.ID,
		TenantID:   entry.TenantID,
		SpiffeID:   entry.SpiffeID,
		ParentID:   entry.ParentID,
		Selectors:  entry.Selectors,
		TTL:        entry.TTL,
		Admin:      entry.Admin,
		Downstream: entry.Downstream,
		CreatedAt:  entry.CreatedAt,
		UpdatedAt:  entry.UpdatedAt,
	}
}

// validateTenantOwnership ensures the tenant_id in the request matches the authenticated caller's tenant
func (ctrl *WorkloadController) validateTenantOwnership(c *gin.Context, requestTenantID string) error {
	callerTenantID, ok := middleware.GetSpireTenantID(c)
	if !ok || callerTenantID == "" {
		return fmt.Errorf("tenant ID not found in authentication context")
	}
	if callerTenantID != requestTenantID {
		return fmt.Errorf("tenant mismatch: authenticated as %s but requesting %s", callerTenantID, requestTenantID)
	}
	return nil
}

// sendError sends an error response
func (ctrl *WorkloadController) sendError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewInternalError("Internal server error", err)
	}

	ctrl.logger.WithFields(logrus.Fields{
		"code":    appErr.Code,
		"message": appErr.Message,
	}).WithError(appErr.Err).Error("Workload request failed")

	c.JSON(appErr.Status, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}
