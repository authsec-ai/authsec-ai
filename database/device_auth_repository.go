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

// CreateDeviceCode creates a new device authorization request.
// TenantID may be nil when the CLI initiates the flow — it is populated during /authorize.
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
		deviceCode.TenantID, // nullable *uuid.UUID — driver handles nil → NULL
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
		       user_id, user_email, tenant_domain, access_token, status, scopes, device_info,
		       expires_at, last_polled_at, authorized_at,
		       created_at, updated_at
		FROM device_codes
		WHERE device_code = $1
	`
	return r.scanDeviceCode(r.db.QueryRow(query, deviceCode))
}

// FindByUserCode retrieves a device code by user_code.
// Accepts both "ABCD-1234" (with hyphen) and "ABCD1234" (without) — normalizes via SQL.
func (r *DeviceAuthRepository) FindByUserCode(userCode string) (*models.DeviceCode, error) {
	query := `
		SELECT id, tenant_id, client_id, device_code, user_code,
		       verification_uri, verification_uri_complete,
		       user_id, user_email, tenant_domain, access_token, status, scopes, device_info,
		       expires_at, last_polled_at, authorized_at,
		       created_at, updated_at
		FROM device_codes
		WHERE REPLACE(user_code, '-', '') = REPLACE($1, '-', '')
	`
	return r.scanDeviceCode(r.db.QueryRow(query, userCode))
}

// scanDeviceCode scans a single device_codes row into a DeviceCode model.
// Handles all nullable columns (tenant_id, client_id, user_id, tenant_domain, access_token, etc.).
func (r *DeviceAuthRepository) scanDeviceCode(row *sql.Row) (*models.DeviceCode, error) {
	dc := &models.DeviceCode{}
	var tenantID, clientID, userID sql.NullString
	var verificationURIComplete, userEmail, tenantDomain, accessToken sql.NullString
	var scopesJSON, deviceInfoJSON []byte
	var lastPolledAt, authorizedAt sql.NullInt64

	err := row.Scan(
		&dc.ID,
		&tenantID,
		&clientID,
		&dc.DeviceCode,
		&dc.UserCode,
		&dc.VerificationURI,
		&verificationURIComplete,
		&userID,
		&userEmail,
		&tenantDomain,
		&accessToken,
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

	if tenantID.Valid {
		id := uuid.MustParse(tenantID.String)
		dc.TenantID = &id
	}
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
	if tenantDomain.Valid {
		dc.TenantDomain = tenantDomain.String
	}
	if accessToken.Valid {
		dc.AccessToken = accessToken.String
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

// UpdateLastPolled updates the last_polled_at timestamp.
// Returns (tooSoon bool, err error).
// tooSoon is true if the previous poll was within the minInterval window (rate limit enforced in DB).
func (r *DeviceAuthRepository) UpdateLastPolled(deviceCode string, minIntervalSeconds int64) (tooSoon bool, err error) {
	// Use a single UPDATE … RETURNING to atomically check interval and update
	query := `
		UPDATE device_codes
		SET last_polled_at = $1, updated_at = $2
		WHERE device_code = $3
		  AND (last_polled_at IS NULL OR last_polled_at <= $4)
		RETURNING id
	`
	now := time.Now().Unix()
	cutoff := now - minIntervalSeconds

	var id uuid.UUID
	scanErr := r.db.QueryRow(query, now, now, deviceCode, cutoff).Scan(&id)
	if scanErr == sql.ErrNoRows {
		// Row exists but last_polled_at is too recent → rate limited
		return true, nil
	}
	if scanErr != nil {
		return false, scanErr
	}
	return false, nil
}

// AuthorizeDeviceCode authorizes or denies a device code with full user + tenant context.
// accessToken is the pre-generated JWT; stored here so /token poll can return it directly.
// tenantID, tenantDomain, and clientID come from the authenticated browser session.
func (r *DeviceAuthRepository) AuthorizeDeviceCode(
	userCode string,
	userID uuid.UUID,
	userEmail string,
	tenantID uuid.UUID,
	tenantDomain string,
	clientID *uuid.UUID,
	accessToken string,
	approve bool,
) error {
	status := "authorized"
	if !approve {
		status = "denied"
	}

	query := `
		UPDATE device_codes
		SET user_id       = $1,
		    user_email    = $2,
		    tenant_id     = $3,
		    tenant_domain = $4,
		    client_id     = $5,
		    access_token  = $6,
		    status        = $7,
		    authorized_at = $8,
		    updated_at    = $9
		WHERE REPLACE(user_code, '-', '') = REPLACE($10, '-', '') AND status = 'pending'
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, userID, userEmail, tenantID, tenantDomain, clientID, accessToken, status, now, now, userCode)
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

// ListPendingDeviceCodes returns all pending device codes for a tenant (optionally filtered by client_id).
// Also auto-expires stale pending codes before returning.
func (r *DeviceAuthRepository) ListPendingDeviceCodes(tenantID uuid.UUID, clientID *uuid.UUID) ([]models.DeviceCode, error) {
	now := time.Now().Unix()

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
			       user_id, user_email, tenant_domain, status, scopes, device_info,
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
			       user_id, user_email, tenant_domain, status, scopes, device_info,
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
		var tID, cID, uID sql.NullString
		var verificationURIComplete, userEmail, tenantDomain sql.NullString
		var scopesJSON, deviceInfoJSON []byte
		var lastPolledAt, authorizedAt sql.NullInt64

		err := rows.Scan(
			&dc.ID,
			&tID,
			&cID,
			&dc.DeviceCode,
			&dc.UserCode,
			&dc.VerificationURI,
			&verificationURIComplete,
			&uID,
			&userEmail,
			&tenantDomain,
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

		if tID.Valid {
			id := uuid.MustParse(tID.String)
			dc.TenantID = &id
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
		if tenantDomain.Valid {
			dc.TenantDomain = tenantDomain.String
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

// ========================================
// Code Generation Helpers
// ========================================

// GenerateDeviceCode generates a secure 32-byte random device code (hex-encoded, 64 chars).
func GenerateDeviceCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// GenerateUserCode generates a human-readable user code (e.g., WDJB-MJHT).
// Format: 4 chars hyphen 4 chars, using an unambiguous charset (no 0/O/I/L/1).
func GenerateUserCode() (string, error) {
	const charset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	code := make([]byte, 9) // 8 chars + hyphen
	for i := 0; i < 4; i++ {
		code[i] = charset[int(b[i])%len(charset)]
	}
	code[4] = '-'
	for i := 5; i < 9; i++ {
		code[i] = charset[int(b[i-1])%len(charset)]
	}
	return string(code), nil
}

// GenerateRefreshToken generates a secure opaque refresh token (32 bytes, base32 encoded).
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)), nil
}
