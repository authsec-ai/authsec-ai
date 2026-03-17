package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/authsec-ai/authsec/config"
)

func init() {
	// Initialize config.AppConfig for tests
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{}
	}
}

func TestRegisterClientWithHydra_Success(t *testing.T) {
	if config.AppConfig.JWTSdkSecret == "" {
		t.Skip("skipping hydra client tests: JWTSdkSecret not configured")
	}

	// Create a mock OOC Manager server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify URL path
		if r.URL.Path != "/oocmgr/tenant/create-base-client" {
			t.Errorf("Expected path /oocmgr/tenant/create-base-client, got %s", r.URL.Path)
		}

		// Verify Content-Type header
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse request body
		var oocManager OOCManager
		if err := json.NewDecoder(r.Body).Decode(&oocManager); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify request body fields
		if oocManager.TenantID != "test-tenant-id" {
			t.Errorf("Expected TenantID test-tenant-id, got %s", oocManager.TenantID)
		}
		if oocManager.TenantName != "Test Client" {
			t.Errorf("Expected TenantName 'Test Client', got %s", oocManager.TenantName)
		}
		if oocManager.ClientID != "test-client-id-main-client" {
			t.Errorf("Expected ClientID test-client-id-main-client, got %s", oocManager.ClientID)
		}
		if oocManager.ClientSecret != "test-secret" {
			t.Errorf("Expected ClientSecret test-secret, got %s", oocManager.ClientSecret)
		}
		if len(oocManager.RedirectURIs) != 1 || oocManager.RedirectURIs[0] != "https://test.example.com/oidc/auth/callback" {
			t.Errorf("Unexpected RedirectURIs: %v", oocManager.RedirectURIs)
		}
		if len(oocManager.Scopes) != 4 {
			t.Errorf("Expected 4 scopes, got %d", len(oocManager.Scopes))
		}

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer mockServer.Close()

	// Set mock OOC Manager URL in config
	config.AppConfig.OOCManagerURL = mockServer.URL

	// Test RegisterClientWithHydra
	err := RegisterClientWithHydra(
		"test-client-id",
		"test-secret",
		"Test Client",
		"test-tenant-id",
		"test.example.com",
	)

	if err != nil {
		t.Errorf("RegisterClientWithHydra failed: %v", err)
	}
}

func TestRegisterClientWithHydra_ErrorResponse(t *testing.T) {
	if config.AppConfig.JWTSdkSecret == "" {
		t.Skip("skipping hydra client tests: JWTSdkSecret not configured")
	}

	// Create a mock OOC Manager server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid request"}`))
	}))
	defer mockServer.Close()

	// Set mock OOC Manager URL in config
	config.AppConfig.OOCManagerURL = mockServer.URL

	// Test RegisterClientWithHydra
	err := RegisterClientWithHydra(
		"test-client-id",
		"test-secret",
		"Test Client",
		"test-tenant-id",
		"test.example.com",
	)

	if err == nil {
		t.Error("Expected error when OOC Manager returns 400, got nil")
	}

	expectedError := "OOC Manager API returned status 400"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestRegisterClientWithHydra_NetworkError(t *testing.T) {
	if config.AppConfig.JWTSdkSecret == "" {
		t.Skip("skipping hydra client tests: JWTSdkSecret not configured")
	}

	// Set invalid OOC Manager URL
	config.AppConfig.OOCManagerURL = "http://invalid-host:9999"

	// Test RegisterClientWithHydra
	err := RegisterClientWithHydra(
		"test-client-id",
		"test-secret",
		"Test Client",
		"test-tenant-id",
		"test.example.com",
	)

	if err == nil {
		t.Error("Expected error when OOC Manager is unreachable, got nil")
	}
}

func TestRegisterClientWithHydra_FallbackURL(t *testing.T) {
	if config.AppConfig.JWTSdkSecret == "" {
		t.Skip("skipping hydra client tests: JWTSdkSecret not configured")
	}

	// Create a mock server on the default port
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// Clear OOC Manager URL to test fallback
	originalURL := config.AppConfig.OOCManagerURL
	config.AppConfig.OOCManagerURL = ""
	defer func() { config.AppConfig.OOCManagerURL = originalURL }()

	// This will use the fallback URL http://localhost:7467
	// We expect it to fail since there's no server on that port
	err := RegisterClientWithHydra(
		"test-client-id",
		"test-secret",
		"Test Client",
		"test-tenant-id",
		"test.example.com",
	)

	// We expect an error because localhost:7467 won't be running
	if err == nil {
		t.Error("Expected error when using fallback URL with no server")
	}
}

