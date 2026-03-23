package admin

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIScopesController manages API Scopes (OAuth scope-to-permission mapping)
// for both admin (primary DB) and end-user (tenant DB) contexts.
// API Scopes are external contracts that allow third-party applications
// to request specific permissions via OAuth.
type APIScopesController struct{}

func NewAPIScopesController() *APIScopesController {
	return &APIScopesController{}
}

// --- Admin endpoints (primary DB) ---

// CreateAPIScopeAdmin godoc
// @Summary Create API Scope (Admin)
// @Description Uses the primary admin database. Creates an OAuth scope contract and maps it to internal permissions. Transaction: 1) insert into api_scopes, 2) insert into api_scope_permissions.
// @Tags OAuth: API Scopes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body models.CreateAPIScopeRequest true "API Scope creation payload"
// @Success 200 {object} models.APIScopeResponse "Created scope with ID and permissions count"
// @Failure 400 {object} map[string]string "Invalid request or permission validation failed"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Scope name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/api_scopes [post]
func (sc *APIScopesController) CreateAPIScopeAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	sc.createAPIScope(c, config.DB, *tenantID)
}

// ListAPIScopesAdmin godoc
// @Summary List API Scopes (Admin)
// @Description Uses the primary admin database. Returns all API scopes for the tenant with permission counts.
// @Tags OAuth: API Scopes
// @Produce json
// @Security BearerAuth
// @Param name query string false "Filter by scope name (partial match)"
// @Success 200 {array} models.APIScopeListItem "List of API scopes"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/api_scopes [get]
func (sc *APIScopesController) ListAPIScopesAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	sc.listAPIScopes(c, config.DB, *tenantID)
}

// GetAPIScopeAdmin godoc
// @Summary Get API Scope Details (Admin)
// @Description Uses the primary admin database. Returns a single API scope with all mapped permissions.
// @Tags OAuth: API Scopes
// @Produce json
// @Security BearerAuth
// @Param scope_id path string true "Scope ID (UUID)"
// @Success 200 {object} models.APIScopeResponse "API scope with permission details"
// @Failure 400 {object} map[string]string "Invalid scope ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Scope not found"
// @Router /uflow/admin/api_scopes/{scope_id} [get]
func (sc *APIScopesController) GetAPIScopeAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	sc.getAPIScope(c, config.DB, *tenantID)
}

// UpdateAPIScopeAdmin godoc
// @Summary Update API Scope (Admin)
// @Description Uses the primary admin database. Updates an API scope and replaces permission mappings. Transaction: 1) update api_scopes, 2) delete + insert api_scope_permissions.
// @Tags OAuth: API Scopes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param scope_id path string true "Scope ID (UUID)"
// @Param input body models.UpdateAPIScopeRequest true "API Scope update payload"
// @Success 200 {object} models.APIScopeResponse "Updated scope with permissions count"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Scope not found"
// @Failure 409 {object} map[string]string "Scope name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/api_scopes/{scope_id} [put]
func (sc *APIScopesController) UpdateAPIScopeAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	sc.updateAPIScope(c, config.DB, *tenantID)
}

// DeleteAPIScopeAdmin godoc
// @Summary Delete API Scope (Admin)
// @Description Uses the primary admin database. Deletes an API scope and all permission mappings (cascade).
// @Tags OAuth: API Scopes
// @Produce json
// @Security BearerAuth
// @Param scope_id path string true "Scope ID (UUID)"
// @Success 200 {object} map[string]string "Deletion confirmation"
// @Failure 400 {object} map[string]string "Invalid scope ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Scope not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/api_scopes/{scope_id} [delete]
func (sc *APIScopesController) DeleteAPIScopeAdmin(c *gin.Context) {
	tenantID, err := shared.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	sc.deleteAPIScope(c, config.DB, *tenantID)
}

// --- End-user endpoints (tenant DB) ---

// CreateAPIScopeEndUser godoc
// @Summary Create API Scope (End User)
// @Description Uses the tenant database. Creates an OAuth scope contract and maps it to internal permissions.
// @Tags OAuth: API Scopes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body models.CreateAPIScopeRequest true "API Scope creation payload"
// @Success 200 {object} models.APIScopeResponse "Created scope with ID and permissions count"
// @Failure 400 {object} map[string]string "Invalid request or permission validation failed"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Scope name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/user/api_scopes [post]
func (sc *APIScopesController) CreateAPIScopeEndUser(c *gin.Context) {
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
	sc.createAPIScope(c, tenantDB, tenantUUID)
}

// ListAPIScopesEndUser godoc
// @Summary List API Scopes (End User)
// @Description Uses the tenant database. Returns all API scopes for the tenant with permission counts.
// @Tags OAuth: API Scopes
// @Produce json
// @Security BearerAuth
// @Param name query string false "Filter by scope name (partial match)"
// @Success 200 {array} models.APIScopeListItem "List of API scopes"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/user/api_scopes [get]
func (sc *APIScopesController) ListAPIScopesEndUser(c *gin.Context) {
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
	sc.listAPIScopes(c, tenantDB, tenantUUID)
}

