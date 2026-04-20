package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
)

// PostgresCertificateRepository implements the CertificateRepository interface
type PostgresCertificateRepository struct {
	db *sql.DB
}

// NewPostgresCertificateRepository creates a new certificate repository
func NewPostgresCertificateRepository(db *sql.DB) repositories.CertificateRepository {
	return &PostgresCertificateRepository{db: db}
}

// GetByID retrieves a certificate by ID
func (r *PostgresCertificateRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Certificate, error) {
	query := `
		SELECT id, tenant_id, workload_id, serial_number, COALESCE(sha256_fingerprint, '') as sha256_fingerprint, spiffe_id, cert_pem, ca_chain,
			issued_at, expires_at, revoked_at, status, issue_type, created_at
		FROM certificates
		WHERE id = $1 AND tenant_id = $2
	`

	return r.scanCertificate(ctx, query, id, tenantID)
}

// GetBySerialNumber retrieves a certificate by serial number
func (r *PostgresCertificateRepository) GetBySerialNumber(ctx context.Context, tenantID, serialNumber string) (*models.Certificate, error) {
	query := `
		SELECT id, tenant_id, workload_id, serial_number, COALESCE(sha256_fingerprint, '') as sha256_fingerprint, spiffe_id, cert_pem, ca_chain,
			issued_at, expires_at, revoked_at, status, issue_type, created_at
		FROM certificates
		WHERE serial_number = $1 AND tenant_id = $2
	`

	return r.scanCertificate(ctx, query, serialNumber, tenantID)
}

// GetActiveByWorkload retrieves the active certificate for a workload
func (r *PostgresCertificateRepository) GetActiveByWorkload(ctx context.Context, tenantID, workloadID string) (*models.Certificate, error) {
	query := `
		SELECT id, tenant_id, workload_id, serial_number, COALESCE(sha256_fingerprint, '') as sha256_fingerprint, spiffe_id, cert_pem, ca_chain,
			issued_at, expires_at, revoked_at, status, issue_type, created_at
		FROM certificates
		WHERE workload_id = $1 AND tenant_id = $2 AND status = 'active'
		ORDER BY issued_at DESC
		LIMIT 1
	`

	return r.scanCertificate(ctx, query, workloadID, tenantID)
}

// Create creates a new certificate record
func (r *PostgresCertificateRepository) Create(ctx context.Context, cert *models.Certificate) error {
	query := `
		INSERT INTO certificates (id, tenant_id, workload_id, serial_number, sha256_fingerprint, spiffe_id, cert_pem, ca_chain,
			issued_at, expires_at, status, issue_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	cert.CreatedAt = time.Now()

	caChainJSON, err := json.Marshal(cert.CAChain)
	if err != nil {
		return errors.NewInternalError("Failed to marshal CA chain", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		cert.ID,
		cert.TenantID,
		cert.WorkloadID,
		cert.SerialNumber,
		cert.SHA256Fingerprint,
		cert.SpiffeID,
		cert.CertPEM,
		caChainJSON,
		cert.IssuedAt,
		cert.ExpiresAt,
		cert.Status,
		cert.IssueType,
		cert.CreatedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to create certificate", err)
	}

	return nil
}

// Update updates a certificate record
func (r *PostgresCertificateRepository) Update(ctx context.Context, cert *models.Certificate) error {
	query := `
		UPDATE certificates
		SET status = $3, revoked_at = $4
		WHERE id = $1 AND tenant_id = $2
	`

	result, err := r.db.ExecContext(ctx, query,
		cert.ID,
		cert.TenantID,
		cert.Status,
		cert.RevokedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to update certificate", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get affected rows", err)
	}

	if rows == 0 {
		return errors.NewNotFoundError("Certificate not found", nil)
	}

	return nil
}

// Revoke marks a certificate as revoked
func (r *PostgresCertificateRepository) Revoke(ctx context.Context, tenantID, id string) error {
	query := `
		UPDATE certificates
		SET status = 'revoked', revoked_at = $3
		WHERE id = $1 AND tenant_id = $2
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, tenantID, now)
	if err != nil {
		return errors.NewInternalError("Failed to revoke certificate", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get affected rows", err)
	}

	if rows == 0 {
		return errors.NewNotFoundError("Certificate not found", nil)
	}

	return nil
}

