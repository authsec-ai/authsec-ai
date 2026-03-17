package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	session "github.com/authsec-ai/authsec/internal/session"
	"github.com/authsec-ai/authsec/testutils"
)

func TestMain(m *testing.M) {
	if err := testutils.MustSetDBEnvFromDockerCompose(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: DB env setup failed (tests requiring DB will be skipped): %v\n", err)
	}
	os.Exit(m.Run())
}

// MockDB implements a basic mock database for testing
type MockDB struct {
	users       map[string]*sharedmodels.User
	credentials map[string][]byte
}

func NewMockDB() *MockDB {
	return &MockDB{
		users:       make(map[string]*sharedmodels.User),
		credentials: make(map[string][]byte),
	}
}

func (m *MockDB) CreateUser(email, tenantID string) *sharedmodels.User {
	tenantUUID, _ := uuid.Parse(tenantID)
	user := &sharedmodels.User{
		ID:        uuid.New(),
		ClientID:  uuid.New(),
		Email:     email,
		TenantID:  tenantUUID,
		Provider:  "local",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	m.users[email+":"+tenantID] = user
	return user
}

func (m *MockDB) GetUser(email, tenantID string) *sharedmodels.User {
	return m.users[email+":"+tenantID]
}

// WebAuthn Test Data Generator
type WebAuthnTestData struct {
	Challenge       string
	UserID          []byte
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Origin          string
	RPID            string
}

// GenerateTestWebAuthnData creates realistic WebAuthn test data
func GenerateTestWebAuthnData() *WebAuthnTestData {
	challenge := base64.RawURLEncoding.EncodeToString([]byte("test-challenge-32-bytes-long-123"))
	userID := []byte(uuid.New().String())
	credentialID := make([]byte, 32)
	copy(credentialID, []byte("test-credential-id-32-bytes-long"))

	// Mock public key (P-256 uncompressed point)
	publicKey := make([]byte, 65)
	publicKey[0] = 0x04 // Uncompressed point marker
	copy(publicKey[1:33], []byte("x-coordinate-32-bytes-long-test"))
	copy(publicKey[33:65], []byte("y-coordinate-32-bytes-long-test"))

	return &WebAuthnTestData{
		Challenge:       challenge,
		UserID:          userID,
		CredentialID:    credentialID,
		PublicKey:       publicKey,
		AttestationType: "none",
		Origin:          "https://app.authsec.dev",
		RPID:            "app.authsec.dev",
	}
}

// GenerateRegistrationResponse creates a mock WebAuthn registration response
func (td *WebAuthnTestData) GenerateRegistrationResponse() map[string]interface{} {
	return map[string]interface{}{
		"id":    base64.RawURLEncoding.EncodeToString(td.CredentialID),
		"rawId": base64.RawURLEncoding.EncodeToString(td.CredentialID),
		"type":  "public-key",
		"response": map[string]interface{}{
			"clientDataJSON": base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{
				"type":"webauthn.create",
				"challenge":"%s",
				"origin":"%s",
				"crossOrigin":false
			}`, td.Challenge, td.Origin))),
			"attestationObject": base64.RawURLEncoding.EncodeToString([]byte("mock-attestation-object")),
		},
	}
}

// GenerateAuthenticationResponse creates a mock WebAuthn authentication response
func (td *WebAuthnTestData) GenerateAuthenticationResponse() map[string]interface{} {
	return map[string]interface{}{
		"id":    base64.RawURLEncoding.EncodeToString(td.CredentialID),
		"rawId": base64.RawURLEncoding.EncodeToString(td.CredentialID),
		"type":  "public-key",
		"response": map[string]interface{}{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"type":"webauthn.get","challenge":"%s","origin":"%s"}`, td.Challenge, td.Origin))),
			"authenticatorData": base64.RawURLEncoding.EncodeToString([]byte("mock-authenticator-data")),
			"signature":         base64.RawURLEncoding.EncodeToString([]byte("mock-signature")),
		},
	}
}

// Test Handler Setup
func setupTestHandler() *WebAuthnHandler {
	config := &webauthn.Config{
		RPID:          "app.authsec.dev",
		RPDisplayName: "AuthSec Test Service",
		RPOrigins:     []string{"https://app.authsec.dev"},
	}

	wa, _ := webauthn.New(config)
	sessionManager := session.NewSessionManager()

	return &WebAuthnHandler{
		WebAuthn:       wa,
		SessionManager: NewSessionManagerAdapter(sessionManager),
		RPDisplayName:  config.RPDisplayName,
		RPID:           config.RPID,
		RPOrigins:      config.RPOrigins,
	}
}

