package sdkmgr

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// OAuthSession represents an MCP OAuth session stored in PostgreSQL.
// Pre-auth sessions live in the master DB; post-auth sessions migrate to
// the tenant-specific database.
type OAuthSession struct {
	SessionID         string         `gorm:"column:session_id;primaryKey;size:36" json:"session_id"`
	UserEmail         *string        `gorm:"column:user_email;size:255" json:"user_email,omitempty"`
	UserInfo          datatypes.JSON `gorm:"column:user_info;type:jsonb" json:"user_info,omitempty"`
	AccessToken       *string        `gorm:"column:access_token;type:text" json:"-"`
	RefreshToken      *string        `gorm:"column:refresh_token;type:text" json:"-"`
	AuthorizationCode *string        `gorm:"column:authorization_code;type:text" json:"-"`
	TokenExpiresAt    *int64         `gorm:"column:token_expires_at" json:"token_expires_at,omitempty"`
	CreatedAt         int64          `gorm:"column:created_at;not null" json:"created_at"`
	LastActivity      int64          `gorm:"column:last_activity;not null" json:"last_activity"`
	OAuthState        *string        `gorm:"column:oauth_state;size:255" json:"-"`
	PKCEVerifier      *string        `gorm:"column:pkce_verifier;type:text" json:"-"`
	PKCEChallenge     *string        `gorm:"column:pkce_challenge;type:text" json:"-"`
	IsActive          bool           `gorm:"column:is_active;default:true" json:"is_active"`
	ClientIdentifier  *string        `gorm:"column:client_identifier;size:255" json:"client_identifier,omitempty"`
	OrgID             *string        `gorm:"column:org_id;size:255" json:"org_id,omitempty"`
	TenantID          *string        `gorm:"column:tenant_id;size:255" json:"tenant_id,omitempty"`
	UserID            *string        `gorm:"column:user_id;size:255" json:"user_id,omitempty"`
	Provider          *string        `gorm:"column:provider;size:100" json:"provider,omitempty"`
	ProviderID        *string        `gorm:"column:provider_id;size:255" json:"provider_id,omitempty"`
	AccessibleTools   datatypes.JSON `gorm:"column:accessible_tools;type:jsonb" json:"accessible_tools,omitempty"`
}

// TableName overrides GORM's default table name.
func (OAuthSession) TableName() string {
	return "oauth_sessions"
}

// NewOAuthSession creates a session with a fresh UUID and timestamps.
func NewOAuthSession() *OAuthSession {
	now := time.Now().Unix()
	id := uuid.New().String()
	return &OAuthSession{
		SessionID: id,
		CreatedAt: now,
		LastActivity: now,
		IsActive:  true,
	}
}

// IsTokenValid returns true if the access token exists and is not expired
// (with a 5-minute buffer).
func (s *OAuthSession) IsTokenValid() bool {
	if s.AccessToken == nil || s.TokenExpiresAt == nil {
		return false
	}
	return time.Now().Unix() < (*s.TokenExpiresAt - 300)
}

// SetJWTToken stores a JWT as the access token with the given TTL.
func (s *OAuthSession) SetJWTToken(jwt string, expiresIn int64) {
	s.AccessToken = &jwt
	exp := time.Now().Unix() + expiresIn
	s.TokenExpiresAt = &exp
}

// Touch updates the last_activity timestamp.
func (s *OAuthSession) Touch() {
	now := time.Now().Unix()
	s.LastActivity = now
}

// UpdateUserInfo populates user-related fields from a claims map.
// Mirrors Python OAuthSession.update_user_info().
func (s *OAuthSession) UpdateUserInfo(info map[string]interface{}) {
	raw, _ := json.Marshal(info)
	s.UserInfo = datatypes.JSON(raw)

	if v, ok := stringFromMap(info, "email_id"); ok {
		s.UserEmail = &v
	} else if v, ok := stringFromMap(info, "email"); ok {
		s.UserEmail = &v
	}

	if v, ok := stringFromMap(info, "project_id"); ok {
		s.OrgID = &v
	} else if v, ok := stringFromMap(info, "org_id"); ok {
		s.OrgID = &v
	}

	if v, ok := stringFromMap(info, "tenant_id"); ok {
		s.TenantID = &v
	}
	if v, ok := stringFromMap(info, "user_id"); ok {
		s.UserID = &v
	}
	if v, ok := stringFromMap(info, "provider"); ok {
		s.Provider = &v
	}
	if v, ok := stringFromMap(info, "provider_id"); ok {
		s.ProviderID = &v
	}
}

// GetUserInfoMap deserialises UserInfo back to a map.
func (s *OAuthSession) GetUserInfoMap() map[string]interface{} {
	if s.UserInfo == nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(s.UserInfo, &m); err != nil {
		return nil
	}
	return m
}

// GetAccessibleToolsList deserialises the AccessibleTools JSON to a string slice.
func (s *OAuthSession) GetAccessibleToolsList() []string {
	if s.AccessibleTools == nil {
		return nil
	}
	var tools []string
	if err := json.Unmarshal(s.AccessibleTools, &tools); err != nil {
		return nil
	}
	return tools
}

// SetAccessibleTools serialises a string slice into the JSONB column.
func (s *OAuthSession) SetAccessibleTools(tools []string) {
	raw, _ := json.Marshal(tools)
	s.AccessibleTools = datatypes.JSON(raw)
}

func stringFromMap(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}
