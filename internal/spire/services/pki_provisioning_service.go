package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/vault"

	"github.com/sirupsen/logrus"
)

// PKIProvisioningService handles PKI provisioning for tenants
type PKIProvisioningService struct {
	tenantRepo  repositories.TenantRepository
	vaultClient *vault.Client
	logger      *logrus.Entry
}

// NewPKIProvisioningService creates a new PKI provisioning service
func NewPKIProvisioningService(tenantRepo repositories.TenantRepository, vaultClient *vault.Client, logger *logrus.Entry) *PKIProvisioningService {
	return &PKIProvisioningService{
		tenantRepo:  tenantRepo,
		vaultClient: vaultClient,
		logger:      logger,
	}
}

// ProvisionPKIRequest represents a request to provision PKI for a tenant
type ProvisionPKIRequest struct {
	TenantID       string
	CommonName     string
	AllowedDomains string
	TTL            string
	MaxTTL         string
}

// ProvisionPKIResponse represents the response after provisioning PKI
type ProvisionPKIResponse struct {
	TenantID    string `json:"tenant_id"`
	PKIMount    string `json:"pki_mount"`
	CACert      string `json:"ca_cert"`
	RoleCreated string `json:"role_created"`
	Message     string `json:"message"`
}

// ProvisionPKI provisions a PKI backend for a tenant in Vault
func (s *PKIProvisioningService) ProvisionPKI(ctx context.Context, req *ProvisionPKIRequest) (*ProvisionPKIResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"tenant_id":   req.TenantID,
		"common_name": req.CommonName,
	}).Info("Starting PKI provisioning")

	// Validate request
	if req.TenantID == "" {
		return nil, errors.NewBadRequestError("tenant_id is required", nil)
	}
	if req.CommonName == "" {
		return nil, errors.NewBadRequestError("common_name is required", nil)
	}

	// 1. Determine PKI mount path (e.g., pki/tenant-id or pki/domain)
	// Use allowed_domains as the mount path if provided, otherwise use tenant_id
	// Always prepend "pki/" prefix for consistency
	var pkiMount string
	if req.AllowedDomains != "" {
		pkiMount = fmt.Sprintf("pki/%s", req.AllowedDomains)
	} else {
		pkiMount = fmt.Sprintf("pki/%s", req.TenantID)
	}

	s.logger.WithField("mount", pkiMount).Info("Using PKI mount path")

	// 2. Enable PKI secrets engine at the tenant-specific path
	s.logger.Info("Enabling PKI secrets engine")
	if err := s.vaultClient.EnablePKIEngine(ctx, pkiMount); err != nil {
		// Check if already enabled
		if !isAlreadyEnabledError(err) {
			return nil, errors.NewInternalError("Failed to enable PKI engine", err)
		}
		s.logger.WithField("mount", pkiMount).Info("PKI engine already enabled")
	}

	// 3. Set URLs for CRL and issuing certificate
	// This is optional but recommended for proper certificate management
	// Skip for now as it requires additional configuration

	// 4. Generate root CA certificate
	s.logger.Info("Generating root CA certificate")
	caCert, err := s.vaultClient.GenerateRootCA(ctx, pkiMount, req.CommonName, req.TTL)
	if err != nil {
		return nil, errors.NewInternalError("Failed to generate root CA", err)
	}

	// 5. Create agent role for agent certificate issuance
	// Restrict CN to SPIFFE ID path components only — do not allow arbitrary names.
	spiffeURISANPattern := fmt.Sprintf("spiffe://%s/*", req.AllowedDomains)
	s.logger.Info("Creating agent role")
	if err := s.vaultClient.CreatePKIRole(ctx, pkiMount, "agent", &vault.PKIRoleConfig{
		AllowedDomains:  []string{req.AllowedDomains},
		AllowedURISANs:  []string{spiffeURISANPattern},
		AllowSubdomains: true,
		AllowAnyName:    false,
		AllowURISANs:    true,
		AllowIPSANs:     false,
		MaxTTL:          "168h", // 7 days max for agents
		TTL:             "24h",  // 24 hours default for agents
		KeyType:         "rsa",
		KeyBits:         2048,
		RequireCN:       false,
	}); err != nil {
		return nil, errors.NewInternalError("Failed to create agent role", err)
	}

	// 6. Create workload role for workload certificate issuance
	s.logger.Info("Creating workload role")
	if err := s.vaultClient.CreatePKIRole(ctx, pkiMount, "workload", &vault.PKIRoleConfig{
		AllowedDomains:  []string{req.AllowedDomains},
		AllowedURISANs:  []string{spiffeURISANPattern},
		AllowSubdomains: true,
		AllowAnyName:    false,
		AllowURISANs:    true,
		AllowIPSANs:     true,
		MaxTTL:          req.MaxTTL, // 24h max for workloads
		TTL:             "1h",       // 1 hour default for workloads
		KeyType:         "rsa",
		KeyBits:         2048,
		RequireCN:       false,
	}); err != nil {
		return nil, errors.NewInternalError("Failed to create workload role", err)
	}

	// 7. Update tenant's vault_mount field (with retry)
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"tenant_id": req.TenantID,
		}).WithError(err).Warn("Tenant not found, skipping vault_mount update")
	} else {
		// Update tenant's vault_mount with retry logic
		tenant.VaultMount = pkiMount
		err = s.retryOperation(ctx, "update tenant vault_mount", func() error {
			return s.tenantRepo.Update(ctx, tenant)
		})
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"tenant_id": req.TenantID,
				"pki_mount": pkiMount,
			}).WithError(err).Error("Failed to update tenant vault_mount after retries")
			// CRITICAL: PKI is fully provisioned in Vault but DB update failed
			// Return descriptive error so admin knows to retry the provisioning call
			// The retry will skip already-created resources and just update the DB
			return nil, errors.NewInternalError(
				fmt.Sprintf("PKI provisioned successfully in Vault at '%s' but failed to update tenant record. "+
					"Please call the provision endpoint again to complete the setup (it's safe to retry).", pkiMount),
				err,
			)
		}
		s.logger.WithFields(logrus.Fields{
			"tenant_id":   req.TenantID,
			"vault_mount": pkiMount,
		}).Info("Updated tenant vault_mount")
	}

	s.logger.WithFields(logrus.Fields{
		"tenant_id": req.TenantID,
		"pki_mount": pkiMount,
	}).Info("PKI provisioning completed successfully")

	return &ProvisionPKIResponse{
		TenantID:    req.TenantID,
		PKIMount:    pkiMount,
		CACert:      caCert,
		RoleCreated: "agent, workload",
		Message:     "PKI provisioned successfully with agent and workload roles",
	}, nil
}

