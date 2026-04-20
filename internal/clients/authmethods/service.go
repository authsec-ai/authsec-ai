// Package authmethods resolves the authentication methods available for clients
// by querying the tenant_hydra_clients table populated by the OIDC configuration manager.
package authmethods

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// TenantHydraClient tracks Hydra OIDC client mappings per tenant.
type TenantHydraClient struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrgID             string         `json:"org_id" gorm:"not null;index"`
	TenantID          string         `json:"tenant_id" gorm:"not null;index"`
	TenantName        string         `json:"tenant_name" gorm:"not null"`
	HydraClientID     string         `json:"hydra_client_id" gorm:"not null;unique"`
	HydraClientSecret string         `json:"hydra_client_secret" gorm:"not null"`
	ClientName        string         `json:"client_name" gorm:"not null"`
	RedirectURIs      pq.StringArray `json:"redirect_uris" gorm:"type:text[];default:'{}'"`
	Scopes            pq.StringArray `json:"scopes" gorm:"type:text[];default:'{openid,profile,email}'"`
	ClientType        string         `json:"client_type" gorm:"not null"`
	ProviderName      string         `json:"provider_name,omitempty"`
	IsActive          bool           `json:"is_active" gorm:"default:true"`
	CreatedAt         time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
	CreatedBy         string         `json:"created_by" gorm:"default:'system'"`
	UpdatedBy         string         `json:"updated_by" gorm:"default:'system'"`
}

func (TenantHydraClient) TableName() string {
	return "tenant_hydra_clients"
}

// GetTenantHydraClientsRequest is the filter struct for listing hydra clients.
type GetTenantHydraClientsRequest struct {
	OrgID      string `json:"org_id,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	ClientType string `json:"client_type,omitempty"`
	IsActive   *bool  `json:"is_active,omitempty"`
}

// tenantHydraRepo is the internal repository interface used by Service.
type tenantHydraRepo interface {
	ListAll(req *GetTenantHydraClientsRequest) ([]*TenantHydraClient, error)
}

// hydraClientRepository is the concrete implementation that queries via GORM.
type hydraClientRepository struct {
	db *gorm.DB
}

func (r *hydraClientRepository) ListAll(req *GetTenantHydraClientsRequest) ([]*TenantHydraClient, error) {
	var clients []*TenantHydraClient

	query := r.db.Model(&TenantHydraClient{})

	if req.OrgID != "" {
		query = query.Where("org_id = ?", req.OrgID)
	}
	if req.TenantID != "" {
		query = query.Where("tenant_id = ?", req.TenantID)
	}
	if req.ClientType != "" {
		query = query.Where("client_type = ?", req.ClientType)
	}
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	if err := query.Order("created_at DESC").Find(&clients).Error; err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	return clients, nil
}

// Service coordinates lookups against the OIDC manager's tenant Hydra client data.
type Service struct {
	repo tenantHydraRepo
}

var (
	initOnce sync.Once
	sharedDB *gorm.DB
)

// NewService wires the OIDC manager repository against the shared DB connection.
func NewService(db *gorm.DB) *Service {
	initOnce.Do(func() {
		sharedDB = db
	})

	return &Service{repo: &hydraClientRepository{db: sharedDB}}
}

// MethodsForClients returns the authentication methods for the provided clients, keyed by ClientID.
func (s *Service) MethodsForClients(tenantID uuid.UUID, clients []sharedmodels.Client) (map[uuid.UUID][]string, error) {
	result := make(map[uuid.UUID][]string, len(clients))
	if len(clients) == 0 {
		return result, nil
	}

	tenantKey := tenantID.String()
	isActive := true
	orgRecords, err := s.repo.ListAll(&GetTenantHydraClientsRequest{
		TenantID:   tenantKey,
		ClientType: "oidc_provider",
		IsActive:   &isActive,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") ||
			strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			for _, client := range clients {
				result[client.ClientID] = buildMethodList(nil)
			}
			return result, nil
		}
		return nil, fmt.Errorf("fetch tenant hydra clients: %w", err)
	}

	baseClientProviders := make(map[string][]string)
	for _, record := range orgRecords {
		if record == nil || !record.IsActive {
			continue
		}

		providerName := strings.TrimSpace(record.ProviderName)
		displayName := providerName
		if displayName == "" {
			displayName = strings.TrimSpace(record.ClientName)
		}
		if displayName == "" {
			displayName = "oidc"
		}

		baseClientID := baseClientIDFromHydra(record.HydraClientID, providerName)
		if baseClientID == "" {
			continue
		}

		baseClientProviders[baseClientID] = append(baseClientProviders[baseClientID], displayName)
	}

	for _, client := range clients {
		baseID := client.ClientID.String()
		result[client.ClientID] = buildMethodList(baseClientProviders[baseID])
	}

	return result, nil
}

func baseClientIDFromHydra(hydraID, providerName string) string {
	if hydraID == "" {
		return ""
	}

	normalized := normalizeProviderName(providerName)
	if normalized != "" {
		suffix := fmt.Sprintf("-%s-oidc", normalized)
		if strings.HasSuffix(hydraID, suffix) {
			return strings.TrimSuffix(hydraID, suffix)
		}
	}

	if strings.HasSuffix(hydraID, "-oidc") {
		return strings.TrimSuffix(hydraID, "-oidc")
	}

	return ""
}

func buildMethodList(providers []string) []string {
	seen := make(map[string]struct{}, len(providers)+1)
	methods := []string{}

	add := func(method string) {
		method = strings.TrimSpace(method)
		if method == "" {
			return
		}
		if _, exists := seen[method]; exists {
			return
		}
		seen[method] = struct{}{}
		methods = append(methods, method)
	}

	add("password")
	for _, provider := range providers {
		add(provider)
	}

	if len(methods) > 1 {
		extras := append([]string{}, methods[1:]...)
		sort.Strings(extras)
		methods = append([]string{methods[0]}, extras...)
	}

	return methods
}

func normalizeProviderName(name string) string {
	lowered := strings.ToLower(strings.TrimSpace(name))
	lowered = strings.ReplaceAll(lowered, " ", "-")
	lowered = strings.ReplaceAll(lowered, "_", "-")
	return lowered
}
