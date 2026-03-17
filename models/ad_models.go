package models

// ADSyncController struct
type ADSyncController struct{}

// ADSyncConfig holds configuration for AD connection
type ADSyncConfig struct {
	Server     string `json:"server"`      // AD server address (e.g., "dc.company.com:636")
	Username   string `json:"username"`    // Service account username
	Password   string `json:"password"`    // Service account password
	BaseDN     string `json:"base_dn"`     // Base DN for user search (e.g., "OU=Users,DC=company,DC=com")
	Filter     string `json:"filter"`      // LDAP filter (e.g., "(&(objectClass=user)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))")
	UseSSL     bool   `json:"use_ssl"`     // Whether to use SSL/TLS
	SkipVerify bool   `json:"skip_verify"` // Skip SSL certificate verification (for testing)
}

// SyncUsersInput represents the input for syncing users from AD
type SyncUsersInput struct {
	TenantID  string        `json:"tenant_id" binding:"required"`
	ClientID  string        `json:"client_id" binding:"required"`
	ProjectID string        `json:"project_id" binding:"required"`
	ConfigID  *string       `json:"config_id,omitempty"` // ID of stored config to use
	Config    *ADSyncConfig `json:"config,omitempty"`     // Or provide config directly (for backward compatibility)
	DryRun    bool          `json:"dry_run,omitempty"`    // Preview changes without applying
}

// SyncResult represents the result of a sync operation using shared ErrorResponse
type SyncResult struct {
	UsersFound   int      `json:"users_found"`
	UsersCreated int      `json:"users_created"`
	UsersUpdated int      `json:"users_updated"`
	Errors       []string `json:"errors,omitempty"`
	PreviewUsers []ADUser `json:"preview_users,omitempty"` // Only populated for dry runs
	Message      string   `json:"message,omitempty"`
}

// ADUser represents a user from Active Directory
type ADUser struct {
	ObjectGUID        string            `json:"object_guid"`
	UserPrincipalName string            `json:"user_principal_name"`
	DisplayName       string            `json:"display_name"`
	Email             string            `json:"email"`
	Username          string            `json:"username"`
	Department        string            `json:"department"`
	Title             string            `json:"title"`
	Groups            []string          `json:"groups"`
	Attributes        map[string]string `json:"attributes"`
	IsActive          bool              `json:"is_active"`
}

// AgentSyncRequest represents the request from AD Agent
type AgentSyncRequest struct {
	TenantID  string          `json:"tenant_id" binding:"required"`
	ProjectID string          `json:"project_id" binding:"required"`
	ClientID  string          `json:"client_id" binding:"required"`
	Users     []AgentUserData `json:"users" binding:"required"`
	DryRun    bool            `json:"dry_run,omitempty"`
}

// AgentUserData represents user data from AD Agent
type AgentUserData struct {
	ExternalID   string                 `json:"external_id"`
	Email        string                 `json:"email"`
	Name         string                 `json:"name"`
	Username     string                 `json:"username"`
	Provider     string                 `json:"provider"`
	ProviderID   string                 `json:"provider_id"`
	ProviderData map[string]interface{} `json:"provider_data"`
	IsActive     bool                   `json:"is_active"`
	IsSyncedUser bool                   `json:"is_synced_user"`
	SyncSource   string                 `json:"sync_source"`
}

// AgentSyncResponse represents the response for agent sync using shared response patterns
type AgentSyncResponse struct {
	Message        string          `json:"message"`
	UsersProcessed int             `json:"users_processed"`
	UsersCreated   int             `json:"users_created"`
	Errors         []ErrorResponse `json:"errors,omitempty"`
}
