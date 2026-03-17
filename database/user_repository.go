package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// UserRepository handles user database operations without GORM
type UserRepository struct {
	db *DBConnection
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *DBConnection) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user record
func (ur *UserRepository) CreateUser(user *models.ExtendedUser) error {
	// Validate user data before creation
	if err := ur.validateUserForCreation(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	query := `
		INSERT INTO users (id, client_id, tenant_id, project_id, name, username, email,
			password_hash, tenant_domain, provider, provider_id, provider_data,
			avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
			mfa_enrolled_at, mfa_verified, last_login,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`

	now := time.Now()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	// Convert datatypes.JSON to interface{} for SQL operations
	var mfaMethodArray interface{}
	if user.MFAMethod != nil {
		mfaMethodArray = user.MFAMethod
	}

	_, err := ur.db.Exec(query,
		user.ID,
		user.ClientID,
		user.TenantID,
		user.ProjectID,
		user.Name,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.TenantDomain,
		user.Provider,
		user.ProviderID,
		user.ProviderData,
		user.AvatarURL,
		user.Active,
		user.MFAEnabled,
		mfaMethodArray,
		user.MFADefaultMethod,
		user.MFAEnrolledAt,
		user.MFAVerified,
		user.LastLogin,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

// GetUserByEmail retrieves a user by email (case-insensitive)
func (ur *UserRepository) GetUserByEmail(email string) (*models.ExtendedUser, error) {
	query := `
		SELECT id, client_id, tenant_id, project_id, name, username, email,
			password_hash, tenant_domain, provider, provider_id, provider_data,
			avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
			mfa_enrolled_at, mfa_verified, last_login,
			created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`

	user := &models.ExtendedUser{}
	var username, providerID, avatarURL, mfaDefaultMethod sql.NullString
	var mfaEnrolledAt, lastLoginAt sql.NullTime
	var mfaMethod pq.StringArray

	err := ur.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.ClientID,
		&user.TenantID,
		&user.ProjectID,
		&user.Name,
		&username,
		&user.Email,
		&user.PasswordHash,
		&user.TenantDomain,
		&user.Provider,
		&providerID,
		&user.ProviderData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethod,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if username.Valid {
		user.Username = &username.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = &mfaDefaultMethod.String
	}
	if mfaEnrolledAt.Valid {
		user.MFAEnrolledAt = &mfaEnrolledAt.Time
	}
	if lastLoginAt.Valid {
		user.LastLogin = &lastLoginAt.Time
	}

	// Assign MFA method directly from TEXT[] array
	user.MFAMethod = mfaMethod

	return user, nil
}

// GetUserByEmailAndClient retrieves a user by email scoped to a client
func (ur *UserRepository) GetUserByEmailAndClient(email string, clientID uuid.UUID) (*models.ExtendedUser, error) {
	query := `
		SELECT id, client_id, tenant_id, project_id, name, username, email,
			password_hash, tenant_domain, provider, provider_id, provider_data,
			avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
			mfa_enrolled_at, mfa_verified, last_login,
			created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1) AND client_id = $2
	`

	user := &models.ExtendedUser{}
	var username, providerID, avatarURL, mfaDefaultMethod sql.NullString
	var mfaEnrolledAt, lastLoginAt sql.NullTime
	var mfaMethod pq.StringArray

	err := ur.db.QueryRow(query, email, clientID).Scan(
		&user.ID,
		&user.ClientID,
		&user.TenantID,
		&user.ProjectID,
		&user.Name,
		&username,
		&user.Email,
		&user.PasswordHash,
		&user.TenantDomain,
		&user.Provider,
		&providerID,
		&user.ProviderData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethod,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	if username.Valid {
		user.Username = &username.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = &mfaDefaultMethod.String
	}
	if mfaEnrolledAt.Valid {
		user.MFAEnrolledAt = &mfaEnrolledAt.Time
	}
	if lastLoginAt.Valid {
		user.LastLogin = &lastLoginAt.Time
	}

	user.MFAMethod = mfaMethod

	return user, nil
}

// GetUserByEmailAndTenant retrieves a user by email scoped to a tenant
func (ur *UserRepository) GetUserByEmailAndTenant(email string, tenantID uuid.UUID) (*models.ExtendedUser, error) {
	query := `
		SELECT id, client_id, tenant_id, project_id, name, username, email,
			password_hash, tenant_domain, provider, provider_id, provider_data,
			avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
			mfa_enrolled_at, mfa_verified, last_login,
			created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1) AND tenant_id = $2
	`

	user := &models.ExtendedUser{}
	var username, providerID, avatarURL, mfaDefaultMethod sql.NullString
	var mfaEnrolledAt, lastLoginAt sql.NullTime
	var mfaMethod pq.StringArray

	err := ur.db.QueryRow(query, email, tenantID).Scan(
		&user.ID,
		&user.ClientID,
		&user.TenantID,
		&user.ProjectID,
		&user.Name,
		&username,
		&user.Email,
		&user.PasswordHash,
		&user.TenantDomain,
		&user.Provider,
		&providerID,
		&user.ProviderData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethod,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	if username.Valid {
		user.Username = &username.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = &mfaDefaultMethod.String
	}
	if mfaEnrolledAt.Valid {
		user.MFAEnrolledAt = &mfaEnrolledAt.Time
	}
	if lastLoginAt.Valid {
		user.LastLogin = &lastLoginAt.Time
	}

	user.MFAMethod = mfaMethod

	return user, nil
}

// GetUserByID retrieves a user by ID
func (ur *UserRepository) GetUserByID(userID uuid.UUID) (*models.ExtendedUser, error) {
	query := `
		SELECT id, client_id, tenant_id, project_id, name, username, email,
			password_hash, tenant_domain, provider, provider_id, provider_data,
			avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
			mfa_enrolled_at, mfa_verified, last_login,
			created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.ExtendedUser{}
	var username, providerID, avatarURL, mfaDefaultMethod sql.NullString
	var mfaEnrolledAt, lastLoginAt sql.NullTime
	var mfaMethod pq.StringArray

	err := ur.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.ClientID,
		&user.TenantID,
		&user.ProjectID,
		&user.Name,
		&username,
		&user.Email,
		&user.PasswordHash,
		&user.TenantDomain,
		&user.Provider,
		&providerID,
		&user.ProviderData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethod,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if username.Valid {
		user.Username = &username.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = &mfaDefaultMethod.String
	}
	if mfaEnrolledAt.Valid {
		user.MFAEnrolledAt = &mfaEnrolledAt.Time
	}
	if lastLoginAt.Valid {
		user.LastLogin = &lastLoginAt.Time
	}

	// Assign MFA method directly from TEXT[] array
	user.MFAMethod = mfaMethod

	return user, nil
}

// GetUserByProvider retrieves a user by OAuth provider and provider ID
func (ur *UserRepository) GetUserByProvider(provider, providerID string) (*models.ExtendedUser, error) {
	query := `
		SELECT id, client_id, tenant_id, project_id, name, username, email,
			password_hash, tenant_domain, provider, provider_id, provider_data,
			avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
			mfa_enrolled_at, mfa_verified, last_login,
			created_at, updated_at
		FROM users
		WHERE provider = $1 AND provider_id = $2
	`

	user := &models.ExtendedUser{}
	var username, providerIDNull, avatarURL, mfaDefaultMethod sql.NullString
	var mfaEnrolledAt, lastLoginAt sql.NullTime
	var mfaMethod pq.StringArray

	err := ur.db.QueryRow(query, provider, providerID).Scan(
		&user.ID,
		&user.ClientID,
		&user.TenantID,
		&user.ProjectID,
		&user.Name,
		&username,
		&user.Email,
		&user.PasswordHash,
		&user.TenantDomain,
		&user.Provider,
		&providerIDNull,
		&user.ProviderData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethod,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if username.Valid {
		user.Username = &username.String
	}
	if providerIDNull.Valid {
		user.ProviderID = providerIDNull.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = &mfaDefaultMethod.String
	}
	if mfaEnrolledAt.Valid {
		user.MFAEnrolledAt = &mfaEnrolledAt.Time
	}
	if lastLoginAt.Valid {
		user.LastLogin = &lastLoginAt.Time
	}

	// Assign MFA method directly from TEXT[] array
	user.MFAMethod = mfaMethod

	return user, nil
}

// UpdateUserLogin updates login-related fields for a user
func (ur *UserRepository) UpdateUserLogin(userID uuid.UUID) error {
	query := `
		UPDATE users
		SET last_login = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	result, err := ur.db.Exec(query, now, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateUserMFA updates MFA-related fields for a user
func (ur *UserRepository) UpdateUserMFA(userID uuid.UUID, mfaEnabled bool, mfaMethods []byte) error {
	query := `
		UPDATE users
		SET mfa_enabled = $1, mfa_method = $2, updated_at = $3
		WHERE id = $4
	`

	now := time.Now()
	result, err := ur.db.Exec(query, mfaEnabled, mfaMethods, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateUserPassword updates a user's password hash
func (ur *UserRepository) UpdateUserPassword(userID uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := ur.db.Exec(query, passwordHash, time.Now(), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetUsersByTenantID retrieves users by tenant ID with pagination
func (ur *UserRepository) GetUsersByTenantID(tenantID uuid.UUID, limit, offset int) ([]*models.ExtendedUser, error) {
	query := `
		SELECT id, client_id, tenant_id, project_id, name, username, email,
			password_hash, tenant_domain, provider, provider_id, provider_data,
			avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
			mfa_enrolled_at, mfa_verified, last_login,
			created_at, updated_at
		FROM users
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := ur.db.Query(query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.ExtendedUser

	for rows.Next() {
		user := &models.ExtendedUser{}
		var username, providerID, avatarURL, mfaDefaultMethod sql.NullString
		var mfaEnrolledAt, lastLoginAt sql.NullTime
		var mfaMethod pq.StringArray

		err := rows.Scan(
			&user.ID,
			&user.ClientID,
			&user.TenantID,
			&user.ProjectID,
			&user.Name,
			&username,
			&user.Email,
			&user.PasswordHash,
			&user.TenantDomain,
			&user.Provider,
			&providerID,
			&user.ProviderData,
			&avatarURL,
			&user.Active,
			&user.MFAEnabled,
			&mfaMethod,
			&mfaDefaultMethod,
			&mfaEnrolledAt,
			&user.MFAVerified,
			&lastLoginAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Handle nullable fields
		if username.Valid {
			user.Username = &username.String
		}
		if providerID.Valid {
			user.ProviderID = providerID.String
		}
		if avatarURL.Valid {
			user.AvatarURL = &avatarURL.String
		}
		if mfaDefaultMethod.Valid {
			user.MFADefaultMethod = &mfaDefaultMethod.String
		}
		if mfaEnrolledAt.Valid {
			user.MFAEnrolledAt = &mfaEnrolledAt.Time
		}
		if lastLoginAt.Valid {
			user.LastLogin = &lastLoginAt.Time
		}

		// Assign MFA method directly from TEXT[] array
		user.MFAMethod = mfaMethod

		users = append(users, user)
	}

	return users, rows.Err()
}

// CountUsersByTenantID counts users for a tenant
func (ur *UserRepository) CountUsersByTenantID(tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE tenant_id = $1`

	var count int
	err := ur.db.QueryRow(query, tenantID).Scan(&count)
	return count, err
}

// UserExists checks if a user exists by email (case-insensitive)
func (ur *UserRepository) UserExists(email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1))`

	var exists bool
	err := ur.db.QueryRow(query, email).Scan(&exists)
	return exists, err
}

// DeleteUser soft deletes a user (marks as inactive) or hard deletes
func (ur *UserRepository) DeleteUser(userID uuid.UUID) error {
	// For now, we'll do a soft delete by marking as inactive
	query := `UPDATE users SET active = false, updated_at = $1 WHERE id = $2`

	result, err := ur.db.Exec(query, time.Now(), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Transaction support

// CreateUserTx creates a user within a transaction
func (ur *UserRepository) CreateUserTx(tx *sql.Tx, user *models.ExtendedUser) error {
	// Validate user data before creation
	if err := ur.validateUserForCreation(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	query := `
		INSERT INTO users (id, client_id, tenant_id, project_id, name, username, email,
			password_hash, tenant_domain, provider, provider_id, provider_data,
			avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
			mfa_enrolled_at, mfa_verified, last_login,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`

	now := time.Now()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	// Convert datatypes.JSON to interface{} for SQL operations
	var mfaMethodArray interface{}
	if user.MFAMethod != nil {
		mfaMethodArray = user.MFAMethod
	}

	_, err := tx.Exec(query,
		user.ID,
		user.ClientID,
		user.TenantID,
		user.ProjectID,
		user.Name,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.TenantDomain,
		user.Provider,
		user.ProviderID,
		user.ProviderData,
		user.AvatarURL,
		user.Active,
		user.MFAEnabled,
		mfaMethodArray,
		user.MFADefaultMethod,
		user.MFAEnrolledAt,
		user.MFAVerified,
		user.LastLogin,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

// initializeUserFields sets default values for optional fields
func (ur *UserRepository) initializeUserFields(user *models.ExtendedUser) {
	now := time.Now()

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	// Initialize ProviderData if nil
	if user.ProviderData == nil {
		user.ProviderData = datatypes.JSON("{}")
	}

	// Ensure required boolean fields have proper defaults
	// MFAEnabled and MFAVerified already have defaults in sharedmodels.User
}

// CreateUserWithValidation creates a user with comprehensive validation and initialization
func (ur *UserRepository) CreateUserWithValidation(user *models.ExtendedUser) error {
	// Initialize default fields
	ur.initializeUserFields(user)

	// Validate user data
	if err := ur.validateUserForCreation(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	// Validate relationships
	if err := ur.validateUserRelationships(user); err != nil {
		return fmt.Errorf("relationship validation failed: %w", err)
	}

	// Create the user
	return ur.CreateUser(user)
}

// validateUserForCreation performs comprehensive validation before user creation
func (ur *UserRepository) validateUserForCreation(user *models.ExtendedUser) error {
	// 1. Required field validation
	if user.ClientID == uuid.Nil {
		return fmt.Errorf("client_id is required")
	}
	if user.TenantID == uuid.Nil {
		return fmt.Errorf("tenant_id is required")
	}
	// Note: project_id is optional and can be nil for admin users
	if strings.TrimSpace(user.Email) == "" {
		return fmt.Errorf("email is required")
	}
	if strings.TrimSpace(user.TenantDomain) == "" {
		return fmt.Errorf("tenant_domain is required")
	}
	if strings.TrimSpace(user.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(user.ProviderID) == "" {
		return fmt.Errorf("provider_id is required")
	}

	// 2. Email format validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(user.Email) {
		return fmt.Errorf("invalid email format: %s", user.Email)
	}

	// 3. ProviderData JSON validation (if provided)
	if user.ProviderData != nil && len(user.ProviderData) > 0 {
		var jsonTest interface{}
		if err := json.Unmarshal(user.ProviderData, &jsonTest); err != nil {
			return fmt.Errorf("invalid JSON in provider_data: %w", err)
		}
	}

	// 4. Check for existing user with same email scoped to client
	if existingUser, err := ur.GetUserByEmailAndClient(user.Email, user.ClientID); err == nil && existingUser != nil {
		return fmt.Errorf("user with email %s already exists for this client", user.Email)
	} else if err != nil && err.Error() != "user not found" {
		return fmt.Errorf("failed to check existing user email: %w", err)
	}

	// 5. Check for existing user with same provider/provider_id combination
	if err := ur.checkProviderUniqueness(user.Provider, user.ProviderID, user.ID); err != nil {
		return err
	}

	// 6. Validate user relationships (scopes, roles, groups, resources)
	if err := ur.validateUserRelationships(user); err != nil {
		return fmt.Errorf("user relationship validation failed: %w", err)
	}

	return nil
}

// checkProviderUniqueness validates that provider/provider_id combination is unique
func (ur *UserRepository) checkProviderUniqueness(provider, providerID string, excludeUserID uuid.UUID) error {
	query := `SELECT id FROM users WHERE provider = $1 AND provider_id = $2`
	args := []interface{}{provider, providerID}

	if excludeUserID != uuid.Nil {
		query += ` AND id != $3`
		args = append(args, excludeUserID)
	}

	var existingID uuid.UUID
	err := ur.db.QueryRow(query, args...).Scan(&existingID)
	if err == nil {
		return fmt.Errorf("user with provider '%s' and provider_id '%s' already exists", provider, providerID)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("error checking provider uniqueness: %w", err)
	}

	return nil
}

// GetUserMFAMethods retrieves enabled MFA methods for a user
func (ur *UserRepository) GetUserMFAMethods(userID uuid.UUID, clientID uuid.UUID) ([]map[string]interface{}, error) {
	query := `
		SELECT method_type, display_name, description, method_data, is_primary, verified
		FROM mfa_methods
		WHERE user_id = $1 AND client_id = $2 AND enabled = true
		ORDER BY is_primary DESC, enrolled_at ASC
	`

	rows, err := ur.db.Query(query, userID, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query MFA methods: %w", err)
	}
	defer rows.Close()

	var methods []map[string]interface{}
	for rows.Next() {
		var methodType, displayName, description string
		var methodData []byte
		var isPrimary, verified bool

		err := rows.Scan(&methodType, &displayName, &description, &methodData, &isPrimary, &verified)
		if err != nil {
			return nil, fmt.Errorf("failed to scan MFA method: %w", err)
		}

		method := map[string]interface{}{
			"method_type":  methodType,
			"display_name": displayName,
			"description":  description,
			"is_primary":   isPrimary,
			"verified":     verified,
		}

		// Parse method_data JSON if present
		if len(methodData) > 0 {
			var data map[string]interface{}
			if err := json.Unmarshal(methodData, &data); err == nil {
				method["method_data"] = data
			}
		}

		methods = append(methods, method)
	}

	return methods, nil
}

// validateUserRelationships checks if referenced scopes, roles, groups, and resources exist
func (ur *UserRepository) validateUserRelationships(user *models.ExtendedUser) error {
	// Note: In sharedmodels.User v0.5.0, relationships are loaded via GORM associations
	// For now, we'll validate that if relationships are populated, they have valid IDs

	// This validation would be more comprehensive if we had access to the relationship tables
	// For now, we trust that the relationships are properly managed through the mapping functions

	return nil
}