// isAlreadyEnabledError checks if the error indicates the PKI engine is already enabled
func isAlreadyEnabledError(err error) bool {
	if err == nil {
		return false
	}
	// Vault returns "path is already in use" when mount exists
	return err.Error() == "path is already in use"
}

// retryOperation retries an operation up to 3 times with exponential backoff
// for transient failures (network issues, temporary unavailability, etc.)
func (s *PKIProvisioningService) retryOperation(ctx context.Context, operationName string, operation func() error) error {
	const maxRetries = 3
	const initialBackoff = 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := initialBackoff * time.Duration(1<<uint(attempt-1)) // Exponential: 1s, 2s, 4s
			s.logger.WithFields(logrus.Fields{
				"operation":   operationName,
				"attempt":     attempt + 1,
				"max_retries": maxRetries,
				"backoff":     backoff,
			}).Info("Retrying operation after backoff")

			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := operation()
		if err == nil {
			if attempt > 0 {
				s.logger.WithFields(logrus.Fields{
					"operation": operationName,
					"attempt":   attempt + 1,
				}).Info("Operation succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			s.logger.WithField("operation", operationName).WithError(err).Warn("Non-retryable error encountered, stopping retries")
			return err
		}

		s.logger.WithFields(logrus.Fields{
			"operation": operationName,
			"attempt":   attempt + 1,
		}).WithError(err).Warn("Retryable error encountered")
	}

	s.logger.WithFields(logrus.Fields{
		"operation":   operationName,
		"max_retries": maxRetries,
	}).WithError(lastErr).Error("Operation failed after all retries")

	return lastErr
}

// isRetryableError checks if an error is retryable (transient)
// Returns true for network errors, timeouts, temporary failures
// Returns false for validation errors, not found, already exists, etc.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Don't retry these permanent failures
	permanentErrors := []string{
		"not found",
		"already exists",
		"invalid",
		"bad request",
		"unauthorized",
		"forbidden",
		"path is already in use", // PKI engine already enabled - not an error
	}

	for _, permanent := range permanentErrors {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(permanent)) {
			return false
		}
	}

	// Retry these transient failures
	retryableErrors := []string{
		"connection refused",
		"timeout",
		"temporary",
		"unavailable",
		"too many requests",
		"rate limit",
		"network",
		"dial",
		"EOF",
		"broken pipe",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(retryable)) {
			return true
		}
	}

	// Default: retry unknown errors (they might be transient)
	return true
}
