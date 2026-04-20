package services

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/database"
	infrarepos "github.com/authsec-ai/authsec/internal/spire/infrastructure/repositories"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/vault"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// NodeAttestationService handles node attestation for agents
type NodeAttestationService struct {
	connManager  *database.ConnectionManager
	tenantRepo   repositories.TenantRepository
	vaultClient  *vault.Client
	k8sValidator *KubernetesValidator
	logger       *logrus.Entry
}

// NodeAttestRequest represents a node attestation request
type NodeAttestRequest struct {
	TenantID        string
	NodeID          string
	AttestationType string
	Evidence        map[string]interface{}
	CSR             string
}

// NodeAttestResponse represents a node attestation response
type NodeAttestResponse struct {
	Certificate string
	CAChain     []string
	SpiffeID    string
	ExpiresAt   time.Time
	AgentID     string
}

// NewNodeAttestationService creates a new node attestation service
func NewNodeAttestationService(
	connManager *database.ConnectionManager,
	tenantRepo repositories.TenantRepository,
	vaultClient *vault.Client,
	logger *logrus.Entry,
) *NodeAttestationService {
	// Initialize Kubernetes validator in development mode (no TokenReview API)
	// This bypasses RBAC requirements for the ICP server
	k8sValidator, err := NewKubernetesValidator(logger, &KubernetesValidatorConfig{
		UseTokenReview: false, // Disable TokenReview API validation
	})
	if err != nil {
		logger.WithError(err).Error("Failed to initialize Kubernetes validator")
		return nil
	}

	return &NodeAttestationService{
		connManager:  connManager,
		tenantRepo:   tenantRepo,
		vaultClient:  vaultClient,
		k8sValidator: k8sValidator,
		logger:       logger,
	}
}

// Attest performs node attestation and issues Agent SVID
func (s *NodeAttestationService) Attest(ctx context.Context, req *NodeAttestRequest) (*NodeAttestResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"tenant_id":        req.TenantID,
		"node_id":          req.NodeID,
		"attestation_type": req.AttestationType,
	}).Info("Node attestation started")

	// 1. Validate tenant
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, errors.NewNotFoundError("Tenant not found", err)
	}
	if tenant.Status != "active" {
		return nil, errors.NewForbiddenError("Tenant is not active", nil)
	}

	// 2. Validate attestation evidence
	nodeSelectors, err := s.validateEvidence(ctx, req.AttestationType, req.Evidence)
	if err != nil {
		s.logger.WithField("attestation_type", req.AttestationType).WithError(err).Error("Attestation validation failed")
		return nil, errors.NewUnauthorizedError("Attestation failed", err)
	}

	// 3. Validate CSR
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

	// 4. Generate SPIFFE ID for agent
	spiffeID := s.generateAgentSpiffeID(req.TenantID, req.NodeID)

	// 5. Ensure Vault PKI role allows URI SANs (automatic configuration)
	if err := s.ensureVaultRoleConfigured(ctx, tenant.VaultMount, req.TenantID); err != nil {
		s.logger.WithFields(logrus.Fields{
			"vault_mount": tenant.VaultMount,
		}).WithError(err).Error("Failed to configure Vault PKI role")
		return nil, errors.NewInternalError("Failed to configure Vault PKI role", err)
	}

	// 6. Issue certificate via Vault
	certReq := &vault.CertificateRequest{
		CSR:        req.CSR,
		CommonName: spiffeID,
		TTL:        "1h", // 1 hour for frequent renewal and better security
		URISANs:    []string{spiffeID},
	}

	certResp, err := s.vaultClient.IssueCertificate(ctx, tenant.VaultMount, "agent", certReq)
	if err != nil {
		s.logger.WithField("vault_mount", tenant.VaultMount).WithError(err).Error("Failed to issue agent certificate")
		return nil, errors.NewInternalError("Failed to issue certificate", err)
	}

	// 6. Get tenant-specific database connection
	tenantDB, err := s.connManager.GetTenantDB(ctx, req.TenantID)
	if err != nil {
		s.logger.WithField("tenant_id", req.TenantID).WithError(err).Error("Failed to connect to tenant database")
		return nil, errors.NewInternalError("Failed to connect to tenant database", err)
	}

	// Create agent repository for this tenant's database
	agentRepo := infrarepos.NewPostgresAgentRepository(tenantDB, s.logger)

	// 7. Create or update agent record
	// Check if agent already exists
	existingAgent, err := agentRepo.GetByTenantAndNode(ctx, req.TenantID, req.NodeID)
	if err == nil {
		// Agent exists, update it
		existingAgent.SpiffeID = spiffeID
		existingAgent.AttestationType = req.AttestationType
		existingAgent.NodeSelectors = nodeSelectors
		existingAgent.CertificateSerial = certResp.SerialNumber
		existingAgent.Status = models.AgentStatusActive
		existingAgent.LastSeen = time.Now()

		if err := agentRepo.Update(ctx, existingAgent); err != nil {
			return nil, errors.NewInternalError("Failed to update agent", err)
		}

		s.logger.WithFields(logrus.Fields{
			"agent_id":  existingAgent.ID,
			"spiffe_id": spiffeID,
		}).Info("Agent updated")

		return &NodeAttestResponse{
			Certificate: certResp.Certificate,
			CAChain:     certResp.CAChain,
			SpiffeID:    spiffeID,
			ExpiresAt:   certResp.ExpirationTime,
			AgentID:     existingAgent.ID,
		}, nil
	}

	// Agent doesn't exist, create new
	agent := &models.Agent{
		ID:                uuid.New().String(),
		TenantID:          req.TenantID,
		NodeID:            req.NodeID,
		SpiffeID:          spiffeID,
		AttestationType:   req.AttestationType,
		NodeSelectors:     nodeSelectors,
		CertificateSerial: certResp.SerialNumber,
		Status:            models.AgentStatusActive,
		LastSeen:          time.Now(),
	}

	if err := agentRepo.Create(ctx, agent); err != nil {
		return nil, errors.NewInternalError("Failed to store agent", err)
	}

	s.logger.WithFields(logrus.Fields{
		"agent_id":  agent.ID,
		"spiffe_id": spiffeID,
		"node_id":   req.NodeID,
	}).Info("Agent created")

	// 8. Return response
	return &NodeAttestResponse{
		Certificate: certResp.Certificate,
		CAChain:     certResp.CAChain,
		SpiffeID:    spiffeID,
		ExpiresAt:   certResp.ExpirationTime,
		AgentID:     agent.ID,
	}, nil
}

