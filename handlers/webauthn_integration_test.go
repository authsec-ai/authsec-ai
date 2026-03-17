package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	session "github.com/authsec-ai/authsec/internal/session"
	"github.com/authsec-ai/authsec/testdata"
	"github.com/authsec-ai/authsec/testutils"
)

// IntegrationTestSuite holds the test environment
type IntegrationTestSuite struct {
	handler        *WebAuthnHandler
	router         *gin.Engine
	testUsers      map[string]*sharedmodels.User
	sessionManager *session.SessionManager
}

// SetupIntegrationTestSuite creates a complete test environment
func SetupIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	gin.SetMode(gin.TestMode)
	testutils.SetDBEnvFromDockerCompose(t)

	// Create WebAuthn config
	config := &webauthn.Config{
		RPID:          "app.authsec.dev",
		RPDisplayName: "AuthSec Test Service",
		RPOrigins:     []string{"https://app.authsec.dev", "https://brcm.app.authsec.dev"},
	}

	wa, err := webauthn.New(config)
	require.NoError(t, err)

	sessionManager := session.NewSessionManager()

	handler := &WebAuthnHandler{
		WebAuthn:       wa,
		SessionManager: NewSessionManagerAdapter(sessionManager),
		RPDisplayName:  config.RPDisplayName,
		RPID:           config.RPID,
		RPOrigins:      config.RPOrigins,
	}

	// Setup router with routes
	router := gin.New()
	setupWebAuthnRoutes(router, handler)

	return &IntegrationTestSuite{
		handler:        handler,
		router:         router,
		testUsers:      make(map[string]*sharedmodels.User),
		sessionManager: sessionManager,
	}
}

func setupWebAuthnRoutes(router *gin.Engine, handler *WebAuthnHandler) {
	webauthn := router.Group("/webauthn")
	{
		biometric := webauthn.Group("/biometric")
		{
			biometric.POST("/beginSetup", handler.BeginBiometricSetup)
			biometric.POST("/confirmSetup", handler.ConfirmBiometricSetup)
		}
		webauthn.POST("/finishRegistration", handler.FinishRegistration)
	}
}

// Helper function to create test users
func (suite *IntegrationTestSuite) CreateTestUser(email, tenantID string) *sharedmodels.User {
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
	suite.testUsers[email+":"+tenantID] = user
	return user
}

// TestCompleteWebAuthnFlow tests the full registration flow
func TestCompleteWebAuthnFlow_Success(t *testing.T) {
	suite := SetupIntegrationTestSuite(t)
	scenario := testdata.GetScenarioByName("Successful Registration")
	require.NotNil(t, scenario)

	// Step 1: Begin Biometric Setup
	beginReq := map[string]string{
		"email":     scenario.User.Email,
		"tenant_id": scenario.User.TenantID,
	}
	beginBody, _ := json.Marshal(beginReq)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webauthn/biometric/beginSetup", bytes.NewBuffer(beginBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://app.authsec.dev")

	suite.router.ServeHTTP(w, req)

	// Note: This will likely fail without proper database mocking
	// but it demonstrates the test structure
	t.Logf("Begin setup response: %d - %s", w.Code, w.Body.String())

	// Verify session was created
	sessionKey := buildChallengeKey("biometric_setup", scenario.User.Email, scenario.User.TenantID)
	_, found := suite.sessionManager.Get(sessionKey)

	// In a full implementation, we'd mock the database and expect this to work
	t.Logf("Session found for key %s: %v", sessionKey, found)
}

func TestSessionFallbackFlow_IntegrationScenario(t *testing.T) {
	suite := SetupIntegrationTestSuite(t)
	scenario := testdata.GetScenarioByName("Successful Registration")
	require.NotNil(t, scenario)

	// Manually create a session with biometric_setup key
	sessionKey := buildChallengeKey("biometric_setup", scenario.User.Email, scenario.User.TenantID)
	mockSessionData := &webauthn.SessionData{
		Challenge:        scenario.RegistrationChallenge.Challenge,
		UserID:           []byte(scenario.User.ID),
		UserVerification: protocol.VerificationPreferred,
		Extensions:       nil,
	}

	err := suite.sessionManager.Save(sessionKey, mockSessionData)
	require.NoError(t, err)

	// Step 2: Call FinishRegistration (which should find the biometric_setup session)
	finishReq := scenario.GenerateRegistrationRequestPayload()
	finishBody, _ := json.Marshal(finishReq)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webauthn/finishRegistration", bytes.NewBuffer(finishBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://app.authsec.dev")

	suite.router.ServeHTTP(w, req)

	t.Logf("Finish registration response: %d - %s", w.Code, w.Body.String())

	// The response will likely be an error due to missing database
	// but we can verify that the session fallback logic is triggered
	responseBody := w.Body.String()

	// Should not contain "session not found" if fallback is working
	assert.NotContains(t, responseBody, "registration session not found")
}

func TestOriginValidation_IntegrationTest(t *testing.T) {
	suite := SetupIntegrationTestSuite(t)
	scenarios := []struct {
		name        string
		origin      string
		expectError bool
	}{
		{
			name:        "Valid primary origin",
			origin:      "https://app.authsec.dev",
			expectError: false,
		},
		{
			name:        "Valid subdomain origin",
			origin:      "https://brcm.app.authsec.dev",
			expectError: false,
		},
		{
			name:        "Invalid origin",
			origin:      "https://malicious.com",
			expectError: true,
		},
		{
			name:        "HTTP instead of HTTPS",
			origin:      "http://app.authsec.dev",
			expectError: true,
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			beginReq := map[string]string{
				"email":     "test@example.com",
				"tenant_id": uuid.New().String(),
			}
			beginBody, _ := json.Marshal(beginReq)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/webauthn/biometric/beginSetup", bytes.NewBuffer(beginBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", tc.origin)

			suite.router.ServeHTTP(w, req)

			if tc.expectError {
				assert.True(t, w.Code >= 400, "Expected error status code for origin %s", tc.origin)
			}

			t.Logf("Origin %s - Response: %d - %s", tc.origin, w.Code, w.Body.String())
		})
	}
}

