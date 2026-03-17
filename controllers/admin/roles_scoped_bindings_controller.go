package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RolesScopedBindingsController manages roles, permissions linkage, and bindings for both admin (primary DB) and end-user (tenant DB) contexts.
type RolesScopedBindingsController struct{}

func NewRolesScopedBindingsController() *RolesScopedBindingsController {
	return &RolesScopedBindingsController{}
}

// CreateRoleRequest represents the payload for creating a role.
type CreateRoleRequest struct {
	Name              string   `json:"name" binding:"required"`
	Description       string   `json:"description"`
	PermissionIDs     []string `json:"permission_ids"`
	PermissionStrings []string `json:"permission_strings"`
}

// CreateRoleResponse represents the response for creating or updating a role.
type CreateRoleResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	PermissionsCount int    `json:"permissions_count"`
}

// RoleListItem enriches a listed role with counts and user details.
type RoleListItem struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	UsersAssigned    int64    `json:"users_assigned"`
	PermissionsCount int      `json:"permissions_count"`
	UserIDs          []string `json:"user_ids,omitempty"`
	Usernames        []string `json:"usernames,omitempty"`
}

// BindingRequest represents the payload for role assignment.
type BindingRequest struct {
	UserID string        `json:"user_id" binding:"required"`
	RoleID string        `json:"role_id" binding:"required"`
	Scope  *BindingScope `json:"scope,omitempty"`
	// Conditions can include metadata such as {"mfa_required": true}
	Conditions map[string]interface{} `json:"conditions"`
}

// BindingScope represents scope information for a role assignment.
// Use "*" for both type and id to indicate tenant-wide (no specific scope).
// The scope_id does NOT reference the scopes table - it's a pointer to any external resource.
type BindingScope struct {
	Type string `json:"type"` // e.g. "project", "organization", or "*" for tenant-wide
	ID   string `json:"id"`   // UUID of the resource or "*" for tenant-wide
}

// BindingResponse represents the response for a role binding creation.
type BindingResponse struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	ScopeDescription string `json:"scope_description"`
	RoleName         string `json:"role_name,omitempty"`
}

