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
	infraRepos "github.com/authsec-ai/authsec/internal/spire/infrastructure/repositories"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/vault"
)

// AttestationService handles workload attestation and certificate issuance
type AttestationService struct {
	workloadRepo repositories.WorkloadRepository
	certRepo     repositories.CertificateRepository
	policyRepo   repositories.PolicyRepository
	auditRepo    repositories.AuditRepository
	tenantRepo   repositories.TenantRepository
	vaultClient  *vault.Client
	connManager  *database.ConnectionManager // For tenant-specific DB connections
	logger       *logrus.Entry
}

// NewAttestationService creates a new attestation service
func NewAttestationService(
	workloadRepo repositories.WorkloadRepository,
	certRepo repositories.CertificateRepository,
	policyRepo repositories.PolicyRepository,
	auditRepo repositories.AuditRepository,
	tenantRepo repositories.TenantRepository,
	vaultClient *vault.Client,
	connManager *database.ConnectionManager,
	logger *logrus.Entry,
) *AttestationService {
	return &AttestationService{
		workloadRepo: workloadRepo,
		certRepo:     certRepo,
		policyRepo:   policyRepo,
		auditRepo:    auditRepo,
		tenantRepo:   tenantRepo,
		vaultClient:  vaultClient,
		connManager:  connManager,
		logger:       logger,
	}
}

