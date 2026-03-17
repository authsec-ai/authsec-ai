package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TenantDomain represents a verified or pending custom domain for a tenant
type TenantDomain struct {
	ID                   uuid.UUID  `db:"id"`
	TenantID             uuid.UUID  `db:"tenant_id"`
	Domain               string     `db:"domain"`
	Kind                 string     `db:"kind"` // 'platform_subdomain' or 'custom'
	IsPrimary            bool       `db:"is_primary"`
	IsVerified           bool       `db:"is_verified"`
	VerificationMethod   string     `db:"verification_method"` // 'dns_txt'
	VerificationToken    string     `db:"verification_token"`
	VerificationTXTName  *string    `db:"verification_txt_name"`  // e.g., _authsec-challenge.domain
	VerificationTXTValue *string    `db:"verification_txt_value"` // e.g., authsec-domain-verification=<token>
	VerifiedAt           *time.Time `db:"verified_at"`
	LastCheckedAt        *time.Time `db:"last_checked_at"`
	FailureReason        *string    `db:"failure_reason"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
	CreatedBy            *uuid.UUID `db:"created_by"`
	UpdatedBy            *uuid.UUID `db:"updated_by"`
}

// TenantDomainsRepository handles database operations for tenant domains
type TenantDomainsRepository struct {
	db *DBConnection
}

// NewTenantDomainsRepository creates a new repository
func NewTenantDomainsRepository(db *DBConnection) *TenantDomainsRepository {
	return &TenantDomainsRepository{db: db}
}

// generateVerificationToken creates a random 32-byte token (hex encoded)
func (tdr *TenantDomainsRepository) generateVerificationToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(token), nil
}

// normalizeDomain converts domain to lowercase and removes trailing dot
func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	return strings.TrimSuffix(domain, ".")
}

// CreateDomain registers a new domain for a tenant (pending verification)
func (tdr *TenantDomainsRepository) CreateDomain(tenantID uuid.UUID, domain string, createdBy *uuid.UUID) (*TenantDomain, error) {
	// Validate and normalize domain
	domain = normalizeDomain(domain)
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}
	if strings.Contains(domain, "/") || strings.Contains(domain, "\\") || strings.Contains(domain, "*") {
		return nil, fmt.Errorf("invalid domain format")
	}

	// Check if domain is already claimed by another tenant
	var existingTenantID uuid.UUID
	err := tdr.db.QueryRow(
		"SELECT tenant_id FROM tenant_domains WHERE domain = $1",
		domain,
	).Scan(&existingTenantID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check domain uniqueness: %w", err)
	}
	if err == nil && existingTenantID != tenantID {
		return nil, fmt.Errorf("domain already claimed by another tenant")
	}

	// Generate verification token
	token, err := tdr.generateVerificationToken()
	if err != nil {
		return nil, err
	}

	// Build TXT record name and value
	txtName := fmt.Sprintf("_authsec-challenge.%s", domain)
	txtValue := fmt.Sprintf("authsec-domain-verification=%s", token)

	id := uuid.New()
	now := time.Now()
	kind := "custom"

	query := `
		INSERT INTO tenant_domains (
			id, tenant_id, domain, kind, is_primary, is_verified,
			verification_method, verification_token, verification_txt_name,
			verification_txt_value, created_at, updated_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, tenant_id, domain, kind, is_primary, is_verified,
			verification_method, verification_token, verification_txt_name,
			verification_txt_value, verified_at, last_checked_at,
			failure_reason, created_at, updated_at, created_by, updated_by
	`

	td := &TenantDomain{
		ID:                   id,
		TenantID:             tenantID,
		Domain:               domain,
		Kind:                 kind,
		IsPrimary:            false,
		IsVerified:           false,
		VerificationMethod:   "dns_txt",
		VerificationToken:    token,
		VerificationTXTName:  &txtName,
		VerificationTXTValue: &txtValue,
		CreatedAt:            now,
		UpdatedAt:            now,
		CreatedBy:            createdBy,
	}

	err = tdr.db.QueryRow(query, id, tenantID, domain, kind, false, false,
		"dns_txt", token, txtName, txtValue, now, now, createdBy).Scan(
		&td.ID, &td.TenantID, &td.Domain, &td.Kind, &td.IsPrimary, &td.IsVerified,
		&td.VerificationMethod, &td.VerificationToken, &td.VerificationTXTName,
		&td.VerificationTXTValue, &td.VerifiedAt, &td.LastCheckedAt,
		&td.FailureReason, &td.CreatedAt, &td.UpdatedAt, &td.CreatedBy, &td.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create domain: %w", err)
	}

	return td, nil
}

// GetDomainByID retrieves a domain by ID
func (tdr *TenantDomainsRepository) GetDomainByID(id uuid.UUID) (*TenantDomain, error) {
	query := `
		SELECT id, tenant_id, domain, kind, is_primary, is_verified,
			verification_method, verification_token, verification_txt_name,
			verification_txt_value, verified_at, last_checked_at, failure_reason,
			created_at, updated_at, created_by, updated_by
		FROM tenant_domains
		WHERE id = $1
	`

	td := &TenantDomain{}
	err := tdr.db.QueryRow(query, id).Scan(
		&td.ID, &td.TenantID, &td.Domain, &td.Kind, &td.IsPrimary, &td.IsVerified,
		&td.VerificationMethod, &td.VerificationToken, &td.VerificationTXTName,
		&td.VerificationTXTValue, &td.VerifiedAt, &td.LastCheckedAt, &td.FailureReason,
		&td.CreatedAt, &td.UpdatedAt, &td.CreatedBy, &td.UpdatedBy,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("domain not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	return td, nil
}

// GetDomainByHostname retrieves a verified domain by hostname (for Host → tenant resolution)
func (tdr *TenantDomainsRepository) GetDomainByHostname(hostname string) (*TenantDomain, error) {
	hostname = normalizeDomain(hostname)

	query := `
		SELECT id, tenant_id, domain, kind, is_primary, is_verified,
			verification_method, verification_token, verification_txt_name,
			verification_txt_value, verified_at, last_checked_at, failure_reason,
			created_at, updated_at, created_by, updated_by
		FROM tenant_domains
		WHERE domain = $1 AND is_verified = true
		LIMIT 1
	`

	td := &TenantDomain{}
	err := tdr.db.QueryRow(query, hostname).Scan(
		&td.ID, &td.TenantID, &td.Domain, &td.Kind, &td.IsPrimary, &td.IsVerified,
		&td.VerificationMethod, &td.VerificationToken, &td.VerificationTXTName,
		&td.VerificationTXTValue, &td.VerifiedAt, &td.LastCheckedAt, &td.FailureReason,
		&td.CreatedAt, &td.UpdatedAt, &td.CreatedBy, &td.UpdatedBy,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("domain not found or not verified")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}

	return td, nil
}

// ListTenantDomains retrieves all domains for a tenant
func (tdr *TenantDomainsRepository) ListTenantDomains(tenantID uuid.UUID) ([]TenantDomain, error) {
	query := `
		SELECT id, tenant_id, domain, kind, is_primary, is_verified,
			verification_method, verification_token, verification_txt_name,
			verification_txt_value, verified_at, last_checked_at, failure_reason,
			created_at, updated_at, created_by, updated_by
		FROM tenant_domains
		WHERE tenant_id = $1
		ORDER BY is_primary DESC, created_at DESC
	`

	rows, err := tdr.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}
	defer rows.Close()

	var domains []TenantDomain
	for rows.Next() {
		td := TenantDomain{}
		err := rows.Scan(
			&td.ID, &td.TenantID, &td.Domain, &td.Kind, &td.IsPrimary, &td.IsVerified,
			&td.VerificationMethod, &td.VerificationToken, &td.VerificationTXTName,
			&td.VerificationTXTValue, &td.VerifiedAt, &td.LastCheckedAt, &td.FailureReason,
			&td.CreatedAt, &td.UpdatedAt, &td.CreatedBy, &td.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, td)
	}

	return domains, nil
}

// GetPrimaryDomainByTenantID retrieves the primary domain for a tenant
func (tdr *TenantDomainsRepository) GetPrimaryDomainByTenantID(tenantID uuid.UUID) (*TenantDomain, error) {
	query := `
		SELECT id, tenant_id, domain, kind, is_primary, is_verified,
			verification_method, verification_token, verification_txt_name,
			verification_txt_value, verified_at, last_checked_at, failure_reason,
			created_at, updated_at, created_by, updated_by
		FROM tenant_domains
		WHERE tenant_id = $1 AND is_primary = true
		LIMIT 1
	`

	td := &TenantDomain{}
	err := tdr.db.QueryRow(query, tenantID).Scan(
		&td.ID, &td.TenantID, &td.Domain, &td.Kind, &td.IsPrimary, &td.IsVerified,
		&td.VerificationMethod, &td.VerificationToken, &td.VerificationTXTName,
		&td.VerificationTXTValue, &td.VerifiedAt, &td.LastCheckedAt, &td.FailureReason,
		&td.CreatedAt, &td.UpdatedAt, &td.CreatedBy, &td.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no primary domain found for tenant")
		}
		return nil, fmt.Errorf("failed to query primary domain: %w", err)
	}

	return td, nil
}

// VerifyDomain marks a domain as verified
func (tdr *TenantDomainsRepository) VerifyDomain(id uuid.UUID, updatedBy *uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE tenant_domains
		SET is_verified = true, verified_at = $1, updated_at = $1, updated_by = $2
		WHERE id = $3
	`

	result, err := tdr.db.Exec(query, now, updatedBy, id)
	if err != nil {
		return fmt.Errorf("failed to verify domain: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("domain not found")
	}

	return nil
}

// SetPrimaryDomain sets a domain as primary for a tenant (and unsets others)
func (tdr *TenantDomainsRepository) SetPrimaryDomain(tenantID, domainID uuid.UUID, updatedBy *uuid.UUID) error {
	now := time.Now()

	// First, unset all other primary domains for this tenant
	query1 := `
		UPDATE tenant_domains
		SET is_primary = false, updated_at = $1, updated_by = $2
		WHERE tenant_id = $3 AND is_primary = true
	`
	_, err := tdr.db.Exec(query1, now, updatedBy, tenantID)
	if err != nil {
		return fmt.Errorf("failed to unset other primary domains: %w", err)
	}

	// Then set this one as primary
	query2 := `
		UPDATE tenant_domains
		SET is_primary = true, updated_at = $1, updated_by = $2
		WHERE id = $3 AND tenant_id = $4
	`
	result, err := tdr.db.Exec(query2, now, updatedBy, domainID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to set primary domain: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("domain not found for tenant")
	}

	return nil
}

// DeleteDomain soft-deletes (or hard-deletes) a domain
func (tdr *TenantDomainsRepository) DeleteDomain(id uuid.UUID) error {
	query := `DELETE FROM tenant_domains WHERE id = $1`
	result, err := tdr.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("domain not found")
	}

	return nil
}

// GetVerifiedDomainsForTenant returns only verified domains for a tenant
func (tdr *TenantDomainsRepository) GetVerifiedDomainsForTenant(tenantID uuid.UUID) ([]string, error) {
	query := `
		SELECT domain
		FROM tenant_domains
		WHERE tenant_id = $1 AND is_verified = true
		ORDER BY is_primary DESC
	`

	rows, err := tdr.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query verified domains: %w", err)
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, domain)
	}

	return domains, nil
}

