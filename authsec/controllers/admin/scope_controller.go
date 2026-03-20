package admin

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ScopeController handles scope management
type ScopeController struct{}

// NewScopeController creates a new scope controller
func NewScopeController() *ScopeController {
	return &ScopeController{}
}

// ScopeMapping represents a scope and its associated resources
type ScopeMapping struct {
	ScopeName string   `json:"scope_name"`
	Resources []string `json:"resources"`
}

// AddScopeInput represents the input for adding a scope
type AddScopeInput struct {
	ScopeName string   `json:"scope_name" binding:"required"`
	Resources []string `json:"resources"` // Resources are optional - empty means full scope ("*")
}

// EditScopeInput represents the input for editing a scope
type EditScopeInput struct {
	Resources []string `json:"resources" binding:"required"`
}

// ========================================
// ADMIN ROUTES - Connect to MAIN/PRIMARY DB
// ========================================

// ListScopes godoc
// @Summary List Scopes (Admin)
// @Description Returns a list of all unique scope names from the main database
// @Tags Admin: Scopes
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Success 200 {object} []string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/scopes [get]
func (sc *ScopeController) ListScopes(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	if !ok || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID is required"})
		return
	}

	// Use main database connection
	sqlDB, err := config.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.listScopesInternal(c, sqlDB, tenantID)
}

// GetMappings godoc
// @Summary Get Scope Mappings (Admin)
// @Description Returns a list of scopes and their associated resources from the main database
// @Tags Admin: Scopes
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Success 200 {object} []ScopeMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/scopes/mappings [get]
func (sc *ScopeController) GetMappings(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	if !ok || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID is required"})
		return
	}

	// Use main database connection
	sqlDB, err := config.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.getMappingsInternal(c, sqlDB, tenantID)
}

// AddScope godoc
// @Summary Add Scope (Admin)
// @Description Adds a new scope with associated resources to the main database
// @Tags Admin: Scopes
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param input body AddScopeInput true "Scope data"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/scopes [post]
func (sc *ScopeController) AddScope(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	if !ok || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID is required"})
		return
	}

	// Use main database connection
	sqlDB, err := config.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.addScopeInternal(c, sqlDB, tenantID)
}

// EditScope godoc
// @Summary Edit Scope (Admin)
// @Description Updates the resources associated with a scope in the main database
// @Tags Admin: Scopes
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param scope_name path string true "Scope Name"
// @Param input body EditScopeInput true "Scope data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/scopes/{scope_name} [put]
func (sc *ScopeController) EditScope(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	scopeName := c.Param("scope_name")
	if !ok || tenantID == "" || scopeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID and Scope Name are required"})
		return
	}

	// Use main database connection
	sqlDB, err := config.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.editScopeInternal(c, sqlDB, tenantID, scopeName)
}

// DeleteScope godoc
// @Summary Delete Scope (Admin)
// @Description Deletes a scope and all its resource mappings from the main database
// @Tags Admin: Scopes
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param scope_name path string true "Scope Name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/scopes/{scope_name} [delete]
func (sc *ScopeController) DeleteScope(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	scopeName := c.Param("scope_name")
	if !ok || tenantID == "" || scopeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID and Scope Name are required"})
		return
	}

	// Use main database connection
	sqlDB, err := config.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.deleteScopeInternal(c, sqlDB, tenantID, scopeName)
}

// ========================================
// USER ROUTES - Connect to TENANT DB
// ========================================

// ListUserScopes godoc
// @Summary List Scopes (End User)
// @Description Returns a list of all unique scope names from the tenant database.
// @Description
// @Description **Authentication Requirements:**
// @Description - `/uflow/user/scopes` - Requires authenticated **end-user JWT token**
// @Description - `/uflow/enduser/scopes` - Requires **admin JWT token** with `admin:access` permission
// @Tags User: Scopes
// @Tags Admin: End-User Scopes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Success 200 {object} []string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/scopes [get]
// @Router /uflow/enduser/scopes [get]
func (sc *ScopeController) ListUserScopes(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	if !ok || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID is required"})
		return
	}

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.listScopesInternal(c, sqlDB, tenantID)
}

