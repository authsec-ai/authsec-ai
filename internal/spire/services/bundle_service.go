package services

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
	"github.com/authsec-ai/authsec/internal/spire/infrastructure/vault"
)

// BundleService handles CA bundle retrieval
type BundleService struct {
	tenantRepo  repositories.TenantRepository
	vaultClient *vault.Client
	logger      *logrus.Entry
}

// NewBundleService creates a new bundle service
func NewBundleService(
	tenantRepo repositories.TenantRepository,
	vaultClient *vault.Client,
	logger *logrus.Entry,
) *BundleService {
	return &BundleService{
		tenantRepo:  tenantRepo,
		vaultClient: vaultClient,
		logger:      logger,
	}
}

// GetBundle retrieves the CA bundle for a tenant
func (s *BundleService) GetBundle(ctx context.Context, tenantIdentifier string) (string, error) {
	s.logger.WithField("tenant_identifier", tenantIdentifier).Debug("Getting CA bundle")

	// Determine if identifier is a UUID or domain
	// UUIDs contain hyphens, domains contain dots
	isUUID := strings.Contains(tenantIdentifier, "-") && len(tenantIdentifier) == 36

	var tenant *models.Tenant
	var err error

	if isUUID {
		// Try to get tenant by ID (UUID)
		tenant, err = s.tenantRepo.GetByID(ctx, tenantIdentifier)
		if err != nil {
			return "", err
		}
	} else {
		// Try to get tenant by domain
		tenant, err = s.tenantRepo.GetByDomain(ctx, tenantIdentifier)
		if err != nil {
			return "", err
		}
	}

	if !tenant.IsActive() {
		return "", errors.NewForbiddenError("Tenant is not active", nil)
	}

	// Remove redundant "pki/" prefix if present
	cleanVaultMount := strings.TrimPrefix(tenant.VaultMount, "pki/")

	// Get CA bundle from Vault
	bundle, err := s.vaultClient.GetCABundle(ctx, cleanVaultMount)
	if err != nil {
		return "", err
	}

	return bundle, nil
}
