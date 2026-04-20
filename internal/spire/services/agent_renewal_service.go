package services

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/database"
	infrarepos "github.com/authsec-ai/authsec/internal/spire/infrastructure/repositories"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/vault"

	"github.com/sirupsen/logrus"
)

// AgentRenewalService handles agent SVID renewal
type AgentRenewalService struct {
	connManager *database.ConnectionManager
	tenantRepo  repositories.TenantRepository
	vaultClient *vault.Client
	logger      *logrus.Entry
}

// AgentRenewRequest represents an agent renewal request
type AgentRenewRequest struct {
	AgentID  string
	TenantID string // Optional: for non-mTLS environments
	CSR      string
}

// AgentRenewResponse represents an agent renewal response
type AgentRenewResponse struct {
	SpiffeID    string
	Certificate string
	CABundle    string
	TTL         int
	CAChain     []string
	ExpiresAt   time.Time
}

// NewAgentRenewalService creates a new agent renewal service
func NewAgentRenewalService(
	connManager *database.ConnectionManager,
	tenantRepo repositories.TenantRepository,
	vaultClient *vault.Client,
	logger *logrus.Entry,
) *AgentRenewalService {
	return &AgentRenewalService{
		connManager: connManager,
		tenantRepo:  tenantRepo,
		vaultClient: vaultClient,
		logger:      logger,
	}
}

// Renew renews an agent's SVID
func (s *AgentRenewalService) Renew(ctx context.Context, req *AgentRenewRequest) (*AgentRenewResponse, error) {
	s.logger.WithField("agent_id", req.AgentID).Info("Agent renewal started")

	// 1. Extract tenant ID from mTLS context only — never trust the request body.
	// The mTLS middleware extracts this from the agent's certificate SPIFFE ID.
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		return nil, errors.NewUnauthorizedError(
			"Tenant ID not found in mTLS context. Agent renewal requires mutual TLS authentication.", nil)
	}

	// 2. Get tenant
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, errors.NewNotFoundError("Tenant not found", err)
	}

	if tenant.Status != "active" {
		return nil, errors.NewForbiddenError("Tenant is not active", nil)
	}

	// 3. Get tenant-specific database connection
	tenantDB, err := s.connManager.GetTenantDB(ctx, tenantID)
	if err != nil {
		s.logger.WithField("tenant_id", tenantID).WithError(err).Error("Failed to connect to tenant database")
		return nil, errors.NewInternalError("Failed to connect to tenant database", err)
	}

	// Create agent repository for this tenant's database
	agentRepo := infrarepos.NewPostgresAgentRepository(tenantDB, s.logger)

	// 4. Get agent from tenant database
	agent, err := agentRepo.GetByID(ctx, req.AgentID)
	if err != nil {
		return nil, errors.NewNotFoundError("Agent not found", err)
	}

	if agent.Status != models.AgentStatusActive {
		return nil, errors.NewForbiddenError("Agent is not active", nil)
	}

	// Verify agent belongs to the authenticated tenant
	if agent.TenantID != tenantID {
		return nil, errors.NewForbiddenError("Agent does not belong to authenticated tenant", nil)
	}

	// 5. Validate CSR
	csrBlock, _ := pem.Decode([]byte(req.CSR))
	if csrBlock == nil {
		return nil, errors.NewBadRequestError("Invalid CSR format", nil)
	}

	csr, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		return nil, errors.NewBadRequestError("Failed to parse CSR", err)
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, errors.NewBadRequestError("Invalid CSR signature", err)
	}

	// 6. Issue new certificate via Vault
	certReq := &vault.CertificateRequest{
		CSR:        req.CSR,
		CommonName: agent.SpiffeID,
		TTL:        "1h", // 1 hour for frequent renewal and better security
		URISANs:    []string{agent.SpiffeID},
	}

	certResp, err := s.vaultClient.IssueCertificate(ctx, tenant.VaultMount, "agent", certReq)
	if err != nil {
		s.logger.WithFields(logrus.Fields{"agent_id": agent.ID, "vault_mount": tenant.VaultMount}).WithError(err).Error("Failed to issue agent certificate")
		return nil, errors.NewInternalError("Failed to issue certificate", err)
	}

	// 7. Update agent record
	agent.CertificateSerial = certResp.SerialNumber
	agent.LastSeen = time.Now()

	if err := agentRepo.Update(ctx, agent); err != nil {
		return nil, errors.NewInternalError("Failed to update agent", err)
	}

	s.logger.WithFields(logrus.Fields{"agent_id": agent.ID, "serial_number": certResp.SerialNumber}).Info("Agent SVID renewed")

	// Calculate TTL in seconds (1 hour = 3600 seconds)
	ttl := 3600 // 1 hour

	// Combine CA chain into single bundle
	caBundle := certResp.CAChain[0] // Root CA
	for i := 1; i < len(certResp.CAChain); i++ {
		caBundle += "\n" + certResp.CAChain[i]
	}

	return &AgentRenewResponse{
		SpiffeID:    agent.SpiffeID,
		Certificate: certResp.Certificate,
		CABundle:    caBundle,
		TTL:         ttl,
		CAChain:     certResp.CAChain,
		ExpiresAt:   certResp.ExpirationTime,
	}, nil
}
