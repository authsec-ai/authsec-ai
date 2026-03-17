package models

import (
	"time"

	"github.com/google/uuid"
)

type TenantDomain struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID             uuid.UUID  `gorm:"type:uuid;not null"`
	Domain               string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	Kind                 string     `gorm:"type:varchar(32);not null;default:'custom'"`
	IsPrimary            bool       `gorm:"type:boolean;not null;default:false"`
	IsVerified           bool       `gorm:"type:boolean;not null;default:false"`
	VerificationToken    string     `gorm:"type:varchar(255);not null"`
	VerificationMethod   string     `gorm:"type:varchar(32);not null;default:'dns_txt'"`
	VerificationTXTName  *string    `gorm:"type:varchar(255)"`
	VerificationTXTValue *string    `gorm:"type:varchar(255)"`
	VerifiedAt           *time.Time `gorm:"type:timestamp"`
	CreatedAt            time.Time  `gorm:"type:timestamp;default:now()"`
	UpdatedAt            time.Time  `gorm:"type:timestamp;default:now()"`
}

func (TenantDomain) TableName() string {
	return "tenant_domains"
}
