package admin

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	amMiddlewares "github.com/authsec-ai/auth-manager/pkg/middlewares"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PermissionController handles both admin (primary DB) and end-user (tenant DB) permission routes.
type PermissionController struct{}

func NewPermissionController() *PermissionController {
	return &PermissionController{}
}

// PermissionRequest represents the payload for creating a permission
type PermissionRequest struct {
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// PermissionResponse represents the response for creating a permission
type PermissionResponse struct {
	ID         string `json:"id"`
	FullString string `json:"full_string"`
}

// PermissionWithRoleCount extends permission with assigned role count.
type PermissionWithRoleCount struct {
	ID            uuid.UUID `json:"id"`
	Resource      string    `json:"resource"`
	Action        string    `json:"action"`
	FullString    string    `json:"full_permission_string"`
	Description   string    `json:"description"`
	RolesAssigned int64     `json:"roles_assigned"`
}

// EndUserPermissionResponse is a simplified response for end-user permissions without ID.
type EndUserPermissionResponse struct {
	Resource             string   `json:"resource"`
	Action               string   `json:"action"`
	FullPermissionString string   `json:"full_permission_string"`
	Description          string   `json:"description"`
	RoleNames            []string `json:"role_names"`
}

// AdminShowResourcesResponse represents a unique list of resources.
type AdminShowResourcesResponse struct {
	Resources []string `json:"resources"`
}

// EndUserShowResourcesResponse mirrors AdminShowResourcesResponse for swagger clarity.
type EndUserShowResourcesResponse struct {
	Resources []string `json:"resources"`
}

// RegisterAtomicPermission godoc
// @Summary Register Atomic Permission (Admin)
// @Description Uses the primary admin database. Insert into 'permissions'. Failure if resource+action pair exists.
// @Tags RBAC: Permissions
// @Accept json
// @Produce json
// @Param input body PermissionRequest true "Permission creation payload"
// @Success 200 {object} PermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/permissions [post]
func (pc *PermissionController) RegisterAtomicPermission(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.Resource == "" || req.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Resource and Action are required"})
		return
	}

	tenantIDStr, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in context"})
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tenant ID format"})
		return
	}

	if err := pc.registerPermission(c, config.DB, tenantID, req); err != nil {
		return
	}
}

// RegisterAtomicPermissionEndUser godoc
// @Summary Register Atomic Permission (End User)
// @Description Uses the tenant database. Insert into 'permissions'. Failure if resource+action pair exists.
// @Tags RBAC: Permissions
// @Accept json
// @Produce json
// @Param input body PermissionRequest true "Permission creation payload"
// @Success 200 {object} PermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/permissions [post]
func (pc *PermissionController) RegisterAtomicPermissionEndUser(c *gin.Context) {
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

	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	if err := pc.registerPermission(c, tenantDB, tenantUUID, req); err != nil {
		return
	}
}

func (pc *PermissionController) registerPermission(c *gin.Context, db *gorm.DB, tenantID uuid.UUID, req PermissionRequest) error {
	perm := &models.RBACPermission{
		TenantID:    &tenantID,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
	}

	// Debug: Log permission creation details
	log.Printf("[RegisterPermission] Creating permission '%s:%s' for tenant_id: %s", req.Resource, req.Action, tenantID.String())

	if err := services.NewRBACService(db).RegisterAtomicPermission(perm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register permission: " + err.Error()})
		return err
	}

	// Debug: Verify permission was created with correct tenant_id
	if perm.TenantID != nil {
		log.Printf("[RegisterPermission] Permission created successfully with ID: %s, tenant_id: %s", perm.ID.String(), perm.TenantID.String())
	} else {
		log.Printf("[RegisterPermission] WARNING: Permission created with ID: %s but tenant_id is NULL!", perm.ID.String())
	}

	// Audit log: Permission registered
	middlewares.Audit(c, "rbac_permission", perm.ID.String(), "register", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"resource":    req.Resource,
			"action":      req.Action,
			"description": req.Description,
			"full_string": fmt.Sprintf("%s:%s", req.Resource, req.Action),
		},
	})

	resp := PermissionResponse{
		ID:         perm.ID.String(),
		FullString: fmt.Sprintf("%s.%s", req.Resource, req.Action),
	}

	c.JSON(http.StatusOK, resp)
	return nil
}

