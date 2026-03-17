package middlewares

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Constants for Log Aggregator to correctly route logs
const (
	LogTypeAuth  = "auth.event"
	LogTypeAudit = "audit.trail"
)

// --- Shared Structs for both log types ---

// Tenant represents tenant information in logs
type Tenant struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// Actor represents the entity performing an action
type Actor struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // "user", "admin", "api", "system"
	Email string `json:"email,omitempty"`
}

// ResultInfo represents the result of an operation
type ResultInfo struct {
	Status     string `json:"status"` // "SUCCESS" or "FAILURE"
	HTTPStatus int    `json:"http_status,omitempty"`
}

// --- Auth Logging Middleware (Automated) ---

// AuthLogSchema defines the schema for API traffic logs
type AuthLogSchema struct {
	TS       string                 `json:"ts"`
	LogType  string                 `json:"log_type"`
	Tenant   Tenant                 `json:"tenant"`
	LogLevel string                 `json:"log_level"`
	Event    AuthEvent              `json:"event"`
	Actor    Actor                  `json:"actor"`
	Client   ClientInfo             `json:"client"`
	Request  RequestInfo            `json:"request"`
	Result   ResultInfo             `json:"result"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AuthEvent represents an authentication/access event
type AuthEvent struct {
	Type     string `json:"type"`
	Category string `json:"category"` // "access_log"
}

// ClientInfo represents client information
type ClientInfo struct {
	IP        string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

// RequestInfo represents HTTP request information
type RequestInfo struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Duration string `json:"duration"`
}

// AuthLoggingMiddleware captures request and response details, printing to stdout.
// This is used for generating high-volume access logs compatible with log aggregators.
//
// Usage: Add to your Gin router's middleware stack:
//
//	r.Use(middlewares.AuthLoggingMiddleware("user-flow"))
func AuthLoggingMiddleware(serviceName string) gin.HandlerFunc {
	// 1. Define paths to exclude
	// Use a map[string]bool for O(1) efficiency
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 1. Check for prefix (excludes /health, /health/v1, /health-check, etc.)
		if strings.HasPrefix(path, "/uflow/health") {
			c.Next()
			return
		}

		// 2. Exact match map (for other random paths)
		skipPaths := map[string]bool{
			"/uflow/auth/voice/device-pending": true,
			"/uflow/favicon.ico":               true,
		}
		if skipPaths[path] {
			c.Next()
			return
		}

		start := time.Now()
		c.Next() // Process request

		duration := time.Since(start)

		// Extract context values (set by your existing AuthMiddleware)
		tenantID := c.GetString("tenant_id")
		userID := c.GetString("user_id")
		userEmail := c.GetString("email_id")

		// Fallback for public endpoints
		if tenantID == "" {
			tenantID = "unknown_tenant"
		}

		status := "SUCCESS"
		if c.Writer.Status() >= 400 {
			status = "FAILURE"
		}

		logEntry := AuthLogSchema{
			TS:       start.UTC().Format(time.RFC3339),
			LogType:  LogTypeAuth,
			Tenant:   Tenant{ID: tenantID},
			LogLevel: "INFO",
			Message:  fmt.Sprintf("%s %s request processed by %s", c.Request.Method, c.Request.URL.Path, serviceName),

			Event: AuthEvent{
				Type:     "api.request",
				Category: "access_log",
			},
			Actor: Actor{
				ID:    userID,
				Type:  "user",
				Email: userEmail,
			},
			Client: ClientInfo{
				IP:        c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
			},
			Request: RequestInfo{
				Method:   c.Request.Method,
				Path:     c.Request.URL.Path,
				Duration: duration.String(),
			},
			Result: ResultInfo{
				Status:     status,
				HTTPStatus: c.Writer.Status(),
			},
		}

		printJSON(logEntry)
	}
}

// --- Audit Trail Logging Function (Manual) ---

// AuditLogSchema defines the schema for business logic audit events
type AuditLogSchema struct {
	TS          string           `json:"ts"`
	LogType     string           `json:"log_type"` // "audit.trail"
	Tenant      Tenant           `json:"tenant"`
	Event       AuditEvent       `json:"event"`
	Actor       Actor            `json:"actor"`
	Object      AuditObject      `json:"object"`
	Action      AuditAction      `json:"action"`
	Changes     *AuditChanges    `json:"changes,omitempty"`
	Result      ResultInfo       `json:"result"`
	Context     AuditContext     `json:"context"`
	Correlation AuditCorrelation `json:"correlation"`
}

// AuditEvent represents an audit event
type AuditEvent struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Category string `json:"category"` // "authorization" | "configuration"
}

// AuditObject represents the object being audited
type AuditObject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// AuditAction represents the action taken
type AuditAction struct {
	Operation       string `json:"operation"`
	Justification   string `json:"justification,omitempty"`
	ChangeRequestID string `json:"change_request_id,omitempty"`
}

// AuditChanges represents before/after changes
type AuditChanges struct {
	Before map[string]interface{} `json:"before,omitempty"`
	After  map[string]interface{} `json:"after,omitempty"`
	Diff   []interface{}          `json:"diff,omitempty"`
}

// AuditContext represents the context of an audit event
type AuditContext struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

// AuditCorrelation represents correlation identifiers
type AuditCorrelation struct {
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

// Audit logs a critical business action to stdout for the log agent.
// This function needs to be called explicitly within your controller logic.
//
// Usage example:
//
//	changes := &middlewares.AuditChanges{
//	    Before: map[string]interface{}{"permissions": []string{"read"}},
//	    After:  map[string]interface{}{"permissions": []string{"read", "write"}},
//	}
//	middlewares.Audit(c, "role", roleID, "update", changes)
func Audit(c *gin.Context, objectType string, objectID string, actionType string, changes *AuditChanges) {
	// Extract context values
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	userEmail := c.GetString("email_id")
	reqID := c.GetString("request_id") // From RequestIDMiddleware

	if tenantID == "" {
		tenantID = "unknown_tenant"
	}

	// Determine actor type based on context
	actorType := "user"
	if c.GetString("is_admin") == "true" {
		actorType = "admin"
	}
	if userID == "" {
		actorType = "system"
	}

	entry := AuditLogSchema{
		TS:      time.Now().UTC().Format(time.RFC3339),
		LogType: LogTypeAudit,
		Tenant:  Tenant{ID: tenantID},

		Event: AuditEvent{
			Type:     fmt.Sprintf("%s.%s", objectType, actionType),
			Category: "authorization", // Set based on the type of change
		},
		Actor: Actor{
			ID:    userID,
			Type:  actorType,
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
		Result: ResultInfo{
			Status: "success",
		},
		Context: AuditContext{
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		},
		Correlation: AuditCorrelation{
			RequestID: reqID,
		},
	}

	printJSON(entry)
}

// --- Helper ---

// printJSON marshals and prints a value as single-line JSON to stdout.
// This format is required for log-agent sidecars to correctly parse logs.
func printJSON(v interface{}) {
	b, err := json.Marshal(v)
	if err == nil {
		// IMPORTANT: Must be single-line JSON for your log-agent sidecar to pick it up correctly.
		fmt.Println(string(b))
	} else {
		// Print errors to stderr to ensure they are captured by the log-agent as raw text
		fmt.Fprintf(os.Stderr, "Error marshalling log: %v\n", err)
	}
}
