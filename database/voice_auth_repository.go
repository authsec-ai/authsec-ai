package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// VoiceAuthRepository handles voice authentication database operations
type VoiceAuthRepository struct {
	db *DBConnection
}

// NewVoiceAuthRepository creates a new voice authentication repository
func NewVoiceAuthRepository(db *DBConnection) *VoiceAuthRepository {
	return &VoiceAuthRepository{db: db}
}

// ========================================
// Voice Session Operations
// ========================================

// CreateVoiceSession creates a new voice authentication session
func (r *VoiceAuthRepository) CreateVoiceSession(session *models.VoiceSession) error {
	query := `
		INSERT INTO voice_sessions (
			id, tenant_id, client_id, session_token, voice_otp,
			otp_attempts, voice_platform, voice_user_id, device_info,
			status, scopes, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	scopesJSON, err := json.Marshal(session.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	deviceInfoJSON, err := json.Marshal(session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal device_info: %w", err)
	}

	now := time.Now().Unix()
	session.CreatedAt = now
	session.UpdatedAt = now

	_, err = r.db.Exec(query,
		session.ID,
		session.TenantID,
		session.ClientID,
		session.SessionToken,
		session.VoiceOTP,
		session.OTPAttempts,
		session.VoicePlatform,
		session.VoiceUserID,
		deviceInfoJSON,
		session.Status,
		scopesJSON,
		session.ExpiresAt,
		session.CreatedAt,
		session.UpdatedAt,
	)

	return err
}

// FindVoiceSessionByToken retrieves a voice session by session_token
func (r *VoiceAuthRepository) FindVoiceSessionByToken(sessionToken string) (*models.VoiceSession, error) {
	query := `
		SELECT id, tenant_id, client_id, session_token, voice_otp,
		       otp_attempts, voice_platform, voice_user_id, device_info,
		       user_id, user_email, status, linked_device_code, scopes,
		       expires_at, verified_at, created_at, updated_at
		FROM voice_sessions
		WHERE session_token = $1
	`

	vs := &models.VoiceSession{}
	var clientID, userID sql.NullString
	var voicePlatform, voiceUserID, userEmail, linkedDeviceCode sql.NullString
	var deviceInfoJSON, scopesJSON []byte
	var verifiedAt sql.NullInt64

	err := r.db.QueryRow(query, sessionToken).Scan(
		&vs.ID,
		&vs.TenantID,
		&clientID,
		&vs.SessionToken,
		&vs.VoiceOTP,
		&vs.OTPAttempts,
		&voicePlatform,
		&voiceUserID,
		&deviceInfoJSON,
		&userID,
		&userEmail,
		&vs.Status,
		&linkedDeviceCode,
		&scopesJSON,
		&vs.ExpiresAt,
		&verifiedAt,
		&vs.CreatedAt,
		&vs.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("voice session not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if clientID.Valid {
		id := uuid.MustParse(clientID.String)
		vs.ClientID = &id
	}
	if userID.Valid {
		id := uuid.MustParse(userID.String)
		vs.UserID = &id
	}
	if voicePlatform.Valid {
		vs.VoicePlatform = voicePlatform.String
	}
	if voiceUserID.Valid {
		vs.VoiceUserID = voiceUserID.String
	}
	if userEmail.Valid {
		vs.UserEmail = userEmail.String
	}
	if linkedDeviceCode.Valid {
		vs.LinkedDeviceCode = linkedDeviceCode.String
	}
	if verifiedAt.Valid {
		vs.VerifiedAt = &verifiedAt.Int64
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(deviceInfoJSON, &vs.DeviceInfo); err != nil {
		vs.DeviceInfo = make(map[string]interface{})
	}
	if err := json.Unmarshal(scopesJSON, &vs.Scopes); err != nil {
		vs.Scopes = []string{}
	}

	return vs, nil
}

// UpdateVoiceSessionStatus updates the status of a voice session
func (r *VoiceAuthRepository) UpdateVoiceSessionStatus(sessionToken string, status string) error {
	query := `
		UPDATE voice_sessions
		SET status = $1, updated_at = $2
		WHERE session_token = $3
	`

	_, err := r.db.Exec(query, status, time.Now().Unix(), sessionToken)
	return err
}

// IncrementOTPAttempts increments the OTP verification attempts
func (r *VoiceAuthRepository) IncrementOTPAttempts(sessionToken string) error {
	query := `
		UPDATE voice_sessions
		SET otp_attempts = otp_attempts + 1, updated_at = $1
		WHERE session_token = $2
	`

	_, err := r.db.Exec(query, time.Now().Unix(), sessionToken)
	return err
}

// VerifyVoiceSession marks a voice session as verified
func (r *VoiceAuthRepository) VerifyVoiceSession(sessionToken string, userID *uuid.UUID, userEmail string) error {
	query := `
		UPDATE voice_sessions
		SET status = 'verified', user_id = $1, user_email = $2, verified_at = $3, updated_at = $4
		WHERE session_token = $5 AND status = 'initiated'
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, userID, userEmail, now, now, sessionToken)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("voice session not found or already processed")
	}

	return nil
}

