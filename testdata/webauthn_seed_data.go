package testdata

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WebAuthnTestScenario represents a complete WebAuthn test scenario
type WebAuthnTestScenario struct {
	Name                    string                 `json:"name"`
	Description             string                 `json:"description"`
	User                    TestUser               `json:"user"`
	RegistrationChallenge   Challenge              `json:"registration_challenge"`
	RegistrationResponse    CredentialCreation     `json:"registration_response"`
	AuthenticationChallenge Challenge              `json:"authentication_challenge"`
	AuthenticationResponse  CredentialAssertion    `json:"authentication_response"`
	ExpectedResults         ExpectedResults        `json:"expected_results"`
}

// TestUser represents a test user for WebAuthn scenarios
type TestUser struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	TenantID    string    `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// Challenge represents a WebAuthn challenge
type Challenge struct {
	Challenge        string   `json:"challenge"`
	Timeout          int      `json:"timeout"`
	UserVerification string   `json:"userVerification"`
	RpID             string   `json:"rpId"`
	Origins          []string `json:"origins"`
}

// CredentialCreation represents a WebAuthn registration response
type CredentialCreation struct {
	ID    string                        `json:"id"`
	RawID string                        `json:"rawId"`
	Type  string                        `json:"type"`
	Response AuthenticatorAttestationResponse `json:"response"`
}

// AuthenticatorAttestationResponse for registration
type AuthenticatorAttestationResponse struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AttestationObject string `json:"attestationObject"`
}

// CredentialAssertion represents a WebAuthn authentication response
type CredentialAssertion struct {
	ID    string                       `json:"id"`
	RawID string                       `json:"rawId"`
	Type  string                       `json:"type"`
	Response AuthenticatorAssertionResponse `json:"response"`
}

// AuthenticatorAssertionResponse for authentication
type AuthenticatorAssertionResponse struct {
	ClientDataJSON     string  `json:"clientDataJSON"`
	AuthenticatorData  string  `json:"authenticatorData"`
	Signature          string  `json:"signature"`
	UserHandle         *string `json:"userHandle,omitempty"`
}

// ExpectedResults defines what should happen in each scenario
type ExpectedResults struct {
	RegistrationShouldSucceed   bool   `json:"registration_should_succeed"`
	AuthenticationShouldSucceed bool   `json:"authentication_should_succeed"`
	ExpectedError               string `json:"expected_error,omitempty"`
}

// Realistic WebAuthn Test Scenarios

// GetStandardScenarios returns a collection of standard WebAuthn test scenarios
func GetStandardScenarios() []WebAuthnTestScenario {
	return []WebAuthnTestScenario{
		createSuccessfulRegistrationScenario(),
		createSuccessfulAuthenticationScenario(),
		createOriginMismatchScenario(),
		createInvalidChallengeScenario(),
		createMultiTenantScenario(),
		createPasskeyScenario(),
	}
}

func createSuccessfulRegistrationScenario() WebAuthnTestScenario {
	userID := uuid.New()
	tenantID := uuid.New()
	challenge := generateChallenge()
	credentialID := generateCredentialID()

	return WebAuthnTestScenario{
		Name:        "Successful Registration",
		Description: "Standard successful WebAuthn credential registration flow",
		User: TestUser{
			ID:          userID.String(),
			Email:       "test@example.com",
			DisplayName: "Test User",
			TenantID:    tenantID.String(),
			CreatedAt:   time.Now().UTC(),
		},
		RegistrationChallenge: Challenge{
			Challenge:        challenge,
			Timeout:          60000,
			UserVerification: "preferred",
			RpID:             "app.authsec.dev",
			Origins:          []string{"https://app.authsec.dev"},
		},
		RegistrationResponse: CredentialCreation{
			ID:    base64.RawURLEncoding.EncodeToString(credentialID),
			RawID: base64.RawURLEncoding.EncodeToString(credentialID),
			Type:  "public-key",
			Response: AuthenticatorAttestationResponse{
				ClientDataJSON: base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{
					"type": "webauthn.create",
					"challenge": "%s",
					"origin": "https://app.authsec.dev",
					"crossOrigin": false
				}`, challenge))),
				AttestationObject: generateAttestationObject(credentialID),
			},
		},
		ExpectedResults: ExpectedResults{
			RegistrationShouldSucceed:   true,
			AuthenticationShouldSucceed: false,
		},
	}
}

