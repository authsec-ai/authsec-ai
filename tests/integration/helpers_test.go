//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// testResponse wraps the HTTP response for convenient assertions.
type testResponse struct {
	Code int
	Body []byte
	JSON map[string]interface{}
}

// doRequest sends an HTTP request through the test router.
func doRequest(method, path string, body interface{}, token string) *testResponse {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBytes)
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	resp := &testResponse{Code: w.Code, Body: w.Body.Bytes()}
	_ = json.Unmarshal(resp.Body, &resp.JSON) // best-effort parse
	return resp
}

// doAdminRequest sends an authenticated admin request.
func doAdminRequest(method, path string, body interface{}) *testResponse {
	return doRequest(method, path, body, generateAdminToken())
}

// doEndUserRequest sends an authenticated end-user request.
func doEndUserRequest(method, path string, body interface{}) *testResponse {
	return doRequest(method, path, body, generateEndUserToken())
}

// doUnauthRequest sends an unauthenticated request.
func doUnauthRequest(method, path string, body interface{}) *testResponse {
	return doRequest(method, path, body, "")
}

// generateAdminToken creates a valid admin JWT token.
func generateAdminToken() string {
	return generateTokenForTenant(
		testTenantID, testAdminUserID, testAdminEmail,
		[]string{"admin", "super_admin"},
		testTenantDomain,
	)
}

// generateEndUserToken creates a valid end-user JWT token.
func generateEndUserToken() string {
	return generateTokenForTenant(
		testTenantID, testEndUserID, testEndUserEmail,
		[]string{"user"},
		testTenantDomain,
	)
}

// generateExpiredToken creates an expired JWT token for auth rejection tests.
func generateExpiredToken() string {
	now := time.Now().Add(-2 * time.Hour)
	claims := jwt.MapClaims{
		"sub":       testAdminUserID.String(),
		"user_id":   testAdminUserID.String(),
		"email":     testAdminEmail,
		"tenant_id": testTenantID.String(),
		"roles":     []string{"admin"},
		"iss":       "authsec-ai/auth-manager",
		"aud":       "authsec-api",
		"iat":       now.Unix(),
		"nbf":       now.Unix(),
		"exp":       now.Add(1 * time.Hour).Unix(), // expired 1 hour ago
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtDefSecret))
	return tokenString
}

// generateTokenForTenant creates a JWT token with configurable claims.
func generateTokenForTenant(tenantID, userID uuid.UUID, email string, roles []string, tenantDomain string) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":           userID.String(),
		"user_id":       userID.String(),
		"email":         email,
		"email_id":      email,
		"tenant_id":     tenantID.String(),
		"client_id":     testClientID.String(),
		"project_id":    testProjectID.String(),
		"tenant_domain": tenantDomain,
		"roles":         roles,
		"scope":         "admin:* users:* tenants:* clients:* roles:* permissions:* external-service:*",
		"token_type":    "default",
		"iss":           "authsec-ai/auth-manager",
		"aud":           "authsec-api",
		"iat":           now.Unix(),
		"nbf":           now.Unix(),
		"exp":           now.Add(1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtDefSecret))
	return tokenString
}

// assertRowExists checks that at least one row matches the query.
func assertRowExists(t *testing.T, table, whereClause string, args ...interface{}) {
	t.Helper()
	query := "SELECT COUNT(*) FROM " + table + " WHERE " + whereClause
	var count int
	err := config.Database.DB.QueryRow(query, args...).Scan(&count)
	assert.NoError(t, err, "query failed for %s", table)
	assert.Greater(t, count, 0, "expected row in %s WHERE %s", table, whereClause)
}

// assertRowCount checks the exact number of rows matching the query.
func assertRowCount(t *testing.T, table string, expected int, whereClause string, args ...interface{}) {
	t.Helper()
	query := "SELECT COUNT(*) FROM " + table + " WHERE " + whereClause
	var count int
	err := config.Database.DB.QueryRow(query, args...).Scan(&count)
	assert.NoError(t, err, "query failed for %s", table)
	assert.Equal(t, expected, count, "row count mismatch in %s WHERE %s", table, whereClause)
}

// assertNotPanic checks that the response is a valid HTTP response (not a panic).
func assertNotPanic(t *testing.T, resp *testResponse) {
	t.Helper()
	assert.NotEqual(t, 0, resp.Code, "response code should not be 0 (panic)")
	assert.True(t, resp.Code >= 200 && resp.Code < 600, "response code %d out of range", resp.Code)
}

// assertJSON checks that the response body is valid JSON.
func assertJSON(t *testing.T, resp *testResponse) {
	t.Helper()
	if len(resp.Body) > 0 {
		var raw json.RawMessage
		err := json.Unmarshal(resp.Body, &raw)
		assert.NoError(t, err, "response body is not valid JSON: %s", string(resp.Body))
	}
}
