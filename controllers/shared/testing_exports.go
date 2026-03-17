package shared

import (
	"net/http"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ConnectToADForTest exposes connectToAD for external tests.
func (asc *ADSyncController) ConnectToADForTest(config models.ADSyncConfig) (*ldap.Conn, error) {
	return asc.connectToAD(config)
}

// FetchADUsersForTest exposes fetchADUsers for external tests.
func (asc *ADSyncController) FetchADUsersForTest(config models.ADSyncConfig) ([]models.ADUser, error) {
	return asc.FetchADUsers(config)
}

// SyncUserToDatabaseForTest exposes syncUserToDatabase for external tests.
func (asc *ADSyncController) SyncUserToDatabaseForTest(tenantDB *gorm.DB, adUser models.ADUser, tenantID, clientID, projectID string) error {
	return asc.syncUserToDatabase(tenantDB, adUser, tenantID, clientID, projectID)
}

// SyncAgentUserToDatabaseForTest exposes syncAgentUserToDatabase for external tests.
func (asc *ADSyncController) SyncAgentUserToDatabaseForTest(tenantDB *gorm.DB, agentUser models.AgentUserData, tenantID, projectID, clientID string) error {
	return asc.syncAgentUserToDatabase(tenantDB, agentUser, tenantID, projectID, clientID)
}

// MapLDAPEntryToUserForTest exposes mapLDAPEntryToUser for external tests.
func (asc *ADSyncController) MapLDAPEntryToUserForTest(entry *ldap.Entry) models.ADUser {
	return asc.mapLDAPEntryToUser(entry)
}

// NewEntraIDServiceForTest exposes newEntraIDService for tests.
func (ec *EntraIDController) NewEntraIDServiceForTest(config *EntraIDConfig) *EntraIDService {
	return ec.NewEntraIDService(config)
}

// NewEntraIDServiceWithClientForTest constructs an EntraIDService with test dependencies.
func NewEntraIDServiceWithClientForTest(config *EntraIDConfig, client *http.Client, token string) *EntraIDService {
	return &EntraIDService{
		config:      config,
		client:      client,
		accessToken: token,
	}
}

// AuthenticateForTest exposes authenticate for tests.
func (es *EntraIDService) AuthenticateForTest() error {
	return es.authenticate()
}

// FetchEntraIDUsersForTest exposes fetchEntraIDUsers for tests.
func (es *EntraIDService) FetchEntraIDUsersForTest() ([]EntraIDUser, error) {
	return es.FetchEntraIDUsers()
}

// FetchUsersWithLimitForTest exposes fetchUsersWithLimit for tests.
func (es *EntraIDService) FetchUsersWithLimitForTest(limit int) ([]GraphUser, error) {
	return es.fetchUsersWithLimit(limit)
}

// CheckPermissionsForTest exposes checkPermissions for tests.
func (es *EntraIDService) CheckPermissionsForTest() (map[string]interface{}, error) {
	return es.checkPermissions()
}

// ConfigForTest returns the EntraIDService config for assertions.
func (es *EntraIDService) ConfigForTest() *EntraIDConfig {
	return es.config
}

// ClientForTest returns the HTTP client used by the EntraIDService.
func (es *EntraIDService) ClientForTest() *http.Client {
	return es.client
}

// AccessTokenForTest returns the stored access token.
func (es *EntraIDService) AccessTokenForTest() string {
	return es.accessToken
}

// TokenExpiryForTest returns the token expiry time.
func (es *EntraIDService) TokenExpiryForTest() time.Time {
	return es.tokenExpiry
}

//SetSeededTenantID allows tests to set the seeded tenant ID.
var SetSeededTenantID = func(id uuid.UUID) {
	seededTenantID = id
}

// GetSeededTenantID returns the seeded tenant ID.
func GetSeededTenantID() uuid.UUID {
	return seededTenantID
}
