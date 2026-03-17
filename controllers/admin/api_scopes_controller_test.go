package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAPIScopesTestDB creates an in-memory SQLite database for testing
func setupAPIScopesTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create tables
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tenants (
			id TEXT PRIMARY KEY,
			name TEXT,
			domain TEXT
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY,
			tenant_id TEXT,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, resource, action)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS api_scopes (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, name)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS api_scope_permissions (
			scope_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			PRIMARY KEY (scope_id, permission_id)
		)
	`).Error
	require.NoError(t, err)

	return db
}

// seedTestTenantAndPermissions creates test data
func seedTestTenantAndPermissions(t *testing.T, db *gorm.DB, tenantID uuid.UUID) []uuid.UUID {
	// Create tenant
	err := db.Exec("INSERT INTO tenants (id, name, domain) VALUES (?, ?, ?)",
		tenantID.String(), "Test Tenant", "test.com").Error
	require.NoError(t, err)

	// Create some permissions
	permIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	perms := []struct {
		id       uuid.UUID
		resource string
		action   string
	}{
		{permIDs[0], "project", "read"},
		{permIDs[1], "project", "write"},
		{permIDs[2], "invoice", "read"},
	}

	for _, p := range perms {
		err := db.Exec("INSERT INTO permissions (id, tenant_id, resource, action) VALUES (?, ?, ?, ?)",
			p.id.String(), tenantID.String(), p.resource, p.action).Error
		require.NoError(t, err)
	}

	return permIDs
}

// setupAPIScopesTestRouter creates a test router with the controller
func setupAPIScopesTestRouter(db *gorm.DB, tenantID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	controller := &APIScopesController{}

	// Mock tenant context middleware
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	})

	// Admin routes
	admin := r.Group("/admin")
	{
		admin.POST("/api_scopes", func(c *gin.Context) {
			controller.createAPIScope(c, db, tenantID)
		})
		admin.GET("/api_scopes", func(c *gin.Context) {
			controller.listAPIScopes(c, db, tenantID)
		})
		admin.GET("/api_scopes/:scope_id", func(c *gin.Context) {
			controller.getAPIScope(c, db, tenantID)
		})
		admin.PUT("/api_scopes/:scope_id", func(c *gin.Context) {
			controller.updateAPIScope(c, db, tenantID)
		})
		admin.DELETE("/api_scopes/:scope_id", func(c *gin.Context) {
			controller.deleteAPIScope(c, db, tenantID)
		})
	}

	return r
}

// TestCreateAPIScope tests the Create API Scope endpoint
func TestCreateAPIScope(t *testing.T) {
	tests := []struct {
		name           string
		payload        models.CreateAPIScopeRequest
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "Success - Create scope with permissions",
			payload: models.CreateAPIScopeRequest{
				Name:                "files:read",
				Description:         "Read access to files",
				MappedPermissionIDs: nil, // filled per test run
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["name"], "files:read")
				assert.Equal(t, float64(2), resp["permissions_linked"])
				assert.NotEmpty(t, resp["id"])
			},
		},
		{
			name: "Success - Create scope without permissions",
			payload: models.CreateAPIScopeRequest{
				Name:        "empty:scope",
				Description: "Scope with no permissions",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["name"], "empty:scope")
				assert.Equal(t, float64(0), resp["permissions_linked"])
			},
		},
		{
			name: "Error - Invalid scope name format",
			payload: models.CreateAPIScopeRequest{
				Name:        "invalidname", // Missing colon
				Description: "Invalid scope",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Error - Invalid permission ID",
			payload: models.CreateAPIScopeRequest{
				Name:                "test:scope",
				MappedPermissionIDs: []string{"not-a-uuid"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Error - Non-existent permission ID",
			payload: models.CreateAPIScopeRequest{
				Name:                "test:scope2",
				MappedPermissionIDs: []string{uuid.New().String()}, // Random UUID that doesn't exist
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAPIScopesTestDB(t)
			tenantID := uuid.New()
			permIDs := seedTestTenantAndPermissions(t, db, tenantID)
			router := setupAPIScopesTestRouter(db, tenantID)

			// ensure unique scope names per test run
			if tt.payload.Name != "" {
				tt.payload.Name = fmt.Sprintf("%s-%s", tt.payload.Name, uuid.NewString())
			}
			if len(tt.payload.MappedPermissionIDs) == 0 &&
				tt.expectedStatus == http.StatusOK &&
				strings.Contains(strings.ToLower(tt.name), "with permissions") {
				tt.payload.MappedPermissionIDs = []string{permIDs[0].String(), permIDs[1].String()}
			}

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				tt.checkResponse(t, resp)
			}
		})
	}
}

// TestCreateAPIScopeDuplicate tests duplicate scope name handling
func TestCreateAPIScopeDuplicate(t *testing.T) {
	db := setupAPIScopesTestDB(t)
	tenantID := uuid.New()
	seedTestTenantAndPermissions(t, db, tenantID)
	router := setupAPIScopesTestRouter(db, tenantID)

	// Create first scope
	payload1 := models.CreateAPIScopeRequest{
		Name:        "files:read",
		Description: "First scope",
	}
	body1, _ := json.Marshal(payload1)
	req1 := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Try to create duplicate scope
	payload2 := models.CreateAPIScopeRequest{
		Name:        "files:read", // Same name
		Description: "Duplicate scope",
	}
	body2, _ := json.Marshal(payload2)
	req2 := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

// TestListAPIScopes tests the List API Scopes endpoint
func TestListAPIScopes(t *testing.T) {
	db := setupAPIScopesTestDB(t)
	tenantID := uuid.New()
	permIDs := seedTestTenantAndPermissions(t, db, tenantID)
	router := setupAPIScopesTestRouter(db, tenantID)

	// Create some scopes
	scopes := []models.CreateAPIScopeRequest{
		{Name: "files:read", Description: "Read files", MappedPermissionIDs: []string{permIDs[0].String()}},
		{Name: "files:write", Description: "Write files", MappedPermissionIDs: []string{permIDs[1].String()}},
		{Name: "projects:read", Description: "Read projects"},
	}

	for _, s := range scopes {
		body, _ := json.Marshal(s)
		req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}

	t.Run("List all scopes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api_scopes", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp, 3)
	})

	t.Run("Filter by name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api_scopes?name=files", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp, 2) // files:read and files:write
	})
}

// TestGetAPIScope tests the Get API Scope endpoint
func TestGetAPIScope(t *testing.T) {
	db := setupAPIScopesTestDB(t)
	tenantID := uuid.New()
	permIDs := seedTestTenantAndPermissions(t, db, tenantID)
	router := setupAPIScopesTestRouter(db, tenantID)

	// Create a scope
	payload := models.CreateAPIScopeRequest{
		Name:                "files:read",
		Description:         "Read access to files",
		MappedPermissionIDs: []string{permIDs[0].String(), permIDs[1].String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	scopeID := createResp["id"].(string)

	t.Run("Get existing scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/api_scopes/%s", scopeID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "files:read", resp["name"])
		assert.Equal(t, float64(2), resp["permissions_linked"])
		assert.Len(t, resp["permission_ids"].([]interface{}), 2)
		assert.Len(t, resp["permission_strings"].([]interface{}), 2)
	})

	t.Run("Get non-existent scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/api_scopes/%s", uuid.New().String()), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Get with invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api_scopes/not-a-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestUpdateAPIScope tests the Update API Scope endpoint
func TestUpdateAPIScope(t *testing.T) {
	db := setupAPIScopesTestDB(t)
	tenantID := uuid.New()
	permIDs := seedTestTenantAndPermissions(t, db, tenantID)
	router := setupAPIScopesTestRouter(db, tenantID)

	// Create a scope
	payload := models.CreateAPIScopeRequest{
		Name:                "files:read",
		Description:         "Read access to files",
		MappedPermissionIDs: []string{permIDs[0].String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	scopeID := createResp["id"].(string)

	t.Run("Update name and description", func(t *testing.T) {
		updatePayload := models.UpdateAPIScopeRequest{
			Name:        "files:write",
			Description: "Write access to files",
		}
		body, _ := json.Marshal(updatePayload)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/api_scopes/%s", scopeID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "files:write", resp["name"])
		assert.Equal(t, "Write access to files", resp["description"])
	})

	t.Run("Update permissions", func(t *testing.T) {
		updatePayload := models.UpdateAPIScopeRequest{
			MappedPermissionIDs: []string{permIDs[0].String(), permIDs[1].String(), permIDs[2].String()},
		}
		body, _ := json.Marshal(updatePayload)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/api_scopes/%s", scopeID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(3), resp["permissions_linked"])
	})

	t.Run("Update non-existent scope", func(t *testing.T) {
		updatePayload := models.UpdateAPIScopeRequest{
			Name: "new:name",
		}
		body, _ := json.Marshal(updatePayload)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/api_scopes/%s", uuid.New().String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestDeleteAPIScope tests the Delete API Scope endpoint
func TestDeleteAPIScope(t *testing.T) {
	db := setupAPIScopesTestDB(t)
	tenantID := uuid.New()
	permIDs := seedTestTenantAndPermissions(t, db, tenantID)
	router := setupAPIScopesTestRouter(db, tenantID)

	// Create a scope
	payload := models.CreateAPIScopeRequest{
		Name:                "files:read",
		Description:         "Read access to files",
		MappedPermissionIDs: []string{permIDs[0].String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	scopeID := createResp["id"].(string)

	t.Run("Delete existing scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/api_scopes/%s", scopeID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "API scope deleted successfully", resp["message"])

		// Verify scope is deleted
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/api_scopes/%s", scopeID), nil)
		getW := httptest.NewRecorder()
		router.ServeHTTP(getW, getReq)
		assert.Equal(t, http.StatusNotFound, getW.Code)
	})

	t.Run("Delete non-existent scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/api_scopes/%s", uuid.New().String()), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Delete with invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/api_scopes/not-a-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestAPIScopeTransactionalIntegrity tests that create operations are transactional
func TestAPIScopeTransactionalIntegrity(t *testing.T) {
	db := setupAPIScopesTestDB(t)
	tenantID := uuid.New()
	permIDs := seedTestTenantAndPermissions(t, db, tenantID)
	router := setupAPIScopesTestRouter(db, tenantID)

	t.Run("Rollback on permission mapping failure", func(t *testing.T) {
		// This would require mocking the DB to fail on permission insert
		// For now, we test that invalid permission IDs don't create the scope
		payload := models.CreateAPIScopeRequest{
			Name:                "test:scope",
			Description:         "Test scope",
			MappedPermissionIDs: []string{permIDs[0].String(), uuid.New().String()}, // Second is invalid
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail validation
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify no scope was created
		listReq := httptest.NewRequest(http.MethodGet, "/admin/api_scopes", nil)
		listW := httptest.NewRecorder()
		router.ServeHTTP(listW, listReq)

		var resp []map[string]interface{}
		json.Unmarshal(listW.Body.Bytes(), &resp)
		assert.Len(t, resp, 0) // No scopes should exist
	})
}

// TestAPIScopeTenantIsolation tests that scopes are isolated per tenant
func TestAPIScopeTenantIsolation(t *testing.T) {
	db := setupAPIScopesTestDB(t)

	// Create two tenants
	tenant1 := uuid.New()
	tenant2 := uuid.New()
	perm1 := seedTestTenantAndPermissions(t, db, tenant1)
	perm2 := seedTestTenantAndPermissions(t, db, tenant2)

	router1 := setupAPIScopesTestRouter(db, tenant1)
	router2 := setupAPIScopesTestRouter(db, tenant2)

	// Create scope in tenant1
	payload := models.CreateAPIScopeRequest{
		Name:                "files:read",
		Description:         "Tenant 1 files",
		MappedPermissionIDs: []string{perm1[0].String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router1.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	scopeID := createResp["id"].(string)

	t.Run("Tenant2 cannot see Tenant1's scopes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api_scopes", nil)
		w := httptest.NewRecorder()
		router2.ServeHTTP(w, req)

		var resp []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Len(t, resp, 0) // Tenant2 sees no scopes
	})

	t.Run("Tenant2 cannot access Tenant1's scope by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/api_scopes/%s", scopeID), nil)
		w := httptest.NewRecorder()
		router2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Tenant2 can create same scope name", func(t *testing.T) {
		payload := models.CreateAPIScopeRequest{
			Name:                "files:read", // Same name as tenant1
			Description:         "Tenant 2 files",
			MappedPermissionIDs: []string{perm2[0].String()},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // Should succeed
	})

	t.Run("Tenant1 cannot use Tenant2's permissions", func(t *testing.T) {
		payload := models.CreateAPIScopeRequest{
			Name:                "cross:tenant",
			Description:         "Cross tenant attempt",
			MappedPermissionIDs: []string{perm2[0].String()}, // Using Tenant2's permission
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/admin/api_scopes", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router1.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code) // Should fail validation
	})
}