// GetUserMappings godoc
// @Summary Get Scope Mappings (End User)
// @Description Returns a list of scopes and their associated resources from the tenant database.
// @Description
// @Description **Authentication Requirements:**
// @Description - `/uflow/user/scopes/mappings` - Requires authenticated **end-user JWT token**
// @Description - `/uflow/enduser/scopes/mappings` - Requires **admin JWT token** with `admin:access` permission
// @Tags User: Scopes
// @Tags Admin: End-User Scopes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Success 200 {object} []ScopeMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/scopes/mappings [get]
// @Router /uflow/enduser/scopes/mappings [get]
func (sc *ScopeController) GetUserMappings(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	if !ok || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID is required"})
		return
	}

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.getMappingsInternal(c, sqlDB, tenantID)
}

// AddUserScope godoc
// @Summary Add Scope (End User)
// @Description Adds a new scope with associated resources to the tenant database.
// @Description
// @Description **Authentication Requirements:**
// @Description - `/uflow/user/scopes` - Requires authenticated **end-user JWT token**
// @Description - `/uflow/enduser/scopes` - Requires **admin JWT token** with `admin:access` permission
// @Tags User: Scopes
// @Tags Admin: End-User Scopes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param input body AddScopeInput true "Scope data"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/scopes [post]
// @Router /uflow/enduser/scopes [post]
func (sc *ScopeController) AddUserScope(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	if !ok || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID is required"})
		return
	}

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.addScopeInternal(c, sqlDB, tenantID)
}

// EditUserScope godoc
// @Summary Edit Scope (End User)
// @Description Updates the resources associated with a scope in the tenant database.
// @Description
// @Description **Authentication Requirements:**
// @Description - `/uflow/user/scopes/{scope_name}` - Requires authenticated **end-user JWT token**
// @Description - `/uflow/enduser/scopes/{scope_name}` - Requires **admin JWT token** with `admin:access` permission
// @Tags User: Scopes
// @Tags Admin: End-User Scopes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param scope_name path string true "Scope Name"
// @Param input body EditScopeInput true "Scope data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/scopes/{scope_name} [put]
// @Router /uflow/enduser/scopes/{scope_name} [put]
func (sc *ScopeController) EditUserScope(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	scopeName := c.Param("scope_name")
	if !ok || tenantID == "" || scopeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID and Scope Name are required"})
		return
	}

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.editScopeInternal(c, sqlDB, tenantID, scopeName)
}

// DeleteUserScope godoc
// @Summary Delete Scope (End User)
// @Description Deletes a scope and all its resource mappings from the tenant database.
// @Description
// @Description **Authentication Requirements:**
// @Description - `/uflow/user/scopes/{scope_name}` - Requires authenticated **end-user JWT token**
// @Description - `/uflow/enduser/scopes/{scope_name}` - Requires **admin JWT token** with `admin:access` permission
// @Tags User: Scopes
// @Tags Admin: End-User Scopes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param scope_name path string true "Scope Name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/scopes/{scope_name} [delete]
// @Router /uflow/enduser/scopes/{scope_name} [delete]
func (sc *ScopeController) DeleteUserScope(c *gin.Context) {
	tenantID, ok := middlewares.GetTenantIDFromToken(c)
	scopeName := c.Param("scope_name")
	if !ok || tenantID == "" || scopeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID and Scope Name are required"})
		return
	}

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database connection"})
		return
	}

	sc.deleteScopeInternal(c, sqlDB, tenantID, scopeName)
}

// ========================================
// INTERNAL SHARED FUNCTIONS
// ========================================

func (sc *ScopeController) listScopesInternal(c *gin.Context, sqlDB *sql.DB, tenantID string) {
	query := `SELECT DISTINCT scope_name FROM scope_resource_mappings WHERE tenant_id = $1 ORDER BY scope_name`
	rows, err := sqlDB.Query(query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scopes"})
		return
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var scopeName string
		if err := rows.Scan(&scopeName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan scope"})
			return
		}
		scopes = append(scopes, scopeName)
	}

	if scopes == nil {
		scopes = []string{}
	}

	c.JSON(http.StatusOK, scopes)
}

