package admin

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	sharedCtrl "github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DelegationPolicyController manages delegation policies that govern
// which roles can delegate trust to AI agent types.
// All operations run against the tenant's database.
type DelegationPolicyController struct {
	spireService *services.SpireService
}

func NewDelegationPolicyController() *DelegationPolicyController {
	return &DelegationPolicyController{
		spireService: services.NewSpireService(),
	}
}

// --- Request/Response types ---

type CreateDelegationPolicyRequest struct {
	RoleName           string   `json:"role_name" binding:"required"`
	AgentType          string   `json:"agent_type" binding:"required"`
	AllowedPermissions []string `json:"allowed_permissions"`
	MaxTTLSeconds      int      `json:"max_ttl_seconds"`
	Enabled            *bool    `json:"enabled"`
	ClientID           string   `json:"client_id"`
	Audience           []string `json:"audience"`
}

type UpdateDelegationPolicyRequest struct {
	RoleName           *string  `json:"role_name"`
	AgentType          *string  `json:"agent_type"`
	AllowedPermissions []string `json:"allowed_permissions"`
	MaxTTLSeconds      *int     `json:"max_ttl_seconds"`
	Enabled            *bool    `json:"enabled"`
	ClientID           *string  `json:"client_id"`
}

// CreateDelegationPolicy creates a new delegation policy for a tenant.
// @Summary     Create delegation policy
// @Description Defines which role can delegate to which AI agent type, with permission scoping and TTL cap.
// @Tags        Delegation
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       input body CreateDelegationPolicyRequest true "Delegation policy"
// @Success     201 {object} models.DelegationPolicy
// @Failure     400 {object} map[string]string
// @Failure     409 {object} map[string]string
// @Router      /uflow/delegation-policies [post]
func (dc *DelegationPolicyController) CreateDelegationPolicy(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	var req CreateDelegationPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Default values
	if req.MaxTTLSeconds <= 0 {
		req.MaxTTLSeconds = 3600
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	// Marshal allowed permissions to JSON
	allowedPerms := req.AllowedPermissions
	if allowedPerms == nil {
		allowedPerms = []string{}
	}
	permsJSON, _ := json.Marshal(allowedPerms)

	// Extract created_by from token
	userIDStr := sharedCtrl.ContextStringValue(c, "user_id")
	var createdBy *uuid.UUID
	if uid, err := uuid.Parse(userIDStr); err == nil {
		createdBy = &uid
	}

	// Validate and parse client_id if provided
	var clientID *uuid.UUID
	if req.ClientID != "" {
		cid, err := uuid.Parse(req.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
			return
		}
		if err := validateClientActive(req.ClientID, tenantID.String()); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Client not found or not active", "details": err.Error()})
			return
		}
		clientID = &cid
	}

	policy := models.DelegationPolicy{
		TenantID:           *tenantID,
		RoleName:           req.RoleName,
		AgentType:          req.AgentType,
		AllowedPermissions: permsJSON,
		MaxTTLSeconds:      req.MaxTTLSeconds,
		Enabled:            enabled,
		ClientID:           clientID,
		CreatedBy:          createdBy,
	}

	result := tenantDB.Create(&policy)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "A delegation policy for this role and agent type already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create delegation policy"})
		return
	}

	// If policy is linked to an AI agent client, auto-provision SPIRE identity if needed
	response := gin.H{"policy": policy}
	if clientID != nil {
		var spiffeID *string
		var clientType string
		var agentType *string
		tenantDB.Table("clients").
			Select("spiffe_id, client_type, agent_type").
			Where("client_id = ? AND tenant_id = ?", clientID, tenantID).
			Row().Scan(&spiffeID, &clientType, &agentType)

		if clientType == "ai_agent" {
			authToken := extractBearerToken(c)
			resolvedSpiffeID := ""

			if spiffeID == nil || *spiffeID == "" {
				// Auto-provision: fetch SPIRE agents to get parent_id
				agents, err := dc.spireService.ListAgents(authToken)
				if err != nil {
					log.Printf("[DelegationPolicy] Failed to list SPIRE agents for auto-provision: %v", err)
					response["identity_provision"] = gin.H{
						"status": "skipped",
						"reason": "Could not list SPIRE agents: " + err.Error(),
					}
				} else if len(agents) == 0 {
					response["identity_provision"] = gin.H{
						"status": "skipped",
						"reason": "No SPIRE agents available to use as parent",
					}
				} else {
					parentID := agents[0].SpiffeID
					agentTypeStr := req.AgentType
					if agentType != nil && *agentType != "" {
						agentTypeStr = *agentType
					}

					spireResp, err := dc.spireService.CreateAgentEntry(&services.CreateAgentEntryRequest{
						TenantID:  tenantID.String(),
						ClientID:  clientID.String(),
						AgentType: agentTypeStr,
						ParentID:  parentID,
					}, authToken)
					if err != nil {
						log.Printf("[DelegationPolicy] Auto-provision failed for agent %s: %v", clientID.String(), err)
						response["identity_provision"] = gin.H{
							"status": "failed",
							"reason": err.Error(),
						}
					} else {
						tenantDB.Table("clients").
							Where("client_id = ? AND tenant_id = ?", clientID, tenantID).
							Update("spiffe_id", spireResp.SpiffeID)

						log.Printf("[DelegationPolicy] Auto-provisioned identity for agent %s: spiffe_id=%s", clientID.String(), spireResp.SpiffeID)
						response["identity_provision"] = gin.H{
							"status":    "provisioned",
							"spiffe_id": spireResp.SpiffeID,
							"entry_id":  spireResp.EntryID,
							"parent_id": spireResp.ParentID,
						}
						resolvedSpiffeID = spireResp.SpiffeID
					}
				}
			} else {
				resolvedSpiffeID = *spiffeID
				response["identity_provision"] = gin.H{
					"status":    "already_provisioned",
					"spiffe_id": *spiffeID,
				}
			}

			// Auto-delegate token if we have a SPIFFE ID
			if resolvedSpiffeID != "" {
				audience := req.Audience
				if len(audience) == 0 {
					audience = []string{"authsec-api"}
				}
				ttlDuration := time.Duration(req.MaxTTLSeconds) * time.Second

				// Use the policy we just created directly
				delegatedPerms := policy.GetAllowedPermissions()
				if len(delegatedPerms) == 0 {
					response["delegate_token"] = gin.H{
						"status": "skipped",
						"reason": "No permissions in policy",
					}
				} else {
					emailID := sharedCtrl.ContextStringValue(c, "email_id")
					customClaims := map[string]interface{}{
						"user_id":     userIDStr,
						"tenant_id":   tenantID.String(),
						"email":       emailID,
						"agent_type":  req.AgentType,
						"permissions": delegatedPerms,
						"client_id":   clientID.String(),
					}

					finalTTL := int(ttlDuration.Seconds())
					jwtResp, jwtErr := dc.spireService.IssueDelegatedJWTSVID(&services.IssueJWTSVIDRequest{
						TenantID:     tenantID.String(),
						SpiffeID:     resolvedSpiffeID,
						Audience:     audience,
						TTL:          finalTTL,
						CustomClaims: customClaims,
					}, authToken)
					if jwtErr != nil {
						log.Printf("[DelegationPolicy] Auto-delegate-token failed for agent %s: %v", clientID.String(), jwtErr)
						response["delegate_token"] = gin.H{
							"status": "failed",
							"reason": jwtErr.Error(),
						}
					} else {
						// Store token in delegation_tokens
						dPermsJSON, _ := json.Marshal(delegatedPerms)
						audJSON, _ := json.Marshal(audience)
						userUUID, _ := uuid.Parse(userIDStr)
						expiresAt := time.Now().Add(time.Duration(finalTTL) * time.Second)

						upsertToken := models.DelegationToken{
							TenantID:    *tenantID,
							ClientID:    *clientID,
							PolicyID:    &policy.ID,
							Token:       jwtResp.Token,
							SpiffeID:    jwtResp.SpiffeID,
							Permissions: dPermsJSON,
							Audience:    audJSON,
							ExpiresAt:   expiresAt,
							DelegatedBy: userUUID,
							TTLSeconds:  finalTTL,
							Status:      "active",
						}

						var existing models.DelegationToken
						upsertResult := tenantDB.
							Where("tenant_id = ? AND client_id = ?", tenantID, clientID).
							First(&existing)
						if upsertResult.Error == nil {
							tenantDB.Model(&existing).Updates(map[string]interface{}{
								"policy_id":    &policy.ID,
								"token":        jwtResp.Token,
								"spiffe_id":    jwtResp.SpiffeID,
								"permissions":  dPermsJSON,
								"audience":     audJSON,
								"expires_at":   expiresAt,
								"delegated_by": userUUID,
								"ttl_seconds":  finalTTL,
								"status":       "active",
								"updated_at":   time.Now(),
							})
						} else {
							if err := tenantDB.Create(&upsertToken).Error; err != nil {
								log.Printf("[DelegationPolicy] Failed to store delegation token: %v", err)
							}
						}

						log.Printf("[DelegationPolicy] Auto-delegated token for agent %s: perms=%d ttl=%ds", clientID.String(), len(delegatedPerms), finalTTL)
						response["delegate_token"] = gin.H{
							"status":      "issued",
							"token":       jwtResp.Token,
							"spiffe_id":   jwtResp.SpiffeID,
							"expires_at":  jwtResp.ExpiresAt,
							"permissions": delegatedPerms,
							"audience":    audience,
							"ttl_seconds": finalTTL,
						}
					}
				}
			}
		}
	}

	// Audit log: delegation policy created
	middlewares.Audit(c, "delegation_policy", policy.ID.String(), "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"role_name":           req.RoleName,
			"agent_type":          req.AgentType,
			"allowed_permissions": req.AllowedPermissions,
			"max_ttl_seconds":     req.MaxTTLSeconds,
			"enabled":             enabled,
			"client_id":           req.ClientID,
		},
	})

	c.JSON(http.StatusCreated, response)
}