// getTenantRepositories creates repositories connected to the tenant's database
func (s *AttestationService) getTenantRepositories(ctx context.Context, tenantID string) (
	repositories.WorkloadRepository,
	repositories.CertificateRepository,
	repositories.PolicyRepository,
	repositories.AuditRepository,
	error,
) {
	// Query tenants table in master DB to get tenant database connection info
	tenantDB, err := s.connManager.GetTenantDB(ctx, tenantID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Create tenant-specific repositories
	workloadRepo := infraRepos.NewPostgresWorkloadRepository(tenantDB)
	certRepo := infraRepos.NewPostgresCertificateRepository(tenantDB)
	policyRepo := infraRepos.NewPostgresPolicyRepository(tenantDB)
	auditRepo := infraRepos.NewPostgresAuditRepository(tenantDB)

	return workloadRepo, certRepo, policyRepo, auditRepo, nil
}

// AttestRequest represents an attestation request
type AttestRequest struct {
	TenantID        string
	CSR             string
	AttestationType string
	Selectors       map[string]string
	VaultMount      string // Optional: PKI mount path (e.g., "pki/auth.authsec.ai")
	IPAddress       string
	UserAgent       string
}

// AttestResponse represents an attestation response
type AttestResponse struct {
	Certificate  string
	CAChain      []string
	SpiffeID     string
	ExpiresAt    time.Time
	WorkloadID   string
	SerialNumber string
}

// Attest performs workload attestation and issues a certificate
func (s *AttestationService) Attest(ctx context.Context, req *AttestRequest) (*AttestResponse, error) {
	s.logger.WithFields(logrus.Fields{"tenant_id": req.TenantID, "attestation_type": req.AttestationType}).Info("Starting attestation")

	// Get tenant-specific repositories (connected to tenant's database)
	workloadRepo, certRepo, policyRepo, auditRepo, err := s.getTenantRepositories(ctx, req.TenantID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tenant repositories")
		return nil, errors.NewInternalError("Failed to connect to tenant database", err)
	}

	// Try to get tenant from local database (optional - tenant may only exist in user-flow DB)
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	var vaultMount string

	if err != nil {
		// Tenant doesn't exist in ICP database - this is OK
		s.logger.WithField("tenant_id", req.TenantID).Info("Tenant not found in ICP database, using request parameters")

		// Use provided vault_mount or generate default
		if req.VaultMount != "" {
			vaultMount = req.VaultMount
		} else {
			// Generate default vault mount path
			vaultMount = fmt.Sprintf("pki/%s", req.TenantID)
		}
	} else {
		// Tenant exists - validate and use stored values
		if !tenant.IsActive() {
			s.auditAttestation(ctx, req, "", false, "tenant not active")
			return nil, errors.NewForbiddenError("Tenant is not active", nil)
		}
		vaultMount = tenant.VaultMount
	}

	// Parse CSR
	csrBlock, _ := pem.Decode([]byte(req.CSR))
	if csrBlock == nil {
		s.auditAttestation(ctx, req, "", false, "invalid CSR format")
		return nil, errors.NewBadRequestError("Invalid CSR format", nil)
	}

	csrParsed, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		s.auditAttestation(ctx, req, "", false, "failed to parse CSR")
		return nil, errors.NewBadRequestError("Failed to parse CSR", err)
	}

	// Validate CSR signature
	if err := csrParsed.CheckSignature(); err != nil {
		s.auditAttestation(ctx, req, "", false, "invalid CSR signature")
		return nil, errors.NewBadRequestError("Invalid CSR signature", err)
	}

	// Find matching policy (optional - use defaults if not found)
	policy, err := policyRepo.FindMatchingPolicy(ctx, req.TenantID, req.AttestationType, req.Selectors)
	var vaultRole string
	var ttl int

	if err != nil {
		// No policy found - use defaults
		s.logger.WithField("tenant_id", req.TenantID).Info("No matching policy found, using defaults")
		vaultRole = "workload" // Default role created during PKI provisioning
		ttl = 86400            // 24 hours default
	} else {
		vaultRole = policy.VaultRole
		ttl = policy.TTL
	}

	// Generate SPIFFE ID
	spiffeID := s.generateSpiffeID(req.TenantID, req.Selectors)

	// Create or get workload (using tenant-specific repo)
	workload, err := s.getOrCreateWorkloadInRepo(ctx, workloadRepo, req.TenantID, spiffeID, req.Selectors, vaultRole, req.AttestationType)
	if err != nil {
		s.auditAttestationWithRepo(ctx, auditRepo, req, spiffeID, false, "failed to create workload")
		return nil, err
	}

	// Issue certificate from Vault
	vaultReq := &vault.CertificateRequest{
		CSR:        req.CSR,
		CommonName: spiffeID,
		TTL:        fmt.Sprintf("%ds", ttl),
		URISANs:    []string{spiffeID},
	}

	// Fix: Remove redundant "pki/" prefix if present
	cleanVaultMount := strings.TrimPrefix(vaultMount, "pki/")
	s.logger.WithFields(logrus.Fields{"original_mount": vaultMount, "cleaned_mount": cleanVaultMount}).Debug("Using cleaned Vault mount path")
	vaultResp, err := s.vaultClient.IssueCertificate(ctx, cleanVaultMount, vaultRole, vaultReq)
	if err != nil {

		s.auditAttestationWithRepo(ctx, auditRepo, req, spiffeID, false, "vault issuance failed")
		return nil, err
	}

	// Store certificate
	cert := &models.Certificate{
		ID:           uuid.New().String(),
		TenantID:     req.TenantID,
		WorkloadID:   workload.ID,
		SerialNumber: vaultResp.SerialNumber,
		SpiffeID:     spiffeID,
		CertPEM:      vaultResp.Certificate,
		CAChain:      vaultResp.CAChain,
		IssuedAt:     time.Now(),
		ExpiresAt:    vaultResp.ExpirationTime,
		Status:       "active",
		IssueType:    "attest",
	}

	if err := certRepo.Create(ctx, cert); err != nil {
		s.logger.WithField("serial_number", vaultResp.SerialNumber).WithError(err).Error("Failed to store certificate")
		// Don't fail the request, certificate is already issued
	}

	// Audit success
	s.auditAttestationWithRepo(ctx, auditRepo, req, spiffeID, true, "")

	return &AttestResponse{
		Certificate:  vaultResp.Certificate,
		CAChain:      vaultResp.CAChain,
		SpiffeID:     spiffeID,
		ExpiresAt:    vaultResp.ExpirationTime,
		WorkloadID:   workload.ID,
		SerialNumber: vaultResp.SerialNumber,
	}, nil
}