func createSuccessfulAuthenticationScenario() WebAuthnTestScenario {
	userID := uuid.New()
	tenantID := uuid.New()
	challenge := generateChallenge()
	credentialID := generateCredentialID()

	return WebAuthnTestScenario{
		Name:        "Successful Authentication",
		Description: "Standard successful WebAuthn authentication flow with existing credential",
		User: TestUser{
			ID:          userID.String(),
			Email:       "existing@example.com",
			DisplayName: "Existing User",
			TenantID:    tenantID.String(),
			CreatedAt:   time.Now().Add(-24 * time.Hour).UTC(),
		},
		AuthenticationChallenge: Challenge{
			Challenge:        challenge,
			Timeout:          60000,
			UserVerification: "preferred",
			RpID:             "app.authsec.dev",
			Origins:          []string{"https://app.authsec.dev"},
		},
		AuthenticationResponse: CredentialAssertion{
			ID:    base64.RawURLEncoding.EncodeToString(credentialID),
			RawID: base64.RawURLEncoding.EncodeToString(credentialID),
			Type:  "public-key",
			Response: AuthenticatorAssertionResponse{
				ClientDataJSON: base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{
					"type": "webauthn.get",
					"challenge": "%s",
					"origin": "https://app.authsec.dev",
					"crossOrigin": false
				}`, challenge))),
				AuthenticatorData: generateAuthenticatorData(),
				Signature:         generateSignature(),
				UserHandle:        nil,
			},
		},
		ExpectedResults: ExpectedResults{
			RegistrationShouldSucceed:   false,
			AuthenticationShouldSucceed: true,
		},
	}
}

func createOriginMismatchScenario() WebAuthnTestScenario {
	userID := uuid.New()
	tenantID := uuid.New()
	challenge := generateChallenge()
	credentialID := generateCredentialID()

	return WebAuthnTestScenario{
		Name:        "Origin Mismatch",
		Description: "Registration attempt from unauthorized origin should fail",
		User: TestUser{
			ID:          userID.String(),
			Email:       "test@malicious.com",
			DisplayName: "Malicious User",
			TenantID:    tenantID.String(),
			CreatedAt:   time.Now().UTC(),
		},
		RegistrationChallenge: Challenge{
			Challenge:        challenge,
			Timeout:          60000,
			UserVerification: "preferred",
			RpID:             "app.authsec.dev",
			Origins:          []string{"https://app.authsec.dev"},
		},
		RegistrationResponse: CredentialCreation{
			ID:    base64.RawURLEncoding.EncodeToString(credentialID),
			RawID: base64.RawURLEncoding.EncodeToString(credentialID),
			Type:  "public-key",
			Response: AuthenticatorAttestationResponse{
				ClientDataJSON: base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{
					"type": "webauthn.create",
					"challenge": "%s",
					"origin": "https://malicious.com",
					"crossOrigin": false
				}`, challenge))),
				AttestationObject: generateAttestationObject(credentialID),
			},
		},
		ExpectedResults: ExpectedResults{
			RegistrationShouldSucceed:   false,
			AuthenticationShouldSucceed: false,
			ExpectedError:               "origin validation failed",
		},
	}
}