// ListDelegationPolicies lists all delegation policies for a tenant.
// @Summary     List delegation policies
// @Description Returns all delegation policies for the tenant. Supports filters: role_name, agent_type, enabled.
// @Tags        Delegation
// @Produce     json
// @Security    BearerAuth
// @Param       role_name  query string false "Filter by role name"
// @Param       agent_type query string false "Filter by agent type"
// @Param       enabled    query string false "Filter by enabled status (true/false)"
// @Success     200 {array} models.DelegationPolicy
// @Router      /uflow/delegation-policies [get]
func (dc *DelegationPolicyController) ListDelegationPolicies(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	query := tenantDB.Where("tenant_id = ?", tenantID)

	if roleName := c.Query("role_name"); roleName != "" {
		query = query.Where("role_name = ?", roleName)
	}
	if agentType := c.Query("agent_type"); agentType != "" {
		query = query.Where("agent_type = ?", agentType)
	}
	if enabled := c.Query("enabled"); enabled == "true" {
		query = query.Where("enabled = true")
	} else if enabled == "false" {
		query = query.Where("enabled = false")
	}

	var policies []models.DelegationPolicy
	if err := query.Order("created_at DESC").Find(&policies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list delegation policies"})
		return
	}

	c.JSON(http.StatusOK, policies)
}

