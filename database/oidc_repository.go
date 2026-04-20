package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// ========================================
// OIDCProviderRepository
// ========================================

// OIDCProviderRepository handles OIDC provider database operations
type OIDCProviderRepository struct {
	db *DBConnection
}

// NewOIDCProviderRepository creates a new OIDC provider repository
func NewOIDCProviderRepository(db *DBConnection) *OIDCProviderRepository {
	return &OIDCProviderRepository{db: db}
}

// GetProviderByName retrieves an OIDC provider by name
func (r *OIDCProviderRepository) GetProviderByName(providerName string) (*models.OIDCProvider, error) {
	query := `
		SELECT id, provider_name, display_name, client_id, client_secret_vault_path,
		       authorization_url, token_url, userinfo_url, scopes, icon_url, is_active,
		       created_at, updated_at
		FROM oidc_providers
		WHERE provider_name = $1
	`

	provider := &models.OIDCProvider{}
	var iconURL sql.NullString

	err := r.db.QueryRow(query, providerName).Scan(
		&provider.ID,
		&provider.ProviderName,
		&provider.DisplayName,
		&provider.ClientID,
		&provider.ClientSecretVaultPath,
		&provider.AuthorizationURL,
		&provider.TokenURL,
		&provider.UserinfoURL,
		&provider.Scopes,
		&iconURL,
		&provider.IsActive,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("OIDC provider not found: %s", providerName)
		}
		return nil, err
	}

	if iconURL.Valid {
		provider.IconURL = iconURL.String
	}

	return provider, nil
}

