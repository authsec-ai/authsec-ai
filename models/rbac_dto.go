package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RoleCreateRequest matches RoleMetadata + permission_ids
type RoleCreateRequest struct {
	Name          string      `json:"name" binding:"required"`
	Description   string      `json:"description"`
	IsSystem      bool        `json:"is_system"`
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

// RoleUpdateRequest matches RoleMetadata + permission_ids
type RoleUpdateRequest struct {
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	IsSystem      bool        `json:"is_system"`
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

// RoleBindingCreateRequest matches RoleBindingCreate
type RoleBindingCreateRequest struct {
	PrincipalIDs  []uuid.UUID     `json:"principal_ids" binding:"required"`
	PrincipalType string          `json:"principal_type" binding:"required,oneof=user service_account"`
	RoleID        uuid.UUID       `json:"role_id" binding:"required"`
	ScopeType     *string         `json:"scope_type"`
	ScopeID       *uuid.UUID      `json:"scope_id"`
	ExpiresAt     *time.Time      `json:"expires_at"`
	Conditions    json.RawMessage `json:"conditions"`
}

// ScopeCreateRequest matches ScopeDefinition
type ScopeCreateRequest struct {
	Name                string      `json:"name" binding:"required"`
	Description         string      `json:"description"`
	MappedPermissionIDs []uuid.UUID `json:"mapped_permission_ids" binding:"required"`
}

// PolicyCheckRequest matches the policy check payload
type PolicyCheckRequest struct {
	PrincipalID uuid.UUID  `json:"principal_id" binding:"required"`
	Resource    string     `json:"resource" binding:"required"`
	Action      string     `json:"action" binding:"required"`
	ScopeID     *uuid.UUID `json:"scope_id"`
}