// GetDelegationPolicy retrieves a single delegation policy by ID.
// @Summary     Get delegation policy
// @Tags        Delegation
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Policy ID (UUID)"
// @Success     200 {object} models.DelegationPolicy
// @Failure     404 {object} map[string]string
// @Router      /uflow/delegation-policies/{id} [get]
func (dc *DelegationPolicyController) GetDelegationPolicy(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	policyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	var policy models.DelegationPolicy
	result := tenantDB.Where("id = ? AND tenant_id = ?", policyID, tenantID).First(&policy)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delegation policy not found"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// UpdateDelegationPolicy updates an existing delegation policy.
// @Summary     Update delegation policy
// @Tags        Delegation
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id    path string                        true "Policy ID (UUID)"
// @Param       input body UpdateDelegationPolicyRequest true "Fields to update"
// @Success     200 {object} models.DelegationPolicy
// @Failure     404 {object} map[string]string
// @Router      /uflow/delegation-policies/{id} [put]
func (dc *DelegationPolicyController) UpdateDelegationPolicy(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	policyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	var policy models.DelegationPolicy
	if err := tenantDB.Where("id = ? AND tenant_id = ?", policyID, tenantID).First(&policy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delegation policy not found"})
		return
	}

	var req UpdateDelegationPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.RoleName != nil {
		policy.RoleName = *req.RoleName
	}
	if req.AgentType != nil {
		policy.AgentType = *req.AgentType
	}
	if req.AllowedPermissions != nil {
		permsJSON, _ := json.Marshal(req.AllowedPermissions)
		policy.AllowedPermissions = permsJSON
	}
	if req.MaxTTLSeconds != nil {
		policy.MaxTTLSeconds = *req.MaxTTLSeconds
	}
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.ClientID != nil {
		if *req.ClientID == "" {
			policy.ClientID = nil // clear client_id
		} else {
			cid, err := uuid.Parse(*req.ClientID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
				return
			}
			if err := validateClientActive(*req.ClientID, tenantID.String()); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Client not found or not active", "details": err.Error()})
				return
			}
			policy.ClientID = &cid
		}
	}
	policy.UpdatedAt = time.Now()

	if err := tenantDB.Save(&policy).Error; err != nil {
		if isDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "A delegation policy for this role and agent type already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update delegation policy"})
		return
	}

	// Audit log: delegation policy updated
	middlewares.Audit(c, "delegation_policy", policyID.String(), "update", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"role_name":       policy.RoleName,
			"agent_type":      policy.AgentType,
			"max_ttl_seconds": policy.MaxTTLSeconds,
			"enabled":         policy.Enabled,
		},
	})

	c.JSON(http.StatusOK, policy)
}

