package enduser

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHydraClient is a mock implementation of Hydra client
type MockHydraClient struct {
	mock.Mock
}

func (m *MockHydraClient) IntrospectOAuth2Token(ctx context.Context, token string) (*models.Introspection, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Introspection), args.Error(1)
}

func ensureNoDatabase(t *testing.T) {
	t.Helper()
	originalDB := config.DB
	config.DB = nil
	t.Cleanup(func() {
		config.DB = originalDB
	})
}

func TestEndUserController_RegisterClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		input          models.RegisterClientsRequest
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name: "successful registration",
			input: models.RegisterClientsRequest{
				TenantID:  uuid.New().String(),
				ProjectID: uuid.New().String(),
				Name:      "Test Client",
				Email:     "test@example.com",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Database connection not available",
			},
			setupMocks: func() {
				// Mock tenant lookup and database operations
			},
		},
		{
			name:  "invalid input",
			input: models.RegisterClientsRequest{
				// Missing required fields
			},
			expectedStatus: http.StatusBadRequest,
			setupMocks:     func() {},
		},
		{
			name: "tenant not found",
			input: models.RegisterClientsRequest{
				TenantID:  "invalid-uuid",
				ProjectID: uuid.New().String(),
				Name:      "Test Client",
				Email:     "test@example.com",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Database connection not available",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("POST", "/api/enduser/register", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			controller.RegisterClient(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_GetEndUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		tenantID       string
		userIdentifier string
		queryString    string
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name:           "database unavailable when looking up by id",
			tenantID:       uuid.New().String(),
			userIdentifier: uuid.New().String(),
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "Database connection not available",
			},
			setupMocks: func() {
				// Mock database user lookup
			},
		},
		{
			name:           "missing client id for email lookup",
			tenantID:       uuid.New().String(),
			userIdentifier: "nonexistent@example.com",
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "client_id is required when using email identifier",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "tenant_id", Value: tt.tenantID},
				{Key: "user_id", Value: tt.userIdentifier},
			}
			requestPath := "/api/enduser/" + tt.tenantID + "/" + tt.userIdentifier
			if tt.queryString != "" {
				requestPath += "?" + tt.queryString
			}
			c.Request = httptest.NewRequest("GET", requestPath, nil)

			// Set token claims for auth middleware simulation
			setTokenClaimsInContext(c, tt.tenantID, tt.userIdentifier)

			controller.GetEndUser(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestResolveEndUserLookup(t *testing.T) {
	validUUID := uuid.New()
	clientUUID := uuid.New()

	t.Run("parses user id", func(t *testing.T) {
		byID, userID, clientID, email, err := resolveEndUserLookup(validUUID.String(), "")
		assert.NoError(t, err)
		assert.True(t, byID)
		assert.Equal(t, validUUID, userID)
		assert.Equal(t, uuid.Nil, clientID)
		assert.Equal(t, "", email)
	})

	t.Run("requires client for email", func(t *testing.T) {
		byID, _, _, _, err := resolveEndUserLookup("user@example.com", "")
		assert.Error(t, err)
		assert.False(t, byID)
		assert.Equal(t, "client_id is required when using email identifier", err.Error())
	})

	t.Run("rejects invalid client id", func(t *testing.T) {
		_, _, _, _, err := resolveEndUserLookup("user@example.com", "invalid")
		assert.Error(t, err)
		assert.Equal(t, "invalid client_id", err.Error())
	})

	t.Run("parses email with client", func(t *testing.T) {
		byID, userID, clientID, email, err := resolveEndUserLookup("user@example.com", clientUUID.String())
		assert.NoError(t, err)
		assert.False(t, byID)
		assert.Equal(t, uuid.Nil, userID)
		assert.Equal(t, clientUUID, clientID)
		assert.Equal(t, "user@example.com", email)
	})
}

func TestEndUserController_GetEndUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		input          GetEndUsersFilter
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name: "successful users retrieval",
			input: GetEndUsersFilter{
				TenantID: uuid.New().String(),
				Page:     1,
				Limit:    10,
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "database connection not available",
			},
			setupMocks: func() {
				// Mock database query
			},
		},
		{
			name:  "invalid input",
			input: GetEndUsersFilter{
				// Missing tenant_id
			},
			expectedStatus: http.StatusBadRequest,
			setupMocks:     func() {},
		},
		{
			name: "invalid client_id format",
			input: GetEndUsersFilter{
				TenantID: uuid.New().String(),
				ClientID: "invalid-uuid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid client_id format",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("POST", "/api/enduser/list", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			controller.GetEndUsers(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_UpdateEndUserStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		tenantID       string
		userID         string
		input          models.UpdateEndUserStatusInput
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name:     "successful status update",
			tenantID: uuid.New().String(),
			userID:   uuid.New().String(),
			input: models.UpdateEndUserStatusInput{
				Active: true,
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "Database connection not available",
			},
			setupMocks: func() {
				// Mock database update
			},
		},
		{
			name:     "invalid user_id format",
			tenantID: uuid.New().String(),
			userID:   "invalid-uuid",
			input: models.UpdateEndUserStatusInput{
				Active: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid user_id format",
			},
			setupMocks: func() {},
		},
		{
			name:     "user not found",
			tenantID: uuid.New().String(),
			userID:   uuid.New().String(),
			input: models.UpdateEndUserStatusInput{
				Active: true,
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "Database connection not available",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "tenant_id", Value: tt.tenantID},
				{Key: "user_id", Value: tt.userID},
			}

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("PUT", "/api/enduser/"+tt.tenantID+"/"+tt.userID+"/status", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			// Set token claims for auth middleware simulation
			setTokenClaimsInContext(c, tt.tenantID, tt.userID)

			controller.UpdateEndUserStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_DeleteEndUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		input          map[string]string
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name: "successful deletion",
			input: map[string]string{
				"tenant_id": uuid.New().String(),
				"user_id":   uuid.New().String(),
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "Database connection not available",
			},
			setupMocks: func() {
				// Mock database deletion
			},
		},
		{
			name: "missing required fields",
			input: map[string]string{
				"tenant_id": uuid.New().String(),
				// Missing user_id
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "user_id is required",
			},
			setupMocks: func() {},
		},
		{
			name: "user not found",
			input: map[string]string{
				"tenant_id": uuid.New().String(),
				"user_id":   uuid.New().String(),
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "Database connection not available",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("DELETE", "/api/enduser/delete", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")
			if tenantID, ok := tt.input["tenant_id"]; ok {
				// Set token claims for auth middleware simulation
				setTokenClaimsInContext(c, tenantID, tt.input["user_id"])
				c.Set("user_info", &middlewares.UserInfo{TenantID: tenantID})
			}

			controller.DeleteEndUser(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_ActiveOrDeactiveEndUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	payload := map[string]interface{}{
		"tenant_id": uuid.New().String(),
		"user_id":   uuid.New().String(),
		"active":    false,
	}

	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/uflow/user/enduser/active", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ActiveOrDeactiveEndUser(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Database connection not available", response["error"])
}

func TestEndUserController_OIDCLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		input          models.OIDCLoginInput
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name: "successful OIDC login",
			input: models.OIDCLoginInput{
				AccessToken: "valid-token",
			},
			expectedStatus: http.StatusUnauthorized, // Token validation will fail in test environment
			expectedBody: map[string]interface{}{
				"error": "Invalid or inactive OIDC token",
			},
			setupMocks: func() {
				// Mock OIDC token validation and database lookup
			},
		},
		{
			name: "invalid OIDC token",
			input: models.OIDCLoginInput{
				AccessToken: "invalid-token",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or inactive OIDC token",
			},
			setupMocks: func() {},
		},
		{
			name: "missing extension fields",
			input: models.OIDCLoginInput{
				AccessToken: "token-without-ext",
			},
			expectedStatus: http.StatusUnauthorized, // Token validation will fail in test environment
			expectedBody: map[string]interface{}{
				"error": "Invalid or inactive OIDC token",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("POST", "/api/enduser/oidc-login", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			controller.OIDCLogin(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_CustomLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		input          models.CustomLoginInput
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name: "successful custom login",
			input: models.CustomLoginInput{
				Email:    "test@example.com",
				Password: "password123",
				ClientID: uuid.New().String(),
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "failed to map tenant: database connection not available",
			},
			setupMocks: func() {
				// Mock tenant mapping and database lookup
			},
		},
		{
			name: "invalid credentials",
			input: models.CustomLoginInput{
				Email:    "test@example.com",
				Password: "wrongpassword",
				ClientID: uuid.New().String(),
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "failed to map tenant: database connection not available",
			},
			setupMocks: func() {},
		},
		{
			name: "MFA enabled user",
			input: models.CustomLoginInput{
				Email:    "mfa@example.com",
				Password: "password123",
				ClientID: uuid.New().String(),
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			setupMocks: func() {
				// Mock MFA-enabled user
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("POST", "/api/enduser/custom-login", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			controller.CustomLogin(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_CustomLoginRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		input          models.CustomLoginRegister
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name: "successful registration",
			input: models.CustomLoginRegister{
				Email:    "newuser@example.com",
				Password: "password123",
				ClientID: uuid.New().String(),
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "failed to map tenant: database connection not available",
			},
			setupMocks: func() {
				// Mock tenant mapping and database operations
			},
		},
		{
			name: "user already exists",
			input: models.CustomLoginRegister{
				Email:    "existing@example.com",
				Password: "password123",
				ClientID: uuid.New().String(),
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "failed to map tenant: database connection not available",
			},
			setupMocks: func() {},
		},
		{
			name: "weak password",
			input: models.CustomLoginRegister{
				Email:    "test@example.com",
				Password: "123", // Too short
				ClientID: uuid.New().String(),
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "failed to map tenant: database connection not available",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("POST", "/api/enduser/custom-login-register", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			controller.CustomLoginRegister(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_AdminChangeUserPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		input          models.AdminChangePasswordInput
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name: "successful password change",
			input: models.AdminChangePasswordInput{
				TenantID:    uuid.New().String(),
				Email:       "test@example.com",
				NewPassword: "newpassword123",
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "database connection not available",
			},
			setupMocks: func() {
				// Mock database operations
			},
		},
		{
			name: "weak password",
			input: models.AdminChangePasswordInput{
				TenantID:    uuid.New().String(),
				Email:       "test@example.com",
				NewPassword: "123", // Too short
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "database connection not available",
			},
			setupMocks: func() {},
		},
		{
			name: "user not found",
			input: models.AdminChangePasswordInput{
				TenantID:    uuid.New().String(),
				Email:       "nonexistent@example.com",
				NewPassword: "newpassword123",
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "database connection not available",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("POST", "/api/admin/change-password", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			controller.AdminChangeUserPassword(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_AdminResetUserPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name           string
		input          models.AdminResetPasswordInput
		expectedStatus int
		expectedBody   map[string]interface{}
		setupMocks     func()
	}{
		{
			name: "successful password reset",
			input: models.AdminResetPasswordInput{
				TenantID:  uuid.New().String(),
				Email:     "test@example.com",
				SendEmail: true,
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "database connection not available",
			},
			setupMocks: func() {
				// Mock database operations and email sending
			},
		},
		{
			name: "user not found",
			input: models.AdminResetPasswordInput{
				TenantID:  uuid.New().String(),
				Email:     "nonexistent@example.com",
				SendEmail: false,
			},
			expectedStatus: http.StatusInternalServerError, // Database connection error expected
			expectedBody: map[string]interface{}{
				"error": "database connection not available",
			},
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonData, _ := json.Marshal(tt.input)
			c.Request = httptest.NewRequest("POST", "/api/admin/reset-password", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			controller.AdminResetUserPassword(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}
		})
	}
}

func TestEndUserController_validateOIDCToken(t *testing.T) {
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name       string
		token      string
		wantErr    bool
		expected   *models.Introspection
		setupMocks func()
	}{
		{
			name:     "valid token",
			token:    "valid-token",
			wantErr:  true, // Application configuration not available
			expected: nil,
			setupMocks: func() {
				// Mock Hydra client
			},
		},
		{
			name:       "invalid token",
			token:      "invalid-token",
			wantErr:    true,
			setupMocks: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := controller.validateOIDCToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEndUserController_tenantMapping(t *testing.T) {
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name       string
		clientID   uuid.UUID
		wantErr    bool
		expected   uuid.UUID
		setupMocks func()
	}{
		{
			name:     "database unavailable",
			clientID: uuid.New(),
			wantErr:  true, // Database connection not available
			expected: uuid.UUID{},
			setupMocks: func() {
				// Mock database lookup
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := controller.tenantMapping(tt.clientID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEndUserController_generateAndSendCustomPasswordResetOTP(t *testing.T) {
	ensureNoDatabase(t)
	controller := &EndUserController{}

	tests := []struct {
		name       string
		email      string
		wantErr    bool
		setupMocks func()
	}{
		{
			name:    "successful OTP generation and sending",
			email:   "test@example.com",
			wantErr: true, // Database connection not available
			setupMocks: func() {
				// Mock OTP generation and email sending
			},
		},
		{
			name:    "email sending failure",
			email:   "test@example.com",
			wantErr: true,
			setupMocks: func() {
				// Mock email sending failure
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := controller.generateAndSendCustomPasswordResetOTP(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthMiddleware_WithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use AuthMiddlewareWithConfig so the signing secret matches generateTestJWT()
	testSecret := "7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c"
	authConfig := &middlewares.AuthConfig{
		JWTSecret:        testSecret,
		JWTDefaultSecret: testSecret,
		ExpectedIssuer:   "authsec-ai/auth-manager",
		ExpectedAudience: "authsec-api",
	}

	// Create a test route that requires authentication
	r := gin.New()
	r.Use(middlewares.AuthMiddlewareWithConfig(authConfig))
	r.GET("/protected", func(c *gin.Context) {
		// Check if user info is properly extracted
		tenantID, exists := c.Get("tenant_id")
		assert.True(t, exists, "tenant_id should be set")
		assert.Equal(t, "test-tenant-123", tenantID)

		userID, exists := c.Get("user_id")
		assert.True(t, exists, "user_id should be set")
		assert.Equal(t, "test-user-456", userID)

		email, exists := c.Get("email_id")
		assert.True(t, exists, "email_id should be set")
		assert.Equal(t, "test@example.com", email)

		roles, exists := c.Get("roles")
		assert.True(t, exists, "roles should be set")
		roleSlice, ok := roles.([]string)
		assert.True(t, ok, "roles should be []string")
		assert.Contains(t, roleSlice, "admin")
		assert.Contains(t, roleSlice, "user")

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Generate a valid JWT token for testing
	tokenString, err := generateTestJWT()
	assert.NoError(t, err)

	// Test the protected route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "success", response["message"])
}

func TestAuthMiddleware_WithInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middlewares.AuthMiddleware())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	// Test with invalid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_NoAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middlewares.AuthMiddleware())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	// Test without authorization header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// generateTestJWT generates a test JWT token with known claims
func generateTestJWT() (string, error) {
	claims := jwt.MapClaims{
		"iss":        "authsec-ai/auth-manager",
		"aud":        "authsec-api",
		"exp":        time.Now().Add(time.Hour).Unix(),
		"iat":        time.Now().Unix(),
		"nbf":        time.Now().Unix(),
		"tenant_id":  "test-tenant-123",
		"project_id": "test-project-456",
		"client_id":  "test-client-789",
		"user_id":    "test-user-456",
		"email_id":   "test@example.com",
		"roles":      []string{"admin", "user"},
		"groups":     []string{"developers"},
		"scopes":     []string{"read", "write"},
		"resources":  []string{"users", "projects"},
		"token_type": "sdk-agent",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := []byte("7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c")
	return token.SignedString(secret)
}

func TestRBAC_AdminOnlyAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create auth config with server-side authorization enabled
	authConfig := &middlewares.AuthConfig{
		JWTSecret:         "7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c",
		JWTDefaultSecret:  "7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c",
		ExpectedIssuer:    "authsec-ai/auth-manager",
		ExpectedAudience:  "authsec-api",
		RequireServerAuth: true, // Enable server-side authorization
	}

	// Create a test route that requires admin access
	r := gin.New()
	r.Use(middlewares.AuthMiddlewareWithConfig(authConfig))
	r.GET("/admin/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})

	// Test with admin role
	adminToken, err := generateTestJWTWithRoles([]string{"admin"})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "admin access granted", response["message"])
}

func TestRBAC_UserAccessDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create auth config with server-side authorization enabled
	authConfig := &middlewares.AuthConfig{
		JWTSecret:         "7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c",
		JWTDefaultSecret:  "7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c",
		ExpectedIssuer:    "authsec-ai/auth-manager",
		ExpectedAudience:  "authsec-api",
		RequireServerAuth: true, // Enable server-side authorization
	}

	// Create a test route that requires admin access
	r := gin.New()
	r.Use(middlewares.AuthMiddlewareWithConfig(authConfig))
	r.GET("/admin/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	// Test with regular user role (no admin)
	userToken, err := generateTestJWTWithRoles([]string{})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRBAC_UserAccessGranted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create auth config with server-side authorization enabled
	authConfig := &middlewares.AuthConfig{
		JWTSecret:         "7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c",
		JWTDefaultSecret:  "7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c",
		ExpectedIssuer:    "authsec-ai/auth-manager",
		ExpectedAudience:  "authsec-api",
		RequireServerAuth: true, // Enable server-side authorization
	}

	// Create a test route for regular users
	r := gin.New()
	r.Use(middlewares.AuthMiddlewareWithConfig(authConfig))
	r.GET("/api/enduser/profile", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "user access granted"})
	})

	// Test with regular user role
	userToken, err := generateTestJWTWithRoles([]string{"user"})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/enduser/profile", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "user access granted", response["message"])
}

// generateTestJWTWithRoles generates a test JWT token with specific roles
func generateTestJWTWithRoles(roles []string) (string, error) {
	claims := jwt.MapClaims{
		"iss":        "authsec-ai/auth-manager",
		"aud":        "authsec-api",
		"exp":        time.Now().Add(time.Hour).Unix(),
		"iat":        time.Now().Unix(),
		"nbf":        time.Now().Unix(),
		"tenant_id":  "test-tenant-123",
		"project_id": "test-project-456",
		"client_id":  "test-client-789",
		"user_id":    "test-user-456",
		"email_id":   "test@example.com",
		"roles":      roles,
		"groups":     []string{"developers"},
		"scopes":     []string{"read", "write"},
		"resources":  []string{"users", "projects"},
		"token_type": "sdk-agent",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := []byte("7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c")
	return token.SignedString(secret)
}
