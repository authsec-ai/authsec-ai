package repositories

import (
	"context"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
)

// CertificateRepository defines the interface for certificate data operations
type CertificateRepository interface {
	GetByID(ctx context.Context, tenantID, id string) (*models.Certificate, error)
	GetBySerialNumber(ctx context.Context, tenantID, serialNumber string) (*models.Certificate, error)
	GetActiveByWorkload(ctx context.Context, tenantID, workloadID string) (*models.Certificate, error)
	Create(ctx context.Context, cert *models.Certificate) error
	Update(ctx context.Context, cert *models.Certificate) error
	Revoke(ctx context.Context, tenantID, id string) error
	ListByWorkload(ctx context.Context, tenantID, workloadID string) ([]*models.Certificate, error)
	ListExpiring(ctx context.Context, tenantID string, within time.Duration) ([]*models.Certificate, error)
}
