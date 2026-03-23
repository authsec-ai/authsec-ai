// repositories/client_repository.go
package repositories

import (
	"errors"
	"fmt"
	"log"
	"time"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	appmodels "github.com/authsec-ai/authsec/models"
	util "github.com/authsec-ai/authsec/utils"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Local Credential struct with correct column mappings for 'credentials' table
type Credential struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid();column:id"`
	ClientID        uuid.UUID      `gorm:"type:uuid;not null;column:client_id"`
	CredentialID    []byte         `gorm:"not null;uniqueIndex;column:credential_id"`
	PublicKey       []byte         `gorm:"not null;column:public_key"`
	AttestationType string         `gorm:"not null;column:attestation_type"`
	AAGUID          *uuid.UUID     `gorm:"type:uuid;column:aaguid"`
	SignCount       int64          `gorm:"not null;default:0;column:sign_count"`
	BackupEligible  bool           `gorm:"default:false;column:backup_eligible"`
	BackupState     bool           `gorm:"default:false;column:backup_state"`
	Transports      pq.StringArray `gorm:"type:text[];column:transports"`
	RPID            *string        `gorm:"type:varchar(255);column:rp_id"`
	CreatedAt       time.Time      `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime;column:updated_at"`
}

// TableName returns the table name for the Credential model
func (Credential) TableName() string {
	return "credentials"
}

// WebAuthnCredential struct for 'webauthn_credentials' table
type WebAuthnCredential struct {
	ID              int            `gorm:"primaryKey;autoIncrement;column:id"`
	UserID          uuid.UUID      `gorm:"type:uuid;column:user_id"`
	CredentialID    string         `gorm:"not null;uniqueIndex;column:credential_id"`
	PublicKey       string         `gorm:"not null;column:public_key"`
	AttestationType string         `gorm:"column:attestation_type"`
	Transports      pq.StringArray `gorm:"type:text[];column:transports"`
	BackupEligible  bool           `gorm:"default:false;column:backup_eligible"`
	BackupState     bool           `gorm:"default:false;column:backup_state"`
	SignCount       int64          `gorm:"default:0;column:sign_count"`
	UserPresent     bool           `gorm:"default:false;column:user_present"`
	UserVerified    bool           `gorm:"default:false;column:user_verified"`
	AAGUID          string         `gorm:"column:aaguid"`
	CreatedAt       time.Time      `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime;column:updated_at"`
}

// TableName returns the table name for the WebAuthnCredential model
func (WebAuthnCredential) TableName() string {
	return "webauthn_credentials"
}

type GlobalRepository struct {
	DB *gorm.DB
}

func NewGlobalRepository(db *gorm.DB) *GlobalRepository {
	return &GlobalRepository{DB: db}
}

func (r *GlobalRepository) GetTenantByID(tenantID string) (*sharedmodels.Tenant, error) {
	var tenant sharedmodels.Tenant
	if err := r.DB.Where("tenant_id = ?", tenantID).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// GetVerifiedCustomDomainForTenant returns the verified custom domain for a tenant, if any
func (r *GlobalRepository) GetVerifiedCustomDomainForTenant(tenantID string) (string, error) {
	var domain string
	err := r.DB.Table("tenant_domains").
		Select("domain").
		Where("tenant_id = ? AND is_verified = ? AND kind = ?", tenantID, true, "custom").
		Order("is_primary DESC, created_at ASC"). // Prefer primary domain, then oldest
		Limit(1).
		Pluck("domain", &domain).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil // No custom domain found
		}
		return "", err
	}
	return domain, nil
}

// GetCustomDomainVerificationTime returns when the custom domain was verified
func (r *GlobalRepository) GetCustomDomainVerificationTime(tenantID string) (*time.Time, error) {
	var domain appmodels.TenantDomain
	err := r.DB.Table("tenant_domains").
		Select("verified_at").
		Where("tenant_id = ? AND is_verified = ? AND kind = ?", tenantID, true, "custom").
		Order("is_primary DESC, created_at ASC").
		Limit(1).
		First(&domain).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return domain.VerifiedAt, nil
}

type ClientRepository struct {
	DB *gorm.DB
}

