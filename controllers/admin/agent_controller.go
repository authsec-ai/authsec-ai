package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	platformCtrl "github.com/authsec-ai/authsec/controllers/platform"
	sharedCtrl "github.com/authsec-ai/authsec/controllers/shared"
	spireservices "github.com/authsec-ai/authsec/internal/spire/services"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AgentController handles AI agent management: listing agents, provisioning
// SPIRE identities, and issuing delegated JWT-SVIDs.
type AgentController struct {
	jwtSvidSvc *spireservices.JWTSVIDService
}

func NewAgentController() *AgentController {
	return &AgentController{}
}

// SetJWTSVIDService injects the JWT-SVID service after bootstrap.
func (ac *AgentController) SetJWTSVIDService(svc *spireservices.JWTSVIDService) {
	ac.jwtSvidSvc = svc
}

// --- Request types ---

type ProvisionIdentityRequest struct {
	ParentID string `json:"parent_id" binding:"required"`
	TTL      *int   `json:"ttl,omitempty"`
}

type DelegateTokenRequest struct {
	AgentType  string   `json:"agent_type" binding:"required"`
	Audience   []string `json:"audience" binding:"required"`
	TTLSeconds int      `json:"ttl_seconds"`
}

// --- Agent client query helpers ---

type agentClient struct {
	ID         uuid.UUID `json:"id"`
	ClientID   uuid.UUID `json:"client_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Name       string    `json:"name"`
	Email      *string   `json:"email,omitempty"`
	Status     string    `json:"status"`
	Active     bool      `json:"active"`
	ClientType string    `json:"client_type"`
	AgentType  *string   `json:"agent_type,omitempty"`
	SpiffeID   *string   `json:"spiffe_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ListAgents lists all AI agent clients for the tenant.
// GET /uflow/admin/agents
func (ac *AgentController) ListAgents(c *gin.Context) {
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

	var agents []agentClient
	query := tenantDB.Table("clients").
		Where("tenant_id = ? AND client_type = 'ai_agent'", tenantID).
		Where("deleted = false OR deleted IS NULL")

	if agentType := c.Query("agent_type"); agentType != "" {
		query = query.Where("agent_type = ?", agentType)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents, "total": len(agents)})
}

// GetAgent gets a single AI agent client by client_id.
// GET /uflow/admin/agents/:id
func (ac *AgentController) GetAgent(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	clientID := c.Param("id")
	if _, err := uuid.Parse(clientID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	var agent agentClient
	result := tenantDB.Table("clients").
		Where("client_id = ? AND tenant_id = ? AND client_type = 'ai_agent'", clientID, tenantID).
		Where("deleted = false OR deleted IS NULL").
		First(&agent)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// ProvisionIdentity creates a SPIRE workload entry for an AI agent.
// Uses the in-process RegisterAgentWorkload which writes to both master DB
// (spire_workloads) and tenant DB (workload_entries), and registers with SPIRE via gRPC.
// The SPIFFE ID is then written back to the client record.
// POST /uflow/admin/agents/:id/provision-identity
func (ac *AgentController) ProvisionIdentity(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	clientID := c.Param("id")
	if _, err := uuid.Parse(clientID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var req ProvisionIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	// Look up the agent client
	var agent agentClient
	result := tenantDB.Table("clients").
		Where("client_id = ? AND tenant_id = ? AND client_type = 'ai_agent'", clientID, tenantID).
		Where("deleted = false OR deleted IS NULL").
		First(&agent)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	if !agent.Active || agent.Status != "Active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent is not active"})
		return
	}

	if agent.SpiffeID != nil && *agent.SpiffeID != "" {
		c.JSON(http.StatusConflict, gin.H{
			"error":     "Agent already has a SPIFFE identity",
			"spiffe_id": *agent.SpiffeID,
		})
		return
	}

	if agent.AgentType == nil || *agent.AgentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent does not have an agent_type set"})
		return
	}

	// Register workload entry via the monolith's platform controller.
	// Writes to both spire_workloads (master) and workload_entries (tenant),
	// and registers with SPIRE server via gRPC if connected.
	spiffeID, err := platformCtrl.RegisterAgentWorkload(
		tenantID.String(),
		clientID,
		*agent.AgentType,
		"", // platform — not provided in provision request
		nil, // selectors — not provided in provision request
	)
	if err != nil {
		log.Printf("[AgentController] Failed to create SPIRE entry for agent %s: %v", clientID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create SPIRE identity", "details": err.Error()})
		return
	}

	// Write the SPIFFE ID back to the client record
	updateResult := tenantDB.Table("clients").
		Where("client_id = ? AND tenant_id = ?", clientID, tenantID).
		Update("spiffe_id", spiffeID)
	if updateResult.Error != nil {
		log.Printf("[AgentController] Failed to update client spiffe_id: %v", updateResult.Error)
	}

	log.Printf("[AgentController] Agent %s provisioned: spiffe_id=%s", clientID, spiffeID)

	// Audit log: agent identity provisioned
	middlewares.Audit(c, "agent_identity", clientID, "provision", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"spiffe_id": spiffeID,
		},
	})

	c.JSON(http.StatusCreated, gin.H{
		"spiffe_id": spiffeID,
		"client_id": clientID,
		"tenant_id": tenantID.String(),
		"message":   "SPIRE identity provisioned successfully",
	})
}

