package database

import (
	"database/sql"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// TOTPRepository handles database operations for TOTP authentication
type TOTPRepository struct {
	db *sql.DB
}

// NewTOTPRepository creates a new TOTP repository
func NewTOTPRepository(db *sql.DB) *TOTPRepository {
	return &TOTPRepository{db: db}
}

// CreateTOTPSecret stores a new TOTP secret
func (r *TOTPRepository) CreateTOTPSecret(secret *models.TOTPSecret) error {
	if secret.ID == uuid.Nil {
		secret.ID = uuid.New()
	}
	now := time.Now().Unix()
	secret.CreatedAt = now
	secret.UpdatedAt = now

	query := `
		INSERT INTO totp_secrets (
			id, user_id, tenant_id, secret, device_name, device_type,
			is_active, is_primary, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(query,
		secret.ID, secret.UserID, secret.TenantID, secret.Secret,
		secret.DeviceName, secret.DeviceType, secret.IsActive,
		secret.IsPrimary, secret.CreatedAt, secret.UpdatedAt,
	)
	return err
}

// UpdateTOTPSecret updates a TOTP secret
func (r *TOTPRepository) UpdateTOTPSecret(secret *models.TOTPSecret) error {
	secret.UpdatedAt = time.Now().Unix()

	query := `
		UPDATE totp_secrets
		SET device_name = $1, is_active = $2, is_primary = $3, updated_at = $4
		WHERE id = $5 AND user_id = $6 AND tenant_id = $7`

	_, err := r.db.Exec(query,
		secret.DeviceName, secret.IsActive, secret.IsPrimary,
		secret.UpdatedAt, secret.ID, secret.UserID, secret.TenantID,
	)
	return err
}

// UpdateLastUsed updates the last_used timestamp
func (r *TOTPRepository) UpdateLastUsed(id uuid.UUID) error {
	now := time.Now().Unix()
	query := `UPDATE totp_secrets SET last_used = $1, updated_at = $1 WHERE id = $2`
	_, err := r.db.Exec(query, now, id)
	return err
}

// GetTOTPSecretByID retrieves a TOTP secret by ID
func (r *TOTPRepository) GetTOTPSecretByID(id uuid.UUID, userID uuid.UUID, tenantID uuid.UUID) (*models.TOTPSecret, error) {
	query := `SELECT id, user_id, tenant_id, secret, device_name, device_type, last_used,
	          is_active, is_primary, created_at, updated_at
	          FROM totp_secrets WHERE id = $1 AND user_id = $2 AND tenant_id = $3`

	row := r.db.QueryRow(query, id, userID, tenantID)
	return r.scanTOTPSecret(row)
}

// GetUserTOTPSecrets retrieves all active TOTP secrets for a user
func (r *TOTPRepository) GetUserTOTPSecrets(userID uuid.UUID, tenantID uuid.UUID) ([]models.TOTPSecret, error) {
	query := `SELECT id, user_id, tenant_id, secret, device_name, device_type, last_used,
	          is_active, is_primary, created_at, updated_at
	          FROM totp_secrets
	          WHERE user_id = $1 AND tenant_id = $2 AND is_active = true
	          ORDER BY is_primary DESC, created_at DESC`

	rows, err := r.db.Query(query, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []models.TOTPSecret
	for rows.Next() {
		s, err := r.scanTOTPSecret(rows)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, *s)
	}
	return secrets, nil
}

// GetPrimaryTOTPSecret retrieves the primary TOTP secret for a user
func (r *TOTPRepository) GetPrimaryTOTPSecret(userID uuid.UUID, tenantID uuid.UUID) (*models.TOTPSecret, error) {
	query := `SELECT id, user_id, tenant_id, secret, device_name, device_type, last_used,
	          is_active, is_primary, created_at, updated_at
	          FROM totp_secrets
	          WHERE user_id = $1 AND tenant_id = $2 AND is_active = true AND is_primary = true
	          LIMIT 1`

	row := r.db.QueryRow(query, userID, tenantID)
	secret, err := r.scanTOTPSecret(row)
	if err == sql.ErrNoRows {
		return nil, nil // No primary device
	}
	return secret, err
}

// DeleteTOTPSecret deletes a TOTP secret
func (r *TOTPRepository) DeleteTOTPSecret(id uuid.UUID, userID uuid.UUID, tenantID uuid.UUID) error {
	query := `DELETE FROM totp_secrets WHERE id = $1 AND user_id = $2 AND tenant_id = $3`
	result, err := r.db.Exec(query, id, userID, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SetPrimaryTOTPSecret sets a device as primary (unsets others)
func (r *TOTPRepository) SetPrimaryTOTPSecret(id uuid.UUID, userID uuid.UUID, tenantID uuid.UUID) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Unset all primary flags for user
	_, err = tx.Exec(`UPDATE totp_secrets SET is_primary = false
	                  WHERE user_id = $1 AND tenant_id = $2`, userID, tenantID)
	if err != nil {
		return err
	}

	// Set this device as primary
	_, err = tx.Exec(`UPDATE totp_secrets SET is_primary = true, updated_at = $1
	                  WHERE id = $2 AND user_id = $3 AND tenant_id = $4`, time.Now().Unix(), id, userID, tenantID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ============================
// Backup Codes
// ============================

// CreateBackupCode stores a new backup code
func (r *TOTPRepository) CreateBackupCode(code *models.BackupCode) error {
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	code.CreatedAt = time.Now().Unix()

	query := `
		INSERT INTO totp_backup_codes (id, user_id, tenant_id, code, is_used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.Exec(query, code.ID, code.UserID, code.TenantID, code.Code, code.IsUsed, code.CreatedAt)
	return err
}

// GetUserBackupCodes retrieves unused backup codes for a user
func (r *TOTPRepository) GetUserBackupCodes(userID uuid.UUID, tenantID uuid.UUID) ([]models.BackupCode, error) {
	query := `SELECT id, user_id, tenant_id, code, is_used, created_at, used_at
	          FROM totp_backup_codes
	          WHERE user_id = $1 AND tenant_id = $2
	          ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []models.BackupCode
	for rows.Next() {
		var c models.BackupCode
		err := rows.Scan(&c.ID, &c.UserID, &c.TenantID, &c.Code, &c.IsUsed, &c.CreatedAt, &c.UsedAt)
		if err != nil {
			return nil, err
		}
		codes = append(codes, c)
	}
	return codes, nil
}

// UseBackupCode marks a backup code as used
func (r *TOTPRepository) UseBackupCode(codeID uuid.UUID) error {
	now := time.Now().Unix()
	query := `UPDATE totp_backup_codes SET is_used = true, used_at = $1 WHERE id = $2`
	_, err := r.db.Exec(query, now, codeID)
	return err
}

// DeleteBackupCodes deletes all backup codes for a user
func (r *TOTPRepository) DeleteBackupCodes(userID uuid.UUID, tenantID uuid.UUID) error {
	query := `DELETE FROM totp_backup_codes WHERE user_id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(query, userID, tenantID)
	return err
}

// scanTOTPSecret scans a TOTP secret from a row
func (r *TOTPRepository) scanTOTPSecret(row scanner) (*models.TOTPSecret, error) {
	var s models.TOTPSecret
	err := row.Scan(
		&s.ID, &s.UserID, &s.TenantID, &s.Secret, &s.DeviceName, &s.DeviceType,
		&s.LastUsed, &s.IsActive, &s.IsPrimary, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// scanner is a common interface for scanning from sql.Row and sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}
