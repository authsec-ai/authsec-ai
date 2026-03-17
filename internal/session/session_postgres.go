package session

import (
	"encoding/json"
	"log"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

// PostgreSQLSessionManager implements session management using PostgreSQL
type PostgreSQLSessionManager struct {
	db *gorm.DB
}

// Ensure PostgreSQLSessionManager implements SessionManagerInterface
var _ SessionManagerInterface = (*PostgreSQLSessionManager)(nil)

// NewPostgreSQLSessionManager creates a new PostgreSQL-based session manager
func NewPostgreSQLSessionManager(db *gorm.DB, migrationsDir string) *PostgreSQLSessionManager {
	manager := &PostgreSQLSessionManager{
		db: db,
	}

	// Note: Migrations have been removed. Database schema should be set up externally.
	log.Printf("PostgreSQLSessionManager: initialized (migrations disabled)")

	return manager
}



// Save stores a WebAuthn session in PostgreSQL with 10-minute expiration
func (s *PostgreSQLSessionManager) Save(key string, data *webauthn.SessionData) error {
	// Serialize extensions if present
	var extensionsData []byte
	if data.Extensions != nil {
		var err error
		extensionsData, err = json.Marshal(data.Extensions)
		if err != nil {
			log.Printf("PostgreSQLSessionManager: failed to marshal extensions: %v", err)
			return err
		}
	}

	// Serialize credential parameters if present
	var credParamsData []byte
	if len(data.CredParams) > 0 {
		var err error
		credParamsData, err = json.Marshal(data.CredParams)
		if err != nil {
			log.Printf("PostgreSQLSessionManager: failed to marshal credential parameters: %v", err)
			return err
		}
	}

	// Serialize allowed credential IDs if present
	var allowedCredsData []byte
	if len(data.AllowedCredentialIDs) > 0 {
		var err error
		allowedCredsData, err = json.Marshal(data.AllowedCredentialIDs)
		if err != nil {
			log.Printf("PostgreSQLSessionManager: failed to marshal allowed credential IDs: %v", err)
			return err
		}
	}

	session := models.Session{
		SessionKey:           key,
		Challenge:            data.Challenge,
		UserID:               data.UserID,
		UserVerification:     string(data.UserVerification),
		Extensions:           extensionsData,
		CredParams:           credParamsData,
		AllowedCredentialIDs: allowedCredsData,
		CreatedAt:            time.Now().UTC(),
		ExpiresAt:            time.Now().UTC().Add(10 * time.Minute), // 10-minute expiration
	}

	// Delete existing session first, then insert new one (workaround for missing unique constraint)
	s.db.Exec("DELETE FROM webauthn_sessions WHERE session_key = ?", session.SessionKey)
	
	result := s.db.Exec(`
		INSERT INTO webauthn_sessions (session_key, challenge, user_id, user_verification, extensions, cred_params, allowed_credential_ids, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, session.SessionKey, session.Challenge, session.UserID, session.UserVerification, session.Extensions, session.CredParams, session.AllowedCredentialIDs, session.CreatedAt, session.ExpiresAt)

	if result.Error != nil {
		log.Printf("PostgreSQLSessionManager: failed to save session for key=%s: %v", key, result.Error)
		return result.Error
	}

	log.Printf("PostgreSQLSessionManager: saved session for key=%s, expires at %s", key, session.ExpiresAt.Format(time.RFC3339))
	return nil
}

// Get retrieves a WebAuthn session from PostgreSQL and validates expiration
func (s *PostgreSQLSessionManager) Get(key string) (*webauthn.SessionData, bool) {
	var session models.Session

	// Query for non-expired session
	result := s.db.Where("session_key = ? AND expires_at > NOW()", key).First(&session)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("PostgreSQLSessionManager: session not found or expired for key=%s", key)
		} else {
			log.Printf("PostgreSQLSessionManager: database error retrieving session for key=%s: %v", key, result.Error)
		}
		return nil, false
	}

	// Deserialize extensions if present
	var extensions protocol.AuthenticationExtensions
	if len(session.Extensions) > 0 {
		if err := json.Unmarshal(session.Extensions, &extensions); err != nil {
			log.Printf("PostgreSQLSessionManager: failed to unmarshal extensions for key=%s: %v", key, err)
			// Continue without extensions rather than failing
		}
	}

	// Deserialize credential parameters if present
	var credParams []protocol.CredentialParameter
	if len(session.CredParams) > 0 {
		if err := json.Unmarshal(session.CredParams, &credParams); err != nil {
			log.Printf("PostgreSQLSessionManager: failed to unmarshal credential parameters for key=%s: %v", key, err)
			// Continue without credential parameters rather than failing
		}
	}

	// Deserialize allowed credential IDs if present
	var allowedCreds [][]byte
	if len(session.AllowedCredentialIDs) > 0 {
		if err := json.Unmarshal(session.AllowedCredentialIDs, &allowedCreds); err != nil {
			log.Printf("PostgreSQLSessionManager: failed to unmarshal allowed credential IDs for key=%s: %v", key, err)
			// Continue without allowed credential IDs rather than failing
		}
	}

	sessionData := &webauthn.SessionData{
		Challenge:            session.Challenge,
		UserID:               session.UserID,
		UserVerification:     protocol.UserVerificationRequirement(session.UserVerification),
		Extensions:           extensions,
		CredParams:           credParams,
		AllowedCredentialIDs: allowedCreds,
	}

	log.Printf("PostgreSQLSessionManager: retrieved session for key=%s, expires at %s", key, session.ExpiresAt.Format(time.RFC3339))
	return sessionData, true
}

// Delete removes a session from PostgreSQL
func (s *PostgreSQLSessionManager) Delete(key string) {
	result := s.db.Where("session_key = ?", key).Delete(&models.Session{})
	if result.Error != nil {
		log.Printf("PostgreSQLSessionManager: failed to delete session for key=%s: %v", key, result.Error)
	} else {
		log.Printf("PostgreSQLSessionManager: deleted session for key=%s (affected rows: %d)", key, result.RowsAffected)
	}
}

// ListKeys returns all active session keys (for debugging/monitoring)
func (s *PostgreSQLSessionManager) ListKeys() []string {
	var sessions []models.Session
	result := s.db.Select("session_key").Where("expires_at > NOW()").Find(&sessions)
	if result.Error != nil {
		log.Printf("PostgreSQLSessionManager: failed to list session keys: %v", result.Error)
		return []string{}
	}

	keys := make([]string, len(sessions))
	for i, session := range sessions {
		keys[i] = session.SessionKey
	}

	log.Printf("PostgreSQLSessionManager: found %d active sessions", len(keys))
	return keys
}

// CleanupExpiredSessions removes expired sessions (can be called periodically)
func (s *PostgreSQLSessionManager) CleanupExpiredSessions() error {
	result := s.db.Where("expires_at < NOW()").Delete(&models.Session{})
	if result.Error != nil {
		log.Printf("PostgreSQLSessionManager: failed to cleanup expired sessions: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("PostgreSQLSessionManager: cleaned up %d expired sessions", result.RowsAffected)
	}
	return nil
}

// GetSessionStats returns statistics about stored sessions
func (s *PostgreSQLSessionManager) GetSessionStats() (int64, int64, error) {
	var totalSessions, activeSessions int64

	// Count total sessions
	if err := s.db.Model(&models.Session{}).Count(&totalSessions).Error; err != nil {
		return 0, 0, err
	}

	// Count active (non-expired) sessions
	if err := s.db.Model(&models.Session{}).Where("expires_at > NOW()").Count(&activeSessions).Error; err != nil {
		return 0, 0, err
	}

	return totalSessions, activeSessions, nil
}
