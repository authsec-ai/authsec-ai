package sdkmgr

import (
	"time"

	"github.com/authsec-ai/authsec/config"
	models "github.com/authsec-ai/authsec/models/sdkmgr"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// OAuthSessionStore manages oauth_sessions across the master and tenant databases.
// Pre-auth sessions live in the master DB; once a tenant_id is known the session
// migrates to the corresponding tenant DB and is removed from master.
type OAuthSessionStore struct{}

// NewOAuthSessionStore creates a new store instance.
func NewOAuthSessionStore() *OAuthSessionStore {
	return &OAuthSessionStore{}
}

// masterDB returns the global GORM instance (master "authsec" DB).
func (s *OAuthSessionStore) masterDB() *gorm.DB {
	return config.DB
}

// tenantDB returns a GORM instance for the given tenant.
func (s *OAuthSessionStore) tenantDB(tenantID string) (*gorm.DB, error) {
	return config.GetTenantGORMDB(tenantID)
}

// InvalidateAllSessions sets is_active=false for every session in the master DB.
// Called once at startup, matching the Python behaviour.
func (s *OAuthSessionStore) InvalidateAllSessions() {
	result := s.masterDB().Model(&models.OAuthSession{}).
		Where("is_active = true").
		Update("is_active", false)
	if result.Error != nil {
		logrus.WithError(result.Error).Error("failed to invalidate master sessions on startup")
	} else {
		logrus.WithField("affected", result.RowsAffected).Info("invalidated all previous master sessions on startup")
	}
}

// SaveSession persists the session. If the session has a tenant_id it goes
// to the tenant DB (and is removed from master if it was there before).
// Otherwise it goes to the master DB.
func (s *OAuthSessionStore) SaveSession(session *models.OAuthSession) error {
	session.Touch()

	if session.TenantID != nil && *session.TenantID != "" {
		return s.saveToTenant(session)
	}
	return s.saveToMaster(session)
}

func (s *OAuthSessionStore) saveToMaster(session *models.OAuthSession) error {
	db := s.masterDB()
	result := db.Save(session) // GORM Save does upsert on primary key
	if result.Error != nil {
		logrus.WithError(result.Error).Error("failed to save session to master DB")
		return result.Error
	}
	logrus.WithField("session_id", session.SessionID).Info("saved session to master DB")
	return nil
}

func (s *OAuthSessionStore) saveToTenant(session *models.OAuthSession) error {
	tenantID := *session.TenantID

	// Check if session exists in master (for migration).
	var existsInMaster bool
	masterDB := s.masterDB()
	var count int64
	masterDB.Model(&models.OAuthSession{}).
		Where("session_id = ?", session.SessionID).
		Count(&count)
	existsInMaster = count > 0

	// Save to tenant DB.
	tdb, err := s.tenantDB(tenantID)
	if err != nil {
		logrus.WithError(err).WithField("tenant_id", tenantID).Error("failed to get tenant DB")
		return err
	}

	if err := tdb.Save(session).Error; err != nil {
		logrus.WithError(err).Error("failed to save session to tenant DB")
		return err
	}
	logrus.WithFields(logrus.Fields{
		"session_id": session.SessionID,
		"tenant_id":  tenantID,
	}).Info("saved session to tenant DB")

	// Upsert user record for dashboard queries.
	s.upsertTenantUser(tdb, session)

	// Migrate: remove from master if it was there.
	if existsInMaster {
		masterDB.Where("session_id = ?", session.SessionID).Delete(&models.OAuthSession{})
		logrus.WithField("session_id", session.SessionID).Info("migrated session from master to tenant DB")
	}

	return nil
}

// upsertTenantUser syncs the authenticated user into the tenant's users table
// so the dashboard has access to user data.
func (s *OAuthSessionStore) upsertTenantUser(tdb *gorm.DB, session *models.OAuthSession) {
	if session.UserID == nil || session.UserEmail == nil {
		return
	}

	var userName *string
	info := session.GetUserInfoMap()
	if info != nil {
		if n, ok := info["name"].(string); ok && n != "" {
			userName = &n
		} else if n, ok := info["full_name"].(string); ok && n != "" {
			userName = &n
		}
	}

	sql := `
		INSERT INTO users (user_id, client_id, tenant_id, name, email, provider, provider_id,
			active, last_login, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			last_login = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			active = EXCLUDED.active,
			name = COALESCE(EXCLUDED.name, users.name),
			provider = COALESCE(EXCLUDED.provider, users.provider),
			provider_id = COALESCE(EXCLUDED.provider_id, users.provider_id)
	`
	tdb.Exec(sql,
		*session.UserID,
		session.ClientIdentifier,
		session.TenantID,
		userName,
		*session.UserEmail,
		session.Provider,
		session.ProviderID,
		true,
	)
}

// GetSession looks up an active session by ID, searching master first then
// all known tenant pools.
func (s *OAuthSessionStore) GetSession(sessionID string) *models.OAuthSession {
	// Search master first.
	var session models.OAuthSession
	result := s.masterDB().
		Where("session_id = ? AND is_active = true", sessionID).
		First(&session)
	if result.Error == nil {
		return &session
	}

	// Search tenant databases.
	return s.searchTenantDBs(func(db *gorm.DB) *models.OAuthSession {
		var ts models.OAuthSession
		if db.Where("session_id = ? AND is_active = true", sessionID).First(&ts).Error == nil {
			return &ts
		}
		return nil
	})
}

// GetSessionByState finds an active session by its oauth_state value.
func (s *OAuthSessionStore) GetSessionByState(oauthState string) *models.OAuthSession {
	var session models.OAuthSession
	result := s.masterDB().
		Where("oauth_state = ? AND is_active = true", oauthState).
		Order("last_activity DESC").
		First(&session)
	if result.Error == nil {
		return &session
	}

	return s.searchTenantDBs(func(db *gorm.DB) *models.OAuthSession {
		var ts models.OAuthSession
		if db.Where("oauth_state = ? AND is_active = true", oauthState).
			Order("last_activity DESC").
			First(&ts).Error == nil {
			return &ts
		}
		return nil
	})
}

// DeleteSession soft-deletes a session (sets is_active=false).
func (s *OAuthSessionStore) DeleteSession(sessionID string) {
	result := s.masterDB().Model(&models.OAuthSession{}).
		Where("session_id = ?", sessionID).
		Update("is_active", false)
	if result.RowsAffected > 0 {
		logrus.WithField("session_id", sessionID).Info("deleted session from master DB")
		return
	}

	// Try tenant databases.
	s.forEachTenantDB(func(db *gorm.DB, _ string) bool {
		r := db.Model(&models.OAuthSession{}).
			Where("session_id = ?", sessionID).
			Update("is_active", false)
		return r.RowsAffected > 0 // stop if found
	})
}

// GetActiveAuthenticatedSessionsCount returns the total count of active,
// authenticated sessions across master and all tenant databases.
func (s *OAuthSessionStore) GetActiveAuthenticatedSessionsCount() int64 {
	now := time.Now().Unix()
	var total int64

	// Clean up expired sessions in master.
	s.masterDB().Model(&models.OAuthSession{}).
		Where("token_expires_at < ?", now).
		Update("is_active", false)

	var masterCount int64
	s.masterDB().Model(&models.OAuthSession{}).
		Where("is_active = true AND access_token IS NOT NULL AND token_expires_at > ?", now).
		Count(&masterCount)
	total += masterCount

	s.forEachTenantDB(func(db *gorm.DB, _ string) bool {
		db.Model(&models.OAuthSession{}).
			Where("token_expires_at < ?", now).
			Update("is_active", false)

		var cnt int64
		db.Model(&models.OAuthSession{}).
			Where("is_active = true AND access_token IS NOT NULL AND token_expires_at > ?", now).
			Count(&cnt)
		total += cnt
		return false // continue iterating
	})

	return total
}

// CleanupClientSessions deactivates all sessions for a given client identifier
// across all databases. Returns total number of affected rows.
func (s *OAuthSessionStore) CleanupClientSessions(clientID string) int64 {
	var total int64

	result := s.masterDB().Model(&models.OAuthSession{}).
		Where("client_identifier = ? AND is_active = true", clientID).
		Update("is_active", false)
	total += result.RowsAffected

	s.forEachTenantDB(func(db *gorm.DB, _ string) bool {
		r := db.Model(&models.OAuthSession{}).
			Where("client_identifier = ? AND is_active = true", clientID).
			Update("is_active", false)
		total += r.RowsAffected
		return false
	})

	logrus.WithFields(logrus.Fields{
		"client_id": clientID,
		"total":     total,
	}).Info("cleaned up client sessions")
	return total
}

// GetActiveSessionsForClient returns all active, authenticated sessions for a
// specific client identifier.
func (s *OAuthSessionStore) GetActiveSessionsForClient(clientID string) []models.OAuthSession {
	now := time.Now().Unix()
	var sessions []models.OAuthSession

	s.masterDB().
		Where("client_identifier = ? AND is_active = true AND access_token IS NOT NULL AND token_expires_at > ?",
			clientID, now).
		Order("last_activity DESC").
		Find(&sessions)

	s.forEachTenantDB(func(db *gorm.DB, _ string) bool {
		var tenantSessions []models.OAuthSession
		db.Where("client_identifier = ? AND is_active = true AND access_token IS NOT NULL AND token_expires_at > ?",
			clientID, now).
			Order("last_activity DESC").
			Find(&tenantSessions)
		sessions = append(sessions, tenantSessions...)
		return false
	})

	return sessions
}

// ---------- helpers for iterating tenant databases ----------

// searchTenantDBs calls fn for each known tenant DB and returns the first
// non-nil result.
func (s *OAuthSessionStore) searchTenantDBs(fn func(*gorm.DB) *models.OAuthSession) *models.OAuthSession {
	// Use the tenant repository to get all known tenants.
	var tenantIDs []string
	s.masterDB().Raw("SELECT tenant_id FROM tenants WHERE is_active = true").Scan(&tenantIDs)

	for _, tid := range tenantIDs {
		tdb, err := s.tenantDB(tid)
		if err != nil {
			logrus.WithError(err).WithField("tenant_id", tid).Warn("skipping tenant DB during session search")
			continue
		}
		if result := fn(tdb); result != nil {
			return result
		}
	}
	return nil
}

// forEachTenantDB iterates known tenant databases. If fn returns true,
// iteration stops early.
func (s *OAuthSessionStore) forEachTenantDB(fn func(db *gorm.DB, tenantID string) bool) {
	var tenantIDs []string
	s.masterDB().Raw("SELECT tenant_id FROM tenants WHERE is_active = true").Scan(&tenantIDs)

	for _, tid := range tenantIDs {
		tdb, err := s.tenantDB(tid)
		if err != nil {
			logrus.WithError(err).WithField("tenant_id", tid).Warn("skipping tenant DB")
			continue
		}
		if fn(tdb, tid) {
			return
		}
	}
}