// RevokeIdentity deletes the SPIRE workload entry for an AI agent and clears its SPIFFE ID.
// DELETE /uflow/admin/agents/:id/revoke-identity
func (ac *AgentController) RevokeIdentity(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	clientID := c.Param("id")
	if _, err := uuid.Parse(clientID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	// Look up the agent
	var agent agentClient
	result := tenantDB.Table("clients").
		Where("client_id = ? AND tenant_id = ? AND client_type = 'ai_agent'", clientID, tenantID).
		Where("deleted = false OR deleted IS NULL").
		First(&agent)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	if agent.SpiffeID == nil || *agent.SpiffeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent does not have a SPIFFE identity"})
		return
	}

	// Clear the SPIFFE ID from the client record
	tenantDB.Table("clients").
		Where("client_id = ? AND tenant_id = ?", clientID, tenantID).
		Update("spiffe_id", nil)

	log.Printf("[AgentController] Agent %s identity revoked (spiffe_id was %s)", clientID, *agent.SpiffeID)

	// Audit log: agent identity revoked
	middlewares.Audit(c, "agent_identity", clientID, "revoke", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"spiffe_id": *agent.SpiffeID,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"status":    "revoked",
		"client_id": clientID,
		"message":   "SPIRE identity revoked successfully",
	})
}