func createInvalidChallengeScenario() WebAuthnTestScenario {
	userID := uuid.New()
	tenantID := uuid.New()
	challenge := generateChallenge()
	wrongChallenge := generateChallenge() // Different challenge
	credentialID := generateCredentialID()

	return WebAuthnTestScenario{
		Name:        "Invalid Challenge",
		Description: "Registration with wrong challenge should fail",
		User: TestUser{
			ID:          userID.String(),
			Email:       "test@challenge.com",
			DisplayName: "Challenge Test User",
			TenantID:    tenantID.String(),
			CreatedAt:   time.Now().UTC(),
		},
		RegistrationChallenge: Challenge{
			Challenge:        challenge,
			Timeout:          60000,
			UserVerification: "preferred",
			RpID:             "app.authsec.dev",
			Origins:          []string{"https://app.authsec.dev"},
		},
		RegistrationResponse: CredentialCreation{
			ID:    base64.RawURLEncoding.EncodeToString(credentialID),
			RawID: base64.RawURLEncoding.EncodeToString(credentialID),
			Type:  "public-key",
			Response: AuthenticatorAttestationResponse{
				ClientDataJSON: base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{
					"type": "webauthn.create",
					"challenge": "%s",
					"origin": "https://app.authsec.dev",
					"crossOrigin": false
				}`, wrongChallenge))), // Wrong challenge here
				AttestationObject: generateAttestationObject(credentialID),
			},
		},
		ExpectedResults: ExpectedResults{
			RegistrationShouldSucceed:   false,
			AuthenticationShouldSucceed: false,
			ExpectedError:               "challenge mismatch",
		},
	}
}

func createMultiTenantScenario() WebAuthnTestScenario {
	userID := uuid.New()
	tenantID := uuid.New()
	challenge := generateChallenge()
	credentialID := generateCredentialID()

	return WebAuthnTestScenario{
		Name:        "Multi-Tenant Registration",
		Description: "Registration in specific tenant context with subdomain origin",
		User: TestUser{
			ID:          userID.String(),
			Email:       "tenant@brcm.com",
			DisplayName: "Tenant User",
			TenantID:    tenantID.String(),
			CreatedAt:   time.Now().UTC(),
		},
		RegistrationChallenge: Challenge{
			Challenge:        challenge,
			Timeout:          60000,
			UserVerification: "required",
			RpID:             "app.authsec.dev",
			Origins:          []string{"https://brcm.app.authsec.dev", "https://app.authsec.dev"},
		},
		RegistrationResponse: CredentialCreation{
			ID:    base64.RawURLEncoding.EncodeToString(credentialID),
			RawID: base64.RawURLEncoding.EncodeToString(credentialID),
			Type:  "public-key",
			Response: AuthenticatorAttestationResponse{
				ClientDataJSON: base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{
					"type": "webauthn.create",
					"challenge": "%s",
					"origin": "https://brcm.app.authsec.dev",
					"crossOrigin": false
				}`, challenge))),
				AttestationObject: generateAttestationObject(credentialID),
			},
		},
		ExpectedResults: ExpectedResults{
			RegistrationShouldSucceed:   true,
			AuthenticationShouldSucceed: false,
		},
	}
}

