package enduser

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestActiveOrDeactiveEndUser_UpdatesStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &EndUserController{}

	tenantID := uuid.New()
	userID := uuid.New()

	db := setupTenantTestDB(t, tenantID, userID, false)

	overrideTenantConnection(t, db)
	overrideTimeNow(t)
	overrideConfigDB(t, db)

	if err := db.Table("users").Create(map[string]interface{}{
		"id":         uuid.New().String(),
		"tenant_id":  tenantID.String(),
		"active":     true,
		"updated_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("failed to insert secondary user: %v", err)
	}

	payload := map[string]interface{}{
		"tenant_id": tenantID.String(),
		"user_id":   userID.String(),
		"active":    true,
	}

	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/uflow/user/enduser/active", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ActiveOrDeactiveEndUser(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "User updated successfully", response["message"])

	var active bool
	if err := db.Table("users").Where("id = ? AND tenant_id = ?", userID.String(), tenantID.String()).Pluck("active", &active).Error; err != nil {
		t.Fatalf("failed to fetch user row: %v", err)
	}
	assert.True(t, active)
}

func TestDeleteEndUser_SoftDeletesRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &EndUserController{}

	tenantID := uuid.New()
	userID := uuid.New()

	db := setupTenantTestDB(t, tenantID, userID, true)

	overrideTenantConnection(t, db)
	overrideTimeNow(t)
	overrideConfigDB(t, db)

	if err := db.Table("users").Create(map[string]interface{}{
		"id":         uuid.New().String(),
		"tenant_id":  tenantID.String(),
		"active":     true,
		"updated_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("failed to insert secondary user: %v", err)
	}

	payload := map[string]string{
		"tenant_id": tenantID.String(),
		"user_id":   userID.String(),
	}

	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/uflow/user/enduser/delete", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// Set token claims for auth middleware simulation
	setTokenClaimsInContext(c, tenantID.String(), userID.String())
	c.Set("user_info", &middlewares.UserInfo{TenantID: tenantID.String()})

	controller.DeleteEndUser(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "User deleted successfully", response["message"])

	var active bool
	if err := db.Table("users").Where("id = ? AND tenant_id = ?", userID.String(), tenantID.String()).Pluck("active", &active).Error; err != nil {
		t.Fatalf("failed to fetch user row: %v", err)
	}
	assert.False(t, active)
}

func TestActiveOrDeactiveEndUser_AcceptsStringBoolean(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &EndUserController{}

	tenantID := uuid.New()
	userID := uuid.New()

	db := setupTenantTestDB(t, tenantID, userID, true)

	overrideTenantConnection(t, db)
	overrideTimeNow(t)
	overrideConfigDB(t, db)

	if err := db.Table("users").Create(map[string]interface{}{
		"id":         uuid.New().String(),
		"tenant_id":  tenantID.String(),
		"active":     true,
		"updated_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("failed to insert secondary user: %v", err)
	}

	payload := map[string]interface{}{
		"tenant_id": tenantID.String(),
		"user_id":   userID.String(),
		"active":    "false",
	}

	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/uflow/user/enduser/active", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ActiveOrDeactiveEndUser(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "User updated successfully", response["message"])

	var active bool
	if err := db.Table("users").Where("id = ? AND tenant_id = ?", userID.String(), tenantID.String()).Pluck("active", &active).Error; err != nil {
		t.Fatalf("failed to fetch user row: %v", err)
	}
	assert.False(t, active)
}

func TestActiveOrDeactiveEndUser_BlocksLastActive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &EndUserController{}

	tenantID := uuid.New()
	userID := uuid.New()

	db := setupTenantTestDB(t, tenantID, userID, true)

	overrideTenantConnection(t, db)
	overrideTimeNow(t)
	overrideConfigDB(t, db)

	payload := map[string]interface{}{
		"tenant_id": tenantID.String(),
		"user_id":   userID.String(),
		"active":    false,
	}

	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/uflow/user/enduser/active", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ActiveOrDeactiveEndUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "cannot deactivate the last active user in this tenant", response["error"])
}

func TestDeleteEndUser_BlocksLastActive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &EndUserController{}

	tenantID := uuid.New()
	userID := uuid.New()

	db := setupTenantTestDB(t, tenantID, userID, true)

	overrideTenantConnection(t, db)
	overrideTimeNow(t)
	overrideConfigDB(t, db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(map[string]string{
		"tenant_id": tenantID.String(),
		"user_id":   userID.String(),
	})

	c.Request = httptest.NewRequest("DELETE", "/uflow/user/enduser/delete", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// Set token claims for auth middleware simulation
	setTokenClaimsInContext(c, tenantID.String(), userID.String())
	c.Set("user_info", &middlewares.UserInfo{TenantID: tenantID.String()})

	controller.DeleteEndUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "cannot deactivate the last active user in this tenant", response["error"])
}

func setupTenantTestDB(t *testing.T, tenantID, userID uuid.UUID, initialActive bool) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}

	if err := db.Exec(`DROP TABLE IF EXISTS users`).Error; err != nil {
		t.Fatalf("failed to drop users table: %v", err)
	}

	if err := db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		active BOOLEAN NOT NULL DEFAULT 1,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	if err := db.Table("users").Create(map[string]interface{}{
		"id":         userID.String(),
		"tenant_id":  tenantID.String(),
		"active":     initialActive,
		"updated_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	return db
}

func overrideTenantConnection(t *testing.T, db *gorm.DB) {
	t.Helper()
	original := tenantConnectionProvider
	t.Cleanup(func() {
		tenantConnectionProvider = original
	})
	tenantConnectionProvider = func(_ interface{}, _ *string, _ *string) (*gorm.DB, error) {
		return db, nil
	}
}

func overrideTimeNow(t *testing.T) {
	t.Helper()
	original := timeNow
	t.Cleanup(func() {
		timeNow = original
	})
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time {
		return fixed
	}
}

func overrideConfigDB(t *testing.T, replacement *gorm.DB) {
	t.Helper()
	original := config.DB
	config.DB = replacement
	t.Cleanup(func() {
		config.DB = original
	})
}