// UpdateVerificationStatus updates verification attempt details
func (tdr *TenantDomainsRepository) UpdateVerificationStatus(id uuid.UUID, isVerified bool, failureReason *string) error {
	now := time.Now()
	query := `
		UPDATE tenant_domains
		SET is_verified = $1, last_checked_at = $2, failure_reason = $3, updated_at = $2
		WHERE id = $4
	`

	result, err := tdr.db.Exec(query, isVerified, now, failureReason, id)
	if err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("domain not found")
	}

	return nil
}

// IsDomainOwnedByTenant checks if a hostname is owned by the tenant and is verified
func (tdr *TenantDomainsRepository) IsDomainOwnedByTenant(tenantID uuid.UUID, hostname string) (bool, error) {
	hostname = normalizeDomain(hostname)

	var count int
	err := tdr.db.QueryRow(
		"SELECT COUNT(*) FROM tenant_domains WHERE tenant_id = $1 AND domain = $2 AND is_verified = true",
		tenantID, hostname,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check domain ownership: %w", err)
	}

	return count > 0, nil
}

// ValidateRedirectURIs validates all redirect URIs for a tenant
func (tdr *TenantDomainsRepository) ValidateRedirectURIs(tenantID uuid.UUID, redirectURIs []string) ([]string, error) {
	// Special case: allow localhost in development (can be made configurable via env)
	isDev := os.Getenv("ENVIRONMENT") == "development" || os.Getenv("ENVIRONMENT") == ""

	var validatedHosts []string
	var errs []error

	for _, uri := range redirectURIs {
		// Skip empty URIs
		uri = strings.TrimSpace(uri)
		if uri == "" {
			continue
		}

		// Allow localhost in development mode
		if isDev && strings.Contains(uri, "localhost") {
			validatedHosts = append(validatedHosts, uri)
			continue
		}

		// Validate and normalize
		host, err := tdr.NormalizeHostnameAndCheckOwnership(tenantID, uri)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		validatedHosts = append(validatedHosts, host)
	}

	if len(errs) > 0 {
		return validatedHosts, &RedirectURIValidationError{Errors: errs}
	}

	return validatedHosts, nil
}

