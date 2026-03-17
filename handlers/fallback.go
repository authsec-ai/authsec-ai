package handlers

import (
	"encoding/base64"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// buildFallbackCredentialFromContainer constructs a minimal credential when attestation validation fails.
// This is primarily used for Windows Hello authenticators that return packed/TPM attestation formats
// without the required metadata.
func buildFallbackCredentialFromContainer(container *CredentialContainer) (*webauthn.Credential, error) {
	if container == nil {
		return nil, fmt.Errorf("credential container is nil")
	}
	if container.RawID == "" {
		return nil, fmt.Errorf("credential rawId is empty")
	}
	if container.Response.AttestationObject == "" {
		return nil, fmt.Errorf("attestation object is missing")
	}

	pubKey, err := parsePublicKeyFromAttestation(container.Response.AttestationObject)
	if err != nil {
		return nil, err
	}

	credentialID, decodeErr := base64.RawURLEncoding.DecodeString(container.RawID)
	if decodeErr != nil {
		credentialID, decodeErr = base64.StdEncoding.DecodeString(container.RawID)
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode credential RawID: %w", decodeErr)
		}
	}

	return &webauthn.Credential{
		ID:              credentialID,
		PublicKey:       pubKey,
		AttestationType: "none",
		Transport:       []protocol.AuthenticatorTransport{},
		Authenticator: webauthn.Authenticator{
			AAGUID:    make([]byte, 16),
			SignCount: 0,
		},
		Flags: webauthn.CredentialFlags{
			BackupEligible: false,
			BackupState:    false,
		},
	}, nil
}
