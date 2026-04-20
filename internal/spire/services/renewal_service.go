package services

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/database"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/vault"
	infraRepos "github.com/authsec-ai/authsec/internal/spire/infrastructure/repositories"
)

// RenewalService handles certificate renewal
type RenewalService struct {
	workloadRepo repositories.WorkloadRepository
	certRepo     repositories.CertificateRepository
	auditRepo    repositories.AuditRepository
	tenantRepo   repositories.TenantRepository
	vaultClient  *vault.Client
	connManager  *database.ConnectionManager // For tenant-specific DB connections
	logger       *logrus.Entry
}

// NewRenewalService creates a new renewal service
func NewRenewalService(
	workloadRepo repositories.WorkloadRepository,
	certRepo repositories.CertificateRepository,
	auditRepo repositories.AuditRepository,
	tenantRepo repositories.TenantRepository,
	vaultClient *vault.Client,
	connManager *database.ConnectionManager,
	logger *logrus.Entry,
) *RenewalService {
	return &RenewalService{
		workloadRepo: workloadRepo,
		certRepo:     certRepo,
		auditRepo:    auditRepo,
		tenantRepo:   tenantRepo,
		vaultClient:  vaultClient,
		connManager:  connManager,
		logger:       logger,
	}
}

// RenewRequest represents a renewal request
type RenewRequest struct {
	TenantID       string
	WorkloadID     string
	CSR            string
	OldCertificate string // For validation
	IPAddress      string
	UserAgent      string
}

// RenewResponse represents a renewal response
type RenewResponse struct {
	Certificate  string
	CAChain      []string
	ExpiresAt    time.Time
	SerialNumber string
}

// Renew renews a certificate
func (s *RenewalService) Renew(ctx context.Context, req *RenewRequest) (*RenewResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"tenant_id":   req.TenantID,
		"workload_id": req.WorkloadID,
	}).Info("Starting certificate renewal")

	// Validate tenant
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		s.auditRenewal(ctx, req, false, err.Error())
		return nil, err
	}

	if !tenant.IsActive() {
		s.auditRenewal(ctx, req, false, "tenant not active")
		return nil, errors.NewForbiddenError("Tenant is not active", nil)
	}

	// Get tenant-specific repositories
	workloadRepo, certRepo, _, err := s.getTenantRepositories(ctx, req.TenantID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tenant repositories")
		return nil, errors.NewInternalError("Failed to connect to tenant database", err)
	}

	// Get workload
	workload, err := workloadRepo.GetByID(ctx, req.TenantID, req.WorkloadID)
	if err != nil {
		s.auditRenewal(ctx, req, false, "workload not found")
		return nil, errors.NewNotFoundError("Workload not found", err)
	}

	if workload.Status != "active" {
		s.auditRenewal(ctx, req, false, "workload not active")
		return nil, errors.NewForbiddenError("Workload is not active", nil)
	}

	// Validate old certificate if provided
	if req.OldCertificate != "" {
		if err := s.validateOldCertificate(ctx, req.TenantID, req.WorkloadID, req.OldCertificate); err != nil {
			s.auditRenewal(ctx, req, false, "old certificate validation failed")
			return nil, err
		}
	}

	// Parse CSR
	csrBlock, _ := pem.Decode([]byte(req.CSR))
	if csrBlock == nil {
		s.auditRenewal(ctx, req, false, "invalid CSR format")
		return nil, errors.NewBadRequestError("Invalid CSR format", nil)
	}

	csrParsed, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		s.auditRenewal(ctx, req, false, "failed to parse CSR")
		return nil, errors.NewBadRequestError("Failed to parse CSR", err)
	}

	// Validate CSR signature
	if err := csrParsed.CheckSignature(); err != nil {
		s.auditRenewal(ctx, req, false, "invalid CSR signature")
		return nil, errors.NewBadRequestError("Invalid CSR signature", err)
	}

	// Issue new certificate from Vault
	vaultReq := &vault.CertificateRequest{
		CSR:        req.CSR,
		CommonName: workload.SpiffeID,
		TTL:        "24h", // Default TTL - should come from policy
		URISANs:    []string{workload.SpiffeID},
	}

	// Remove redundant "pki/" prefix if present
	cleanVaultMount := strings.TrimPrefix(tenant.VaultMount, "pki/")
	vaultResp, err := s.vaultClient.IssueCertificate(ctx, cleanVaultMount, workload.VaultRole, vaultReq)
	if err != nil {
		s.auditRenewal(ctx, req, false, "vault issuance failed")
		return nil, err
	}

	// Store new certificate
	cert := &models.Certificate{
		ID:           uuid.New().String(),
		TenantID:     req.TenantID,
		WorkloadID:   workload.ID,
		SerialNumber: vaultResp.SerialNumber,
		SpiffeID:     workload.SpiffeID,
		CertPEM:      vaultResp.Certificate,
		CAChain:      vaultResp.CAChain,
		IssuedAt:     time.Now(),
		ExpiresAt:    vaultResp.ExpirationTime,
		Status:       "active",
		IssueType:    "renew",
	}

	if err := certRepo.Create(ctx, cert); err != nil {
		s.logger.WithField("serial_number", vaultResp.SerialNumber).WithError(err).Error("Failed to store certificate")
	}

	// Audit success
	s.auditRenewal(ctx, req, true, "")

	return &RenewResponse{
		Certificate:  vaultResp.Certificate,
		CAChain:      vaultResp.CAChain,
		ExpiresAt:    vaultResp.ExpirationTime,
		SerialNumber: vaultResp.SerialNumber,
	}, nil
}