// NormalizeHostnameAndCheckOwnership validates a redirect URI hostname and checks tenant ownership
func (tdr *TenantDomainsRepository) NormalizeHostnameAndCheckOwnership(tenantID uuid.UUID, redirectURI string) (string, error) {
	// Parse redirect URI to extract host
	if !strings.HasPrefix(redirectURI, "http://") && !strings.HasPrefix(redirectURI, "https://") {
		return "", &InvalidRedirectURIError{Message: "invalid redirect URI: must start with http:// or https://"}
	}

	// Simple host extraction
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
		return "", &InvalidRedirectURIError{Message: "wildcard hosts not allowed"}
	}

	// Validate hostname format
	if err := validateHostnameFormat(host); err != nil {
		return "", err
	}

	// Check if domain is owned by tenant and is verified
	owned, err := tdr.IsDomainOwnedByTenant(tenantID, host)
	if err != nil {
		return "", err
	}
	if !owned {
		return "", &DomainOwnershipError{Hostname: host, TenantID: tenantID}
	}

	return host, nil
}

// validateHostnameFormat performs basic validation on hostname format
func validateHostnameFormat(hostname string) error {
	// Reject if empty
	if hostname == "" {
		return &InvalidRedirectURIError{Message: "empty hostname"}
	}

	// Reject if contains invalid characters (path separators, backslashes, wildcards, spaces)
	if strings.ContainsAny(hostname, "/\\* ") {
		return &InvalidRedirectURIError{Message: "invalid hostname characters"}
	}

	// Validate domain format (basic check: at least one dot, no consecutive dots, reasonable length)
	// This is a minimal check - real validation is done by DNS and DB
	if !strings.Contains(hostname, ".") || len(hostname) < 3 || len(hostname) > 253 {
		return &InvalidRedirectURIError{Message: "invalid hostname format"}
	}

	// Reject IP addresses in production (optional - can be configurable via env)
	isDev := os.Getenv("ENVIRONMENT") == "development" || os.Getenv("ENVIRONMENT") == ""
	if !isDev {
		if ip := net.ParseIP(hostname); ip != nil {
			return &InvalidRedirectURIError{Message: "IP addresses not allowed in redirect URIs"}
		}
	}

	return nil
}

// Custom error types

type InvalidRedirectURIError struct {
	Message string
}

func (e *InvalidRedirectURIError) Error() string {
	return e.Message
}

type DomainOwnershipError struct {
	Hostname string
	TenantID uuid.UUID
}

func (e *DomainOwnershipError) Error() string {
	return fmt.Sprintf("redirect URI host %s is not owned by tenant %s", e.Hostname, e.TenantID)
}

type RedirectURIValidationError struct {
	Errors []error
}

func (e *RedirectURIValidationError) Error() string {
	var msgs []string
	for _, err := range e.Errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}