func NewClientRepository(db *gorm.DB) *ClientRepository {
	return &ClientRepository{DB: db}
}

func (r *ClientRepository) GetClientByEmail(email string) (*sharedmodels.User, error) {
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := r.DB.Scopes(util.WithUsersMFAMethodArray).Where("email = ?", email).First(&userWithJSONMFA).Error; err != nil {
		return nil, err
	}

	user := userWithJSONMFA.ToShared()
	return &user, nil
}

type CredentialRepository struct {
	DB *gorm.DB
}

func NewCredentialRepository(db *gorm.DB) *CredentialRepository {
	return &CredentialRepository{DB: db}
}

func (r *CredentialRepository) AddCredential(userID string, cred *webauthn.Credential) error {
	// Parse userID as UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	// Convert AAGUID bytes to UUID
	var aaguid *uuid.UUID
	if len(cred.Authenticator.AAGUID) == 16 {
		parsedAAGUID, err := uuid.FromBytes(cred.Authenticator.AAGUID)
		if err != nil {
			log.Printf("Warning: Invalid AAGUID format: %v", err)
			// Use nil for invalid AAGUID (database allows null)
			aaguid = nil
		} else {
			aaguid = &parsedAAGUID
		}
	}

	// Convert transports to string array
	var transports pq.StringArray
	if cred.Transport != nil {
		transports = make(pq.StringArray, len(cred.Transport))
		for i, transport := range cred.Transport {
			transports[i] = string(transport)
		}
	}

	// Generate UUID for the credential record
	credentialUUID := uuid.New()

	credential := Credential{
		ID:              credentialUUID,                      // Use proper UUID type
		ClientID:        userUUID,                            // Map userID to ClientID
		CredentialID:    cred.ID,                             // WebAuthn credential ID
		PublicKey:       cred.PublicKey,                      // Public key bytes
		AttestationType: cred.AttestationType,                // Attestation type
		AAGUID:          aaguid,                              // UUID pointer (nullable)
		SignCount:       int64(cred.Authenticator.SignCount), // Convert uint32 to int64
		BackupEligible:  cred.Flags.BackupEligible,           // Persist BE flag
		BackupState:     cred.Flags.BackupState,              // Persist BS flag
		Transports:      transports,                          // Transport methods
		CreatedAt:       time.Now(),                          // Creation timestamp
		UpdatedAt:       time.Now(),                          // Update timestamp
	}

	// Omit columns if the DB schema hasn't been migrated yet
	db := r.DB
	mig := db.Migrator()
	if !mig.HasColumn(&Credential{}, "backup_eligible") {
		db = db.Omit("backup_eligible")
	}
	if !mig.HasColumn(&Credential{}, "backup_state") {
		db = db.Omit("backup_state")
	}

	return db.Create(&credential).Error
}

// AddWebAuthnCredential saves credential to webauthn_credentials table
func (r *CredentialRepository) AddWebAuthnCredential(userID string, cred *webauthn.Credential) error {
	// Parse userID as UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	// Convert credential ID bytes to hex string
	credentialIDHex := fmt.Sprintf("%x", cred.ID)

	// Convert public key bytes to hex string
	publicKeyHex := fmt.Sprintf("%x", cred.PublicKey)

	// Convert AAGUID bytes to hex string
	var aaguidHex string
	if len(cred.Authenticator.AAGUID) == 16 {
		aaguidHex = fmt.Sprintf("%x", cred.Authenticator.AAGUID)
	}

	// Convert transports to string array
	var transports pq.StringArray
	if cred.Transport != nil {
		transports = make(pq.StringArray, len(cred.Transport))
		for i, transport := range cred.Transport {
			transports[i] = string(transport)
		}
	}

	webauthnCred := WebAuthnCredential{
		UserID:          userUUID,
		CredentialID:    credentialIDHex,
		PublicKey:       publicKeyHex,
		AttestationType: cred.AttestationType,
		Transports:      transports,
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
		SignCount:       int64(cred.Authenticator.SignCount),
		UserPresent:     cred.Flags.UserPresent,
		UserVerified:    cred.Flags.UserVerified,
		AAGUID:          aaguidHex,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	return r.DB.Create(&webauthnCred).Error
}

func (r *ClientRepository) GetClientByEmailAndTenant(email, tenantID, clientID *string) (*sharedmodels.User, error) {
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods

	query := r.DB.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", email, tenantID)
	if clientID != nil && *clientID != "" {
		query = query.Where("client_id = ?", *clientID)
	}

	err := query.First(&userWithJSONMFA).Error
	if err != nil {
		// MFA check failed: log and fallback to query without MFA condition
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("MFA not enabled or user not found for email: %v client: %v", valueOrNil(email), valueOrNil(clientID))
		} else {
			log.Printf("Database error during MFA-enabled user query for email: %v, error: %v", email, err)
			return nil, err
		}
		// Fallback query without MFA condition
		err = r.DB.Scopes(util.WithUsersMFAMethodArray).
			Where("email = ? AND tenant_id = ?", email, tenantID).First(&userWithJSONMFA).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("User not found in fallback query for email: %v", email)
			} else {
				log.Printf("Database error in fallback user query for email: %v, error: %v", email, err)
			}
			return nil, err
		}
		// At this point, user exists but MFA is not enabled
		fallbackUser := userWithJSONMFA.ToShared()
		log.Printf("Fallback successful: User found with MFA disabled for email: %s", fallbackUser.Email)
	} else {
		// MFA-enabled user found
		enabledUser := userWithJSONMFA.ToShared()
		log.Printf("MFA-enabled user found for email: %s", enabledUser.Email)
	}

	user := userWithJSONMFA.ToShared()
	return &user, nil
}