func createPasskeyScenario() WebAuthnTestScenario {
	userID := uuid.New()
	tenantID := uuid.New()
	challenge := generateChallenge()
	credentialID := generateCredentialID()

	return WebAuthnTestScenario{
		Name:        "Passkey Flow",
		Description: "Discoverable credential (passkey) registration and authentication",
		User: TestUser{
			ID:          userID.String(),
			Email:       "passkey@example.com",
			DisplayName: "Passkey User",
			TenantID:    tenantID.String(),
			CreatedAt:   time.Now().UTC(),
		},
		RegistrationChallenge: Challenge{
			Challenge:        challenge,
			Timeout:          300000, // Longer timeout for passkeys
			UserVerification: "required",
			RpID:             "app.authsec.dev",
			Origins:          []string{"https://app.authsec.dev"},
		},
		RegistrationResponse: CredentialCreation{
			ID:    base64.RawURLEncoding.EncodeToString(credentialID),
			RawID: base64.RawURLEncoding.EncodeToString(credentialID),
			Type:  "public-key",
			Response: AuthenticatorAttestationResponse{
				ClientDataJSON: base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{
					"type": "webauthn.create",
					"challenge": "%s",
					"origin": "https://app.authsec.dev",
					"crossOrigin": false
				}`, challenge))),
				AttestationObject: generatePasskeyAttestationObject(credentialID),
			},
		},
		ExpectedResults: ExpectedResults{
			RegistrationShouldSucceed:   true,
			AuthenticationShouldSucceed: false,
		},
	}
}

// Helper functions to generate realistic test data

func generateChallenge() string {
	challengeBytes := make([]byte, 32)
	// Fill with predictable but varied data
	for i := range challengeBytes {
		challengeBytes[i] = byte(i + 65) // ASCII starting from 'A'
	}
	return base64.RawURLEncoding.EncodeToString(challengeBytes)
}

func generateCredentialID() []byte {
	credID := make([]byte, 32)
	for i := range credID {
		credID[i] = byte((i * 7) % 256) // Predictable pattern
	}
	return credID
}

func generateAttestationObject(credentialID []byte) string {
	// Simplified CBOR-like structure for testing
	attestation := map[string]interface{}{
		"fmt":      "none",
		"attStmt":  map[string]interface{}{},
		"authData": hex.EncodeToString(credentialID),
	}

	jsonData, _ := json.Marshal(attestation)
	return base64.RawURLEncoding.EncodeToString(jsonData)
}

func generatePasskeyAttestationObject(credentialID []byte) string {
	// Passkey-specific attestation with resident key flag
	attestation := map[string]interface{}{
		"fmt":      "none",
		"attStmt":  map[string]interface{}{},
		"authData": hex.EncodeToString(credentialID),
		"flags":    "resident_key",
	}

	jsonData, _ := json.Marshal(attestation)
	return base64.RawURLEncoding.EncodeToString(jsonData)
}

func generateAuthenticatorData() string {
	// Mock authenticator data for authentication
	authData := make([]byte, 37) // Standard length
	// RP ID hash (32 bytes) + flags (1 byte) + sign count (4 bytes)
	copy(authData[:32], []byte("app.authsec.dev.................")) // 32 bytes
	authData[32] = 0x01 // User present flag
	// Sign count in bytes 33-36
	authData[33] = 0x00
	authData[34] = 0x00
	authData[35] = 0x00
	authData[36] = 0x01 // Count = 1

	return base64.RawURLEncoding.EncodeToString(authData)
}

func generateSignature() string {
	// Mock signature for testing
	signature := make([]byte, 64) // Typical P-256 signature length
	for i := range signature {
		signature[i] = byte((i * 3) % 256)
	}
	return base64.RawURLEncoding.EncodeToString(signature)
}

// Utility functions for tests

// SerializeScenarios converts scenarios to JSON for external use
func SerializeScenarios(scenarios []WebAuthnTestScenario) ([]byte, error) {
	return json.MarshalIndent(scenarios, "", "  ")
}

// LoadScenariosFromJSON loads scenarios from JSON data
func LoadScenariosFromJSON(data []byte) ([]WebAuthnTestScenario, error) {
	var scenarios []WebAuthnTestScenario
	err := json.Unmarshal(data, &scenarios)
	return scenarios, err
}

// GetScenarioByName returns a specific test scenario by name
func GetScenarioByName(name string) *WebAuthnTestScenario {
	scenarios := GetStandardScenarios()
	for _, scenario := range scenarios {
		if scenario.Name == name {
			return &scenario
		}
	}
	return nil
}

// GenerateRequestPayload creates HTTP request payload for a scenario
func (s *WebAuthnTestScenario) GenerateRegistrationRequestPayload() map[string]interface{} {
	return map[string]interface{}{
		"email":     s.User.Email,
		"tenant_id": s.User.TenantID,
		"response":  s.RegistrationResponse,
	}
}

func (s *WebAuthnTestScenario) GenerateAuthenticationRequestPayload() map[string]interface{} {
	return map[string]interface{}{
		"email":     s.User.Email,
		"tenant_id": s.User.TenantID,
		"response":  s.AuthenticationResponse,
	}
}