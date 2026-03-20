package middlewares

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditWithStatus logs an audit event with an explicit status string.
// Used by WebAuthn and MFA handlers.
func AuditWithStatus(c *gin.Context, objectType string, objectID string, actionType string, status string, changes *AuditChanges) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	userEmail := c.GetString("email_id")
	reqID := c.Writer.Header().Get("X-Request-ID")

	if tenantID == "" {
		tenantID = "unknown_tenant"
	}

	entry := AuditLogSchema{
		TS:      time.Now().UTC().Format(time.RFC3339),
		LogType: LogTypeAudit,
		Tenant:  Tenant{ID: tenantID},
		Event: AuditEvent{
			ID:       uuid.New().String(),
			Type:     fmt.Sprintf("%s.%s", objectType, actionType),
			Category: "authorization",
		},
		Actor: Actor{
			ID:    userID,
			Type:  "user",
			Email: userEmail,
		},
		Object: AuditObject{
			Type: objectType,
			ID:   objectID,
		},
		Action: AuditAction{
			Operation: actionType,
		},
		Changes: changes,
		Result:  ResultInfo{Status: status},
		Context: AuditContext{
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		},
		Correlation: AuditCorrelation{RequestID: reqID},
	}

	printJSON(entry)
}

// AuditAuthentication logs authentication-specific audit events.
// Used by WebAuthn, TOTP, and SMS MFA handlers.
func AuditAuthentication(c *gin.Context, userID string, authMethod string, actionType string, success bool, metadata map[string]interface{}) {
	tenantID := c.GetString("tenant_id")
	userEmail := c.GetString("email_id")
	reqID := c.Writer.Header().Get("X-Request-ID")

	if tenantID == "" {
		tenantID = "unknown_tenant"
	}

	status := "SUCCESS"
	if !success {
		status = "FAILURE"
	}

	var changes *AuditChanges
	if metadata != nil {
		changes = &AuditChanges{After: metadata}
	}

	entry := AuditLogSchema{
		TS:      time.Now().UTC().Format(time.RFC3339),
		LogType: LogTypeAudit,
		Tenant:  Tenant{ID: tenantID},
		Event: AuditEvent{
			ID:       uuid.New().String(),
			Type:     fmt.Sprintf("%s.%s", authMethod, actionType),
			Category: "authentication",
		},
		Actor: Actor{
			ID:    userID,
			Type:  "user",
			Email: userEmail,
		},
		Object: AuditObject{
			Type: authMethod,
			ID:   userID,
		},
		Action: AuditAction{
			Operation: actionType,
		},
		Changes: changes,
		Result:  ResultInfo{Status: status},
		Context: AuditContext{
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		},
		Correlation: AuditCorrelation{RequestID: reqID},
	}

	printJSON(entry)
}