func valueOrNil(value *string) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func (r *ClientRepository) GetClientByEmailAndTenantForLogin(email, tenantID string) (*sharedmodels.User, error) {
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	err := r.DB.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ? AND provider = ?", email, tenantID, "local").First(&userWithJSONMFA).Error
	if err != nil {
		return nil, err
	}

	// Return the embedded User
	user := userWithJSONMFA.ToShared()
	return &user, nil
}

// GetClientByEmailTenantAndClient retrieves a user by email, tenant_id, and client_id
// This method takes values instead of pointers to avoid SQL NULL issues
func (r *ClientRepository) GetClientByEmailTenantAndClient(email, tenantID, clientID string) (*sharedmodels.User, error) {
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	err := r.DB.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ? AND client_id = ?", email, tenantID, clientID).First(&userWithJSONMFA).Error
	if err != nil {
		return nil, err
	}

	// Return the embedded User
	user := userWithJSONMFA.ToShared()
	return &user, nil
}

// GetClientForTOTP fetches client for TOTP operations with proper filters
func (r *ClientRepository) GetClientForTOTP(email, tenantID, clientID string) (*sharedmodels.User, error) {
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	query := r.DB.Scopes(util.WithUsersMFAMethodArray).Where("email = ? AND tenant_id = ?", email, tenantID)

	// Add clientID filter if provided
	if clientID != "" {
		query = query.Where("client_id = ?", clientID)
	}

	// Check for TOTP-enabled users using text array contains
	err := query.Where("mfa_method @> ARRAY[?]::text[]", "totp").First(&userWithJSONMFA).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Try without TOTP filter for new setup
			query = r.DB.Scopes(util.WithUsersMFAMethodArray).Where("email = ? AND tenant_id = ?", email, tenantID)
			if clientID != "" {
				query = query.Where("client_id = ?", clientID)
			}
			err = query.First(&userWithJSONMFA).Error
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	user := userWithJSONMFA.ToShared()
	return &user, nil
}

// GetClientForTOTPLogin fetches client for TOTP login operations
func (r *ClientRepository) GetClientForTOTPLogin(email, tenantID string) (*sharedmodels.User, error) {
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods

	// First try to find user with TOTP enabled using text array contains
	err := r.DB.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ? AND provider = ? AND mfa_method @> ARRAY[?]::text[]",
			email, tenantID, "local", "totp").First(&userWithJSONMFA).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Fallback to find user without TOTP for setup
			err = r.DB.Scopes(util.WithUsersMFAMethodArray).
				Where("email = ? AND tenant_id = ? AND provider = ?",
					email, tenantID, "local").First(&userWithJSONMFA).Error
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	user := userWithJSONMFA.ToShared()
	return &user, nil
}