func (sc *ScopeController) getMappingsInternal(c *gin.Context, sqlDB *sql.DB, tenantID string) {
	query := `
		SELECT scope_name, array_agg(resource_name) 
		FROM scope_resource_mappings 
		WHERE tenant_id = $1
		GROUP BY scope_name 
		ORDER BY scope_name
	`
	rows, err := sqlDB.Query(query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scope mappings"})
		return
	}
	defer rows.Close()

	var mappings []ScopeMapping
	for rows.Next() {
		var scopeName string
		var resources []string
		if err := rows.Scan(&scopeName, (*pq.StringArray)(&resources)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan mapping"})
			return
		}
		mappings = append(mappings, ScopeMapping{
			ScopeName: scopeName,
			Resources: resources,
		})
	}

	if mappings == nil {
		mappings = []ScopeMapping{}
	}

	c.JSON(http.StatusOK, mappings)
}

func (sc *ScopeController) addScopeInternal(c *gin.Context, sqlDB *sql.DB, tenantID string) {
	var input AddScopeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle empty/blank resources as wildcard ("*" = full scope)
	if len(input.Resources) == 0 {
		input.Resources = []string{"*"}
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	// Check if scope already exists for this tenant
	var exists int
	err = tx.QueryRow("SELECT 1 FROM scope_resource_mappings WHERE scope_name = $1 AND tenant_id = $2 LIMIT 1", input.ScopeName, tenantID).Scan(&exists)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Scope already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Insert mappings
	stmt, err := tx.Prepare("INSERT INTO scope_resource_mappings (tenant_id, scope_name, resource_name) VALUES ($1, $2, $3)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare statement"})
		return
	}
	defer stmt.Close()

	for _, resource := range input.Resources {
		// Treat empty resource string as wildcard
		if resource == "" {
			resource = "*"
		}
		if _, err := stmt.Exec(tenantID, input.ScopeName, resource); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add resource %s: %v", resource, err)})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Audit log: Scope created
	middlewares.Audit(c, "scope", input.ScopeName, "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"scope_name":      input.ScopeName,
			"resources":       input.Resources,
			"resources_count": len(input.Resources),
		},
	})

	c.JSON(http.StatusCreated, gin.H{"message": "Scope created successfully"})
}

func (sc *ScopeController) editScopeInternal(c *gin.Context, sqlDB *sql.DB, tenantID string, scopeName string) {
	var input EditScopeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	// Check if scope exists for this tenant
	var exists int
	err = tx.QueryRow("SELECT 1 FROM scope_resource_mappings WHERE scope_name = $1 AND tenant_id = $2 LIMIT 1", scopeName, tenantID).Scan(&exists)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scope not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Delete existing mappings for this scope and tenant
	if _, err := tx.Exec("DELETE FROM scope_resource_mappings WHERE scope_name = $1 AND tenant_id = $2", scopeName, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing mappings"})
		return
	}

	// Insert new mappings
	stmt, err := tx.Prepare("INSERT INTO scope_resource_mappings (tenant_id, scope_name, resource_name) VALUES ($1, $2, $3)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare statement"})
		return
	}
	defer stmt.Close()

	for _, resource := range input.Resources {
		if _, err := stmt.Exec(tenantID, scopeName, resource); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add resource %s: %v", resource, err)})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Scope updated successfully"})
}

func (sc *ScopeController) deleteScopeInternal(c *gin.Context, sqlDB *sql.DB, tenantID string, scopeName string) {
	// Check if scope exists for this tenant
	var exists int
	err := sqlDB.QueryRow("SELECT 1 FROM scope_resource_mappings WHERE scope_name = $1 AND tenant_id = $2 LIMIT 1", scopeName, tenantID).Scan(&exists)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scope not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Delete scope mappings for this tenant
	if _, err := sqlDB.Exec("DELETE FROM scope_resource_mappings WHERE scope_name = $1 AND tenant_id = $2", scopeName, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete scope"})
		return
	}

	// Audit log: Scope deleted
	middlewares.Audit(c, "scope", scopeName, "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"scope_name": scopeName,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Scope deleted successfully"})
}

// Helper to validate UUID
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
