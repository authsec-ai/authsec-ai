package handlers

import (
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/authsec-ai/authsec/services"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// SMSHandler handles SMS-related operations
type SMSHandler struct {
	Service *services.SMSService
}

// TOTPHandler handles TOTP-related operations
type TOTPHandler struct {
	Service *services.WebAuthnTOTPService
}

// SessionManagerInterface defines session management methods
type SessionManagerInterface interface {
	Save(key string, data interface{}) error
	Get(key string) (interface{}, bool)
	Delete(key string)
}

// WebAuthnHandler handles WebAuthn operations
type WebAuthnHandler struct {
	WebAuthn       *webauthn.WebAuthn
	SessionManager SessionManagerInterface
	RPDisplayName  string
	RPID           string
	RPOrigins      []string
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SMS Request/Response structs
type SMSSetupRequest struct {
	TenantID    string `json:"tenant_id" binding:"required"`
	ClientID    string `json:"client_id" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type SMSSetupResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	PhoneDisplay      string `json:"phone_display"`
	ExpiresInMinutes  int    `json:"expires_in_minutes"`
	AttemptsRemaining int    `json:"attempts_remaining"`
}

type SMSConfirmRequest struct {
	TenantID    string `json:"tenant_id" binding:"required"`
	ClientID    string `json:"client_id" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	PhoneNumber string `json:"phone_number" binding:"required"`
	Code        string `json:"code" binding:"required"`
}

type SMSConfirmResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	PhoneDisplay string `json:"phone_display"`
}

type RequestSMSCodeRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	ClientID string `json:"client_id" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type SMSCodeResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	PhoneDisplay      string `json:"phone_display"`
	ExpiresInMinutes  int    `json:"expires_in_minutes"`
	AttemptsRemaining int    `json:"attempts_remaining"`
}

type VerifySMSRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	ClientID string `json:"client_id" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required"`
}

// TOTP Request/Response structs
type TOTPSetupRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	ClientID string `json:"client_id,omitempty"`
	Email    string `json:"email" binding:"required,email"`
}

type TOTPSetupResponse struct {
	Secret      string `json:"secret"`
	QRCode      string `json:"qr_code"`
	ManualEntry string `json:"manual_entry"`
	Issuer      string `json:"issuer"`
	Account     string `json:"account"`
	OTPAuthURL  string `json:"otp_auth_url"`
}

type TOTPConfirmRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	ClientID string `json:"client_id,omitempty"`
	Email    string `json:"email" binding:"required,email"`
	Secret   string `json:"secret" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

type TOTPConfirmResponse struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	BackupCodes []string `json:"backup_codes"`
}

type TOTPVerifyRequest struct {
	TenantID   string `json:"tenant_id" binding:"required"`
	ClientID   string `json:"client_id,omitempty"`
	Email      string `json:"email" binding:"required,email"`
	Code       string `json:"code"`
	BackupCode string `json:"backup_code"`
}

// TOTPLoginVerifyRequest for login verification (doesn't require client_id)
type TOTPLoginVerifyRequest struct {
	TenantID   string `json:"tenant_id" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Code       string `json:"code"`
	BackupCode string `json:"backup_code"`
}

// WebAuthn Request/Response structs
type BeginRegistrationRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type BeginAuthenticationRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type FinishAuthenticationRequest struct {
	TenantID   string                               `json:"tenant_id" binding:"required"`
	Email      string                               `json:"email" binding:"required,email"`
	Credential protocol.CredentialAssertionResponse `json:"credential" binding:"required"`
}

type RegistrationSuccessResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	CredentialID string `json:"credential_id,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type AuthenticationSuccessResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	CredentialID string `json:"credential_id,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// WebAuthnUser wraps a User for WebAuthn operations
type WebAuthnUser struct {
	*sharedmodels.User
	credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	return []byte(u.ID.String())
}

func (u *WebAuthnUser) WebAuthnName() string {
	return u.Email
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.Email
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u *WebAuthnUser) SetCredentials(creds []webauthn.Credential) {
	u.credentials = creds
}

// SessionData represents data stored in sessions - matches webauthn.SessionData
type SessionData struct {
	Challenge        string                 `json:"challenge"`
	UserID           []byte                 `json:"user_id,omitempty"`
	UserVerification string                 `json:"user_verification,omitempty"`
	Extensions       map[string]interface{} `json:"extensions,omitempty"`
}

// ToWebAuthnSessionData converts our SessionData to webauthn.SessionData
func (s *SessionData) ToWebAuthnSessionData() *webauthn.SessionData {
	if s == nil {
		return nil
	}
	return &webauthn.SessionData{
		Challenge:        s.Challenge,
		UserID:           s.UserID,
		UserVerification: protocol.UserVerificationRequirement(s.UserVerification),
		Extensions:       s.Extensions,
	}
}