// GetAPIScopeEndUser godoc
// @Summary Get API Scope Details (End User)
// @Description Uses the tenant database. Returns a single API scope with all mapped permissions.
// @Tags OAuth: API Scopes
// @Produce json
// @Security BearerAuth
// @Param scope_id path string true "Scope ID (UUID)"
// @Success 200 {object} models.APIScopeResponse "API scope with permission details"
// @Failure 400 {object} map[string]string "Invalid scope ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Scope not found"
// @Router /uflow/user/api_scopes/{scope_id} [get]
func (sc *APIScopesController) GetAPIScopeEndUser(c *gin.Context) {
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
	sc.getAPIScope(c, tenantDB, tenantUUID)
}

// UpdateAPIScopeEndUser godoc
// @Summary Update API Scope (End User)
// @Description Uses the tenant database. Updates an API scope and replaces permission mappings.
// @Tags OAuth: API Scopes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param scope_id path string true "Scope ID (UUID)"
// @Param input body models.UpdateAPIScopeRequest true "API Scope update payload"
// @Success 200 {object} models.APIScopeResponse "Updated scope with permissions count"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Scope not found"
// @Failure 409 {object} map[string]string "Scope name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/user/api_scopes/{scope_id} [put]
func (sc *APIScopesController) UpdateAPIScopeEndUser(c *gin.Context) {
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
	sc.updateAPIScope(c, tenantDB, tenantUUID)
}

// DeleteAPIScopeEndUser godoc
// @Summary Delete API Scope (End User)
// @Description Uses the tenant database. Deletes an API scope and all permission mappings (cascade).
// @Tags OAuth: API Scopes
// @Produce json
// @Security BearerAuth
// @Param scope_id path string true "Scope ID (UUID)"
// @Success 200 {object} map[string]string "Deletion confirmation"
// @Failure 400 {object} map[string]string "Invalid scope ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Scope not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/user/api_scopes/{scope_id} [delete]
func (sc *APIScopesController) DeleteAPIScopeEndUser(c *gin.Context) {
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
	sc.deleteAPIScope(c, tenantDB, tenantUUID)
}

// --- Shared implementation helpers ---

func (sc *APIScopesController) createAPIScope(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	var req models.CreateAPIScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate scope name format (should be like "resource:action")
	if !strings.Contains(req.Name, ":") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Scope name should be in format 'resource:action' (e.g., 'files:read')"})
		return
	}

	// Parse and validate permission IDs
	permUUIDs, err := sc.validatePermissionIDs(db, tenantID, req.MappedPermissionIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var scope models.APIScope
	var permStrings []string

	// Transaction: insert scope + permission mappings
	if err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Create API scope
		scope = models.APIScope{
			ID:          uuid.New(), // Generate UUID in Go for cross-database compatibility
			TenantID:    &tenantID,
			Name:        req.Name,
			Description: req.Description,
		}
		if err := tx.Create(&scope).Error; err != nil {
			errLower := strings.ToLower(err.Error())
			if strings.Contains(errLower, "duplicate") || strings.Contains(errLower, "unique") {
				return fmt.Errorf("CONFLICT: scope name '%s' already exists", req.Name)
			}
			return err
		}

		// 2. Create permission mappings
		for _, permID := range permUUIDs {
			mapping := models.APIScopePermission{
				ScopeID:      scope.ID,
				PermissionID: permID,
			}
			if err := tx.Create(&mapping).Error; err != nil {
				return err
			}
		}

		// 3. Fetch permission strings for response
		var perms []models.RBACPermission
		if len(permUUIDs) > 0 {
			if err := tx.Where("id IN ?", permUUIDs).Find(&perms).Error; err != nil {
				return err
			}
			for _, p := range perms {
				permStrings = append(permStrings, fmt.Sprintf("%s:%s", p.Resource, p.Action))
			}
		}

		return nil
	}); err != nil {
		if strings.HasPrefix(err.Error(), "CONFLICT:") {
			c.JSON(http.StatusConflict, gin.H{"error": strings.TrimPrefix(err.Error(), "CONFLICT: ")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API scope: " + err.Error()})
		return
	}

	// Build response
	permIDStrings := make([]string, len(permUUIDs))
	for i, id := range permUUIDs {
		permIDStrings[i] = id.String()
	}

	// Audit log: API scope created
	middlewares.Audit(c, "api_scope", scope.ID.String(), "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"scope_id":          scope.ID.String(),
			"name":              scope.Name,
			"description":       scope.Description,
			"tenant_id":         tenantID.String(),
			"permissions_count": len(permUUIDs),
		},
	})

	resp := models.APIScopeResponse{
		ID:                scope.ID.String(),
		Name:              scope.Name,
		Description:       scope.Description,
		PermissionsLinked: len(permUUIDs),
		PermissionIDs:     permIDStrings,
		PermissionStrings: permStrings,
		CreatedAt:         scope.CreatedAt.Format(time.RFC3339),
	}
	c.JSON(http.StatusOK, resp)
}

