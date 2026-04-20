package middleware

import (
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TenantDBProvider is an interface for obtaining a tenant-scoped database connection.
type TenantDBProvider interface {
	GetTenantDB(tenantID string) (*sql.DB, error)
}

// AgentCertMiddleware authenticates AI agents via client certificates
// forwarded by nginx ingress in the ssl-client-cert header or via direct mTLS.
//
// When strict=true, requests without a valid agent cert are rejected.
// When strict=false (permissive), requests without the header are allowed
// through so the system works before ingress mTLS forwarding is configured.
type AgentCertMiddleware struct {
	dbProvider TenantDBProvider
	logger     *logrus.Logger
	strict     bool
}

// NewAgentCertMiddleware creates a new agent certificate middleware.
// Set strict=true to require a valid agent cert on every request.
func NewAgentCertMiddleware(dbProvider TenantDBProvider, logger *logrus.Logger, strict bool) *AgentCertMiddleware {
	return &AgentCertMiddleware{
		dbProvider: dbProvider,
		logger:     logger,
		strict:     strict,
	}
}

// Authenticate returns a Gin middleware that validates agent certificates.
func (m *AgentCertMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Try ssl-client-cert header (URL-encoded PEM set by nginx ingress)
		certHeader := c.GetHeader("ssl-client-cert")
		if certHeader == "" {
			certHeader = c.GetHeader("Ssl-Client-Cert")
		}

		if certHeader != "" {
			if m.authenticateFromCertHeader(c, certHeader) {
				c.Next()
				return
			}
			// Auth failed, error already sent via c.AbortWithStatusJSON
			return
		}

		// 2. Try direct TLS peer certificate (when server terminates TLS itself)
		if c.Request.TLS != nil && len(c.Request.TLS.PeerCertificates) > 0 {
			if m.authenticateFromTLSCert(c, c.Request.TLS.PeerCertificates[0]) {
				c.Next()
				return
			}
			return
		}

		// 3. No certificate available
		if m.strict {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Agent certificate required: provide client certificate via mTLS or ssl-client-cert header",
				},
			})
			return
		}

		// Permissive mode: allow through with a warning
		m.logger.WithFields(logrus.Fields{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"remote_addr": c.ClientIP(),
		}).Warn("Agent request without certificate (permissive mode)")

		c.Next()
	}
}

// authenticateFromCertHeader parses the URL-encoded PEM from nginx's ssl-client-cert header.
func (m *AgentCertMiddleware) authenticateFromCertHeader(c *gin.Context, certHeader string) bool {
	// Nginx URL-encodes the PEM certificate
	decoded, err := url.QueryUnescape(certHeader)
	if err != nil {
		m.logger.WithError(err).Warn("Failed to URL-decode ssl-client-cert header")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid ssl-client-cert header encoding",
			},
		})
		return false
	}

	block, _ := pem.Decode([]byte(decoded))
	if block == nil {
		m.logger.Warn("No PEM block in ssl-client-cert header")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid certificate PEM in ssl-client-cert header",
			},
		})
		return false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		m.logger.WithError(err).Warn("Failed to parse certificate from ssl-client-cert header")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Failed to parse agent certificate",
			},
		})
		return false
	}

	return m.authenticateFromTLSCert(c, cert)
}

// authenticateFromTLSCert validates an agent's X.509 certificate and sets context values.
func (m *AgentCertMiddleware) authenticateFromTLSCert(c *gin.Context, cert *x509.Certificate) bool {
	// Extract SPIFFE ID from URI SANs
	var spiffeID string
	for _, uri := range cert.URIs {
		if strings.HasPrefix(uri.String(), "spiffe://") {
			spiffeID = uri.String()
			break
		}
	}

	if spiffeID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "No SPIFFE ID in agent certificate URI SANs",
			},
		})
		return false
	}

	// Parse SPIFFE ID: spiffe://<tenant_id>/agent/<node_id>
	tenantID, isAgent, err := parseAgentSpiffeID(spiffeID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Invalid SPIFFE ID format in agent certificate",
			},
		})
		return false
	}

	if !isAgent {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Certificate does not belong to an agent (expected spiffe://<tenant>/agent/...)",
			},
		})
		return false
	}

	// Look up the agent in the tenant database
	agentRecord, err := m.lookupAgent(c, tenantID, spiffeID)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"spiffe_id": spiffeID,
			"tenant_id": tenantID,
		}).WithError(err).Error("Failed to look up agent in database")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to verify agent identity",
			},
		})
		return false
	}

	if agentRecord == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Agent not found: " + spiffeID,
			},
		})
		return false
	}

	if agentRecord.Status != "active" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Agent is not active (status: " + agentRecord.Status + ")",
			},
		})
		return false
	}

	// Set context for downstream handlers
	c.Set(SpireAgentIDKey, agentRecord.ID)
	c.Set(SpireTenantIDKey, tenantID)
	c.Set(SpireSpiffeIDKey, spiffeID)
	c.Set(SpireIsAgentKey, true)
	c.Set(SpireClientCertKey, cert)

	m.logger.WithFields(logrus.Fields{
		"agent_id":  agentRecord.ID,
		"spiffe_id": spiffeID,
		"tenant_id": tenantID,
	}).Debug("Agent authenticated via certificate")

	return true
}

// lookupAgent queries the tenant database for an agent by SPIFFE ID.
func (m *AgentCertMiddleware) lookupAgent(c *gin.Context, tenantID, spiffeID string) (*agentLookupResult, error) {
	tenantDB, err := m.dbProvider.GetTenantDB(tenantID)
	if err != nil {
		return nil, err
	}

	var result agentLookupResult
	err = tenantDB.QueryRowContext(c.Request.Context(),
		`SELECT id, status FROM agents WHERE spiffe_id = $1 LIMIT 1`,
		spiffeID,
	).Scan(&result.ID, &result.Status)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &result, nil
}

// agentLookupResult holds the minimal fields needed for agent auth.
type agentLookupResult struct {
	ID     string
	Status string
}

// parseAgentSpiffeID extracts tenant ID and validates agent path from a SPIFFE ID.
func parseAgentSpiffeID(spiffeID string) (tenantID string, isAgent bool, err error) {
	if !strings.HasPrefix(spiffeID, "spiffe://") {
		return "", false, &spireError{Code: "BAD_REQUEST", Message: "Not a SPIFFE ID"}
	}

	remainder := strings.TrimPrefix(spiffeID, "spiffe://")
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		return "", false, &spireError{Code: "BAD_REQUEST", Message: "Tenant ID missing in SPIFFE ID"}
	}

	tenantID = parts[0]
	if len(parts) >= 2 {
		pathParts := strings.Split(parts[1], "/")
		if len(pathParts) >= 1 && pathParts[0] == "agent" {
			isAgent = true
		}
	}

	return tenantID, isAgent, nil
}