// generateSpiffeID generates a SPIFFE ID based on tenant and selectors
func (s *AttestationService) generateSpiffeID(tenantID string, selectors map[string]string) string {
	// Simple implementation - customize based on your requirements
	base := fmt.Sprintf("spiffe://%s", tenantID)

	if ns, ok := selectors["k8s:namespace"]; ok {
		base += "/ns/" + ns
	}
	if sa, ok := selectors["k8s:service-account"]; ok {
		base += "/sa/" + sa
	}

	return base
}

// getOrCreateWorkload retrieves or creates a workload
func (s *AttestationService) getOrCreateWorkload(
	ctx context.Context,
	tenantID, spiffeID string,
	selectors map[string]string,
	vaultRole, attestationType string,
) (*models.Workload, error) {
	// Try to find existing workload
	workload, err := s.workloadRepo.GetBySpiffeID(ctx, tenantID, spiffeID)
	if err == nil {
		return workload, nil
	}

	// Create new workload
	workload = &models.Workload{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		SpiffeID:        spiffeID,
		Selectors:       selectors,
		VaultRole:       vaultRole,
		Status:          "active",
		AttestationType: attestationType,
	}

	if err := s.workloadRepo.Create(ctx, workload); err != nil {
		return nil, errors.NewInternalError("Failed to create workload", err)
	}

	s.logger.WithFields(logrus.Fields{"workload_id": workload.ID, "spiffe_id": spiffeID}).Info("Created new workload")

	return workload, nil
}

// getOrCreateWorkloadInRepo retrieves or creates a workload using a specific repository
func (s *AttestationService) getOrCreateWorkloadInRepo(
	ctx context.Context,
	workloadRepo repositories.WorkloadRepository,
	tenantID, spiffeID string,
	selectors map[string]string,
	vaultRole, attestationType string,
) (*models.Workload, error) {
	// Try to find existing workload
	workload, err := workloadRepo.GetBySpiffeID(ctx, tenantID, spiffeID)
	if err == nil {
		return workload, nil
	}

	// Create new workload
	workload = &models.Workload{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		SpiffeID:        spiffeID,
		Selectors:       selectors,
		VaultRole:       vaultRole,
		Status:          "active",
		AttestationType: attestationType,
	}

	if err := workloadRepo.Create(ctx, workload); err != nil {
		return nil, errors.NewInternalError("Failed to create workload", err)
	}

	s.logger.WithFields(logrus.Fields{"workload_id": workload.ID, "spiffe_id": spiffeID}).Info("Created new workload")

	return workload, nil
}

// auditAttestation creates an audit log entry for attestation
func (s *AttestationService) auditAttestation(ctx context.Context, req *AttestRequest, spiffeID string, success bool, errorMsg string) {
	audit := &models.AuditLog{
		TenantID:     req.TenantID,
		EventType:    models.EventAttest,
		SpiffeID:     spiffeID,
		Success:      success,
		ErrorMessage: errorMsg,
		Metadata: map[string]interface{}{
			"attestation_type": req.AttestationType,
			"selectors":        req.Selectors,
		},
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	}

	if err := s.auditRepo.Create(ctx, audit); err != nil {
		s.logger.WithError(err).Error("Failed to create audit log")
	}
}

// auditAttestationWithRepo creates an audit log entry using a specific repository
func (s *AttestationService) auditAttestationWithRepo(
	ctx context.Context,
	auditRepo repositories.AuditRepository,
	req *AttestRequest,
	spiffeID string,
	success bool,
	errorMsg string,
) {
	audit := &models.AuditLog{
		TenantID:     req.TenantID,
		EventType:    models.EventAttest,
		SpiffeID:     spiffeID,
		Success:      success,
		ErrorMessage: errorMsg,
		Metadata: map[string]interface{}{
			"attestation_type": req.AttestationType,
			"selectors":        req.Selectors,
		},
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	}

	if err := auditRepo.Create(ctx, audit); err != nil {
		s.logger.WithError(err).Error("Failed to create audit log")
	}
}