// validateEvidence validates attestation evidence based on type
func (s *NodeAttestationService) validateEvidence(ctx context.Context, attestationType string, evidence map[string]interface{}) (map[string]string, error) {
	switch attestationType {
	case models.AttestationTypeKubernetes:
		return s.k8sValidator.Validate(ctx, evidence)

	case models.AttestationTypeUnix:
		// Unix attestation - minimal validation for development/testing
		// In production, this should verify process identity and permissions
		nodeSelectors := make(map[string]string)
		if pid, ok := evidence["pid"].(float64); ok {
			nodeSelectors["unix:pid"] = fmt.Sprintf("%d", int(pid))
		}
		if username, ok := evidence["username"].(string); ok {
			nodeSelectors["unix:username"] = username
		}
		if hostname, ok := evidence["hostname"].(string); ok {
			nodeSelectors["unix:hostname"] = hostname
		}
		return nodeSelectors, nil

	case models.AttestationTypeDocker:
		// Docker attestation - minimal validation for development/testing
		// In production, this should verify container identity
		nodeSelectors := make(map[string]string)
		if containerID, ok := evidence["container_id"].(string); ok {
			nodeSelectors["docker:container_id"] = containerID
		}
		if imageName, ok := evidence["image_name"].(string); ok {
			nodeSelectors["docker:image_name"] = imageName
		}
		return nodeSelectors, nil

	case models.AttestationTypeTPM:
		// TODO: Implement TPM validator in future sprint
		return nil, fmt.Errorf("TPM attestation not yet implemented")

	case models.AttestationTypeAWS:
		// TODO: Implement AWS validator in future sprint
		return nil, fmt.Errorf("AWS attestation not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported attestation type: %s", attestationType)
	}
}

// generateAgentSpiffeID generates a SPIFFE ID for an agent
// Format: spiffe://{tenant-id}/agent/{node-id}
func (s *NodeAttestationService) generateAgentSpiffeID(tenantID, nodeID string) string {
	return fmt.Sprintf("spiffe://%s/agent/%s", tenantID, nodeID)
}

// ensureVaultRoleConfigured ensures the Vault PKI role allows URI SANs for SPIFFE IDs
// This is called automatically during node attestation to configure the role if needed
func (s *NodeAttestationService) ensureVaultRoleConfigured(ctx context.Context, vaultMount, tenantID string) error {
	s.logger.WithFields(logrus.Fields{
		"vault_mount": vaultMount,
		"tenant_id":   tenantID,
	}).Info("Ensuring Vault PKI role is configured for URI SANs")

	// Configure the agent role to allow URI SANs
	// This allows SPIFFE IDs in the form: spiffe://{tenant-id}/agent/{node-id}
	roleConfig := &vault.PKIRoleConfig{
		AllowedDomains:  []string{},
		AllowedURISANs:  []string{fmt.Sprintf("spiffe://%s/*", tenantID)},
		AllowSubdomains: false,
		AllowAnyName:    false,
		AllowURISANs:    true,
		AllowIPSANs:     false,
		MaxTTL:          "2h",
		TTL:             "1h",
		KeyType:         "rsa",
		KeyBits:         2048,
		RequireCN:       false,
	}

	// Create or update the agent role
	// This is idempotent - it will update the role if it already exists
	// Note: vaultMount already includes the full path (e.g., "pki/spire.app.authsec.dev")
	if err := s.vaultClient.CreatePKIRole(ctx, vaultMount, "agent", roleConfig); err != nil {
		s.logger.WithField("vault_mount", vaultMount).WithError(err).Error("Failed to create/update PKI role")
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"vault_mount": vaultMount,
		"role":        "agent",
	}).Info("Vault PKI role configured successfully")

	return nil
}
