package utils

import (
	appmodels "github.com/authsec-ai/authsec/models"
	"gorm.io/gorm"
)

const mfaSelectExpr = "COALESCE(to_jsonb(users.mfa_method), '[]'::jsonb) AS mfa_method"

// WithUsersMFAMethodArray ensures user queries target the users table while
// normalizing the mfa_method column to JSON for consistent scanning.
func WithUsersMFAMethodArray(db *gorm.DB) *gorm.DB {
	if db == nil {
		return db
	}

	return db.Model(&appmodels.UserWithJSONMFAMethods{}).
		Select("users.*", mfaSelectExpr)
}