// LinkDeviceCode links a voice session to a device authorization code
func (r *VoiceAuthRepository) LinkDeviceCode(sessionToken string, deviceCode string) error {
	query := `
		UPDATE voice_sessions
		SET linked_device_code = $1, updated_at = $2
		WHERE session_token = $3
	`

	_, err := r.db.Exec(query, deviceCode, time.Now().Unix(), sessionToken)
	return err
}

// ExpireOldVoiceSessions marks expired voice sessions as expired
func (r *VoiceAuthRepository) ExpireOldVoiceSessions() (int64, error) {
	query := `
		UPDATE voice_sessions
		SET status = 'expired', updated_at = $1
		WHERE status = 'initiated' AND expires_at < $2
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, now, now)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// DeleteExpiredVoiceSessions deletes old expired voice sessions for cleanup
func (r *VoiceAuthRepository) DeleteExpiredVoiceSessions(olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM voice_sessions
		WHERE status IN ('expired', 'failed', 'verified')
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
// Voice Identity Link Operations
// ========================================

// CreateVoiceIdentityLink creates a new voice identity link
func (r *VoiceAuthRepository) CreateVoiceIdentityLink(link *models.VoiceIdentityLink) error {
	query := `
		INSERT INTO voice_identity_links (
			id, tenant_id, voice_platform, voice_user_id, voice_user_name,
			user_id, user_email, is_active, link_method, linked_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	now := time.Now().Unix()
	link.CreatedAt = now
	link.UpdatedAt = now
	link.LinkedAt = now

	_, err := r.db.Exec(query,
		link.ID,
		link.TenantID,
		link.VoicePlatform,
		link.VoiceUserID,
		link.VoiceUserName,
		link.UserID,
		link.UserEmail,
		link.IsActive,
		link.LinkMethod,
		link.LinkedAt,
		link.CreatedAt,
		link.UpdatedAt,
	)

	return err
}

// FindVoiceIdentityLink retrieves a voice identity link
func (r *VoiceAuthRepository) FindVoiceIdentityLink(tenantID uuid.UUID, voicePlatform string, voiceUserID string) (*models.VoiceIdentityLink, error) {
	query := `
		SELECT id, tenant_id, voice_platform, voice_user_id, voice_user_name,
		       user_id, user_email, is_active, link_method, last_used_at,
		       linked_at, created_at, updated_at
		FROM voice_identity_links
		WHERE tenant_id = $1 AND voice_platform = $2 AND voice_user_id = $3
	`

	link := &models.VoiceIdentityLink{}
	var voiceUserName, linkMethod sql.NullString
	var lastUsedAt sql.NullInt64

	err := r.db.QueryRow(query, tenantID, voicePlatform, voiceUserID).Scan(
		&link.ID,
		&link.TenantID,
		&link.VoicePlatform,
		&link.VoiceUserID,
		&voiceUserName,
		&link.UserID,
		&link.UserEmail,
		&link.IsActive,
		&linkMethod,
		&lastUsedAt,
		&link.LinkedAt,
		&link.CreatedAt,
		&link.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("voice identity link not found")
		}
		return nil, err
	}

	if voiceUserName.Valid {
		link.VoiceUserName = voiceUserName.String
	}
	if linkMethod.Valid {
		link.LinkMethod = linkMethod.String
	}
	if lastUsedAt.Valid {
		link.LastUsedAt = &lastUsedAt.Int64
	}

	return link, nil
}

