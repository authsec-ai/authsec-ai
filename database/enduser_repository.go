package database

import (
"database/sql"
"fmt"
"time"

"github.com/authsec-ai/authsec/models"
"github.com/google/uuid"
"golang.org/x/crypto/bcrypt"
)

// EndUserRepository handles end-user database operations on tenant DB
type EndUserRepository struct {
	db interface{} // Can be *DBConnection or *sql.DB depending on tenant connection
}

// NewEndUserRepository creates a new end-user repository
func NewEndUserRepository(db interface{}) *EndUserRepository {
	return &EndUserRepository{db: db}
}

// setDB sets the database connection (for dynamic tenant connections)
func (eur *EndUserRepository) setDB(db interface{}) {
	eur.db = db
}

// executeQuery executes a query on the current database connection
func (eur *EndUserRepository) executeQuery(query string, args ...interface{}) (*sql.Rows, error) {
	switch db := eur.db.(type) {
	case *DBConnection:
		return db.Query(query, args...)
	case *sql.DB:
		return db.Query(query, args...)
	default:
		return nil, fmt.Errorf("unsupported database connection type")
	}
}

// executeQueryRow executes a query that returns a single row
func (eur *EndUserRepository) executeQueryRow(query string, args ...interface{}) *sql.Row {
	switch db := eur.db.(type) {
	case *DBConnection:
		return db.QueryRow(query, args...)
	case *sql.DB:
		return db.QueryRow(query, args...)
	default:
		// Return a row that will error when scanned
		return nil
	}
}

// executeExec executes a command that doesn't return rows
func (eur *EndUserRepository) executeExec(query string, args ...interface{}) (sql.Result, error) {
switch db := eur.db.(type) {
case *DBConnection:
return db.Exec(query, args...)
case *sql.DB:
return db.Exec(query, args...)
default:
return nil, fmt.Errorf("unsupported database connection type")
}
}

// CreateUser creates a new end-user in tenant database
func (eur *EndUserRepository) CreateUser(user *models.ExtendedUser) error {
	// Password should already be hashed in PasswordHash field
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

_, err := eur.executeExec(query,
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
user.MFAMethod,
user.MFADefaultMethod,
user.MFAEnrolledAt,
user.MFAVerified,
user.LastLogin,
user.CreatedAt,
user.UpdatedAt,
)

if err != nil {
return fmt.Errorf("failed to create end-user: %w", err)
}

return nil
}

// GetUserByEmail retrieves an end-user by email from tenant database (case-insensitive)
func (eur *EndUserRepository) GetUserByEmail(email string, clientID string) (*models.User, error) {
query := `
SELECT id, client_id, tenant_id, project_id, name, username, email,
password_hash, tenant_domain, provider, provider_id, provider_data,
avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
mfa_enrolled_at, mfa_verified, last_login, created_at, updated_at
FROM users
WHERE LOWER(email) = LOWER($1) AND client_id = $2 AND active = true
`

var user models.User
row := eur.executeQueryRow(query, email, clientID)
if row == nil {
return nil, fmt.Errorf("database connection error")
}

err := row.Scan(
&user.ID,
&user.ClientID,
&user.TenantID,
&user.ProjectID,
&user.Name,
&user.Username,
&user.Email,
&user.PasswordHash,
&user.TenantDomain,
&user.Provider,
&user.ProviderID,
&user.ProviderData,
&user.AvatarURL,
&user.Active,
&user.MFAEnabled,
&user.MFAMethod,
&user.MFADefaultMethod,
&user.MFAEnrolledAt,
&user.MFAVerified,
&user.LastLogin,
&user.CreatedAt,
&user.UpdatedAt,
)

if err != nil {
if err == sql.ErrNoRows {
return nil, fmt.Errorf("user not found")
}
return nil, fmt.Errorf("failed to get user: %w", err)
}

return &user, nil
}

