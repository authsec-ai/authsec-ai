package repositories

import (
	"context"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
)

// AuditRepository defines the interface for audit log operations
type AuditRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	List(ctx context.Context, tenantID string, limit, offset int) ([]*models.AuditLog, error)
	ListByWorkload(ctx context.Context, tenantID, workloadID string, limit, offset int) ([]*models.AuditLog, error)
	ListByEventType(ctx context.Context, tenantID string, eventType models.AuditEventType, limit, offset int) ([]*models.AuditLog, error)
	ListByDateRange(ctx context.Context, tenantID string, from, to time.Time, limit, offset int) ([]*models.AuditLog, error)
}