// ListByWorkload retrieves all certificates for a workload
func (r *PostgresCertificateRepository) ListByWorkload(ctx context.Context, tenantID, workloadID string) ([]*models.Certificate, error) {
	query := `
		SELECT id, tenant_id, workload_id, serial_number, COALESCE(sha256_fingerprint, '') as sha256_fingerprint, spiffe_id, cert_pem, ca_chain,
			issued_at, expires_at, revoked_at, status, issue_type, created_at
		FROM certificates
		WHERE workload_id = $1 AND tenant_id = $2
		ORDER BY issued_at DESC
	`

	return r.queryCertificates(ctx, query, workloadID, tenantID)
}

// ListExpiring retrieves certificates expiring within the given duration
func (r *PostgresCertificateRepository) ListExpiring(ctx context.Context, tenantID string, within time.Duration) ([]*models.Certificate, error) {
	query := `
		SELECT id, tenant_id, workload_id, serial_number, COALESCE(sha256_fingerprint, '') as sha256_fingerprint, spiffe_id, cert_pem, ca_chain,
			issued_at, expires_at, revoked_at, status, issue_type, created_at
		FROM certificates
		WHERE tenant_id = $1 AND status = 'active' AND expires_at < $2
		ORDER BY expires_at ASC
	`

	expiryThreshold := time.Now().Add(within)
	return r.queryCertificates(ctx, query, tenantID, expiryThreshold)
}

// scanCertificate scans a single certificate
func (r *PostgresCertificateRepository) scanCertificate(ctx context.Context, query string, args ...interface{}) (*models.Certificate, error) {
	cert := &models.Certificate{}
	var caChainJSON []byte
	var revokedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&cert.ID,
		&cert.TenantID,
		&cert.WorkloadID,
		&cert.SerialNumber,
		&cert.SHA256Fingerprint,
		&cert.SpiffeID,
		&cert.CertPEM,
		&caChainJSON,
		&cert.IssuedAt,
		&cert.ExpiresAt,
		&revokedAt,
		&cert.Status,
		&cert.IssueType,
		&cert.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("Certificate not found", err)
	}
	if err != nil {
		return nil, errors.NewInternalError("Failed to get certificate", err)
	}

	if revokedAt.Valid {
		cert.RevokedAt = &revokedAt.Time
	}

	if err := json.Unmarshal(caChainJSON, &cert.CAChain); err != nil {
		return nil, errors.NewInternalError("Failed to unmarshal CA chain", err)
	}

	return cert, nil
}

// queryCertificates queries multiple certificates
func (r *PostgresCertificateRepository) queryCertificates(ctx context.Context, query string, args ...interface{}) ([]*models.Certificate, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewInternalError("Failed to query certificates", err)
	}
	defer rows.Close()

	var certs []*models.Certificate
	for rows.Next() {
		cert := &models.Certificate{}
		var caChainJSON []byte
		var revokedAt sql.NullTime

		err := rows.Scan(
			&cert.ID,
			&cert.TenantID,
			&cert.WorkloadID,
			&cert.SerialNumber,
			&cert.SHA256Fingerprint,
			&cert.SpiffeID,
			&cert.CertPEM,
			&caChainJSON,
			&cert.IssuedAt,
			&cert.ExpiresAt,
			&revokedAt,
			&cert.Status,
			&cert.IssueType,
			&cert.CreatedAt,
		)
		if err != nil {
			return nil, errors.NewInternalError("Failed to scan certificate", err)
		}

		if revokedAt.Valid {
			cert.RevokedAt = &revokedAt.Time
		}

		if err := json.Unmarshal(caChainJSON, &cert.CAChain); err != nil {
			return nil, errors.NewInternalError("Failed to unmarshal CA chain", err)
		}

		certs = append(certs, cert)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.NewInternalError("Error iterating certificates", err)
	}

	return certs, nil
}
