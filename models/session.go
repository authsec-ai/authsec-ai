package models

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a WebAuthn session stored in PostgreSQL
type Session struct {
	ID                   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionKey           string    `gorm:"type:varchar(255);unique;not null;index" json:"session_key"`
	Challenge            string    `gorm:"type:text;not null" json:"challenge"`
	UserID               []byte    `gorm:"type:bytea;not null" json:"user_id"`
	UserVerification     string    `gorm:"type:varchar(50)" json:"user_verification"`
	Extensions           []byte    `gorm:"type:bytea" json:"extensions,omitempty"`
	CredParams           []byte    `gorm:"type:bytea" json:"cred_params,omitempty"`
	AllowedCredentialIDs []byte    `gorm:"type:bytea" json:"allowed_credential_ids,omitempty"`
	CreatedAt            time.Time `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt            time.Time `gorm:"not null;index" json:"expires_at"`
}

// TableName returns the table name for the Session model
func (Session) TableName() string {
	return "webauthn_sessions"
}