// RoleBindingListItem represents a single role binding in the list response.
type RoleBindingListItem struct {
	ID               string                 `json:"id"`
	UserID           string                 `json:"user_id,omitempty"`
	Username         string                 `json:"username,omitempty"`
	Email            string                 `json:"email,omitempty"`
	ServiceAccountID string                 `json:"service_account_id,omitempty"`
	RoleID           string                 `json:"role_id"`
	RoleName         string                 `json:"role_name"`
	ScopeType        string                 `json:"scope_type,omitempty"`
	ScopeID          string                 `json:"scope_id,omitempty"`
	Conditions       map[string]interface{} `json:"conditions,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	ExpiresAt        string                 `json:"expires_at,omitempty"`
}

// --- Admin endpoints (primary DB) ---

// CreateRoleCompositeAdmin godoc
// @Summary Create Role (Admin)
// @Description Uses the primary admin database. Transaction: 1) insert into roles, 2) insert into role_permissions with provided permission_ids/permission_strings.
// @Tags RBAC: Roles & Bindings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body CreateRoleRequest true "Role creation payload (name, description, permission_ids[] and/or permission_strings[] resource:action; mapped into role_permissions)"
// @Success 200 {object} CreateRoleResponse "Created role with ID and permissions count"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/roles [post]
func (rc *RolesScopedBindingsController) CreateRoleCompositeAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	rc.createRole(c, config.DB, services.NewRBACService(config.DB), *tenantID, true)
}

// ListRolesAdmin godoc
// @Summary List Roles (Admin)
// @Description Uses the primary admin database. Returns role summary (id, name, description, permissions_count, users_assigned, user_ids/usernames). Optional filters: resource, role_id, user_id. For full permission details, call GET /uflow/admin/roles/{role_id}. Optional grouping by role_id or user_id via query params.
// @Tags RBAC: Roles & Bindings
// @Produce json
// @Security BearerAuth
// @Param resource query string false "Filter by resource"
// @Param role_id query string false "Filter by role_id"
// @Param user_id query string false "Filter by user_id"
// @Param group_by query string false "Group results by 'role_id' or 'user_id'"
// @Success 200 {array} RoleListItem
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/roles [get]
func (rc *RolesScopedBindingsController) ListRolesAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	rc.listRoles(c, config.DB, *tenantID)
}

// GetRoleAdmin godoc
// @Summary Get Role by ID (Admin)
// @Description Returns a single role with its permissions and assigned users. Uses the primary admin database.
// @Tags RBAC: Roles & Bindings
// @Produce json
// @Security BearerAuth
// @Param role_id path string true "Role ID (UUID)"
// @Success 200 {object} RoleListItem
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/roles/{role_id} [get]
func (rc *RolesScopedBindingsController) GetRoleAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	roleIDStr := c.Param("role_id")
	if _, err := uuid.Parse(roleIDStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}
	// Inject role_id as query param so listRoles filters to this single role
	c.Request.URL.RawQuery = "role_id=" + roleIDStr
	rc.listRoles(c, config.DB, *tenantID)
}

// UpdateRoleCompositeAdmin godoc
// @Summary Update Role (Admin)
// @Description Uses the primary admin database. Transaction: update role then replace role_permissions.
// @Tags RBAC: Roles & Bindings
// @Accept json
// @Produce json
// @Param role_id path string true "Role ID (UUID)"
// @Param input body CreateRoleRequest true "Role update payload"
// @Success 200 {object} CreateRoleResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/roles/{role_id} [put]
func (rc *RolesScopedBindingsController) UpdateRoleCompositeAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	rc.updateRole(c, config.DB, *tenantID)
}

// DeleteRoleAdmin godoc
// @Summary Delete Role (Admin)
// @Description Uses the primary admin database. Deletes a role and all its permission mappings.
// @Tags RBAC: Roles & Bindings
// @Produce json
// @Param role_id path string true "Role ID (UUID)"
// @Success 200 {object} map[string]string "Role deleted successfully"
// @Failure 400 {object} map[string]string "Invalid role ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Role not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/roles/{role_id} [delete]
func (rc *RolesScopedBindingsController) DeleteRoleAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	rc.deleteRole(c, config.DB, *tenantID)
}

// AssignRoleScopedAdmin godoc
// @Summary Assign Role (Admin)
// @Description Uses the primary admin database. Inserts into role_bindings; scope can be tenant-wide (null) or scoped (type/id). Conditions JSON stored on binding.
// @Tags RBAC: Roles & Bindings
// @Accept json
// @Produce json
// @Param input body BindingRequest true "Role assignment payload"
// @Success 200 {object} BindingResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/bindings [post]
func (rc *RolesScopedBindingsController) AssignRoleScopedAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	rc.assignRoleScoped(c, config.DB, services.NewRBACService(config.DB), *tenantID)
}

// ListRoleBindingsAdmin godoc
// @Summary List Role Bindings (Admin)
// @Description Uses the primary admin database. Returns all role bindings with user/role details. Optional filters: user_id, role_id, scope_type.
// @Tags RBAC: Roles & Bindings
// @Produce json
// @Security BearerAuth
// @Param user_id query string false "Filter by user_id"
// @Param role_id query string false "Filter by role_id"
// @Param scope_type query string false "Filter by scope_type"
// @Success 200 {array} RoleBindingListItem
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/bindings [get]
func (rc *RolesScopedBindingsController) ListRoleBindingsAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	rc.listRoleBindings(c, config.DB, *tenantID)
}

// --- End-user endpoints (tenant DB) ---

// CreateRoleCompositeEndUser godoc
// @Summary Create Role (End User)
// @Description Uses the tenant database. Transaction: insert into roles then role_permissions with provided permission_ids/permission_strings. Tenant context required from token.
// @Tags RBAC: Roles & Bindings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body CreateRoleRequest true "Role creation payload"
// @Success 200 {object} CreateRoleResponse "Created role with ID and permissions count"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/roles [post]
func (rc *RolesScopedBindingsController) CreateRoleCompositeEndUser(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	rc.createRole(c, tenantDB, services.NewRBACService(tenantDB), tenantUUID, false)
}

// ListRolesEndUser godoc
// @Summary List Roles (End User)
// @Description Uses the tenant database. Returns role summary (id, name, description, permissions_count, users_assigned, user_ids/usernames). Optional filters: resource, role_id, user_id. For full permission details, call GET /uflow/user/rbac/roles/{role_id}.
// @Tags RBAC: Roles & Bindings
// @Produce json
// @Security BearerAuth
// @Success 200 {array} RoleListItem "List of roles with assignment counts"
// @Failure 401 {object} map[string]string
// @Router /uflow/user/rbac/roles [get]
func (rc *RolesScopedBindingsController) ListRolesEndUser(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	rc.listRoles(c, tenantDB, tenantUUID)
}

// UpdateRoleCompositeEndUser godoc
// @Summary Update Role (End User)
// @Description Uses the tenant database. Transaction: update role then replace role_permissions.
// @Tags RBAC: Roles & Bindings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param role_id path string true "Role ID (UUID)"
// @Param input body CreateRoleRequest true "Role update payload"
// @Success 200 {object} CreateRoleResponse "Updated role with permissions count"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/roles/{role_id} [put]
func (rc *RolesScopedBindingsController) UpdateRoleCompositeEndUser(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	rc.updateRole(c, tenantDB, tenantUUID)
}

// DeleteRoleEndUser godoc
// @Summary Delete Role (End User)
// @Description Uses the tenant database. Deletes a role and all its permission mappings.
// @Tags RBAC: Roles & Bindings
// @Produce json
// @Security BearerAuth
// @Param role_id path string true "Role ID (UUID)"
// @Success 200 {object} map[string]string "Role deleted successfully"
// @Failure 400 {object} map[string]string "Invalid role ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Role not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/user/rbac/roles/{role_id} [delete]
func (rc *RolesScopedBindingsController) DeleteRoleEndUser(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	rc.deleteRole(c, tenantDB, tenantUUID)
}

// AssignRoleScopedEndUser godoc
// @Summary Assign Role (End User)
// @Description Uses the tenant database. Inserts into role_bindings; scope can be tenant-wide (null) or scoped (type/id). Conditions JSON stored on binding.
// @Tags RBAC: Roles & Bindings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body BindingRequest true "Role assignment payload (user_id, role_id, optional scope{type,id}, conditions)"
// @Success 200 {object} BindingResponse "Binding ID, status, scope description, role name"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /uflow/user/rbac/bindings [post]
func (rc *RolesScopedBindingsController) AssignRoleScopedEndUser(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	rc.assignRoleScoped(c, tenantDB, services.NewRBACService(tenantDB), tenantUUID)
}

// ListRoleBindingsEndUser godoc
// @Summary List Role Bindings (End User)
// @Description Uses the tenant database. Returns all role bindings with user/role details. Optional filters: user_id, role_id, scope_type.
// @Tags RBAC: Roles & Bindings
// @Produce json
// @Security BearerAuth
// @Param user_id query string false "Filter by user_id"
// @Param role_id query string false "Filter by role_id"
// @Param scope_type query string false "Filter by scope_type"
// @Success 200 {array} RoleBindingListItem
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/bindings [get]
func (rc *RolesScopedBindingsController) ListRoleBindingsEndUser(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	rc.listRoleBindings(c, tenantDB, tenantUUID)
}

// --- Shared helpers ---

func (rc *RolesScopedBindingsController) createRole(c *gin.Context, db *gorm.DB, rbac *services.RBACService, tenantID uuid.UUID, isAdmin bool) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Use a fresh session to avoid stale transaction states from connection pooling
	// This prevents "transaction is aborted" errors (SQLSTATE 25P02)
	freshDB := db.Session(&gorm.Session{NewDB: true})
	freshRBAC := services.NewRBACService(freshDB)

	permUUIDs, err := shared.ResolvePermissionUUIDs(freshDB, tenantID, req.PermissionIDs, req.PermissionStrings)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := &models.RBACRole{
		ID:          uuid.New(), // Explicitly generate UUID to ensure it's available in response
		TenantID:    &tenantID,
		Name:        req.Name,
		Description: req.Description,
	}

	// Debug: Log role creation details
	log.Printf("[CreateRole] Creating role '%s' with ID: %s, tenant_id: %s", req.Name, role.ID.String(), tenantID.String())

	if err := freshRBAC.CreateRoleComposite(role, permUUIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role: " + err.Error()})
		return
	}

	// Debug: Verify role was created with correct tenant_id
	if role.TenantID != nil {
		log.Printf("[CreateRole] Role created successfully with tenant_id: %s", role.TenantID.String())
	} else {
		log.Printf("[CreateRole] WARNING: Role created but tenant_id is NULL!")
	}

	// Audit log: Role created
	middlewares.Audit(c, "rbac_role", role.ID.String(), "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"name":              role.Name,
			"description":       role.Description,
			"permissions_count": len(permUUIDs),
		},
	})

	resp := CreateRoleResponse{
		ID:               role.ID.String(),
		Name:             role.Name,
		PermissionsCount: len(permUUIDs),
	}
	c.JSON(http.StatusOK, resp)
	_ = isAdmin
}

func (rc *RolesScopedBindingsController) listRoles(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	resourceFilter := c.Query("resource")
	roleFilter := c.Query("role_id")
	userFilter := c.Query("user_id")
	_ = c.Query("group_by") // Reserved for future grouping functionality

	// Use a fresh session to avoid stale transaction states from connection pooling
	freshDB := db.Session(&gorm.Session{NewDB: true})

	var roles []models.RBACRole
	// Debug: Log the query parameters
	log.Printf("[ListRoles] Querying with tenant_id: %s", tenantID.String())

	roleQuery := freshDB.Where("tenant_id = ?", tenantID)
	if roleFilter != "" {
		if _, err := uuid.Parse(roleFilter); err == nil {
			roleQuery = roleQuery.Where("id = ?", roleFilter)
		}
	}
	if err := roleQuery.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list roles: " + err.Error()})
		return
	}

	// Debug: Log the number of roles found
	log.Printf("[ListRoles] Found %d roles for tenant_id: %s", len(roles), tenantID.String())

	type permRow struct {
		RoleID     uuid.UUID
		Resource   string
		Action     string
		Permission string
	}
	var permRows []permRow
	permQuery := freshDB.Table("role_permissions rp").
		Select("rp.role_id, p.resource, p.action, concat(p.resource, ':', p.action) as permission").
		Joins("JOIN permissions p ON rp.permission_id = p.id").
		Where("p.tenant_id = ?", tenantID)
	if resourceFilter != "" {
		permQuery = permQuery.Where("p.resource = ?", resourceFilter)
	}
	_ = permQuery.Find(&permRows)

	permsByRole := map[uuid.UUID][]string{}
	for _, row := range permRows {
		permsByRole[row.RoleID] = append(permsByRole[row.RoleID], row.Permission)
	}

	type countRow struct {
		RoleID   uuid.UUID
		Count    int64
		UserID   *uuid.UUID
		Username *string
	}
	var counts []countRow
	// Get UNIQUE users per role (not total bindings)
	// A user may have multiple bindings for the same role with different scopes
	bindingQuery := freshDB.Table("role_bindings").
		Select("role_bindings.role_id, COUNT(DISTINCT role_bindings.user_id) AS count, role_bindings.user_id, COALESCE(users.username, '') AS username").
		Joins("LEFT JOIN users ON role_bindings.user_id = users.id").
		Where("role_bindings.tenant_id = ? AND role_bindings.user_id IS NOT NULL", tenantID)
	if roleFilter != "" {
		if _, err := uuid.Parse(roleFilter); err == nil {
			bindingQuery = bindingQuery.Where("role_bindings.role_id = ?", roleFilter)
		}
	}
	if userFilter != "" {
		if _, err := uuid.Parse(userFilter); err == nil {
			bindingQuery = bindingQuery.Where("role_bindings.user_id = ?", userFilter)
		}
	}
	// Group by role_id only to get unique user count, then get distinct users
	bindingQuery = bindingQuery.Group("role_bindings.role_id, role_bindings.user_id, users.username")
	_ = bindingQuery.Scan(&counts)

	countByRole := map[uuid.UUID]int64{}
	usersByRole := map[uuid.UUID][]string{}
	usernamesByRole := map[uuid.UUID][]string{}
	seenUsersByRole := map[uuid.UUID]map[string]bool{} // Track seen users to avoid duplicates
	for _, row := range counts {
		if row.UserID != nil {
			userIDStr := row.UserID.String()
			if seenUsersByRole[row.RoleID] == nil {
				seenUsersByRole[row.RoleID] = make(map[string]bool)
			}
			// Only add user if not already seen for this role
			if !seenUsersByRole[row.RoleID][userIDStr] {
				seenUsersByRole[row.RoleID][userIDStr] = true
				countByRole[row.RoleID]++
				usersByRole[row.RoleID] = append(usersByRole[row.RoleID], userIDStr)
				if row.Username != nil {
					usernamesByRole[row.RoleID] = append(usernamesByRole[row.RoleID], *row.Username)
				}
			}
		}
	}

	var resp []RoleListItem
	for _, role := range roles {
		resp = append(resp, RoleListItem{
			ID:               role.ID.String(),
			Name:             role.Name,
			Description:      role.Description,
			PermissionsCount: len(permsByRole[role.ID]),
			UsersAssigned:    countByRole[role.ID],
			UserIDs:          usersByRole[role.ID],
			Usernames:        usernamesByRole[role.ID],
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (rc *RolesScopedBindingsController) updateRole(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Role ID format"})
		return
	}

	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Use a fresh session to avoid stale transaction states from connection pooling
	freshDB := db.Session(&gorm.Session{NewDB: true})

	// Capture old values for audit log
	var oldRole models.RBACRole
	var oldPermCount int64
	freshDB.Where("id = ? AND tenant_id = ?", roleID, tenantID).First(&oldRole)
	freshDB.Model(&models.RolePermission{}).Where("role_id = ?", roleID).Count(&oldPermCount)

	permUUIDs, err := shared.ResolvePermissionUUIDs(freshDB, tenantID, req.PermissionIDs, req.PermissionStrings)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := freshDB.Transaction(func(tx *gorm.DB) error {
		// Ensure role belongs to tenant
		res := tx.Where("id = ? AND tenant_id = ?", roleID, tenantID).Model(&models.RBACRole{}).
			Updates(map[string]interface{}{
				"name":        req.Name,
				"description": req.Description,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("role not found")
		}

		// Replace role_permissions
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		var rolePerms []models.RolePermission
		for _, pid := range permUUIDs {
			rolePerms = append(rolePerms, models.RolePermission{
				RoleID:       roleID,
				PermissionID: pid,
			})
		}
		if len(rolePerms) > 0 {
			if err := tx.Create(&rolePerms).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if err.Error() == "role not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role: " + err.Error()})
		return
	}

	// Audit log: Role updated
	middlewares.Audit(c, "rbac_role", roleID.String(), "update", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"name":              oldRole.Name,
			"description":       oldRole.Description,
			"permissions_count": oldPermCount,
		},
		After: map[string]interface{}{
			"name":              req.Name,
			"description":       req.Description,
			"permissions_count": len(permUUIDs),
		},
	})

	resp := CreateRoleResponse{
		ID:               roleID.String(),
		Name:             req.Name,
		PermissionsCount: len(permUUIDs),
	}
	c.JSON(http.StatusOK, resp)
}

func (rc *RolesScopedBindingsController) assignRoleScoped(c *gin.Context, db *gorm.DB, rbac *services.RBACService, tenantID uuid.UUID) {
	var req BindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID format"})
		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Role ID format"})
		return
	}

	// Use a fresh session to avoid stale transaction states from connection pooling
	// This prevents "transaction is aborted" errors (SQLSTATE 25P02)
	freshDB := db.Session(&gorm.Session{NewDB: true})

	// Create a new RBAC service with the fresh session
	freshRBAC := services.NewRBACService(freshDB)

	// Validate role exists for tenant and capture role name
	var role models.RBACRole
	if err := freshDB.Where("id = ? AND tenant_id = ?", roleID, tenantID).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found for tenant"})
		return
	}
	var user models.User
	if err := freshDB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var scopeType *string
	var scopeID *uuid.UUID
	scopeDesc := "Tenant-Wide"
	if req.Scope != nil {
		st := req.Scope.Type
		// Handle wildcard scope: "*" or empty means tenant-wide (no specific scope)
		if st != "" && st != "*" {
			scopeType = &st
		}
		// Handle wildcard scope ID: "*" means tenant-wide (null scope_id)
		if req.Scope.ID != "" && req.Scope.ID != "*" {
			sid, err := uuid.Parse(req.Scope.ID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Scope ID format (must be UUID or '*' for tenant-wide)"})
				return
			}
			scopeID = &sid
			scopeDesc = fmt.Sprintf("%s: %s", st, req.Scope.ID)
		} else {
			// Wildcard scope - tenant-wide
			scopeDesc = "Tenant-Wide (wildcard)"
		}
	}

	conditionsJSON, _ := services.MapToJSON(req.Conditions)

	binding := &models.RoleBinding{
		ID:         uuid.New(), // Generate UUID in Go, don't rely on DB default
		TenantID:   &tenantID,
		UserID:     &userID,
		Username:   shared.DerefString(user.Username),
		RoleID:     roleID,
		RoleName:   role.Name,
		ScopeType:  scopeType,
		ScopeID:    scopeID,
		Conditions: []byte(conditionsJSON),
	}

	if err := freshRBAC.AssignRoleScoped(binding); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign role: " + err.Error()})
		return
	}

	// Audit log: Role binding created
	middlewares.Audit(c, "role_binding", binding.ID.String(), "assign", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":          req.UserID,
			"username":         user.Username,
			"role_id":          req.RoleID,
			"role_name":        role.Name,
			"scope":            scopeDesc,
			"scope_type":       scopeType,
			"scope_id":         scopeID,
			"conditions_count": len(req.Conditions),
		},
	})

	resp := BindingResponse{
		ID:               binding.ID.String(),
		Status:           "active",
		ScopeDescription: scopeDesc,
		RoleName:         role.Name,
	}
	c.JSON(http.StatusOK, resp)
}

// listRoleBindings retrieves role bindings with optional filters
func (rc *RolesScopedBindingsController) listRoleBindings(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	userIDFilter := c.Query("user_id")
	roleIDFilter := c.Query("role_id")
	scopeTypeFilter := c.Query("scope_type")

	// Use a fresh session to avoid stale transaction states from connection pooling
	freshDB := db.Session(&gorm.Session{NewDB: true})

	query := freshDB.Table("role_bindings").
		Select(`
			role_bindings.id,
			role_bindings.user_id,
			COALESCE(users.username, '') AS username,
			role_bindings.service_account_id,
			role_bindings.role_id,
			COALESCE(role_bindings.role_name, roles.name, '') AS role_name,
			role_bindings.scope_type,
			role_bindings.scope_id,
			role_bindings.conditions,
			role_bindings.created_at,
			role_bindings.expires_at,
			COALESCE(users.email, '') AS email
		`).
		Joins("LEFT JOIN users ON role_bindings.user_id = users.id").
		Joins("LEFT JOIN roles ON role_bindings.role_id = roles.id").
		Where("role_bindings.tenant_id = ?", tenantID)

	if userIDFilter != "" {
		if uid, err := uuid.Parse(userIDFilter); err == nil {
			query = query.Where("role_bindings.user_id = ?", uid)
		}
	}
	if roleIDFilter != "" {
		if rid, err := uuid.Parse(roleIDFilter); err == nil {
			query = query.Where("role_bindings.role_id = ?", rid)
		}
	}
	if scopeTypeFilter != "" {
		query = query.Where("role_bindings.scope_type = ?", scopeTypeFilter)
	}

	type bindingRow struct {
		ID               uuid.UUID       `gorm:"column:id"`
		UserID           *uuid.UUID      `gorm:"column:user_id"`
		Username         string          `gorm:"column:username"`
		ServiceAccountID *uuid.UUID      `gorm:"column:service_account_id"`
		RoleID           uuid.UUID       `gorm:"column:role_id"`
		RoleName         string          `gorm:"column:role_name"`
		ScopeType        *string         `gorm:"column:scope_type"`
		ScopeID          *uuid.UUID      `gorm:"column:scope_id"`
		Conditions       json.RawMessage `gorm:"column:conditions"`
		CreatedAt        time.Time       `gorm:"column:created_at"`
		ExpiresAt        *time.Time      `gorm:"column:expires_at"`
		Email            string          `gorm:"column:email"`
	}

	var rows []bindingRow
	if err := query.Order("role_bindings.created_at DESC").Find(&rows).Error; err != nil {
		// Fallback for tenants without denormalized columns (e.g., role_bindings.username)
		if strings.Contains(err.Error(), "role_bindings.username") || strings.Contains(err.Error(), "role_bindings.role_name") {
			legacyQuery := freshDB.Table("role_bindings").
				Select(`
					role_bindings.id,
					role_bindings.user_id,
					role_bindings.service_account_id,
					role_bindings.role_id,
					COALESCE(roles.name, '') AS role_name,
					role_bindings.scope_type,
					role_bindings.scope_id,
					role_bindings.conditions,
					role_bindings.created_at,
					role_bindings.expires_at,
					COALESCE(users.username, '') AS username,
					COALESCE(users.email, '') AS email
				`).
				Joins("LEFT JOIN users ON role_bindings.user_id = users.id").
				Joins("LEFT JOIN roles ON role_bindings.role_id = roles.id").
				Where("role_bindings.tenant_id = ?", tenantID)

			if userIDFilter != "" {
				if uid, parseErr := uuid.Parse(userIDFilter); parseErr == nil {
					legacyQuery = legacyQuery.Where("role_bindings.user_id = ?", uid)
				}
			}
			if roleIDFilter != "" {
				if rid, parseErr := uuid.Parse(roleIDFilter); parseErr == nil {
					legacyQuery = legacyQuery.Where("role_bindings.role_id = ?", rid)
				}
			}
			if scopeTypeFilter != "" {
				legacyQuery = legacyQuery.Where("role_bindings.scope_type = ?", scopeTypeFilter)
			}

			if legacyErr := legacyQuery.Order("role_bindings.created_at DESC").Find(&rows).Error; legacyErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list role bindings: " + legacyErr.Error()})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list role bindings: " + err.Error()})
			return
		}
	}

	result := make([]RoleBindingListItem, 0, len(rows))
	for _, r := range rows {
		item := RoleBindingListItem{
			ID:        r.ID.String(),
			Username:  r.Username,
			Email:     r.Email,
			RoleID:    r.RoleID.String(),
			RoleName:  r.RoleName,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		}
		if r.UserID != nil {
			item.UserID = r.UserID.String()
		}
		if r.ServiceAccountID != nil {
			item.ServiceAccountID = r.ServiceAccountID.String()
		}
		if r.ScopeType != nil {
			item.ScopeType = *r.ScopeType
		}
		if r.ScopeID != nil {
			item.ScopeID = r.ScopeID.String()
		}
		if r.ExpiresAt != nil {
			item.ExpiresAt = r.ExpiresAt.Format(time.RFC3339)
		}
		if len(r.Conditions) > 0 && string(r.Conditions) != "{}" && string(r.Conditions) != "null" {
			var cond map[string]interface{}
			if json.Unmarshal(r.Conditions, &cond) == nil {
				item.Conditions = cond
			}
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}
func (rc *RolesScopedBindingsController) deleteRole(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		log.Printf("ERROR: Delete role - Invalid role ID format: %s", roleIDStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
		return
	}

	log.Printf("INFO: Deleting role %s for tenant %s", roleID, tenantID)

	// Use a fresh session to avoid stale transaction states from connection pooling
	freshDB := db.Session(&gorm.Session{NewDB: true})

	// Verify role exists and belongs to tenant
	var role models.RBACRole
	if err := freshDB.Where("id = ? AND tenant_id = ?", roleID, tenantID).First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("WARN: Role %s not found for tenant %s", roleID, tenantID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		log.Printf("ERROR: Failed to fetch role %s: %v", roleID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role: " + err.Error()})
		return
	}

	log.Printf("INFO: Found role %s (name: %s), proceeding to delete with role_permissions and role_bindings", roleID, role.Name)

	// Delete role (cascades to role_permissions due to FK constraint)
	if err := freshDB.Transaction(func(tx *gorm.DB) error {
		// Delete role permissions first (in case cascade doesn't work)
		result := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{})
		if result.Error != nil {
			log.Printf("ERROR: Failed to delete role_permissions for role %s: %v", roleID, result.Error)
			return result.Error
		}
		log.Printf("INFO: Deleted %d role_permissions for role %s", result.RowsAffected, roleID)

		// Delete role bindings for this role
		result = tx.Where("role_id = ?", roleID).Delete(&models.RoleBinding{})
		if result.Error != nil {
			log.Printf("ERROR: Failed to delete role_bindings for role %s: %v", roleID, result.Error)
			return result.Error
		}
		log.Printf("INFO: Deleted %d role_bindings for role %s", result.RowsAffected, roleID)

		// Delete the role
		if err := tx.Delete(&role).Error; err != nil {
			log.Printf("ERROR: Failed to delete role %s: %v", roleID, err)
			return err
		}
		log.Printf("INFO: Successfully deleted role %s", roleID)
		return nil
	}); err != nil {
		log.Printf("ERROR: Transaction failed for deleting role %s: %v", roleID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role: " + err.Error()})
		return
	}

	// Audit log: Role deleted
	middlewares.Audit(c, "rbac_role", roleID.String(), "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"name":        role.Name,
			"description": role.Description,
		},
		After: nil,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Role deleted successfully",
		"id":      roleID.String(),
	})
}