// DeleteDelegationPolicy deletes a delegation policy.
// @Summary     Delete delegation policy
// @Tags        Delegation
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Policy ID (UUID)"
// @Success     200 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /uflow/delegation-policies/{id} [delete]
func (dc *DelegationPolicyController) DeleteDelegationPolicy(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	policyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	result := tenantDB.Where("id = ? AND tenant_id = ?", policyID, tenantID).Delete(&models.DelegationPolicy{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delegation policy not found"})
		return
	}

	// Audit log: delegation policy deleted
	middlewares.Audit(c, "delegation_policy", policyID.String(), "delete", nil)

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": policyID.String()})
}

// GetMyRolesAndPermissions returns the authenticated admin's roles and permissions.
// This is used by the delegation UI to show what permissions can be delegated.
// @Summary     Get my roles and permissions
// @Tags        Delegation
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Router      /uflow/admin/me/roles-permissions [get]
func (dc *DelegationPolicyController) GetMyRolesAndPermissions(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	userID := sharedCtrl.ContextStringValue(c, "user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	tid := tenantID.String()

	// Get all tenant roles
	roles, err := getTenantRoleNames(tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get roles: " + err.Error()})
		return
	}

	// Get all tenant permissions
	permissions, err := getTenantPermissionStrings(tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get permissions: " + err.Error()})
		return
	}

	// Get all tenant scopes
	scopes, err := getTenantScopes(tid)
	if err != nil {
		scopes = []string{} // non-fatal, return empty
	}

	// Get all tenant resources
	resources, err := getTenantResources(tid)
	if err != nil {
		resources = []string{} // non-fatal, return empty
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     userID,
		"tenant_id":   tid,
		"roles":       roles,
		"permissions": permissions,
		"scopes":      scopes,
		"resources":   resources,
	})
}

// isDuplicateKeyError checks if a GORM error is a PostgreSQL unique constraint violation.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "23505")
}