// getTenantRepositories creates repositories connected to the tenant's database
func (s *RenewalService) getTenantRepositories(ctx context.Context, tenantID string) (
	repositories.WorkloadRepository,
	repositories.CertificateRepository,
	repositories.AuditRepository,
	error,
) {
	// Get tenant database connection
	tenantDB, err := s.connManager.GetTenantDB(ctx, tenantID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Create tenant-specific repositories
	workloadRepo := infraRepos.NewPostgresWorkloadRepository(tenantDB)
	certRepo := infraRepos.NewPostgresCertificateRepository(tenantDB)
	auditRepo := infraRepos.NewPostgresAuditRepository(tenantDB)

	return workloadRepo, certRepo, auditRepo, nil
}

// validateOldCertificate validates the old certificate
func (s *RenewalService) validateOldCertificate(ctx context.Context, tenantID, workloadID, certPEM string) error {
	// Parse certificate
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return errors.NewBadRequestError("Invalid certificate format", nil)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return errors.NewBadRequestError("Failed to parse certificate", err)
	}

	// Get active certificate for workload
	activeCert, err := s.certRepo.GetActiveByWorkload(ctx, tenantID, workloadID)
	if err != nil {
		return errors.NewForbiddenError("No active certificate found for workload", err)
	}

	// Verify serial numbers match
	if cert.SerialNumber.String() != activeCert.SerialNumber {
		return errors.NewForbiddenError("Certificate serial number mismatch", nil)
	}

	// Check if certificate is expiring (within renewal window)
	renewalThreshold := 24 * time.Hour // Allow renewal within 24 hours of expiry
	if !activeCert.IsExpiringSoon(renewalThreshold) {
		s.logger.WithFields(logrus.Fields{
			"serial_number": activeCert.SerialNumber,
			"expires_at":    activeCert.ExpiresAt,
		}).Warn("Certificate renewal requested but not yet in renewal window")
	}

	return nil
}

// auditRenewal creates an audit log entry for renewal
func (s *RenewalService) auditRenewal(ctx context.Context, req *RenewRequest, success bool, errorMsg string) {
	audit := &models.AuditLog{
		TenantID:     req.TenantID,
		EventType:    models.EventRenew,
		WorkloadID:   req.WorkloadID,
		Success:      success,
		ErrorMessage: errorMsg,
		Metadata: map[string]interface{}{
			"workload_id": req.WorkloadID,
		},
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	}

	if err := s.auditRepo.Create(ctx, audit); err != nil {
		s.logger.WithError(err).Error("Failed to create audit log")
	}
}
