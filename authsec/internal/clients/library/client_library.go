package library

import (
	"fmt"
	"strings"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// ClientLibrary provides programmatic access to client management operations
type ClientLibrary struct {
	db *gorm.DB
}

// NewClientLibrary creates a new client library instance
func NewClientLibrary(db *gorm.DB) *ClientLibrary {
	return &ClientLibrary{db: db}
}

// ClientCreateRequest represents the data needed to create a client record.
type ClientCreateRequest struct {
	TenantID      uuid.UUID
	ProjectID     uuid.UUID
	OwnerID       uuid.UUID
	OrgID         uuid.UUID
	Name          string
	Email         *string
	Active        bool
	Status        string
	Tags          []string
	HydraClientID *string
	OIDCEnabled   bool
}

// ClientUpdateRequest represents mutable fields for a client.
type ClientUpdateRequest struct {
	Name          *string
	Email         *string
	Active        *bool
	Status        *string
	Tags          *[]string
	HydraClientID *string
	OIDCEnabled   *bool
}

// ClientListFilters represents filters for listing clients.
type ClientListFilters struct {
	TenantID       uuid.UUID
	Status         string
	Tags           []string
	Name           string
	Email          string
	Active         *bool
	Page           int
	Limit          int
	IncludeDeleted *bool // When true, include soft-deleted clients (deleted_at IS NOT NULL)
}

// CreateClient creates a new client using shared models
func (cl *ClientLibrary) CreateClient(req *ClientCreateRequest) (*sharedmodels.Client, error) {
	if req == nil {
		return nil, fmt.Errorf("client request is required")
	}

	status := req.Status
	if status == "" {
		status = sharedmodels.StatusActive
	}

	clientID := uuid.New()
	client := &sharedmodels.Client{
		ID:            uuid.New(),
		ClientID:      clientID,
		TenantID:      req.TenantID,
		ProjectID:     req.ProjectID,
		OwnerID:       req.OwnerID,
		OrgID:         req.OrgID,
		Name:          req.Name,
		Email:         req.Email,
		Active:        req.Active,
		Status:        status,
		Tags:          pq.StringArray(req.Tags),
		HydraClientID: fmt.Sprintf("%s-main-client", clientID.String()),
		OIDCEnabled:   req.OIDCEnabled,
	}

	if req.HydraClientID != nil {
		if value := strings.TrimSpace(*req.HydraClientID); value != "" {
			client.HydraClientID = value
		}
	}

	if err := cl.db.Create(client).Error; err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

// GetClient retrieves a client by ID with tenant validation
func (cl *ClientLibrary) GetClient(id uuid.UUID, tenantID uuid.UUID) (*sharedmodels.Client, error) {
	var client sharedmodels.Client
	err := cl.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&client).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("client not found")
		}
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return &client, nil
}