// DelegateToken resolves delegation permissions and issues a JWT-SVID for the agent.
// POST /uflow/admin/agents/:id/delegate-token
func (ac *AgentController) DelegateToken(c *gin.Context) {
	tenantID, err := sharedCtrl.ResolveTenantIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	clientID := c.Param("id")
	if _, err := uuid.Parse(clientID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var req DelegateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.TTLSeconds <= 0 {
		req.TTLSeconds = 3600
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	// Look up the agent client
	var agent agentClient
	result := tenantDB.Table("clients").
		Where("client_id = ? AND tenant_id = ? AND client_type = 'ai_agent'", clientID, tenantID).
		Where("deleted = false OR deleted IS NULL").
		First(&agent)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	if !agent.Active || agent.Status != "Active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent is not active"})
		return
	}

	if agent.SpiffeID == nil || *agent.SpiffeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent does not have a SPIFFE identity. Provision identity first."})
		return
	}

	// Get admin user ID from token
	userID := sharedCtrl.ContextStringValue(c, "user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Could not determine user ID from token"})
		return
	}

	// Resolve delegation permissions (intersection of admin perms + policy allowed perms)
	ttl := time.Duration(req.TTLSeconds) * time.Second
	delegatedPerms, policyClientID, err := resolveDelegationPermissions(userID, tenantID.String(), req.AgentType, &ttl)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Delegation not allowed", "details": err.Error()})
		return
	}

	if len(delegatedPerms) == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permissions to delegate after intersection"})
		return
	}

	// If the policy is pinned to a specific client, verify it matches
	if policyClientID != "" && policyClientID != clientID {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Delegation policy is bound to client %s, not %s", policyClientID, clientID)})
		return
	}

	// Get admin email for claims
	emailID := sharedCtrl.ContextStringValue(c, "email_id")

	// Build custom claims for the JWT-SVID
	customClaims := map[string]interface{}{
		"user_id":     userID,
		"tenant_id":   tenantID.String(),
		"email":       emailID,
		"agent_type":  req.AgentType,
		"permissions": delegatedPerms,
		"client_id":   clientID,
	}

	// Issue JWT-SVID directly via merged service
	finalTTL := int(ttl.Seconds())
	if ac.jwtSvidSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JWT-SVID service not initialized"})
		return
	}
	jwtResp, err := ac.jwtSvidSvc.IssueJWTSVID(c.Request.Context(), &spireservices.IssueJWTSVIDRequest{
		TenantID:     tenantID.String(),
		SpiffeID:     *agent.SpiffeID,
		Audience:     req.Audience,
		TTL:          finalTTL,
		CustomClaims: customClaims,
	})
	if err != nil {
		log.Printf("[AgentController] Failed to issue JWT-SVID for agent %s: %v", clientID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue JWT-SVID", "details": err.Error()})
		return
	}

	log.Printf("[AgentController] Delegated JWT-SVID issued: agent=%s spiffe_id=%s perms=%d ttl=%ds",
		clientID, *agent.SpiffeID, len(delegatedPerms), finalTTL)

	// Upsert into delegation_tokens so SDK/agent can pull it later
	permsJSON, _ := json.Marshal(delegatedPerms)
	audJSON, _ := json.Marshal(req.Audience)
	userUUID, _ := uuid.Parse(userID)
	clientUUID, _ := uuid.Parse(clientID)
	expiresAt := time.Now().Add(time.Duration(finalTTL) * time.Second)

	// Find matching policy ID for the record
	var policyID *uuid.UUID
	roleNames, _ := getUserRoleNames(userID, tenantID.String())
	if policy, err := findDelegationPolicy(tenantID.String(), roleNames, req.AgentType); err == nil {
		policyID = &policy.ID
	}

	upsertToken := models.DelegationToken{
		TenantID:    *tenantID,
		ClientID:    clientUUID,
		PolicyID:    policyID,
		Token:       jwtResp.Token,
		SpiffeID:    jwtResp.SpiffeID,
		Permissions: permsJSON,
		Audience:    audJSON,
		ExpiresAt:   expiresAt,
		DelegatedBy: userUUID,
		TTLSeconds:  finalTTL,
		Status:      "active",
	}

	// Upsert: update if (tenant_id, client_id) exists, else insert
	var existing models.DelegationToken
	upsertResult := tenantDB.
		Where("tenant_id = ? AND client_id = ?", tenantID, clientUUID).
		First(&existing)
	if upsertResult.Error == nil {
		// Update existing row
		tenantDB.Model(&existing).Updates(map[string]interface{}{
			"policy_id":    policyID,
			"token":        jwtResp.Token,
			"spiffe_id":    jwtResp.SpiffeID,
			"permissions":  permsJSON,
			"audience":     audJSON,
			"expires_at":   expiresAt,
			"delegated_by": userUUID,
			"ttl_seconds":  finalTTL,
			"status":       "active",
			"updated_at":   time.Now(),
		})
		log.Printf("[AgentController] Delegation token updated for agent %s", clientID)
	} else {
		// Insert new row
		if err := tenantDB.Create(&upsertToken).Error; err != nil {
			log.Printf("[AgentController] Failed to store delegation token: %v", err)
		} else {
			log.Printf("[AgentController] Delegation token stored for agent %s", clientID)
		}
	}

	// Audit log: delegation token issued
	middlewares.Audit(c, "delegation_token", clientID, "delegate", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"agent_type":  req.AgentType,
			"permissions": delegatedPerms,
			"ttl_seconds": finalTTL,
			"audience":    req.Audience,
			"spiffe_id":   jwtResp.SpiffeID,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"token":       jwtResp.Token,
		"spiffe_id":   jwtResp.SpiffeID,
		"expires_at":  jwtResp.ExpiresAt,
		"permissions": delegatedPerms,
		"audience":    req.Audience,
		"ttl_seconds": finalTTL,
	})
}

// extractBearerToken gets the Bearer token from the Authorization header.
func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