// ListVoiceIdentityLinks lists all voice identity links for a user
func (r *VoiceAuthRepository) ListVoiceIdentityLinks(tenantID uuid.UUID, userID uuid.UUID) ([]models.VoiceIdentityLink, error) {
	query := `
		SELECT id, tenant_id, voice_platform, voice_user_id, voice_user_name,
		       user_id, user_email, is_active, link_method, last_used_at,
		       linked_at, created_at, updated_at
		FROM voice_identity_links
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY linked_at DESC
	`

	rows, err := r.db.Query(query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.VoiceIdentityLink
	for rows.Next() {
		var link models.VoiceIdentityLink
		var voiceUserName, linkMethod sql.NullString
		var lastUsedAt sql.NullInt64

		err := rows.Scan(
			&link.ID,
			&link.TenantID,
			&link.VoicePlatform,
			&link.VoiceUserID,
			&voiceUserName,
			&link.UserID,
			&link.UserEmail,
			&link.IsActive,
			&linkMethod,
			&lastUsedAt,
			&link.LinkedAt,
			&link.CreatedAt,
			&link.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if voiceUserName.Valid {
			link.VoiceUserName = voiceUserName.String
		}
		if linkMethod.Valid {
			link.LinkMethod = linkMethod.String
		}
		if lastUsedAt.Valid {
			link.LastUsedAt = &lastUsedAt.Int64
		}

		links = append(links, link)
	}

	return links, nil
}

// UpdateVoiceIdentityLinkLastUsed updates the last_used_at timestamp
func (r *VoiceAuthRepository) UpdateVoiceIdentityLinkLastUsed(linkID uuid.UUID) error {
	query := `
		UPDATE voice_identity_links
		SET last_used_at = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now().Unix()
	_, err := r.db.Exec(query, now, now, linkID)
	return err
}

// DeactivateVoiceIdentityLink deactivates a voice identity link
func (r *VoiceAuthRepository) DeactivateVoiceIdentityLink(tenantID uuid.UUID, voicePlatform string, voiceUserID string) error {
	query := `
		UPDATE voice_identity_links
		SET is_active = false, updated_at = $1
		WHERE tenant_id = $2 AND voice_platform = $3 AND voice_user_id = $4
	`

	result, err := r.db.Exec(query, time.Now().Unix(), tenantID, voicePlatform, voiceUserID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("voice identity link not found")
	}

	return nil
}

// DeleteVoiceIdentityLink permanently deletes a voice identity link
func (r *VoiceAuthRepository) DeleteVoiceIdentityLink(linkID uuid.UUID) error {
	query := `DELETE FROM voice_identity_links WHERE id = $1`

	result, err := r.db.Exec(query, linkID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("voice identity link not found")
	}

	return nil
}

// ========================================
// Code Generation Helpers
// ========================================

// GenerateSessionToken generates a secure random session token (128 characters max)
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 64) // 64 bytes = ~102 chars in base32 (well under 128)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)), nil
}

// GenerateVoiceOTP generates a 4-digit numeric OTP for voice
// Returns a string like "8532" which can be spoken as "eight-five-three-two"
func GenerateVoiceOTP() (string, error) {
	// Generate a random number between 0000 and 9999
	max := big.NewInt(10000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// Format as 4-digit string with leading zeros
	return fmt.Sprintf("%04d", n.Int64()), nil
}

// ========================================
// Voice Active Session Operations
// ========================================

// CreateVoiceActiveSession creates a new active session record
func (r *VoiceAuthRepository) CreateVoiceActiveSession(session *models.VoiceActiveSession) error {
	query := `
		INSERT INTO voice_active_sessions (
			id, tenant_id, client_id, user_id, user_email, session_id,
			voice_platform, voice_user_id, device_info, device_name,
			access_token_hash, refresh_token_hash,
			login_at, last_activity_at, expires_at,
			is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	deviceInfoJSON, err := json.Marshal(session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal device_info: %w", err)
	}

	now := time.Now().Unix()
	session.CreatedAt = now
	session.UpdatedAt = now
	if session.LoginAt == 0 {
		session.LoginAt = now
	}
	if session.LastActivityAt == 0 {
		session.LastActivityAt = now
	}

	_, err = r.db.Exec(query,
		session.ID,
		session.TenantID,
		session.ClientID,
		session.UserID,
		session.UserEmail,
		session.SessionID,
		session.VoicePlatform,
		session.VoiceUserID,
		deviceInfoJSON,
		session.DeviceName,
		session.AccessTokenHash,
		session.RefreshTokenHash,
		session.LoginAt,
		session.LastActivityAt,
		session.ExpiresAt,
		session.IsActive,
		session.CreatedAt,
		session.UpdatedAt,
	)

	return err
}

// FindVoiceActiveSessionByID retrieves an active session by ID
func (r *VoiceAuthRepository) FindVoiceActiveSessionByID(sessionID uuid.UUID) (*models.VoiceActiveSession, error) {
	query := `
		SELECT id, tenant_id, client_id, user_id, user_email, session_id,
		       voice_platform, voice_user_id, device_info, device_name,
		       access_token_hash, refresh_token_hash,
		       login_at, last_activity_at, expires_at,
		       is_active, revoked_at, revoked_reason, created_at, updated_at
		FROM voice_active_sessions
		WHERE id = $1
	`

	return r.scanVoiceActiveSession(r.db.QueryRow(query, sessionID))
}

// FindVoiceActiveSessionBySessionID retrieves an active session by session_id (jti)
func (r *VoiceAuthRepository) FindVoiceActiveSessionBySessionID(sessionID string) (*models.VoiceActiveSession, error) {
	query := `
		SELECT id, tenant_id, client_id, user_id, user_email, session_id,
		       voice_platform, voice_user_id, device_info, device_name,
		       access_token_hash, refresh_token_hash,
		       login_at, last_activity_at, expires_at,
		       is_active, revoked_at, revoked_reason, created_at, updated_at
		FROM voice_active_sessions
		WHERE session_id = $1
	`

	return r.scanVoiceActiveSession(r.db.QueryRow(query, sessionID))
}

// FindVoiceActiveSessionByTokenHash retrieves an active session by token hash
func (r *VoiceAuthRepository) FindVoiceActiveSessionByTokenHash(tokenHash string) (*models.VoiceActiveSession, error) {
	query := `
		SELECT id, tenant_id, client_id, user_id, user_email, session_id,
		       voice_platform, voice_user_id, device_info, device_name,
		       access_token_hash, refresh_token_hash,
		       login_at, last_activity_at, expires_at,
		       is_active, revoked_at, revoked_reason, created_at, updated_at
		FROM voice_active_sessions
		WHERE access_token_hash = $1
	`

	return r.scanVoiceActiveSession(r.db.QueryRow(query, tokenHash))
}

// ListVoiceActiveSessions lists all active sessions for a user
func (r *VoiceAuthRepository) ListVoiceActiveSessions(tenantID uuid.UUID, userID uuid.UUID) ([]models.VoiceActiveSession, error) {
	query := `
		SELECT id, tenant_id, client_id, user_id, user_email, session_id,
		       voice_platform, voice_user_id, device_info, device_name,
		       access_token_hash, refresh_token_hash,
		       login_at, last_activity_at, expires_at,
		       is_active, revoked_at, revoked_reason, created_at, updated_at
		FROM voice_active_sessions
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY login_at DESC
	`

	rows, err := r.db.Query(query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.VoiceActiveSession
	for rows.Next() {
		session, err := r.scanVoiceActiveSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}

	return sessions, nil
}

// ListActiveVoiceSessions lists only active (non-revoked, non-expired) sessions
func (r *VoiceAuthRepository) ListActiveVoiceSessions(tenantID uuid.UUID, userID uuid.UUID) ([]models.VoiceActiveSession, error) {
	now := time.Now().Unix()
	query := `
		SELECT id, tenant_id, client_id, user_id, user_email, session_id,
		       voice_platform, voice_user_id, device_info, device_name,
		       access_token_hash, refresh_token_hash,
		       login_at, last_activity_at, expires_at,
		       is_active, revoked_at, revoked_reason, created_at, updated_at
		FROM voice_active_sessions
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true AND expires_at > $3
		ORDER BY login_at DESC
	`

	rows, err := r.db.Query(query, tenantID, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.VoiceActiveSession
	for rows.Next() {
		session, err := r.scanVoiceActiveSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}

	return sessions, nil
}

// RevokeVoiceActiveSession revokes a specific session
func (r *VoiceAuthRepository) RevokeVoiceActiveSession(sessionID string, reason string) error {
	query := `
		UPDATE voice_active_sessions
		SET is_active = false, revoked_at = $1, revoked_reason = $2, updated_at = $3
		WHERE session_id = $4 AND is_active = true
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, now, reason, now, sessionID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found or already revoked")
	}

	return nil
}

// RevokeAllVoiceActiveSessions revokes all active sessions for a user
func (r *VoiceAuthRepository) RevokeAllVoiceActiveSessions(tenantID uuid.UUID, userID uuid.UUID, reason string, exceptSessionID string) (int64, error) {
	query := `
		UPDATE voice_active_sessions
		SET is_active = false, revoked_at = $1, revoked_reason = $2, updated_at = $3
		WHERE tenant_id = $4 AND user_id = $5 AND is_active = true AND session_id != $6
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, now, reason, now, tenantID, userID, exceptSessionID)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// UpdateVoiceActiveSessionActivity updates the last_activity_at timestamp
func (r *VoiceAuthRepository) UpdateVoiceActiveSessionActivity(sessionID string) error {
	query := `
		UPDATE voice_active_sessions
		SET last_activity_at = $1, updated_at = $2
		WHERE session_id = $3 AND is_active = true
	`

	now := time.Now().Unix()
	_, err := r.db.Exec(query, now, now, sessionID)
	return err
}

// IsSessionRevoked checks if a session is revoked
func (r *VoiceAuthRepository) IsSessionRevoked(sessionID string) (bool, error) {
	query := `
		SELECT is_active, expires_at FROM voice_active_sessions WHERE session_id = $1
	`

	var isActive bool
	var expiresAt int64
	err := r.db.QueryRow(query, sessionID).Scan(&isActive, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil // Session not found = treated as revoked
		}
		return false, err
	}

	// Session is revoked if not active OR expired
	return !isActive || time.Now().Unix() > expiresAt, nil
}

// CountActiveSessionsForUser counts active sessions for a user
func (r *VoiceAuthRepository) CountActiveSessionsForUser(tenantID uuid.UUID, userID uuid.UUID) (int, error) {
	now := time.Now().Unix()
	query := `
		SELECT COUNT(*) FROM voice_active_sessions
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true AND expires_at > $3
	`

	var count int
	err := r.db.QueryRow(query, tenantID, userID, now).Scan(&count)
	return count, err
}

// CleanupExpiredVoiceActiveSessions marks expired sessions as inactive
func (r *VoiceAuthRepository) CleanupExpiredVoiceActiveSessions() (int64, error) {
	query := `
		UPDATE voice_active_sessions
		SET is_active = false, revoked_at = $1, revoked_reason = 'expired', updated_at = $2
		WHERE is_active = true AND expires_at < $3
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, now, now, now)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// DeleteOldVoiceActiveSessions deletes old inactive sessions
func (r *VoiceAuthRepository) DeleteOldVoiceActiveSessions(olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM voice_active_sessions
		WHERE is_active = false AND revoked_at < $1
	`

	cutoff := time.Now().Add(-olderThan).Unix()
	result, err := r.db.Exec(query, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Helper function to scan a voice active session from a row
func (r *VoiceAuthRepository) scanVoiceActiveSession(row *sql.Row) (*models.VoiceActiveSession, error) {
	session := &models.VoiceActiveSession{}
	var clientID sql.NullString
	var voicePlatform, voiceUserID, deviceName sql.NullString
	var deviceInfoJSON []byte
	var accessTokenHash, refreshTokenHash sql.NullString
	var revokedAt sql.NullInt64
	var revokedReason sql.NullString

	err := row.Scan(
		&session.ID,
		&session.TenantID,
		&clientID,
		&session.UserID,
		&session.UserEmail,
		&session.SessionID,
		&voicePlatform,
		&voiceUserID,
		&deviceInfoJSON,
		&deviceName,
		&accessTokenHash,
		&refreshTokenHash,
		&session.LoginAt,
		&session.LastActivityAt,
		&session.ExpiresAt,
		&session.IsActive,
		&revokedAt,
		&revokedReason,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("voice active session not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if clientID.Valid {
		id := uuid.MustParse(clientID.String)
		session.ClientID = &id
	}
	if voicePlatform.Valid {
		session.VoicePlatform = voicePlatform.String
	}
	if voiceUserID.Valid {
		session.VoiceUserID = voiceUserID.String
	}
	if deviceName.Valid {
		session.DeviceName = deviceName.String
	}
	if accessTokenHash.Valid {
		session.AccessTokenHash = accessTokenHash.String
	}
	if refreshTokenHash.Valid {
		session.RefreshTokenHash = refreshTokenHash.String
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Int64
	}
	if revokedReason.Valid {
		session.RevokedReason = revokedReason.String
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
		session.DeviceInfo = make(map[string]interface{})
	}

	return session, nil
}

// Helper function to scan a voice active session from rows
func (r *VoiceAuthRepository) scanVoiceActiveSessionRow(rows *sql.Rows) (*models.VoiceActiveSession, error) {
	session := &models.VoiceActiveSession{}
	var clientID sql.NullString
	var voicePlatform, voiceUserID, deviceName sql.NullString
	var deviceInfoJSON []byte
	var accessTokenHash, refreshTokenHash sql.NullString
	var revokedAt sql.NullInt64
	var revokedReason sql.NullString

	err := rows.Scan(
		&session.ID,
		&session.TenantID,
		&clientID,
		&session.UserID,
		&session.UserEmail,
		&session.SessionID,
		&voicePlatform,
		&voiceUserID,
		&deviceInfoJSON,
		&deviceName,
		&accessTokenHash,
		&refreshTokenHash,
		&session.LoginAt,
		&session.LastActivityAt,
		&session.ExpiresAt,
		&session.IsActive,
		&revokedAt,
		&revokedReason,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if clientID.Valid {
		id := uuid.MustParse(clientID.String)
		session.ClientID = &id
	}
	if voicePlatform.Valid {
		session.VoicePlatform = voicePlatform.String
	}
	if voiceUserID.Valid {
		session.VoiceUserID = voiceUserID.String
	}
	if deviceName.Valid {
		session.DeviceName = deviceName.String
	}
	if accessTokenHash.Valid {
		session.AccessTokenHash = accessTokenHash.String
	}
	if refreshTokenHash.Valid {
		session.RefreshTokenHash = refreshTokenHash.String
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Int64
	}
	if revokedReason.Valid {
		session.RevokedReason = revokedReason.String
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
		session.DeviceInfo = make(map[string]interface{})
	}

	return session, nil
}

// ========================================
// Pending Voice Auth Request Operations
// ========================================

// SetVoiceSessionPendingApproval marks a voice session as pending approval
func (r *VoiceAuthRepository) SetVoiceSessionPendingApproval(sessionToken string, pending bool) error {
	query := `
		UPDATE voice_sessions
		SET pending_approval = $1, approval_status = $2, updated_at = $3
		WHERE session_token = $4
	`

	status := ""
	if pending {
		status = "pending"
	}

	_, err := r.db.Exec(query, pending, status, time.Now().Unix(), sessionToken)
	return err
}

// ApproveVoiceSession approves or denies a pending voice session
func (r *VoiceAuthRepository) ApproveVoiceSession(sessionToken string, approve bool, approverUserID uuid.UUID) error {
	status := "denied"
	if approve {
		status = "approved"
	}

	query := `
		UPDATE voice_sessions
		SET pending_approval = false, approval_status = $1, approved_at = $2, approved_by = $3, updated_at = $4
		WHERE session_token = $5 AND pending_approval = true
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(query, status, now, approverUserID, now, sessionToken)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("voice session not found or not pending approval")
	}

	return nil
}

// ListPendingVoiceSessions lists pending voice auth requests for a tenant
func (r *VoiceAuthRepository) ListPendingVoiceSessions(tenantID uuid.UUID) ([]models.VoiceSession, error) {
	now := time.Now().Unix()
	query := `
		SELECT id, tenant_id, client_id, session_token, voice_otp,
		       otp_attempts, voice_platform, voice_user_id, device_info,
		       user_id, user_email, status, linked_device_code, scopes,
		       expires_at, verified_at, created_at, updated_at
		FROM voice_sessions
		WHERE tenant_id = $1 AND pending_approval = true AND approval_status = 'pending' AND expires_at > $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, tenantID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.VoiceSession
	for rows.Next() {
		vs := models.VoiceSession{}
		var clientID, userID sql.NullString
		var voicePlatform, voiceUserID, userEmail, linkedDeviceCode sql.NullString
		var deviceInfoJSON, scopesJSON []byte
		var verifiedAt sql.NullInt64

		err := rows.Scan(
			&vs.ID,
			&vs.TenantID,
			&clientID,
			&vs.SessionToken,
			&vs.VoiceOTP,
			&vs.OTPAttempts,
			&voicePlatform,
			&voiceUserID,
			&deviceInfoJSON,
			&userID,
			&userEmail,
			&vs.Status,
			&linkedDeviceCode,
			&scopesJSON,
			&vs.ExpiresAt,
			&verifiedAt,
			&vs.CreatedAt,
			&vs.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Handle nullable fields
		if clientID.Valid {
			id := uuid.MustParse(clientID.String)
			vs.ClientID = &id
		}
		if userID.Valid {
			id := uuid.MustParse(userID.String)
			vs.UserID = &id
		}
		if voicePlatform.Valid {
			vs.VoicePlatform = voicePlatform.String
		}
		if voiceUserID.Valid {
			vs.VoiceUserID = voiceUserID.String
		}
		if userEmail.Valid {
			vs.UserEmail = userEmail.String
		}
		if linkedDeviceCode.Valid {
			vs.LinkedDeviceCode = linkedDeviceCode.String
		}
		if verifiedAt.Valid {
			vs.VerifiedAt = &verifiedAt.Int64
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(deviceInfoJSON, &vs.DeviceInfo); err != nil {
			vs.DeviceInfo = make(map[string]interface{})
		}
		if err := json.Unmarshal(scopesJSON, &vs.Scopes); err != nil {
			vs.Scopes = []string{}
		}

		sessions = append(sessions, vs)
	}

	return sessions, nil
}

// GetVoiceSessionApprovalStatus gets the approval status of a voice session
func (r *VoiceAuthRepository) GetVoiceSessionApprovalStatus(sessionToken string) (string, error) {
	query := `
		SELECT COALESCE(approval_status, '') FROM voice_sessions WHERE session_token = $1
	`

	var status string
	err := r.db.QueryRow(query, sessionToken).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("voice session not found")
		}
		return "", err
	}

	return status, nil
}

// GenerateSessionID generates a unique session ID for JWT jti claim
func GenerateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)), nil
}