// GetUserByID retrieves an end-user by ID from tenant database
func (eur *EndUserRepository) GetUserByID(id uuid.UUID) (*models.User, error) {
query := `
SELECT id, client_id, tenant_id, project_id, name, username, email,
password_hash, tenant_domain, provider, provider_id, provider_data,
avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
mfa_enrolled_at, mfa_verified, last_login, created_at, updated_at
FROM users
WHERE id = $1 AND active = true
`

var user models.User
row := eur.executeQueryRow(query, id)
if row == nil {
return nil, fmt.Errorf("database connection error")
}

err := row.Scan(
&user.ID,
&user.ClientID,
&user.TenantID,
&user.ProjectID,
&user.Name,
&user.Username,
&user.Email,
&user.PasswordHash,
&user.TenantDomain,
&user.Provider,
&user.ProviderID,
&user.ProviderData,
&user.AvatarURL,
&user.Active,
&user.MFAEnabled,
&user.MFAMethod,
&user.MFADefaultMethod,
&user.MFAEnrolledAt,
&user.MFAVerified,
&user.LastLogin,
&user.CreatedAt,
&user.UpdatedAt,
)

if err != nil {
if err == sql.ErrNoRows {
return nil, fmt.Errorf("user not found")
}
return nil, fmt.Errorf("failed to get user: %w", err)
}

return &user, nil
}

// UpdateUser updates an end-user in tenant database
func (eur *EndUserRepository) UpdateUser(id uuid.UUID, updates map[string]interface{}) error {
if len(updates) == 0 {
return fmt.Errorf("no updates provided")
}

query := "UPDATE users SET "
args := []interface{}{}
argCount := 1

for field, value := range updates {
query += field + " = $" + fmt.Sprintf("%d", argCount) + ", "
args = append(args, value)
argCount++
}

query += "updated_at = $" + fmt.Sprintf("%d", argCount)
args = append(args, time.Now())
argCount++

query += " WHERE id = $" + fmt.Sprintf("%d", argCount)
args = append(args, id)

_, err := eur.executeExec(query, args...)
if err != nil {
return fmt.Errorf("failed to update user: %w", err)
}

return nil
}

// DeleteUser soft deletes an end-user in tenant database
func (eur *EndUserRepository) DeleteUser(id uuid.UUID) error {
query := "UPDATE users SET active = false, updated_at = $1 WHERE id = $2"

_, err := eur.executeExec(query, time.Now(), id)
if err != nil {
return fmt.Errorf("failed to delete user: %w", err)
}

return nil
}

// GetUsersByTenant retrieves all users for a tenant from tenant database
func (eur *EndUserRepository) GetUsersByTenant(tenantID string, limit, offset int) ([]models.User, error) {
query := `
SELECT id, client_id, tenant_id, project_id, name, username, email,
password_hash, tenant_domain, provider, provider_id, provider_data,
avatar_url, active, mfa_enabled, mfa_method, mfa_default_method,
mfa_enrolled_at, mfa_verified, last_login, created_at, updated_at
FROM users
WHERE tenant_id = $1 AND active = true
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`

rows, err := eur.executeQuery(query, tenantID, limit, offset)
if err != nil {
return nil, fmt.Errorf("failed to query users: %w", err)
}
defer rows.Close()

var users []models.User
for rows.Next() {
var user models.User
err := rows.Scan(
&user.ID,
&user.ClientID,
&user.TenantID,
&user.ProjectID,
&user.Name,
&user.Username,
&user.Email,
&user.PasswordHash,
&user.TenantDomain,
&user.Provider,
&user.ProviderID,
&user.ProviderData,
&user.AvatarURL,
&user.Active,
&user.MFAEnabled,
&user.MFAMethod,
&user.MFADefaultMethod,
&user.MFAEnrolledAt,
&user.MFAVerified,
&user.LastLogin,
&user.CreatedAt,
&user.UpdatedAt,
)
if err != nil {
return nil, fmt.Errorf("failed to scan user: %w", err)
}
users = append(users, user)
}

return users, nil
}

// UpdateLastLogin updates the last login time for an end-user
func (eur *EndUserRepository) UpdateLastLogin(id uuid.UUID) error {
query := "UPDATE users SET last_login = $1, updated_at = $1 WHERE id = $2"

_, err := eur.executeExec(query, time.Now(), id)
if err != nil {
return fmt.Errorf("failed to update last login: %w", err)
}

return nil
}

// VerifyPassword verifies an end-user's password
func (eur *EndUserRepository) VerifyPassword(email, password, clientID string) (*models.User, error) {
	user, err := eur.GetUserByEmail(email, clientID)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}
