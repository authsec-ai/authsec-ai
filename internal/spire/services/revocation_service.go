package services

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/database"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/vault"
	infraRepos "github.com/authsec-ai/authsec/internal/spire/infrastructure/repositories"
)

// RevocationService handles certificate revocation
type RevocationService struct {
	certRepo    repositories.CertificateRepository
	auditRepo   repositories.AuditRepository
	tenantRepo  repositories.TenantRepository
	vaultClient *vault.Client
	connManager *database.ConnectionManager // For tenant-specific DB connections
	logger      *logrus.Entry
}

// NewRevocationService creates a new revocation service
func NewRevocationService(
	certRepo repositories.CertificateRepository,
	auditRepo repositories.AuditRepository,
	tenantRepo repositories.TenantRepository,
	vaultClient *vault.Client,
	connManager *database.ConnectionManager,
	logger *logrus.Entry,
) *RevocationService {
	return &RevocationService{
		certRepo:    certRepo,
		auditRepo:   auditRepo,
		tenantRepo:  tenantRepo,
		vaultClient: vaultClient,
		connManager: connManager,
		logger:      logger,
	}
}

// RevokeRequest represents a revocation request
type RevokeRequest struct {
	TenantID     string
	SerialNumber string
	Reason       string
	IPAddress    string
	UserAgent    string
}

// Revoke revokes a certificate
func (s *RevocationService) Revoke(ctx context.Context, req *RevokeRequest) error {
	s.logger.WithFields(logrus.Fields{
		"tenant_id":     req.TenantID,
		"serial_number": req.SerialNumber,
	}).Info("Starting certificate revocation")

	// Validate tenant
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		s.auditRevocation(ctx, req, false, err.Error())
		return err
	}

	if !tenant.IsActive() {
		s.auditRevocation(ctx, req, false, "tenant not active")
		return errors.NewForbiddenError("Tenant is not active", nil)
	}

	// Get tenant-specific repositories
	certRepo, _, err := s.getTenantRepositories(ctx, req.TenantID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tenant repositories")
		return errors.NewInternalError("Failed to connect to tenant database", err)
	}

	// Get certificate
	cert, err := certRepo.GetBySerialNumber(ctx, req.TenantID, req.SerialNumber)
	if err != nil {
		s.auditRevocation(ctx, req, false, "certificate not found")
		return errors.NewNotFoundError("Certificate not found", err)
	}

	// Check if already revoked
	if cert.RevokedAt != nil {
		s.auditRevocation(ctx, req, false, "certificate already revoked")
		return errors.NewConflictError("Certificate already revoked", nil)
	}

	// Revoke in Vault
	if err := s.vaultClient.RevokeCertificate(ctx, tenant.VaultMount, req.SerialNumber); err != nil {
		s.auditRevocation(ctx, req, false, "vault revocation failed")
		return err
	}

	// Update certificate status
	if err := certRepo.Revoke(ctx, req.TenantID, cert.ID); err != nil {
		s.logger.WithField("serial_number", req.SerialNumber).WithError(err).Error("Failed to update certificate status")
		// Don't fail - certificate is already revoked in Vault
	}

	// Audit success
	s.auditRevocation(ctx, req, true, "")

	s.logger.WithField("serial_number", req.SerialNumber).Info("Certificate revoked successfully")

	return nil
}

// getTenantRepositories creates repositories connected to the tenant's database
func (s *RevocationService) getTenantRepositories(ctx context.Context, tenantID string) (
	repositories.CertificateRepository,
	repositories.AuditRepository,
	error,
) {
	// Get tenant database connection
	tenantDB, err := s.connManager.GetTenantDB(ctx, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Create tenant-specific repositories
	certRepo := infraRepos.NewPostgresCertificateRepository(tenantDB)
	auditRepo := infraRepos.NewPostgresAuditRepository(tenantDB)

	return certRepo, auditRepo, nil
}

// auditRevocation creates an audit log entry for revocation
func (s *RevocationService) auditRevocation(ctx context.Context, req *RevokeRequest, success bool, errorMsg string) {
	audit := &models.AuditLog{
		TenantID:     req.TenantID,
		EventType:    models.EventRevoke,
		Success:      success,
		ErrorMessage: errorMsg,
		Metadata: map[string]interface{}{
			"serial_number": req.SerialNumber,
			"reason":        req.Reason,
		},
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	}

	if err := s.auditRepo.Create(ctx, audit); err != nil {
		s.logger.WithError(err).Error("Failed to create audit log")
	}
}