// DeletePermission godoc
// @Summary Delete Permission (Admin)
// @Description Deletes a permission from the primary database.
// @Tags RBAC: Permissions
// @Produce json
// @Param id path string true "Permission ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/permissions/{id} [delete]
func (pc *PermissionController) DeletePermission(c *gin.Context) {
	tenantIDStr, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in context"})
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tenant ID format"})
		return
	}

	permIDStr := c.Param("id")
	permID, err := uuid.Parse(permIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Permission ID format"})
		return
	}

	// Verify permission belongs to tenant
	var perm models.RBACPermission
	if err := config.DB.Where("id = ? AND tenant_id = ?", permID, tenantID).First(&perm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve permission: " + err.Error()})
		return
	}

	if err := services.NewRBACService(config.DB).DeletePermission(permID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission: " + err.Error()})
		return
	}

	// Audit log: Permission deleted
	middlewares.Audit(c, "rbac_permission", permID.String(), "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"resource":    perm.Resource,
			"action":      perm.Action,
			"description": perm.Description,
			"full_string": fmt.Sprintf("%s:%s", perm.Resource, perm.Action),
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

// DeletePermissionEndUser godoc
// @Summary Delete Permission (End User)
// @Description Deletes a permission from the tenant database.
// @Tags RBAC: Permissions
// @Produce json
// @Param id path string true "Permission ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/permissions/{id} [delete]
func (pc *PermissionController) DeletePermissionEndUser(c *gin.Context) {
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

	permIDStr := c.Param("id")
	permID, err := uuid.Parse(permIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Permission ID format"})
		return
	}

	// Verify permission belongs to tenant and capture for audit log
	var perm models.RBACPermission
	if err := tenantDB.Where("id = ? AND tenant_id = ?", permID, tenantID).First(&perm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve permission: " + err.Error()})
		return
	}

	if err := services.NewRBACService(tenantDB).DeletePermission(permID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission: " + err.Error()})
		return
	}

	// Audit log: Permission deleted (end-user)
	middlewares.Audit(c, "rbac_permission", permID.String(), "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"resource":    perm.Resource,
			"action":      perm.Action,
			"description": perm.Description,
			"full_string": fmt.Sprintf("%s:%s", perm.Resource, perm.Action),
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

// DeletePermissionByBody godoc
// @Summary Delete Permission by Body (Admin)
// @Description Deletes a permission from the primary database using resource and action in body.
// @Tags RBAC: Permissions
// @Accept json
// @Produce json
// @Param input body PermissionRequest true "Permission deletion payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/permissions [delete]
func (pc *PermissionController) DeletePermissionByBody(c *gin.Context) {
	tenantIDStr, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in context"})
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tenant ID format"})
		return
	}

	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.Resource == "" || req.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Resource and Action are required"})
		return
	}

	// Find permission
	var perm models.RBACPermission
	if err := config.DB.Where("tenant_id = ? AND resource = ? AND action = ?", tenantID, req.Resource, req.Action).First(&perm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve permission: " + err.Error()})
		return
	}

	if err := services.NewRBACService(config.DB).DeletePermission(perm.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission: " + err.Error()})
		return
	}

	// Audit log: Permission deleted by body
	middlewares.Audit(c, "rbac_permission", perm.ID.String(), "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"resource":    perm.Resource,
			"action":      perm.Action,
			"description": perm.Description,
			"full_string": fmt.Sprintf("%s:%s", perm.Resource, perm.Action),
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

// DeletePermissionEndUserByBody godoc
// @Summary Delete Permission by Body (End User)
// @Description Deletes a permission from the tenant database using resource and action in body.
// @Tags RBAC: Permissions
// @Accept json
// @Produce json
// @Param input body PermissionRequest true "Permission deletion payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/permissions [delete]
func (pc *PermissionController) DeletePermissionEndUserByBody(c *gin.Context) {
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

	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.Resource == "" || req.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Resource and Action are required"})
		return
	}

	// Find permission
	var perm models.RBACPermission
	if err := tenantDB.Where("tenant_id = ? AND resource = ? AND action = ?", tenantID, req.Resource, req.Action).First(&perm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve permission: " + err.Error()})
		return
	}

	if err := services.NewRBACService(tenantDB).DeletePermission(perm.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission: " + err.Error()})
		return
	}

	// Audit log: Permission deleted by body (end-user)
	middlewares.Audit(c, "rbac_permission", perm.ID.String(), "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"resource":    perm.Resource,
			"action":      perm.Action,
			"description": perm.Description,
			"full_string": fmt.Sprintf("%s:%s", perm.Resource, perm.Action),
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

// ListPermissions godoc
// @Summary List Permissions (Admin)
// @Description Lists permissions from the primary database.
// @Tags RBAC: Permissions
// @Produce json
// @Security BearerAuth
// @Param resource query string false "Filter by resource"
// @Success 200 {array} PermissionWithRoleCount
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/permissions [get]
func (pc *PermissionController) ListPermissions(c *gin.Context) {
	tenantIDStr, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in context"})
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tenant ID format"})
		return
	}

	resource := c.Query("resource")
	pc.listPermissions(c, config.DB, tenantID, resource)
}

// ListPermissionsEndUser godoc
// @Summary List Permissions (End User)
// @Description Lists permissions from the tenant database.
// @Tags RBAC: Permissions
// @Produce json
// @Security BearerAuth
// @Param resource query string false "Filter by resource"
// @Success 200 {array} EndUserPermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/permissions [get]
func (pc *PermissionController) ListPermissionsEndUser(c *gin.Context) {
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
	resource := c.Query("resource")
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	pc.listPermissions(c, tenantDB, tenantUUID, resource)
}

func (pc *PermissionController) listPermissions(c *gin.Context, db *gorm.DB, tenantID uuid.UUID, resource string) {
	// Debug: Log query parameters
	log.Printf("[ListPermissions] Querying with tenant_id: %s, resource: %s", tenantID.String(), resource)

	var perms []models.RBACPermission
	query := db.Where("tenant_id = ?", tenantID)
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if err := query.Find(&perms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions: " + err.Error()})
		return
	}

	// Debug: Log number of permissions found
	log.Printf("[ListPermissions] Found %d permissions for tenant_id: %s", len(perms), tenantID.String())

	type countRow struct {
		PermissionID uuid.UUID
		Count        int64
	}
	var counts []countRow
	_ = db.Table("role_permissions rp").
		Select("rp.permission_id, count(*) as count").
		Joins("JOIN permissions p ON rp.permission_id = p.id").
		Where("p.tenant_id = ?", tenantID).
		Group("rp.permission_id").
		Scan(&counts)

	countMap := map[uuid.UUID]int64{}
	for _, row := range counts {
		countMap[row.PermissionID] = row.Count
	}

	resp := make([]PermissionWithRoleCount, 0, len(perms))
	for _, p := range perms {
		resp = append(resp, PermissionWithRoleCount{
			ID:            p.ID,
			Resource:      p.Resource,
			Action:        p.Action,
			FullString:    fmt.Sprintf("%s:%s", p.Resource, p.Action),
			Description:   p.Description,
			RolesAssigned: countMap[p.ID],
		})
	}

	c.JSON(http.StatusOK, resp)
}

// listPermissionsEndUser returns permissions with role names for end-user context
func (pc *PermissionController) listPermissionsEndUser(c *gin.Context, db *gorm.DB, tenantID uuid.UUID, resource string) {
	var perms []models.RBACPermission
	query := db.Where("tenant_id = ?", tenantID)
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if err := query.Find(&perms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions: " + err.Error()})
		return
	}

	// Get role names for each permission
	type roleRow struct {
		PermissionID uuid.UUID
		RoleName     string
	}
	var roleRows []roleRow
	_ = db.Table("role_permissions rp").
		Select("rp.permission_id, r.name as role_name").
		Joins("JOIN roles r ON rp.role_id = r.id").
		Joins("JOIN permissions p ON rp.permission_id = p.id").
		Where("p.tenant_id = ?", tenantID).
		Scan(&roleRows)

	// Build map of permission_id -> role names
	roleMap := map[uuid.UUID][]string{}
	for _, row := range roleRows {
		roleMap[row.PermissionID] = append(roleMap[row.PermissionID], row.RoleName)
	}

	resp := make([]EndUserPermissionResponse, 0, len(perms))
	for _, p := range perms {
		fullPermStr := fmt.Sprintf("%s:%s", p.Resource, p.Action)
		roles := roleMap[p.ID]
		if roles == nil {
			roles = []string{}
		}
		resp = append(resp, EndUserPermissionResponse{
			Resource:             p.Resource,
			Action:               p.Action,
			FullPermissionString: fullPermStr,
			Description:          p.Description,
			RoleNames:            roles,
		})
	}

	c.JSON(http.StatusOK, resp)
}

// ShowResources godoc
// @Summary List Resources (Admin)
// @Description Returns unique resource names from the permissions table for the authenticated tenant.
// @Tags RBAC: Permissions
// @Produce json
// @Security BearerAuth
// @Success 200 {object} AdminShowResourcesResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/permissions/resources [get]
func (pc *PermissionController) ShowResources(c *gin.Context) {
	tenantIDStr, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in context"})
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tenant ID format"})
		return
	}

	pc.showResources(c, config.DB, tenantID, true)
}

// ShowResourcesEndUser godoc
// @Summary List Resources (End User)
// @Description Returns unique resource names from the permissions table for the authenticated tenant.
// @Tags RBAC: Permissions
// @Produce json
// @Security BearerAuth
// @Success 200 {object} EndUserShowResourcesResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/rbac/permissions/resources [get]
func (pc *PermissionController) ShowResourcesEndUser(c *gin.Context) {
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
	pc.showResources(c, tenantDB, tenantUUID, false)
}

func (pc *PermissionController) showResources(c *gin.Context, db *gorm.DB, tenantID uuid.UUID, isAdmin bool) {
	var resources []string
	if err := db.Table("permissions").Where("tenant_id = ?", tenantID).Distinct().Pluck("resource", &resources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch resources from permissions: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"resources": resources})
	_ = isAdmin
}

// End-user-only helper endpoints reused from former enduser permission controller

// GetMyPermissions godoc
// @Summary Get My Permissions (End User)
// @Description Get all permissions assigned to the current authenticated user (tenant DB)
// @Tags RBAC: Permissions
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Permission
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/permissions [get]
func (pc *PermissionController) GetMyPermissions(c *gin.Context) {
	userID, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	permissions, err := GetUserPermissionsInTenant(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user permissions: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

// GetMyEffectivePermissions godoc
// @Summary Get My Effective Permissions (End User)
// @Description Get all effective permissions (including group inheritance) for the current user (tenant DB)
// @Tags RBAC: Permissions
// @Produce json
// @Security BearerAuth
// @Success 200 {object} EffectivePermissionsResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/permissions/effective [get]
func (pc *PermissionController) GetMyEffectivePermissions(c *gin.Context) {
	userID, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Get direct permissions
	directPermissions, err := GetUserPermissionsInTenant(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get direct permissions: " + err.Error()})
		return
	}

	// Get role-based permissions
	rolePermissions, err := GetUserRolePermissionsInTenant(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role permissions: " + err.Error()})
		return
	}

	// Combine and deduplicate permissions
	allPermissions := append(directPermissions, rolePermissions...)

	// Create response
	response := EffectivePermissionsResponse{
		DirectPermissions: directPermissions,
		RolePermissions:   rolePermissions,
		AllPermissions:    allPermissions,
	}

	c.JSON(http.StatusOK, response)
}

// CheckPermission godoc
// @Summary Check Permission (End User)
// @Description Check if the current user has a specific permission (tenant DB)
// @Tags RBAC: Permissions
// @Produce json
// @Security BearerAuth
// @Param resource query string true "Resource name"
// @Param scope query string true "Scope name"
// @Success 200 {object} PermissionCheckResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/permissions/check [get]
func (pc *PermissionController) CheckPermission(c *gin.Context) {
	// Get user ID from context
	userID, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	resource := c.Query("resource")
	scope := c.Query("scope")

	if resource == "" || scope == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Resource and scope parameters are required"})
		return
	}

	hasPermission, err := CheckUserPermission(userID, tenantID, resource, scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permission: " + err.Error()})
		return
	}

	response := PermissionCheckResponse{
		UserID:        userID,
		TenantID:      tenantID,
		Resource:      resource,
		Scope:         scope,
		HasPermission: hasPermission,
	}

	c.JSON(http.StatusOK, response)
}

// End-user request/response structs reused from previous controller
type EffectivePermissionsResponse struct {
	DirectPermissions []models.Permission `json:"direct_permissions"`
	RolePermissions   []models.Permission `json:"role_permissions"`
	AllPermissions    []models.Permission `json:"all_permissions"`
}

type PermissionCheckResponse struct {
	UserID        string `json:"user_id"`
	TenantID      string `json:"tenant_id"`
	Resource      string `json:"resource"`
	Scope         string `json:"scope"`
	HasPermission bool   `json:"has_permission"`
}

// Database helper functions (tenant DB)
// Uses role_bindings for role assignments (user_roles is deprecated)
func GetUserPermissionsInTenant(userID, tenantID string) ([]models.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.tenant_id, p.resource, p.action, p.description, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN role_bindings rb ON rp.role_id = rb.role_id
		WHERE rb.user_id = $1 AND rb.tenant_id = $2
		ORDER BY p.created_at DESC
	`
	rows, err := config.Database.DB.Query(query, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(&permission.ID, &permission.TenantID, &permission.Resource, &permission.Action,
			&permission.Description, &permission.CreatedAt, &permission.UpdatedAt)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	return permissions, nil
}

func GetUserRolePermissionsInTenant(userID, tenantID string) ([]models.Permission, error) {
	// Uses role_bindings for role assignments (user_roles is deprecated)
	query := `
		SELECT DISTINCT p.id, p.tenant_id, p.resource, p.action, p.description, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN role_bindings rb ON rp.role_id = rb.role_id
		WHERE rb.user_id = $1 AND rb.tenant_id = $2
		ORDER BY p.created_at DESC
	`
	rows, err := config.Database.DB.Query(query, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(&permission.ID, &permission.TenantID, &permission.Resource, &permission.Action,
			&permission.Description, &permission.CreatedAt, &permission.UpdatedAt)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	return permissions, nil
}

func CheckUserPermission(userID, tenantID, resource, scope string) (bool, error) {
	// Uses role_bindings for role assignments (user_roles is deprecated)
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM permissions p
			INNER JOIN role_permissions rp ON p.id = rp.permission_id
			INNER JOIN role_bindings rb ON rp.role_id = rb.role_id
			WHERE rb.user_id = $1
			AND rb.tenant_id = $2
			AND p.resource = $3
			AND p.action = $4
		)
	`
	var exists bool
	err := config.Database.DB.QueryRow(query, userID, tenantID, resource, scope).Scan(&exists)
	return exists, err
}