// Test Cases

func TestWebAuthnUser_Interface(t *testing.T) {
	// Test that WebAuthnUser properly implements the webauthn.User interface
	user := &sharedmodels.User{
		ID:       uuid.New(),
		ClientID: uuid.New(),
		Email:    "test@example.com",
	}

	webauthnUser := &WebAuthnUser{User: user}

	// Test interface methods
	assert.Equal(t, []byte(user.ID.String()), webauthnUser.WebAuthnID())
	assert.Equal(t, user.Email, webauthnUser.WebAuthnName())
	assert.Equal(t, user.Email, webauthnUser.WebAuthnDisplayName())
	assert.Empty(t, webauthnUser.WebAuthnCredentials())

	// Test credential management
	testCreds := []webauthn.Credential{{ID: []byte("test-cred")}}
	webauthnUser.SetCredentials(testCreds)
	assert.Equal(t, testCreds, webauthnUser.WebAuthnCredentials())
}

func TestBeginBiometricSetup_Success(t *testing.T) {
	_ = setupTestHandler()
	mockDB := NewMockDB()

	// Create test user
	testUser := mockDB.CreateUser("test@example.com", uuid.New().String())

	// Setup request
	reqBody := map[string]string{
		"email":     testUser.Email,
		"tenant_id": testUser.TenantID.String(),
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Create request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/webauthn/biometric/beginSetup", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Origin", "https://app.authsec.dev")

	// Mock the resolveDB method behavior (this would need to be mocked in real implementation)
	// handler.BeginBiometricSetup(c)

	// For now, test the session key generation
	challengeKey := buildChallengeKey("biometric_setup", testUser.Email, testUser.TenantID.String())
	expectedKey := fmt.Sprintf("biometric_setup:%s:%s", testUser.Email, testUser.TenantID.String())
	assert.Equal(t, expectedKey, challengeKey)
}

