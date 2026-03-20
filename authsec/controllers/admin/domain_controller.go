package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DomainController struct {
	service *services.DomainService
	db      *database.DBConnection
}

func NewDomainController(db *database.DBConnection) *DomainController {
	repo := database.NewTenantDomainsRepository(db)
	svc := services.NewDomainService(repo)
	return &DomainController{
		service: svc,
		db:      db,
	}
}

type CreateDomainRequest struct {
	Domain    string `json:"domain" binding:"required"`
	IsPrimary bool   `json:"is_primary"`
}

type DomainResponse struct {
	ID                    string `json:"id"`
	TenantID              string `json:"tenant_id"`
	Domain                string `json:"domain"`
	Kind                  string `json:"kind"`
	IsPrimary             bool   `json:"is_primary"`
	IsVerified            bool   `json:"is_verified"`
	VerificationMethod    string `json:"verification_method"`
	VerificationToken     string `json:"verification_token"`
	VerificationTXTName   string `json:"verification_txt_name,omitempty"`
	VerificationTXTValue  string `json:"verification_txt_value,omitempty"`
	VerifiedAt            string `json:"verified_at,omitempty"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

func getTenantIDFromRequest(c *gin.Context) (uuid.UUID, error) {
	// Get tenant_id from context (set by ValidateTenantFromPath middleware)
	tenantIDAny, ok := c.Get("tenant_id")
	if !ok {
		return uuid.Nil, fmt.Errorf("tenant_id not found in request context")
	}

	tenantIDStr, ok := tenantIDAny.(string)
	if !ok || strings.TrimSpace(tenantIDStr) == "" {
		return uuid.Nil, fmt.Errorf("tenant_id not found in request context")
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid tenant_id format: %w", err)
	}

	return tenantID, nil
}

func getUserIDFromRequest(c *gin.Context) *uuid.UUID {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	if userIDStr, ok := userIDStr.(string); ok {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return &id
		}
	}
	return nil
}

func domainToResponse(td *database.TenantDomain) DomainResponse {
	resp := DomainResponse{
		ID:                 td.ID.String(),
		TenantID:           td.TenantID.String(),
		Domain:             td.Domain,
		Kind:               td.Kind,
		IsPrimary:          td.IsPrimary,
		IsVerified:         td.IsVerified,
		VerificationMethod: td.VerificationMethod,
		VerificationToken:  td.VerificationToken,
		CreatedAt:          td.CreatedAt.String(),
		UpdatedAt:          td.UpdatedAt.String(),
	}

	if td.VerificationTXTName != nil {
		resp.VerificationTXTName = *td.VerificationTXTName
	}
	if td.VerificationTXTValue != nil {
		resp.VerificationTXTValue = *td.VerificationTXTValue
	}
	if td.VerifiedAt != nil {
		resp.VerifiedAt = td.VerifiedAt.String()
	}

	return resp
}

// ListDomains retrieves all domains for the authenticated tenant
func (dc *DomainController) ListDomains(c *gin.Context) {
	tenantID, err := getTenantIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	domains, err := dc.service.ListTenantDomains(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list domains", "details": err.Error()})
		return
	}

	responses := make([]DomainResponse, len(domains))
	for i, d := range domains {
		responses[i] = domainToResponse(&d)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"domains": responses,
		"count":   len(responses),
	})
}

// GetDomainByID retrieves a specific domain
func (dc *DomainController) GetDomainByID(c *gin.Context) {
	domainIDStr := strings.TrimSpace(c.Param("domain_id"))
	if domainIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain_id is required"})
		return
	}

	domainID, err := uuid.Parse(domainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain_id format"})
		return
	}

	tenantID, err := getTenantIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	repo := database.NewTenantDomainsRepository(dc.db)
	td, err := repo.GetDomainByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	if td.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to access this domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"domain":  domainToResponse(td),
	})
}

// CreateDomain registers a new custom domain (pending verification)
func (dc *DomainController) CreateDomain(c *gin.Context) {
	tenantID, err := getTenantIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdBy := getUserIDFromRequest(c)

	td, err := dc.service.RegisterDomain(tenantID, req.Domain, createdBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"domain":  domainToResponse(td),
		"verification": gin.H{
			"method":    "dns_txt",
			"txt_name":  *td.VerificationTXTName,
			"txt_value": *td.VerificationTXTValue,
			"token":     td.VerificationToken,
		},
	})
}

// VerifyDomain performs DNS verification for a domain
func (dc *DomainController) VerifyDomain(c *gin.Context) {
	domainIDStr := strings.TrimSpace(c.Param("domain_id"))
	if domainIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain_id is required"})
		return
	}

	domainID, err := uuid.Parse(domainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain_id format"})
		return
	}

	// Get tenant_id from context (set by ValidateTenantFromPath middleware)
	tenantIDAny, ok := c.Get("tenant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in request"})
		return
	}

	tenantIDStr, ok := tenantIDAny.(string)
	if !ok || strings.TrimSpace(tenantIDStr) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in request"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	repo := database.NewTenantDomainsRepository(dc.db)
	td, err := repo.GetDomainByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if td.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to verify this domain"})
		return
	}

	err = dc.service.VerifyDomainOwnership(domainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification failed", "details": err.Error()})
		return
	}

	td, _ = repo.GetDomainByID(domainID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "domain verified successfully",
		"domain":  domainToResponse(td),
	})
}

// SetPrimaryDomain sets a domain as the primary for a tenant
func (dc *DomainController) SetPrimaryDomain(c *gin.Context) {
	domainIDStr := strings.TrimSpace(c.Param("domain_id"))
	if domainIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain_id is required"})
		return
	}

	domainID, err := uuid.Parse(domainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain_id format"})
		return
	}

	tenantID, err := getTenantIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	repo := database.NewTenantDomainsRepository(dc.db)
	td, err := repo.GetDomainByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if td.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to modify this domain"})
		return
	}
	if !td.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain must be verified before setting as primary"})
		return
	}

	updatedBy := getUserIDFromRequest(c)

	err = dc.service.SetPrimaryDomain(tenantID, domainID, updatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set primary domain", "details": err.Error()})
		return
	}

	td, _ = repo.GetDomainByID(domainID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "domain set as primary",
		"domain":  domainToResponse(td),
	})
}

// DeleteDomain removes a domain
func (dc *DomainController) DeleteDomain(c *gin.Context) {
	domainIDStr := strings.TrimSpace(c.Param("domain_id"))
	if domainIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain_id is required"})
		return
	}

	domainID, err := uuid.Parse(domainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain_id format"})
		return
	}

	tenantID, err := getTenantIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	repo := database.NewTenantDomainsRepository(dc.db)
	td, err := repo.GetDomainByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if td.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to delete this domain"})
		return
	}

	err = dc.service.DeleteDomain(domainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete domain", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "domain deleted successfully",
	})
}
