package repositories

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	sharedmodels "github.com/authsec-ai/sharedmodels"
)

type MFARepository struct {
	DB *gorm.DB
}

func NewMFARepository(db *gorm.DB) *MFARepository {
	repo := &MFARepository{DB: db}

	// Ensure the mfa_methods table exists
	if err := repo.ensureTableExists(); err != nil {
		log.Printf("Warning: Failed to ensure mfa_methods table exists: %v", err)
	}

	return repo
}

// ensureTableExists creates the mfa_methods table if it doesn't exist
func (r *MFARepository) ensureTableExists() error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS mfa_methods (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID NOT NULL,
			user_id UUID,
			method_type VARCHAR(50) NOT NULL,
			display_name VARCHAR(255),
			description VARCHAR(255),
			recommended BOOLEAN DEFAULT FALSE,
			method_data JSONB,
			enabled BOOLEAN DEFAULT FALSE,
			method_subtype VARCHAR(255),
			is_primary BOOLEAN DEFAULT FALSE,
			verified BOOLEAN DEFAULT FALSE,
			backup_codes TEXT,
			enrolled_at TIMESTAMPTZ DEFAULT NOW(),
			expires_at TIMESTAMPTZ,
			last_used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		
		-- Align legacy installations with TEXT-based backup_codes
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'mfa_methods'
					AND column_name = 'backup_codes'
					AND data_type <> 'text'
			) THEN
				ALTER TABLE mfa_methods
					ALTER COLUMN backup_codes TYPE TEXT USING backup_codes::text;
			END IF;
		END $$;
		
		-- Add unique constraint if it doesn't exist
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'mfa_methods_client_id_method_type_key'
			) THEN
				ALTER TABLE mfa_methods ADD CONSTRAINT mfa_methods_client_id_method_type_key UNIQUE(client_id, method_type);
			END IF;
		END $$;
		
		CREATE INDEX IF NOT EXISTS idx_mfa_methods_client_id ON mfa_methods(client_id);
		CREATE INDEX IF NOT EXISTS idx_mfa_methods_user_id ON mfa_methods(user_id);
		CREATE INDEX IF NOT EXISTS idx_mfa_methods_type ON mfa_methods(method_type);
		CREATE INDEX IF NOT EXISTS idx_mfa_methods_enabled ON mfa_methods(enabled);
	`

	return r.DB.Exec(createTableSQL).Error
}

// EnableMethod creates or updates an MFA method for a client
func (r *MFARepository) EnableMethod(clientID string, methodType string, data interface{}, userID uuid.UUID) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	now := time.Now()

	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_id"}, {Name: "method_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"method_data", "enabled", "verified", "enrolled_at", "updated_at"}),
	}).Create(&sharedmodels.MFAMethod{
		ClientID:   clientUUID,
		MethodType: methodType,
		MethodData: jsonData,
		Enabled:    true,
		Verified:   true,
		EnrolledAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
		UserID:     &userID,
	}).Error
}

// GetUserMethods returns all enabled MFA methods for a client
func (r *MFARepository) GetUserMethods(clientID string) ([]sharedmodels.MFAMethod, error) {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client ID format: %w", err)
	}

	var methods []sharedmodels.MFAMethod
	err = r.DB.Where("client_id = ? AND enabled = true", clientUUID).Find(&methods).Error
	return methods, err
}

// GetMethod returns a specific MFA method for a client
func (r *MFARepository) GetMethod(clientID string, methodType string) (*sharedmodels.MFAMethod, error) {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client ID format: %w", err)
	}

	var method sharedmodels.MFAMethod
	err = r.DB.Where("client_id = ? AND method_type = ?", clientUUID, methodType).First(&method).Error
	return &method, err
}

// UpdateLastUsed updates the last_used_at timestamp for a method
func (r *MFARepository) UpdateLastUsed(clientID string, methodType string) error {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	now := time.Now()
	return r.DB.Model(&sharedmodels.MFAMethod{}).
		Where("client_id = ? AND method_type = ?", clientUUID, methodType).
		Updates(map[string]interface{}{
			"last_used_at": &now,
			"updated_at":   now,
		}).Error
}

// DisableMethod disables an MFA method
func (r *MFARepository) DisableMethod(clientID string, methodType string) error {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	return r.DB.Model(&sharedmodels.MFAMethod{}).
		Where("client_id = ? AND method_type = ?", clientUUID, methodType).
		Updates(map[string]interface{}{
			"enabled":    false,
			"updated_at": time.Now(),
		}).Error
}

// HasMethod checks if a client has a specific MFA method enabled
func (r *MFARepository) HasMethod(clientID string, methodType string) (bool, error) {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return false, fmt.Errorf("invalid client ID format: %w", err)
	}

	var count int64
	err = r.DB.Model(&sharedmodels.MFAMethod{}).
		Where("client_id = ? AND method_type = ? AND enabled = true", clientUUID, methodType).
		Count(&count).Error
	return count > 0, err
}

// EnableMethodWithBackupCodes creates an MFA method with backup codes
func (r *MFARepository) EnableMethodWithBackupCodes(clientID string, methodType string, data interface{}, backupCodes pq.StringArray, userID uuid.UUID) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	backupCodesJSON, err := json.Marshal(backupCodes)
	if err != nil {
		return fmt.Errorf("failed to marshal backup codes: %w", err)
	}
	backupCodesStr := string(backupCodesJSON)

	now := time.Now()

	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_id"}, {Name: "method_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"method_data", "backup_codes", "enabled", "verified", "enrolled_at", "updated_at"}),
	}).Create(&sharedmodels.MFAMethod{
		ClientID:    clientUUID,
		MethodType:  methodType,
		MethodData:  jsonData,
		BackupCodes: &backupCodesStr,
		Enabled:     true,
		Verified:    true,
		EnrolledAt:  now,
		CreatedAt:   now,
		UpdatedAt:   now,
		UserID:      &userID,
	}).Error
}

// UpdateBackupCodes updates the backup codes for a method
func (r *MFARepository) UpdateBackupCodes(clientID, methodType string, backupCodes pq.StringArray) error {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	backupCodesJSON, err := json.Marshal(backupCodes)
	if err != nil {
		return fmt.Errorf("failed to marshal backup codes: %w", err)
	}
	backupCodesStr := string(backupCodesJSON)

	return r.DB.Model(&sharedmodels.MFAMethod{}).
		Where("client_id = ? AND method_type = ?", clientUUID, methodType).
		Updates(map[string]interface{}{
			"backup_codes": &backupCodesStr,
			"last_used_at": time.Now(),
			"updated_at":   time.Now(),
		}).Error
}

// EnableMethodWithExpiry creates an MFA method with expiration
func (r *MFARepository) EnableMethodWithExpiry(clientID string, methodType string, data interface{}, enabled bool, expiresAt time.Time, userID uuid.UUID) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	now := time.Now()

	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_id"}, {Name: "method_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"method_data", "enabled", "expires_at", "updated_at"}),
	}).Create(&sharedmodels.MFAMethod{
		ClientID:   clientUUID,
		MethodType: methodType,
		MethodData: jsonData,
		Enabled:    enabled,
		ExpiresAt:  &expiresAt,
		CreatedAt:  now,
		UpdatedAt:  now,
		UserID:     &userID,
	}).Error
}

// UpdateMethodData updates only the method_data field
func (r *MFARepository) UpdateMethodData(clientID, methodType string, data []byte) error {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	return r.DB.Model(&sharedmodels.MFAMethod{}).
		Where("client_id = ? AND method_type = ?", clientUUID, methodType).
		Update("method_data", data).Error
}