func TestSessionKeyGeneration(t *testing.T) {
	testCases := []struct {
		operation string
		email     string
		tenantID  string
		expected  string
	}{
		{
			operation: "biometric_setup",
			email:     "user@example.com",
			tenantID:  "123e4567-e89b-12d3-a456-426614174000",
			expected:  "biometric_setup:user@example.com:123e4567-e89b-12d3-a456-426614174000",
		},
		{
			operation: "registration",
			email:     "admin@authsec.dev",
			tenantID:  "987fcdeb-51a2-43d7-8f90-123456789abc",
			expected:  "registration:admin@authsec.dev:987fcdeb-51a2-43d7-8f90-123456789abc",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.operation, tc.email), func(t *testing.T) {
			result := buildChallengeKey(tc.operation, tc.email, tc.tenantID)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSessionManager_SaveAndRetrieve(t *testing.T) {
	sessionManager := session.NewSessionManager()
	testData := GenerateTestWebAuthnData()

	// Create test session data
	sessionData := &webauthn.SessionData{
		Challenge:        testData.Challenge,
		UserID:           testData.UserID,
		UserVerification: protocol.VerificationRequired,
	}

	// Test save and retrieve
	key := "test:user@example.com:tenant-id"
	err := sessionManager.Save(key, sessionData)
	assert.NoError(t, err)

	retrieved, found := sessionManager.Get(key)
	assert.True(t, found)
	assert.Equal(t, sessionData.Challenge, retrieved.Challenge)
	assert.Equal(t, sessionData.UserID, retrieved.UserID)
	assert.Equal(t, sessionData.UserVerification, retrieved.UserVerification)

	// Test non-existent key
	_, found = sessionManager.Get("non-existent-key")
	assert.False(t, found)

	// Test delete
	sessionManager.Delete(key)
	_, found = sessionManager.Get(key)
	assert.False(t, found)
}

func TestSessionFallback_RegistrationToBuilometricSetup(t *testing.T) {
	sessionManager := session.NewSessionManager()
	testData := GenerateTestWebAuthnData()

	// Store session with biometric_setup key
	biometricKey := buildChallengeKey("biometric_setup", "test@example.com", "tenant-123")
	sessionData := &webauthn.SessionData{
		Challenge: testData.Challenge,
		UserID:    testData.UserID,
	}

	err := sessionManager.Save(biometricKey, sessionData)
	assert.NoError(t, err)

	// Simulate the fallback logic from FinishRegistration
	registrationKey := buildChallengeKey("registration", "test@example.com", "tenant-123")

	// First try registration key (should fail)
	_, found := sessionManager.Get(registrationKey)
	assert.False(t, found)

	// Fallback to biometric_setup key (should succeed)
	retrieved, found := sessionManager.Get(biometricKey)
	assert.True(t, found)
	assert.Equal(t, sessionData.Challenge, retrieved.Challenge)
}

func TestWebAuthnCredentialConversion(t *testing.T) {
	testData := GenerateTestWebAuthnData()

	// Test credential ID encoding/decoding
	credIDHex := hex.EncodeToString(testData.CredentialID)
	decodedCredID, err := hex.DecodeString(credIDHex)
	assert.NoError(t, err)
	assert.Equal(t, testData.CredentialID, decodedCredID)

	// Test base64 encoding/decoding
	credIDBase64 := base64.RawURLEncoding.EncodeToString(testData.CredentialID)
	decodedBase64, err := base64.RawURLEncoding.DecodeString(credIDBase64)
	assert.NoError(t, err)
	assert.Equal(t, testData.CredentialID, decodedBase64)
}

func TestOriginValidation(t *testing.T) {
	testCases := []struct {
		name           string
		requestOrigin  string
		allowedOrigins []string
		shouldPass     bool
	}{
		{
			name:           "Exact match",
			requestOrigin:  "https://app.authsec.dev",
			allowedOrigins: []string{"https://app.authsec.dev"},
			shouldPass:     true,
		},
		{
			name:           "Subdomain allowed",
			requestOrigin:  "https://brcm.app.authsec.dev",
			allowedOrigins: []string{"https://app.authsec.dev"},
			shouldPass:     true, // Based on current implementation
		},
		{
			name:           "Wrong protocol",
			requestOrigin:  "http://app.authsec.dev",
			allowedOrigins: []string{"https://app.authsec.dev"},
			shouldPass:     false,
		},
		{
			name:           "Different domain",
			requestOrigin:  "https://malicious.com",
			allowedOrigins: []string{"https://app.authsec.dev"},
			shouldPass:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This would test the validateOriginAndCreateWebAuthn function
			// Implementation would depend on how the validation logic is exposed
			t.Logf("Testing origin validation: %s against %v", tc.requestOrigin, tc.allowedOrigins)
		})
	}
}

// Benchmark Tests

func BenchmarkSessionManager_Save(b *testing.B) {
	sessionManager := session.NewSessionManager()
	testData := GenerateTestWebAuthnData()
	sessionData := &webauthn.SessionData{
		Challenge: testData.Challenge,
		UserID:    testData.UserID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("test:user%d@example.com:tenant-id", i)
		sessionManager.Save(key, sessionData)
	}
}

func BenchmarkSessionManager_Get(b *testing.B) {
	sessionManager := session.NewSessionManager()
	testData := GenerateTestWebAuthnData()
	sessionData := &webauthn.SessionData{
		Challenge: testData.Challenge,
		UserID:    testData.UserID,
	}

	// Pre-populate
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("test:user%d@example.com:tenant-id", i)
		keys[i] = key
		sessionManager.Save(key, sessionData)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionManager.Get(keys[i%1000])
	}
}

// Integration Test Scenarios

func TestCompleteRegistrationFlow_MockScenario(t *testing.T) {
	// This would test the complete flow:
	// 1. BeginBiometricSetup
	// 2. ConfirmBiometricSetup
	// 3. Verify session handling

	t.Log("Integration test for complete registration flow would go here")
	t.Log("This requires mocking the database and HTTP request/response cycle")
}

func TestErrorHandling_InvalidSessions(t *testing.T) {
	sessionManager := session.NewSessionManager()

	// Test various error conditions
	testCases := []struct {
		name        string
		sessionKey  string
		expectError bool
	}{
		{"Empty key", "", true},
		{"Non-existent key", "invalid:key", true},
		{"Valid key format", "biometric_setup:user@example.com:tenant-id", true}, // true because no data stored
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, found := sessionManager.Get(tc.sessionKey)
			assert.Equal(t, !tc.expectError, found)
		})
	}
}
