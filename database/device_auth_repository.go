package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// DeviceAuthRepository handles device authorization database operations
type DeviceAuthRepository struct {
	db *DBConnection
}

// NewDeviceAuthRepository creates a new device authorization repository
func NewDeviceAuthRepository(db *DBConnection) *DeviceAuthRepository {
	return &DeviceAuthRepository{db: db}
}

// CreateDeviceCode creates a new device authorization request
func (r *DeviceAuthRepository) CreateDeviceCode(deviceCode *models.DeviceCode) error {
	query := `
		INSERT INTO device_codes (
			id, tenant_id, client_id, device_code, user_code,
			verification_uri, verification_uri_complete,
			status, scopes, device_info, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	scopesJSON, err := json.Marshal(deviceCode.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	deviceInfoJSON, err := json.Marshal(deviceCode.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal device_info: %w", err)
	}

	now := time.Now().Unix()
	deviceCode.CreatedAt = now
	deviceCode.UpdatedAt = now

	_, err = r.db.Exec(query,
		deviceCode.ID,
		deviceCode.TenantID,
		deviceCode.ClientID,
		deviceCode.DeviceCode,
		deviceCode.UserCode,
		deviceCode.VerificationURI,
		deviceCode.VerificationURIComplete,
		deviceCode.Status,
		scopesJSON,
		deviceInfoJSON,
		deviceCode.ExpiresAt,
		deviceCode.CreatedAt,
		deviceCode.UpdatedAt,
	)

	return err
}

// FindByDeviceCode retrieves a device code by device_code
func (r *DeviceAuthRepository) FindByDeviceCode(deviceCode string) (*models.DeviceCode, error) {
	query := `
		SELECT id, tenant_id, client_id, device_code, user_code,
		       verification_uri, verification_uri_complete,
		       user_id, user_email, status, scopes, device_info,
		       expires_at, last_polled_at, authorized_at,
		       created_at, updated_at
		FROM device_codes
		WHERE device_code = $1
	`

	dc := &models.DeviceCode{}
	var clientID, userID sql.NullString
	var verificationURIComplete, userEmail sql.NullString
	var scopesJSON, deviceInfoJSON []byte
	var lastPolledAt, authorizedAt sql.NullInt64

	err := r.db.QueryRow(query, deviceCode).Scan(
		&dc.ID,
		&dc.TenantID,
		&clientID,
		&dc.DeviceCode,
		&dc.UserCode,
		&dc.VerificationURI,
		&verificationURIComplete,
		&userID,
		&userEmail,
		&dc.Status,
		&scopesJSON,
		&deviceInfoJSON,
		&dc.ExpiresAt,
		&lastPolledAt,
		&authorizedAt,
		&dc.CreatedAt,
		&dc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("device code not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if clientID.Valid {
		id := uuid.MustParse(clientID.String)
		dc.ClientID = &id
	}
	if userID.Valid {
		id := uuid.MustParse(userID.String)
		dc.UserID = &id
	}
	if verificationURIComplete.Valid {
		dc.VerificationURIComplete = verificationURIComplete.String
	}
	if userEmail.Valid {
		dc.UserEmail = userEmail.String
	}
	if lastPolledAt.Valid {
		dc.LastPolledAt = &lastPolledAt.Int64
	}
	if authorizedAt.Valid {
		dc.AuthorizedAt = &authorizedAt.Int64
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(scopesJSON, &dc.Scopes); err != nil {
		dc.Scopes = []string{}
	}
	if err := json.Unmarshal(deviceInfoJSON, &dc.DeviceInfo); err != nil {
		dc.DeviceInfo = make(map[string]interface{})
	}

	return dc, nil
}

// FindByUserCode retrieves a device code by user_code
func (r *DeviceAuthRepository) FindByUserCode(userCode string) (*models.DeviceCode, error) {
	query := `
		SELECT id, tenant_id, client_id, device_code, user_code,
		       verification_uri, verification_uri_complete,
		       user_id, user_email, status, scopes, device_info,
		       expires_at, last_polled_at, authorized_at,
		       created_at, updated_at
		FROM device_codes
		WHERE user_code = $1
	`

	dc := &models.DeviceCode{}
	var clientID, userID sql.NullString
	var verificationURIComplete, userEmail sql.NullString
	var scopesJSON, deviceInfoJSON []byte
	var lastPolledAt, authorizedAt sql.NullInt64

	err := r.db.QueryRow(query, userCode).Scan(
		&dc.ID,
		&dc.TenantID,
		&clientID,
		&dc.DeviceCode,
		&dc.UserCode,
		&dc.VerificationURI,
		&verificationURIComplete,
		&userID,
		&userEmail,
		&dc.Status,
		&scopesJSON,
		&deviceInfoJSON,
		&dc.ExpiresAt,
		&lastPolledAt,
		&authorizedAt,
		&dc.CreatedAt,
		&dc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user code not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if clientID.Valid {
		id := uuid.MustParse(clientID.String)
		dc.ClientID = &id
	}
	if userID.Valid {
		id := uuid.MustParse(userID.String)
		dc.UserID = &id
	}
	if verificationURIComplete.Valid {
		dc.VerificationURIComplete = verificationURIComplete.String
	}
	if userEmail.Valid {
		dc.UserEmail = userEmail.String
	}
	if lastPolledAt.Valid {
		dc.LastPolledAt = &lastPolledAt.Int64
	}
	if authorizedAt.Valid {
		dc.AuthorizedAt = &authorizedAt.Int64
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(scopesJSON, &dc.Scopes); err != nil {
		dc.Scopes = []string{}
	}
	if err := json.Unmarshal(deviceInfoJSON, &dc.DeviceInfo); err != nil {
		dc.DeviceInfo = make(map[string]interface{})
	}

	return dc, nil
}

// UpdateStatus updates the status of a device code
func (r *DeviceAuthRepository) UpdateStatus(deviceCode string, status string) error {
	query := `
		UPDATE device_codes
		SET status = $1, updated_at = $2
		WHERE device_code = $3
	`

	_, err := r.db.Exec(query, status, time.Now().Unix(), deviceCode)
	return err
}

// UpdateLastPolled updates the last_polled_at timestamp
func (r *DeviceAuthRepository) UpdateLastPolled(deviceCode string) error {
	query := `
		UPDATE device_codes
		SET last_polled_at = $1, updated_at = $2
		WHERE device_code = $3
	`

	now := time.Now().Unix()
	_, err := r.db.Exec(query, now, now, deviceCode)
	return err
}

// AuthorizeDeviceCode authorizes a device code with user information
func (r *DeviceAuthRepository) AuthorizeDeviceCode(userCode string, userID uuid.UUID, userEmail string, approve bool) error {
	status := "authorized"
	if !approve {
		status = "denied"
	}

	query := `
		UPDATE device_codes
		SET user_id = $1, user_email = $2, status = $3, authorized_at = $4, updated_at = $5
		WHERE user_code = $6 AND status = 'pending'
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, userID, userEmail, status, now, now, userCode)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device code not found or already processed")
	}

	return nil
}