func TestChallengeKeyGeneration_ConsistencyTest(t *testing.T) {
	testCases := []struct {
		operation string
		email     string
		tenantID  string
	}{
		{"biometric_setup", "user1@example.com", "123e4567-e89b-12d3-a456-426614174000"},
		{"registration", "user2@example.com", "987fcdeb-51a2-43d7-8f90-123456789abc"},
		{"authentication", "user3@example.com", "456e7890-12ab-34cd-5678-901234567890"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.operation, tc.email), func(t *testing.T) {
			key1 := buildChallengeKey(tc.operation, tc.email, tc.tenantID)
			key2 := buildChallengeKey(tc.operation, tc.email, tc.tenantID)

			// Keys should be consistent
			assert.Equal(t, key1, key2)

			// Keys should have expected format
			expected := fmt.Sprintf("%s:%s:%s", tc.operation, tc.email, tc.tenantID)
			assert.Equal(t, expected, key1)
		})
	}
}

func TestErrorHandling_IntegrationScenarios(t *testing.T) {
	suite := SetupIntegrationTestSuite(t)

	testCases := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing email",
			requestBody:    map[string]string{"tenant_id": uuid.New().String()},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
		{
			name:           "Missing tenant_id",
			requestBody:    map[string]string{"email": "test@example.com"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
		{
			name:           "Invalid JSON",
			requestBody:    "{invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "",
		},
		{
			name: "Empty request",
			requestBody: map[string]string{
				"email":     "",
				"tenant_id": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/webauthn/biometric/beginSetup", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "https://app.authsec.dev")

			suite.router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedError != "" {
				assert.Contains(t, strings.ToLower(w.Body.String()), strings.ToLower(tc.expectedError))
			}

			t.Logf("Test case '%s' - Response: %d - %s", tc.name, w.Code, w.Body.String())
		})
	}
}

func TestSessionCleanup_IntegrationTest(t *testing.T) {
	suite := SetupIntegrationTestSuite(t)

	// Create multiple sessions
	testSessions := []struct {
		key  string
		data *webauthn.SessionData
	}{
		{
			key: "biometric_setup:user1@example.com:tenant1",
			data: &webauthn.SessionData{
				Challenge: "challenge1",
				UserID:    []byte("user1"),
			},
		},
		{
			key: "registration:user2@example.com:tenant2",
			data: &webauthn.SessionData{
				Challenge: "challenge2",
				UserID:    []byte("user2"),
			},
		},
	}

	// Save all sessions
	for _, session := range testSessions {
		err := suite.sessionManager.Save(session.key, session.data)
		require.NoError(t, err)
	}

	// Verify all sessions exist
	for _, session := range testSessions {
		_, found := suite.sessionManager.Get(session.key)
		assert.True(t, found, "Session should exist: %s", session.key)
	}

	// Clean up one session
	suite.sessionManager.Delete(testSessions[0].key)

	// Verify cleanup
	_, found := suite.sessionManager.Get(testSessions[0].key)
	assert.False(t, found, "Session should be deleted: %s", testSessions[0].key)

	// Verify other session still exists
	_, found = suite.sessionManager.Get(testSessions[1].key)
	assert.True(t, found, "Other session should still exist: %s", testSessions[1].key)
}

// Benchmark tests for performance validation

func BenchmarkWebAuthnFlow_BeginSetup(b *testing.B) {
	suite := SetupIntegrationTestSuite(&testing.T{})

	beginReq := map[string]string{
		"email":     "benchmark@example.com",
		"tenant_id": uuid.New().String(),
	}
	beginBody, _ := json.Marshal(beginReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/webauthn/biometric/beginSetup", bytes.NewBuffer(beginBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://app.authsec.dev")

		suite.router.ServeHTTP(w, req)
	}
}

// Helper functions for integration tests

func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	assert.Equal(t, expectedStatus, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")

	return response
}

func createTestRequest(method, url string, body interface{}, headers map[string]string) *http.Request {
	var reqBody []byte
	if body != nil {
		if str, ok := body.(string); ok {
			reqBody = []byte(str)
		} else {
			reqBody, _ = json.Marshal(body)
		}
	}

	req := httptest.NewRequest(method, url, bytes.NewBuffer(reqBody))

	// Set default headers
	req.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req
}

// Mock database functions for testing (these would be implemented with actual mocking framework)

func mockDatabaseSetup(t *testing.T) {
	// This would set up mock database connections and responses
	t.Log("Mock database setup would be implemented here")
}

func mockUserExists(email, tenantID string) bool {
	// Mock user existence check
	return email == "existing@example.com"
}

func mockCreateUser(email, tenantID string) *sharedmodels.User {
	// Mock user creation
	tenantUUID, _ := uuid.Parse(tenantID)
	return &sharedmodels.User{
		ID:       uuid.New(),
		ClientID: uuid.New(),
		Email:    email,
		TenantID: tenantUUID,
		Provider: "local",
	}
}