func (sc *APIScopesController) listAPIScopes(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	nameFilter := c.Query("name")

	var scopes []models.APIScope
	query := db.Where("tenant_id = ?", tenantID)
	if nameFilter != "" {
		// Use LOWER() for cross-database compatibility (SQLite doesn't support ILIKE)
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+nameFilter+"%")
	}
	if err := query.Order("created_at DESC").Find(&scopes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API scopes: " + err.Error()})
		return
	}

	// Get permission counts for each scope
	type countRow struct {
		ScopeID uuid.UUID
		Count   int
	}
	var counts []countRow
	if len(scopes) > 0 {
		scopeIDs := make([]uuid.UUID, len(scopes))
		for i, s := range scopes {
			scopeIDs[i] = s.ID
		}
		db.Table("api_scope_permissions").
			Select("scope_id, count(*) as count").
			Where("scope_id IN ?", scopeIDs).
			Group("scope_id").
			Scan(&counts)
	}

	countMap := make(map[uuid.UUID]int)
	for _, c := range counts {
		countMap[c.ScopeID] = c.Count
	}

	// Initialize resp as empty slice to avoid null in JSON when no scopes exist
	resp := make([]models.APIScopeListItem, 0)
	for _, scope := range scopes {
		resp = append(resp, models.APIScopeListItem{
			ID:                scope.ID.String(),
			Name:              scope.Name,
			Description:       scope.Description,
			PermissionsLinked: countMap[scope.ID],
			CreatedAt:         scope.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (sc *APIScopesController) getAPIScope(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	scopeIDStr := c.Param("scope_id")
	scopeID, err := uuid.Parse(scopeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scope ID format"})
		return
	}

	var scope models.APIScope
	if err := db.Where("id = ? AND tenant_id = ?", scopeID, tenantID).First(&scope).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "API scope not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API scope: " + err.Error()})
		return
	}

	// Get mapped permissions
	var mappings []models.APIScopePermission
	db.Where("scope_id = ?", scopeID).Find(&mappings)

	var permIDs []string
	var permStrings []string
	if len(mappings) > 0 {
		permUUIDs := make([]uuid.UUID, len(mappings))
		for i, m := range mappings {
			permUUIDs[i] = m.PermissionID
			permIDs = append(permIDs, m.PermissionID.String())
		}

		var perms []models.RBACPermission
		db.Where("id IN ?", permUUIDs).Find(&perms)
		for _, p := range perms {
			permStrings = append(permStrings, fmt.Sprintf("%s:%s", p.Resource, p.Action))
		}
	}

	resp := models.APIScopeResponse{
		ID:                scope.ID.String(),
		Name:              scope.Name,
		Description:       scope.Description,
		PermissionsLinked: len(mappings),
		PermissionIDs:     permIDs,
		PermissionStrings: permStrings,
		CreatedAt:         scope.CreatedAt.Format(time.RFC3339),
	}
	c.JSON(http.StatusOK, resp)
}

