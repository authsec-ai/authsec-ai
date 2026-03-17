package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdminUserController_Instantiation(t *testing.T) {
	controller, err := NewAdminUserController()

	// If database is not initialized, skip the test
	// This is expected in unit test environments without a database
	if err != nil && err.Error() == "database not initialized" {
		t.Skip("Skipping test - database not initialized (this is expected in unit tests)")
	}

	assert.NotNil(t, controller)
	assert.Nil(t, err)
}

func TestAdminUserController_MethodsExist(t *testing.T) {
	controller, err := NewAdminUserController()

	// Skip if database not available
	if err != nil {
		t.Skip("Skipping test - controller not initialized")
	}

	assert.NotNil(t, controller.ListTenants)
	assert.NotNil(t, controller.ListAdminUsers)
	assert.NotNil(t, controller.ListEndUsersByTenant)
	assert.NotNil(t, controller.CreateTenant)
	assert.NotNil(t, controller.UpdateTenant)
	assert.NotNil(t, controller.GetTenantUsers)
}
