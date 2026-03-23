package platform

import (
	"fmt"
	"net/http"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SDKTokenController handles SDK/agent token retrieval endpoints.
// AI agents use these endpoints to pull their delegated JWT-SVID tokens.
type SDKTokenController struct{}

func NewSDKTokenController() *SDKTokenController {
	return &SDKTokenController{}
}

// GetDelegationToken returns the active delegation token for an AI agent client.
// The SDK authenticates with its client_id (passed as query param or header).
//
// GET /uflow/sdk/delegation-token?client_id=<uuid>
//
// Flow:
//  1. SDK sends client_id
//  2. Look up tenant_id via tenant_mappings table (client_id → tenant_id)
//  3. Query delegation_tokens in tenant DB
//  4. Return token + permissions if active and not expired
func (sc *SDKTokenController) GetDelegationToken(c *gin.Context) {
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = c.GetHeader("X-Client-ID")
	}
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id is required (query param or X-Client-ID header)"})
		return
	}

	if _, err := uuid.Parse(clientID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
		return
	}

	// Resolve tenant_id from client_id via tenant_mappings in master DB
	tenantID, err := resolveTenantIDFromClientID(clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found", "details": err.Error()})
		return
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	var dt models.DelegationToken
	result := tenantDB.
		Where("client_id::text = ? AND tenant_id::text = ? AND status = 'active'", clientID, tenantID).
		First(&dt)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No active delegation token found",
			"details": "An admin must delegate a token first via POST /uflow/admin/agents/:id/delegate-token",
		})
		return
	}

	// Check expiry
	if dt.IsExpired() {
		// Mark as expired
		tenantDB.Model(&dt).Update("status", "expired")
		c.JSON(http.StatusGone, gin.H{
			"error":      "Delegation token has expired",
			"expired_at": dt.ExpiresAt,
			"details":    "Admin must re-delegate via POST /uflow/admin/agents/:id/delegate-token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":       dt.Token,
		"spiffe_id":   dt.SpiffeID,
		"permissions": dt.GetPermissions(),
		"audience":    dt.GetAudience(),
		"expires_at":  dt.ExpiresAt,
		"ttl_seconds": dt.TTLSeconds,
		"client_id":   dt.ClientID,
		"tenant_id":   dt.TenantID,
		"status":      dt.Status,
		"issued_at":   dt.CreatedAt,
		"updated_at":  dt.UpdatedAt,
	})
}

// RevokeDelegationToken revokes the active delegation token for an AI agent.
// POST /uflow/admin/agents/:id/revoke-token
func (sc *SDKTokenController) RevokeDelegationToken(c *gin.Context) {
	tenantID, err := resolveDelegationTenantID(c)
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

	result := tenantDB.Model(&models.DelegationToken{}).
		Where("client_id::text = ? AND tenant_id = ? AND status = 'active'", clientID, tenantID).
		Updates(map[string]interface{}{
			"status":     "revoked",
			"updated_at": time.Now(),
		})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active delegation token found for this agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "revoked",
		"client_id": clientID,
		"message":   "Delegation token revoked successfully",
	})
}

// resolveTenantIDFromClientID looks up the tenant_id for a client_id
// via the tenant_mappings table in the master database.
func resolveTenantIDFromClientID(clientID string) (string, error) {
	masterDB := config.GetDatabase()
	if masterDB == nil {
		return "", fmt.Errorf("master database not initialized")
	}

	var tenantID string
	err := masterDB.DB.QueryRow(
		`SELECT tenant_id FROM tenant_mappings WHERE client_id = $1 LIMIT 1`,
		clientID,
	).Scan(&tenantID)
	if err != nil {
		return "", fmt.Errorf("client_id %s not found in tenant_mappings", clientID)
	}
	return tenantID, nil
}