func TestOOCManagerStruct(t *testing.T) {
	oocManager := OOCManager{
		TenantID:     "tenant-123",
		TenantName:   "Test Tenant",
		ClientID:     "client-123",
		ClientSecret: "secret-123",
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"openid", "profile"},
		CreatedBy:    "system",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(oocManager)
	if err != nil {
		t.Errorf("Failed to marshal OOCManager: %v", err)
	}

	// Unmarshal back
	var decoded OOCManager
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Errorf("Failed to unmarshal OOCManager: %v", err)
	}

	// Verify fields
	if decoded.TenantID != oocManager.TenantID {
		t.Errorf("TenantID mismatch: expected %s, got %s", oocManager.TenantID, decoded.TenantID)
	}
	if decoded.ClientID != oocManager.ClientID {
		t.Errorf("ClientID mismatch: expected %s, got %s", oocManager.ClientID, decoded.ClientID)
	}
}

func TestDeleteClientFromHydra_Success(t *testing.T) {
	if config.AppConfig.JWTSdkSecret == "" {
		t.Skip("skipping hydra client tests: JWTSdkSecret not configured")
	}

	// Create a mock OOC Manager server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify URL path
		if r.URL.Path != "/oocmgr/tenant/delete-complete" {
			t.Errorf("Expected path /oocmgr/tenant/delete-complete, got %s", r.URL.Path)
		}

		// Parse request body
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify client_id
		expectedClientID := "test-client-id-main-client"
		if payload["client_id"] != expectedClientID {
			t.Errorf("Expected client_id %s, got %s", expectedClientID, payload["client_id"])
		}

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"deleted"}`))
	}))
	defer mockServer.Close()

	// Set mock OOC Manager URL in config
	config.AppConfig.OOCManagerURL = mockServer.URL

	// Test DeleteClientFromHydra
	err := DeleteClientFromHydra("test-client-id")

	if err != nil {
		t.Errorf("DeleteClientFromHydra failed: %v", err)
	}
}

func TestDeleteClientFromHydra_ErrorResponse(t *testing.T) {
	if config.AppConfig.JWTSdkSecret == "" {
		t.Skip("skipping hydra client tests: JWTSdkSecret not configured")
	}

	// Create a mock OOC Manager server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"client not found"}`))
	}))
	defer mockServer.Close()

	// Set mock OOC Manager URL in config
	config.AppConfig.OOCManagerURL = mockServer.URL

	// Test DeleteClientFromHydra
	err := DeleteClientFromHydra("test-client-id")

	if err == nil {
		t.Error("Expected error when OOC Manager returns 404, got nil")
	}

	expectedError := "OOC Manager API returned status 404 during deletion"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestUpdateClientInHydra_Success(t *testing.T) {
	if config.AppConfig.JWTSdkSecret == "" {
		t.Skip("skipping hydra client tests: JWTSdkSecret not configured")
	}

	// Create a mock OOC Manager server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify URL path
		if r.URL.Path != "/oocmgr/tenant/update-complete" {
			t.Errorf("Expected path /oocmgr/tenant/update-complete, got %s", r.URL.Path)
		}

		// Parse request body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify payload fields
		if payload["client_id"] != "test-client-id-main-client" {
			t.Errorf("Unexpected client_id: %v", payload["client_id"])
		}
		if payload["client_secret"] != "new-secret" {
			t.Errorf("Unexpected client_secret: %v", payload["client_secret"])
		}
		if payload["tenant_id"] != "test-tenant-id" {
			t.Errorf("Unexpected tenant_id: %v", payload["tenant_id"])
		}
		if payload["email"] != "test@example.com" {
			t.Errorf("Unexpected email: %v", payload["email"])
		}

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"updated"}`))
	}))
	defer mockServer.Close()

	// Set mock OOC Manager URL in config
	config.AppConfig.OOCManagerURL = mockServer.URL

	// Test UpdateClientInHydra
	err := UpdateClientInHydra("test-client-id", "new-secret", "test@example.com", "test-tenant-id")

	if err != nil {
		t.Errorf("UpdateClientInHydra failed: %v", err)
	}
}

func TestUpdateClientInHydra_ErrorResponse(t *testing.T) {
	if config.AppConfig.JWTSdkSecret == "" {
		t.Skip("skipping hydra client tests: JWTSdkSecret not configured")
	}

	// Create a mock OOC Manager server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid update"}`))
	}))
	defer mockServer.Close()

	// Set mock OOC Manager URL in config
	config.AppConfig.OOCManagerURL = mockServer.URL

	// Test UpdateClientInHydra
	err := UpdateClientInHydra("test-client-id", "new-secret", "test@example.com", "test-tenant-id")

	if err == nil {
		t.Error("Expected error when OOC Manager returns 400, got nil")
	}

	expectedError := "OOC Manager API returned status 400 during update"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}
