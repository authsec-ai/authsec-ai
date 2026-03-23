package services

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/authsec-ai/authsec/database"
	"github.com/google/uuid"
)

// DomainService handles domain operations including DNS verification
type DomainService struct {
	repo *database.TenantDomainsRepository
}

// NewDomainService creates a new domain service
func NewDomainService(repo *database.TenantDomainsRepository) *DomainService {
	return &DomainService{repo: repo}
}

// skipRealDNSLookup checks if we're in development/mock mode
func (ds *DomainService) skipRealDNSLookup() bool {
	// Allow skipping real DNS for local development/testing
	// In production, this should always be false
	return os.Getenv("SKIP_DNS_VERIFICATION") == "true"
}

// RegisterDomain registers a new domain for a tenant
func (ds *DomainService) RegisterDomain(tenantID uuid.UUID, domain string, createdBy *uuid.UUID) (*database.TenantDomain, error) {
	return ds.repo.CreateDomain(tenantID, domain, createdBy)
}

// VerifyDomainOwnership performs DNS TXT verification
func (ds *DomainService) VerifyDomainOwnership(domainID uuid.UUID) error {
	// Get domain details
	td, err := ds.repo.GetDomainByID(domainID)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}

	if td.IsVerified {
		return fmt.Errorf("domain already verified")
	}

	if td.VerificationTXTName == nil || td.VerificationTXTValue == nil {
		return fmt.Errorf("verification details not set")
	}

	// Perform DNS TXT lookup
	verified, failureReason := ds.verifyDNSTXT(*td.VerificationTXTName, *td.VerificationTXTValue)

	if verified {
		// Mark as verified
		err := ds.repo.VerifyDomain(domainID, nil)
		if err != nil {
			return fmt.Errorf("failed to mark domain as verified: %w", err)
		}
		return nil
	}

	// Record failure reason
	err = ds.repo.UpdateVerificationStatus(domainID, false, &failureReason)
	if err != nil {
		// Log but don't fail
		fmt.Printf("Failed to update verification status: %v\n", err)
	}

	return fmt.Errorf("verification failed: %s", failureReason)
}

// verifyDNSTXT looks up TXT records and checks for verification token
func (ds *DomainService) verifyDNSTXT(txtName, expectedValue string) (bool, string) {
	// Check if we should skip real DNS lookup (for local testing)
	if ds.skipRealDNSLookup() {
		// Mock mode: always succeed verification
		return true, ""
	}

	// Normalize: remove trailing dot if present
	txtName = strings.TrimSuffix(txtName, ".")

	// Perform DNS TXT lookup
	txtRecords, err := net.LookupTXT(txtName)
	if err != nil {
		return false, fmt.Sprintf("DNS TXT lookup failed: %v", err)
	}

	if len(txtRecords) == 0 {
		return false, fmt.Sprintf("No TXT records found for %s", txtName)
	}

	// Check if any record matches
	for _, record := range txtRecords {
		if record == expectedValue {
			return true, ""
		}
	}

	return false, fmt.Sprintf("TXT record value mismatch. Expected: %s, Found: %v", expectedValue, txtRecords)
}

// GetVerifiedDomainsForTenant returns list of verified domains (for use in redirect URI validation)
func (ds *DomainService) GetVerifiedDomainsForTenant(tenantID uuid.UUID) ([]string, error) {
	return ds.repo.GetVerifiedDomainsForTenant(tenantID)
}

// ResolveTenantByHost resolves tenant from Host header (for Host-based tenant resolution)
func (ds *DomainService) ResolveTenantByHost(hostname string) (uuid.UUID, error) {
	td, err := ds.repo.GetDomainByHostname(hostname)
	if err != nil {
		return uuid.Nil, err
	}
	return td.TenantID, nil
}

// ListTenantDomains lists all domains for a tenant
func (ds *DomainService) ListTenantDomains(tenantID uuid.UUID) ([]database.TenantDomain, error) {
	return ds.repo.ListTenantDomains(tenantID)
}

// SetPrimaryDomain sets a domain as the primary for a tenant
func (ds *DomainService) SetPrimaryDomain(tenantID, domainID uuid.UUID, updatedBy *uuid.UUID) error {
	return ds.repo.SetPrimaryDomain(tenantID, domainID, updatedBy)
}

// DeleteDomain deletes a domain
func (ds *DomainService) DeleteDomain(domainID uuid.UUID) error {
	return ds.repo.DeleteDomain(domainID)
}

// ValidateRedirectURIHost checks if a redirect URI's host is owned by tenant
func (ds *DomainService) ValidateRedirectURIHost(tenantID uuid.UUID, redirectURI string) (bool, error) {
	// Parse redirect URI to extract host
	if !strings.HasPrefix(redirectURI, "http://") && !strings.HasPrefix(redirectURI, "https://") {
		return false, fmt.Errorf("invalid redirect URI: must start with http:// or https://")
	}

	// Simple host extraction (not a full URL parser, but adequate for this use case)
	uriWithoutScheme := redirectURI
	if strings.HasPrefix(redirectURI, "https://") {
		uriWithoutScheme = redirectURI[8:]
	} else if strings.HasPrefix(redirectURI, "http://") {
		uriWithoutScheme = redirectURI[7:]
	}

	// Extract host (before first /, ?, or :)
	host := uriWithoutScheme
	if idx := strings.IndexAny(host, "/?"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		// Only strip port if it's not part of IPv6 address
		if !strings.Contains(host[:idx], "[") {
			host = host[:idx]
		}
	}

	// Reject wildcards and dangerous patterns
	if strings.Contains(host, "*") || strings.Contains(host, "%") {
		return false, fmt.Errorf("wildcard hosts not allowed")
	}

	// Get verified domains for tenant
	verifiedDomains, err := ds.repo.GetVerifiedDomainsForTenant(tenantID)
	if err != nil {
		return false, err
	}

	if len(verifiedDomains) == 0 {
		return false, fmt.Errorf("no verified domains for tenant")
	}

	// Check if host matches one of the verified domains
	hostNormalized := strings.ToLower(strings.TrimSuffix(host, "."))
	for _, verifiedDomain := range verifiedDomains {
		if hostNormalized == verifiedDomain {
			return true, nil
		}
	}

	return false, fmt.Errorf("redirect URI host %s not verified for this tenant", host)
}