func (sc *APIScopesController) updateAPIScope(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	scopeIDStr := c.Param("scope_id")
	scopeID, err := uuid.Parse(scopeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scope ID format"})
		return
	}

	var req models.UpdateAPIScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate scope name format if provided
	if req.Name != "" && !strings.Contains(req.Name, ":") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Scope name should be in format 'resource:action' (e.g., 'files:read')"})
		return
	}

	// Parse and validate permission IDs if provided
	var permUUIDs []uuid.UUID
	if len(req.MappedPermissionIDs) > 0 {
		permUUIDs, err = sc.validatePermissionIDs(db, tenantID, req.MappedPermissionIDs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	var scope models.APIScope
	var permStrings []string

	if err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Check scope exists for tenant
		if err := tx.Where("id = ? AND tenant_id = ?", scopeID, tenantID).First(&scope).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("NOT_FOUND: API scope not found")
			}
			return err
		}

		// 2. Update scope fields
		updates := make(map[string]interface{})
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		if len(updates) > 0 {
			if err := tx.Model(&scope).Updates(updates).Error; err != nil {
				errLower := strings.ToLower(err.Error())
				if strings.Contains(errLower, "duplicate") || strings.Contains(errLower, "unique") {
					return fmt.Errorf("CONFLICT: scope name '%s' already exists", req.Name)
				}
				return err
			}
		}

		// 3. Replace permission mappings if provided
		if len(req.MappedPermissionIDs) > 0 {
			// Delete existing mappings
			if err := tx.Where("scope_id = ?", scopeID).Delete(&models.APIScopePermission{}).Error; err != nil {
				return err
			}

			// Create new mappings
			for _, permID := range permUUIDs {
				mapping := models.APIScopePermission{
					ScopeID:      scopeID,
					PermissionID: permID,
				}
				if err := tx.Create(&mapping).Error; err != nil {
					return err
				}
			}

			// Fetch permission strings for response
			var perms []models.RBACPermission
			if err := tx.Where("id IN ?", permUUIDs).Find(&perms).Error; err != nil {
				return err
			}
			for _, p := range perms {
				permStrings = append(permStrings, fmt.Sprintf("%s:%s", p.Resource, p.Action))
			}
		}

		// Refresh scope data
		tx.First(&scope, scopeID)
		return nil
	}); err != nil {
		if strings.HasPrefix(err.Error(), "NOT_FOUND:") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API scope not found"})
			return
		}
		if strings.HasPrefix(err.Error(), "CONFLICT:") {
			c.JSON(http.StatusConflict, gin.H{"error": strings.TrimPrefix(err.Error(), "CONFLICT: ")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API scope: " + err.Error()})
		return
	}

	// Get current permission count if not updated
	if len(permStrings) == 0 {
		var count int64
		db.Model(&models.APIScopePermission{}).Where("scope_id = ?", scopeID).Count(&count)

		var mappings []models.APIScopePermission
		db.Where("scope_id = ?", scopeID).Find(&mappings)
		if len(mappings) > 0 {
			permUUIDs = make([]uuid.UUID, len(mappings))
			for i, m := range mappings {
				permUUIDs[i] = m.PermissionID
			}
			var perms []models.RBACPermission
			db.Where("id IN ?", permUUIDs).Find(&perms)
			for _, p := range perms {
				permStrings = append(permStrings, fmt.Sprintf("%s:%s", p.Resource, p.Action))
			}
		}
	}

	permIDStrings := make([]string, len(permUUIDs))
	for i, id := range permUUIDs {
		permIDStrings[i] = id.String()
	}

	// Audit log: API scope updated
	middlewares.Audit(c, "api_scope", scope.ID.String(), "update", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"scope_id":          scope.ID.String(),
			"name":              scope.Name,
			"description":       scope.Description,
			"tenant_id":         tenantID.String(),
			"permissions_count": len(permUUIDs),
		},
	})

	resp := models.APIScopeResponse{
		ID:                scope.ID.String(),
		Name:              scope.Name,
		Description:       scope.Description,
		PermissionsLinked: len(permUUIDs),
		PermissionIDs:     permIDStrings,
		PermissionStrings: permStrings,
		CreatedAt:         scope.CreatedAt.Format(time.RFC3339),
	}
	c.JSON(http.StatusOK, resp)
}

func (sc *APIScopesController) deleteAPIScope(c *gin.Context, db *gorm.DB, tenantID uuid.UUID) {
	scopeIDStr := c.Param("scope_id")
	scopeID, err := uuid.Parse(scopeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scope ID format"})
		return
	}

	// Check scope exists for tenant
	var scope models.APIScope
	if err := db.Where("id = ? AND tenant_id = ?", scopeID, tenantID).First(&scope).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "API scope not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find API scope: " + err.Error()})
		return
	}

	// Delete scope (cascade will delete permission mappings)
	if err := db.Delete(&scope).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API scope: " + err.Error()})
		return
	}

	// Audit log: API scope deleted
	middlewares.Audit(c, "api_scope", scopeID.String(), "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"scope_id":    scopeID.String(),
			"name":        scope.Name,
			"description": scope.Description,
			"tenant_id":   tenantID.String(),
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message":  "API scope deleted successfully",
		"scope_id": scopeID.String(),
		"name":     scope.Name,
	})
}

// validatePermissionIDs validates that all permission IDs exist and belong to the tenant
func (sc *APIScopesController) validatePermissionIDs(db *gorm.DB, tenantID uuid.UUID, permIDStrings []string) ([]uuid.UUID, error) {
	if len(permIDStrings) == 0 {
		return nil, nil
	}

	permUUIDs := make([]uuid.UUID, 0, len(permIDStrings))
	for _, idStr := range permIDStrings {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid permission ID format: %s", idStr)
		}
		permUUIDs = append(permUUIDs, id)
	}

	// Validate all permissions exist and belong to tenant
	var count int64
	db.Model(&models.RBACPermission{}).Where("id IN ? AND tenant_id = ?", permUUIDs, tenantID).Count(&count)
	if int(count) != len(permUUIDs) {
		return nil, fmt.Errorf("one or more permission IDs do not exist or do not belong to this tenant")
	}

	return permUUIDs, nil
}