// MarkAsConsumed marks a device code as consumed after token issuance
func (r *DeviceAuthRepository) MarkAsConsumed(deviceCode string) error {
	return r.UpdateStatus(deviceCode, "consumed")
}

// ExpireOldDeviceCodes marks expired device codes as expired
func (r *DeviceAuthRepository) ExpireOldDeviceCodes() (int64, error) {
	query := `
		UPDATE device_codes
		SET status = 'expired', updated_at = $1
		WHERE status = 'pending' AND expires_at < $2
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, now, now)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// DeleteExpiredDeviceCodes deletes old expired device codes for cleanup
func (r *DeviceAuthRepository) DeleteExpiredDeviceCodes(olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM device_codes
		WHERE status IN ('expired', 'consumed', 'denied')
		AND expires_at < $1
	`

	cutoff := time.Now().Add(-olderThan).Unix()
	result, err := r.db.Exec(query, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ========================================
// Code Generation Helpers
// ========================================

// GenerateDeviceCode generates a secure random device code (128 characters max)
func GenerateDeviceCode() (string, error) {
	bytes := make([]byte, 64) // 64 bytes = ~102 chars in base32 (well under 128)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)), nil
}

// GenerateUserCode generates a human-readable user code (e.g., WDJB-MJHT)
// Format: 4 chars - 4 chars (e.g., ABCD-1234 or WDJB-MJHT)
func GenerateUserCode() (string, error) {
	// Use base32 charset without ambiguous characters (0, O, I, L, 1)
	const charset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	code := make([]byte, 9) // 8 chars + 1 hyphen
	for i := 0; i < 4; i++ {
		code[i] = charset[int(bytes[i])%len(charset)]
	}
	code[4] = '-'
	for i := 5; i < 9; i++ {
		code[i] = charset[int(bytes[i-1])%len(charset)]
	}

	return string(code), nil
}

// ListPendingDeviceCodes returns all pending device codes for a tenant (optionally filtered by client_id)
// It also automatically marks expired codes as 'expired' before returning results
func (r *DeviceAuthRepository) ListPendingDeviceCodes(tenantID uuid.UUID, clientID *uuid.UUID) ([]models.DeviceCode, error) {
	now := time.Now().Unix()

	// First, auto-expire any expired pending codes for this tenant
	expireQuery := `
		UPDATE device_codes
		SET status = 'expired', updated_at = $1
		WHERE tenant_id = $2 AND status = 'pending' AND expires_at < $3
	`
	r.db.Exec(expireQuery, now, tenantID, now)

	var query string
	var args []interface{}

	if clientID != nil {
		query = `
			SELECT id, tenant_id, client_id, device_code, user_code,
			       verification_uri, verification_uri_complete,
			       user_id, user_email, status, scopes, device_info,
			       expires_at, last_polled_at, authorized_at,
			       created_at, updated_at
			FROM device_codes
			WHERE tenant_id = $1 AND client_id = $2 AND status = 'pending' AND expires_at > $3
			ORDER BY created_at DESC
		`
		args = []interface{}{tenantID, *clientID, now}
	} else {
		query = `
			SELECT id, tenant_id, client_id, device_code, user_code,
			       verification_uri, verification_uri_complete,
			       user_id, user_email, status, scopes, device_info,
			       expires_at, last_polled_at, authorized_at,
			       created_at, updated_at
			FROM device_codes
			WHERE tenant_id = $1 AND status = 'pending' AND expires_at > $2
			ORDER BY created_at DESC
		`
		args = []interface{}{tenantID, now}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []models.DeviceCode
	for rows.Next() {
		dc := models.DeviceCode{}
		var cID, uID sql.NullString
		var verificationURIComplete, userEmail sql.NullString
		var scopesJSON, deviceInfoJSON []byte
		var lastPolledAt, authorizedAt sql.NullInt64

		err := rows.Scan(
			&dc.ID,
			&dc.TenantID,
			&cID,
			&dc.DeviceCode,
			&dc.UserCode,
			&dc.VerificationURI,
			&verificationURIComplete,
			&uID,
			&userEmail,
			&dc.Status,
			&scopesJSON,
			&deviceInfoJSON,
			&dc.ExpiresAt,
			&lastPolledAt,
			&authorizedAt,
			&dc.CreatedAt,
			&dc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if cID.Valid {
			id := uuid.MustParse(cID.String)
			dc.ClientID = &id
		}
		if uID.Valid {
			id := uuid.MustParse(uID.String)
			dc.UserID = &id
		}
		if verificationURIComplete.Valid {
			dc.VerificationURIComplete = verificationURIComplete.String
		}
		if userEmail.Valid {
			dc.UserEmail = userEmail.String
		}
		if lastPolledAt.Valid {
			dc.LastPolledAt = &lastPolledAt.Int64
		}
		if authorizedAt.Valid {
			dc.AuthorizedAt = &authorizedAt.Int64
		}
		if err := json.Unmarshal(scopesJSON, &dc.Scopes); err != nil {
			dc.Scopes = []string{}
		}
		if err := json.Unmarshal(deviceInfoJSON, &dc.DeviceInfo); err != nil {
			dc.DeviceInfo = make(map[string]interface{})
		}

		codes = append(codes, dc)
	}

	return codes, nil
}