// UpdateClient updates an existing client
func (cl *ClientLibrary) UpdateClient(id uuid.UUID, tenantID uuid.UUID, req *ClientUpdateRequest) (*sharedmodels.Client, error) {
	client, err := cl.GetClient(id, tenantID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = req.Email
	}
	if req.Status != nil {
		statusValue := strings.TrimSpace(*req.Status)
		updates["status"] = statusValue
		if strings.EqualFold(statusValue, sharedmodels.StatusDeleted) {
			updates["active"] = false
		}
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if req.Tags != nil {
		updates["tags"] = pq.StringArray(*req.Tags)
	}
	if req.HydraClientID != nil {
		value := strings.TrimSpace(*req.HydraClientID)
		if value == "" {
			value = client.ClientID.String()
		}
		updates["hydra_client_id"] = value
	}
	if req.OIDCEnabled != nil {
		updates["oidc_enabled"] = *req.OIDCEnabled
	}

	if len(updates) == 0 {
		return client, nil
	}

	if err := cl.db.Model(&sharedmodels.Client{}).Where("id = ? AND tenant_id = ?", id, tenantID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	return cl.GetClient(id, tenantID)
}

// DeleteClient soft-deletes a client via GORM (sets deleted_at) and marks it inactive
func (cl *ClientLibrary) DeleteClient(id uuid.UUID, tenantID uuid.UUID) error {
	_, err := cl.GetClient(id, tenantID)
	if err != nil {
		return err
	}

	// Mark status/active first, then let GORM set deleted_at
	if err := cl.db.Model(&sharedmodels.Client{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(map[string]interface{}{"status": sharedmodels.StatusDeleted, "active": false}).Error; err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	if err := cl.db.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&sharedmodels.Client{}).Error; err != nil {
		return fmt.Errorf("failed to soft-delete client: %w", err)
	}

	return nil
}

// ListClients retrieves clients with filtering and pagination
func (cl *ClientLibrary) ListClients(filters *ClientListFilters) ([]sharedmodels.Client, int64, error) {
	base := cl.db.Model(&sharedmodels.Client{})
	if filters.IncludeDeleted != nil && *filters.IncludeDeleted {
		base = base.Unscoped()
	}

	query := base.Where("tenant_id = ?", filters.TenantID)

	if filters.IncludeDeleted == nil || !*filters.IncludeDeleted {
		query = query.Where("status != ?", sharedmodels.StatusDeleted)
	}

	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Email != "" {
		query = query.Where("email LIKE ?", "%"+filters.Email+"%")
	}
	if filters.Active != nil {
		query = query.Where("active = ?", *filters.Active)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if len(filters.Tags) > 0 && cl.db.Dialector.Name() == "postgres" {
		query = query.Where("tags && ?", pq.Array(filters.Tags))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count clients: %w", err)
	}

	offset := 0
	if filters.Page > 1 {
		offset = (filters.Page - 1) * filters.Limit
	}
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit).Offset(offset)
	}

	var clients []sharedmodels.Client
	if err := query.Find(&clients).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list clients: %w", err)
	}

	return clients, total, nil
}

// ActivateClient sets a client's active status to true
func (cl *ClientLibrary) ActivateClient(id uuid.UUID, tenantID uuid.UUID) (*sharedmodels.Client, error) {
	active := true
	status := sharedmodels.StatusActive
	req := &ClientUpdateRequest{Active: &active, Status: &status}
	return cl.UpdateClient(id, tenantID, req)
}

// DeactivateClient sets a client's active status to false
func (cl *ClientLibrary) DeactivateClient(id uuid.UUID, tenantID uuid.UUID) (*sharedmodels.Client, error) {
	active := false
	status := sharedmodels.StatusInactive
	req := &ClientUpdateRequest{Active: &active, Status: &status}
	return cl.UpdateClient(id, tenantID, req)
}

// GetClientByClientID retrieves a client by ClientID with tenant validation
func (cl *ClientLibrary) GetClientByClientID(clientID uuid.UUID, tenantID uuid.UUID) (*sharedmodels.Client, error) {
	var client sharedmodels.Client
	err := cl.db.Where("client_id = ? AND tenant_id = ?", clientID, tenantID).First(&client).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("client not found")
		}
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return &client, nil
}

// UpdateClientByClientID updates an existing client using client_id field
func (cl *ClientLibrary) UpdateClientByClientID(clientID uuid.UUID, tenantID uuid.UUID, req *ClientUpdateRequest) (*sharedmodels.Client, error) {
	client, err := cl.GetClientByClientID(clientID, tenantID)
	if err != nil {
		return nil, err
	}

	if client.Name == "Default client" || client.Name == "default" {
		return nil, fmt.Errorf("default client cannot be updated")
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = req.Email
	}
	if req.Status != nil {
		statusValue := strings.TrimSpace(*req.Status)
		updates["status"] = statusValue
		if strings.EqualFold(statusValue, sharedmodels.StatusDeleted) {
			updates["active"] = false
		}
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if req.Tags != nil {
		updates["tags"] = pq.StringArray(*req.Tags)
	}
	if req.HydraClientID != nil {
		value := strings.TrimSpace(*req.HydraClientID)
		if value == "" {
			value = client.ClientID.String()
		}
		updates["hydra_client_id"] = value
	}
	if req.OIDCEnabled != nil {
		updates["oidc_enabled"] = *req.OIDCEnabled
	}

	if len(updates) == 0 {
		return client, nil
	}

	if err := cl.db.Model(&sharedmodels.Client{}).Where("client_id = ? AND tenant_id = ?", clientID, tenantID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	return cl.GetClientByClientID(clientID, tenantID)
}

// DeleteClientByClientID soft-deletes a client using client_id field
func (cl *ClientLibrary) DeleteClientByClientID(clientID uuid.UUID, tenantID uuid.UUID) error {
	c, err := cl.GetClientByClientID(clientID, tenantID)
	if err != nil {
		return err
	}

	if c.Name == "Default client" || c.Name == "default" {
		return fmt.Errorf("default client cannot be deleted")
	}

	if err := cl.db.Model(&sharedmodels.Client{}).
		Where("client_id = ? AND tenant_id = ?", clientID, tenantID).
		Updates(map[string]interface{}{"status": sharedmodels.StatusDeleted, "active": false}).Error; err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	if err := cl.db.Where("client_id = ? AND tenant_id = ?", clientID, tenantID).Delete(&sharedmodels.Client{}).Error; err != nil {
		return fmt.Errorf("failed to soft-delete client: %w", err)
	}

	return nil
}