// GetActiveProviders retrieves all active OIDC providers
func (r *OIDCProviderRepository) GetActiveProviders() ([]models.OIDCProvider, error) {
	query := `
		SELECT id, provider_name, display_name, client_id, client_secret_vault_path,
		       authorization_url, token_url, userinfo_url, scopes, icon_url, is_active,
		       created_at, updated_at
		FROM oidc_providers
		WHERE is_active = true
		ORDER BY display_name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []models.OIDCProvider
	for rows.Next() {
		var provider models.OIDCProvider
		var iconURL sql.NullString

		err := rows.Scan(
			&provider.ID,
			&provider.ProviderName,
			&provider.DisplayName,
			&provider.ClientID,
			&provider.ClientSecretVaultPath,
			&provider.AuthorizationURL,
			&provider.TokenURL,
			&provider.UserinfoURL,
			&provider.Scopes,
			&iconURL,
			&provider.IsActive,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if iconURL.Valid {
			provider.IconURL = iconURL.String
		}

		providers = append(providers, provider)
	}

	return providers, rows.Err()
}

// GetAllProviders retrieves all OIDC providers (for admin)
func (r *OIDCProviderRepository) GetAllProviders() ([]models.OIDCProvider, error) {
	query := `
		SELECT id, provider_name, display_name, client_id, client_secret_vault_path,
		       authorization_url, token_url, userinfo_url, scopes, icon_url, is_active,
		       created_at, updated_at
		FROM oidc_providers
		ORDER BY display_name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []models.OIDCProvider
	for rows.Next() {
		var provider models.OIDCProvider
		var iconURL sql.NullString

		err := rows.Scan(
			&provider.ID,
			&provider.ProviderName,
			&provider.DisplayName,
			&provider.ClientID,
			&provider.ClientSecretVaultPath,
			&provider.AuthorizationURL,
			&provider.TokenURL,
			&provider.UserinfoURL,
			&provider.Scopes,
			&iconURL,
			&provider.IsActive,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if iconURL.Valid {
			provider.IconURL = iconURL.String
		}

		providers = append(providers, provider)
	}

	return providers, rows.Err()
}

// UpdateProvider updates an OIDC provider configuration
func (r *OIDCProviderRepository) UpdateProvider(providerName string, input *models.OIDCProviderUpdateInput) error {
	query := `
		UPDATE oidc_providers
		SET client_id = COALESCE(NULLIF($1, ''), client_id),
		    client_secret_vault_path = COALESCE(NULLIF($2, ''), client_secret_vault_path),
		    is_active = COALESCE($3, is_active),
		    icon_url = COALESCE(NULLIF($4, ''), icon_url),
		    updated_at = $5
		WHERE provider_name = $6
	`

	result, err := r.db.Exec(query,
		input.ClientID,
		input.ClientSecretVaultPath,
		input.IsActive,
		input.IconURL,
		time.Now(),
		providerName,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OIDC provider not found: %s", providerName)
	}

	return nil
}

// ========================================
// OIDCStateRepository
// ========================================

// OIDCStateRepository handles OIDC state database operations
type OIDCStateRepository struct {
	db *DBConnection
}

// NewOIDCStateRepository creates a new OIDC state repository
func NewOIDCStateRepository(db *DBConnection) *OIDCStateRepository {
	return &OIDCStateRepository{db: db}
}

// CreateState creates a new OIDC state entry
func (r *OIDCStateRepository) CreateState(state *models.OIDCState) error {
	query := `
		INSERT INTO oidc_states (state_token, tenant_id, tenant_domain, request_host, provider_name,
		                         action, code_verifier, redirect_after, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	now := time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}

	log.Printf("DEBUG CreateState: Inserting state with OriginDomain='%s' (will be stored in request_host)", state.OriginDomain)
	log.Printf("DEBUG CreateState: All params - StateToken=%s, TenantID=%v, TenantDomain=%s, OriginDomain=%s, ProviderName=%s, Action=%s",
		state.StateToken, state.TenantID, state.TenantDomain, state.OriginDomain, state.ProviderName, state.Action)

	// Use sql.NullString for proper NULL handling
	requestHostParam := sql.NullString{
		String: state.OriginDomain,
		Valid:  state.OriginDomain != "",
	}
	log.Printf("DEBUG CreateState: requestHostParam={String: '%s', Valid: %v}", requestHostParam.String, requestHostParam.Valid)

	err := r.db.QueryRow(query,
		state.StateToken,
		state.TenantID,
		state.TenantDomain,
		requestHostParam, // Maps to request_host column
		state.ProviderName,
		state.Action,
		state.CodeVerifier,
		state.RedirectAfter,
		state.ExpiresAt,
		state.CreatedAt,
	).Scan(&state.ID)

	if err != nil {
		log.Printf("ERROR CreateState: Failed to insert state: %v", err)
	} else {
		log.Printf("DEBUG CreateState: Successfully inserted state with ID=%s", state.ID)
	}

	return err
}

// GetStateByToken retrieves a valid (non-expired) state by token
func (r *OIDCStateRepository) GetStateByToken(stateToken string) (*models.OIDCState, error) {
	query := `
		SELECT id, state_token, tenant_id, tenant_domain, request_host, provider_name,
		       action, code_verifier, redirect_after, expires_at, created_at
		FROM oidc_states
		WHERE state_token = $1 AND expires_at > $2
	`

	state := &models.OIDCState{}
	var tenantID sql.NullString
	var requestHost, codeVerifier, redirectAfter sql.NullString

	err := r.db.QueryRow(query, stateToken, time.Now()).Scan(
		&state.ID,
		&state.StateToken,
		&tenantID,
		&state.TenantDomain,
		&requestHost,
		&state.ProviderName,
		&state.Action,
		&codeVerifier,
		&redirectAfter,
		&state.ExpiresAt,
		&state.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("OIDC state not found or expired")
		}
		return nil, err
	}

	if tenantID.Valid {
		id, _ := uuid.Parse(tenantID.String)
		state.TenantID = &id
	}
	if requestHost.Valid {
		state.OriginDomain = requestHost.String
		log.Printf("DEBUG GetStateByToken: Read request_host='%s' from DB, set OriginDomain='%s'", requestHost.String, state.OriginDomain)
	} else {
		log.Printf("DEBUG GetStateByToken: request_host is NULL in DB")
	}
	if codeVerifier.Valid {
		state.CodeVerifier = codeVerifier.String
	}
	if redirectAfter.Valid {
		state.RedirectAfter = redirectAfter.String
	}

	return state, nil
}

// DeleteState deletes a state entry (after use or cleanup)
func (r *OIDCStateRepository) DeleteState(stateToken string) error {
	query := `DELETE FROM oidc_states WHERE state_token = $1`
	_, err := r.db.Exec(query, stateToken)
	return err
}

// DeleteExpiredStates deletes all expired state entries (cleanup job)
func (r *OIDCStateRepository) DeleteExpiredStates() error {
	query := `DELETE FROM oidc_states WHERE expires_at < $1`
	_, err := r.db.Exec(query, time.Now())
	return err
}

// ========================================
// OIDCUserIdentityRepository
// ========================================

// OIDCUserIdentityRepository handles OIDC user identity database operations
type OIDCUserIdentityRepository struct {
	db *DBConnection
}

// NewOIDCUserIdentityRepository creates a new OIDC user identity repository
func NewOIDCUserIdentityRepository(db *DBConnection) *OIDCUserIdentityRepository {
	return &OIDCUserIdentityRepository{db: db}
}

// CreateIdentity creates a new OIDC user identity link or updates it if it already exists.
func (r *OIDCUserIdentityRepository) CreateIdentity(identity *models.OIDCUserIdentity) error {
	query := `
		INSERT INTO oidc_user_identities (tenant_id, user_id, provider_name, provider_user_id,
		                                  email, profile_data, created_at, updated_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (provider_name, provider_user_id) DO UPDATE
		SET email = EXCLUDED.email,
		    profile_data = EXCLUDED.profile_data,
		    updated_at = EXCLUDED.updated_at,
			last_login_at = EXCLUDED.updated_at
		RETURNING id
	`

	now := time.Now()
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = now
	}
	identity.UpdatedAt = now

	err := r.db.QueryRow(query,
		identity.TenantID,
		identity.UserID,
		identity.ProviderName,
		identity.ProviderUserID,
		identity.Email,
		identity.ProfileData,
		identity.CreatedAt,
		identity.UpdatedAt,
		identity.CreatedAt, // $9 = last_login_at (same as created_at on first insert)
	).Scan(&identity.ID)

	return err
}

// GetIdentityByProviderUser retrieves identity by provider and provider user ID
// This answers: "Does this Google user exist anywhere?"
func (r *OIDCUserIdentityRepository) GetIdentityByProviderUser(providerName, providerUserID string) (*models.OIDCUserIdentity, error) {
	query := `
		SELECT id, tenant_id, user_id, provider_name, provider_user_id,
		       email, profile_data, last_login_at, created_at, updated_at
		FROM oidc_user_identities
		WHERE provider_name = $1 AND provider_user_id = $2
	`

	identity := &models.OIDCUserIdentity{}
	var profileData sql.NullString
	var email sql.NullString
	var lastLoginAt sql.NullTime

	err := r.db.QueryRow(query, providerName, providerUserID).Scan(
		&identity.ID,
		&identity.TenantID,
		&identity.UserID,
		&identity.ProviderName,
		&identity.ProviderUserID,
		&email,
		&profileData,
		&lastLoginAt,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is valid - user doesn't have OIDC linked
		}
		return nil, err
	}

	if email.Valid {
		identity.Email = email.String
	}
	if profileData.Valid {
		identity.ProfileData = profileData.String
	}
	if lastLoginAt.Valid {
		identity.LastLoginAt = &lastLoginAt.Time
	}

	return identity, nil
}

// GetIdentityByTenantAndProviderUser retrieves identity for a specific tenant
// This answers: "Does this Google user exist in THIS tenant?"
func (r *OIDCUserIdentityRepository) GetIdentityByTenantAndProviderUser(tenantID uuid.UUID, providerName, providerUserID string) (*models.OIDCUserIdentity, error) {
	query := `
		SELECT id, tenant_id, user_id, provider_name, provider_user_id,
		       email, profile_data, last_login_at, created_at, updated_at
		FROM oidc_user_identities
		WHERE tenant_id = $1 AND provider_name = $2 AND provider_user_id = $3
	`

	identity := &models.OIDCUserIdentity{}
	var profileData sql.NullString
	var email sql.NullString
	var lastLoginAt sql.NullTime

	err := r.db.QueryRow(query, tenantID, providerName, providerUserID).Scan(
		&identity.ID,
		&identity.TenantID,
		&identity.UserID,
		&identity.ProviderName,
		&identity.ProviderUserID,
		&email,
		&profileData,
		&lastLoginAt,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found - user not in this tenant with this provider
		}
		return nil, err
	}

	if email.Valid {
		identity.Email = email.String
	}
	if profileData.Valid {
		identity.ProfileData = profileData.String
	}
	if lastLoginAt.Valid {
		identity.LastLoginAt = &lastLoginAt.Time
	}

	return identity, nil
}

// GetIdentitiesByUserID retrieves all OIDC identities for a user
func (r *OIDCUserIdentityRepository) GetIdentitiesByUserID(tenantID, userID uuid.UUID) ([]models.OIDCUserIdentity, error) {
	query := `
		SELECT id, tenant_id, user_id, provider_name, provider_user_id,
		       email, profile_data, last_login_at, created_at, updated_at
		FROM oidc_user_identities
		WHERE tenant_id = $1 AND user_id = $2
	`

	rows, err := r.db.Query(query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []models.OIDCUserIdentity
	for rows.Next() {
		var identity models.OIDCUserIdentity
		var profileData sql.NullString
		var email sql.NullString
		var lastLoginAt sql.NullTime

		err := rows.Scan(
			&identity.ID,
			&identity.TenantID,
			&identity.UserID,
			&identity.ProviderName,
			&identity.ProviderUserID,
			&email,
			&profileData,
			&lastLoginAt,
			&identity.CreatedAt,
			&identity.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if email.Valid {
			identity.Email = email.String
		}
		if profileData.Valid {
			identity.ProfileData = profileData.String
		}
		if lastLoginAt.Valid {
			identity.LastLoginAt = &lastLoginAt.Time
		}

		identities = append(identities, identity)
	}

	return identities, rows.Err()
}

// UpdateLastLogin updates the last login timestamp for an identity
func (r *OIDCUserIdentityRepository) UpdateLastLogin(identityID uuid.UUID) error {
	query := `
		UPDATE oidc_user_identities
		SET last_login_at = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	_, err := r.db.Exec(query, now, now, identityID)
	return err
}

// UpdateProfileData updates the profile data for an identity
func (r *OIDCUserIdentityRepository) UpdateProfileData(identityID uuid.UUID, profileData map[string]interface{}) error {
	jsonData, err := json.Marshal(profileData)
	if err != nil {
		return err
	}

	query := `
		UPDATE oidc_user_identities
		SET profile_data = $1, updated_at = $2
		WHERE id = $3
	`

	_, err = r.db.Exec(query, string(jsonData), time.Now(), identityID)
	return err
}

// DeleteIdentity deletes an OIDC identity link (unlink provider)
func (r *OIDCUserIdentityRepository) DeleteIdentity(tenantID, userID uuid.UUID, providerName string) error {
	query := `
		DELETE FROM oidc_user_identities
		WHERE tenant_id = $1 AND user_id = $2 AND provider_name = $3
	`

	result, err := r.db.Exec(query, tenantID, userID, providerName)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OIDC identity not found")
	}

	return nil
}

// GetTenantsByProviderEmail retrieves all tenants where this email has OIDC identity
// Useful for "find my workspace" feature
func (r *OIDCUserIdentityRepository) GetTenantsByProviderEmail(email string) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT tenant_id
		FROM oidc_user_identities
		WHERE email = $1
	`

	rows, err := r.db.Query(query, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenantIDs []uuid.UUID
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, err
		}
		tenantIDs = append(tenantIDs, tenantID)
	}

	return tenantIDs, rows.Err()
}
