package repositories

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// ExternalService is the GORM model for the services table (per-tenant DB).
type ExternalService struct {
	ID              string         `json:"id" gorm:"primaryKey"`
	Name            string         `json:"name" gorm:"not null"`
	Type            string         `json:"type"`
	URL             string         `json:"url"`
	Description     string         `json:"description"`
	Tags            pq.StringArray `json:"tags" gorm:"type:text[]" swaggertype:"array,string"`
	ResourceID      string         `json:"resource_id" gorm:"not null"`
	AuthType        string         `json:"auth_type" gorm:"not null"`
	AuthConfig      string         `json:"auth_config"` // JSON blob
	VaultPath       string         `json:"vault_path"`
	CreatedBy       string         `json:"created_by" gorm:"not null"`
	AgentAccessible bool           `json:"agent_accessible" gorm:"default:true"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (ExternalService) TableName() string { return "services" }

// ExternalServiceRepository provides CRUD operations for ExternalService.
type ExternalServiceRepository interface {
	Create(svc *ExternalService) error
	GetByID(id string) (*ExternalService, error)
	ListByClient(clientID string) ([]ExternalService, error)
	Update(svc *ExternalService) error
	Delete(id string) error
}

type externalServiceRepository struct{ db *gorm.DB }

func NewExternalServiceRepository(db *gorm.DB) ExternalServiceRepository {
	return &externalServiceRepository{db}
}

func (r *externalServiceRepository) Create(svc *ExternalService) error {
	return r.db.Create(svc).Error
}

func (r *externalServiceRepository) GetByID(id string) (*ExternalService, error) {
	var svc ExternalService
	err := r.db.First(&svc, "id = ?", id).Error
	return &svc, err
}

func (r *externalServiceRepository) ListByClient(clientID string) ([]ExternalService, error) {
	var svcs []ExternalService
	err := r.db.Where("created_by = ?", clientID).Find(&svcs).Error
	return svcs, err
}

func (r *externalServiceRepository) Update(svc *ExternalService) error {
	return r.db.Save(svc).Error
}

func (r *externalServiceRepository) Delete(id string) error {
	return r.db.Delete(&ExternalService{}, "id = ?", id).Error
}