func (r *ClientRepository) SaveCredentialWithMFA(clientID uuid.UUID, cred *webauthn.Credential, method string) error {
	tx := r.DB.Begin()

	// Map webauthn.Credential → sharedmodels.Credential
	credRecord := sharedmodels.Credential{
		ID:              uuid.New(), // internal DB primary key
		ClientID:        clientID,
		CredentialID:    cred.ID, // from webauthn
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		SignCount:       int64(cred.Authenticator.SignCount),
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
		Transports:      pq.StringArray{}, // optional — fill if available
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	// Map AAGUID if present
	if cred.Authenticator.AAGUID != nil {
		if parsed, err := uuid.FromBytes(cred.Authenticator.AAGUID); err == nil {
			credRecord.AAGUID = &parsed
		}
	}

	if err := tx.Create(&credRecord).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Save MFA method
	now := time.Now().UTC()
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_id"}, {Name: "method_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "verified", "updated_at", "enrolled_at"}),
	}).Create(&sharedmodels.MFAMethod{
		ClientID:   clientID,
		MethodType: method,
		Enabled:    true,
		Verified:   true,
		EnrolledAt: now,
		UpdatedAt:  now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *ClientRepository) SaveCredential(credential *Credential) error {
	// Simple save - migrations ensure all columns exist
	return r.DB.Create(credential).Error
}

func (r *ClientRepository) GetCredentialsByClientID(clientID string) ([]Credential, error) {
	var credentials []Credential

	// Simple query - the migrations ensure the table and columns exist
	err := r.DB.Where("client_id = ?", clientID).Find(&credentials).Error
	if err != nil {
		log.Printf("GetCredentialsByClientID: error querying credentials for client_id=%s: %v", clientID, err)
		return nil, err
	}

	log.Printf("GetCredentialsByClientID: found %d credentials for client_id=%s", len(credentials), clientID)
	return credentials, nil
}

// Update credential sign count after successful authentication
func (r *ClientRepository) UpdateCredentialSignCount(credentialID []byte, newSignCount uint32) error {
	return r.DB.Model(&Credential{}).
		Where("credential_id = ?", credentialID).
		Update("sign_count", int64(newSignCount)).Error
}

// UpdateCredentialFlags updates BE/BS flags if the schema supports these columns.
func (r *ClientRepository) UpdateCredentialFlags(credentialID []byte, be, bs bool) error {
	// Simple update - migrations ensure all columns exist
	return r.DB.Model(&Credential{}).
		Where("credential_id = ?", credentialID).
		Updates(map[string]interface{}{
			"backup_eligible": be,
			"backup_state":    bs,
		}).Error
}

// Check if client has WebAuthn enabled and has credentials
func (r *ClientRepository) HasWebAuthnCredentials(clientID string) (bool, error) {
	var count int64
	err := r.DB.Model(&Credential{}).
		Where("client_id = ?", clientID).
		Count(&count).Error
	return count > 0, err
}

// In repositories/client_repository.go
func (r *ClientRepository) GetCredentialCountByClientID(clientID string) (int, error) {
	var count int64
	err := r.DB.Model(&Credential{}).Where("client_id = ?", clientID).Count(&count).Error
	return int(count), err
}

// HasCredentialsForRPID checks if a user has WebAuthn credentials registered for a specific RP ID
// This checks the rp_id column to ensure credentials were created for the specified domain.
// For legacy credentials (rp_id IS NULL), they are considered invalid for specific domain checking.
func (r *ClientRepository) HasCredentialsForRPID(clientID, rpID string) (bool, error) {
	var count int64
	err := r.DB.Model(&Credential{}).
		Where("client_id = ? AND rp_id = ?", clientID, rpID).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("error checking credentials for RP ID: %w", err)
	}

	return count > 0, nil
}

// GetMostRecentCredentialCreationTime returns the creation time of the most recent credential
func (r *ClientRepository) GetMostRecentCredentialCreationTime(clientID string) (*time.Time, error) {
	var createdAt *time.Time
	err := r.DB.Model(&Credential{}).
		Select("created_at").
		Where("client_id = ?", clientID).
		Order("created_at DESC").
		Limit(1).
		Pluck("created_at", &createdAt).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return createdAt, nil
}
