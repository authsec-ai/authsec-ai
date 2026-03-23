package models

import (
	"github.com/google/uuid"
)

// WebAuthnRegistrationInput represents the input for WebAuthn credential registration
type WebAuthnRegistrationInput struct {
	TenantID         uuid.UUID `json:"tenant_id" binding:"required"`
	Email            string    `json:"email" binding:"required,email"`
	CredentialID     []byte    `json:"credential_id,omitempty"`
	PublicKey        []byte    `json:"public_key,omitempty"`
	AttestationType  string    `json:"attestation_type,omitempty"`
	AAGUID           *uuid.UUID `json:"aaguid,omitempty"`
	SignCount        int64     `json:"sign_count,omitempty"`
	Transports       []string  `json:"transports,omitempty"`
	BackupEligible   bool      `json:"backup_eligible,omitempty"`
	BackupState      bool      `json:"backup_state,omitempty"`
	MFAVerified      *bool     `json:"mfa_verified,omitempty"`
	FlowContext      string    `json:"flow_context,omitempty"`
	VerificationMethod string   `json:"verification_method,omitempty"`
}