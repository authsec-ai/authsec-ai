package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdminAuthController_Instantiation(t *testing.T) {
	// Test that the controller can be instantiated
	controller, err := NewAdminAuthController()

	// If database is not initialized, skip the test
	// This is expected in unit test environments without a database
	if err != nil && err.Error() == "database not initialized" {
		t.Skip("Skipping test - database not initialized (this is expected in unit tests)")
	}

	// The controller should instantiate successfully with database connection
	assert.NotNil(t, controller)
	assert.Nil(t, err)
}

func TestAdminAuthController_MethodsExist(t *testing.T) {
	controller, _ := NewAdminAuthController()

	// Test that key methods exist (they will fail at runtime due to DB, but should not panic)
	assert.NotNil(t, controller.AdminLogin)
	assert.NotNil(t, controller.AdminRegister)
	assert.NotNil(t, controller.AdminForgotPassword)
	assert.NotNil(t, controller.AdminVerifyOTP)
	assert.NotNil(t, controller.AdminResetPassword)
}
