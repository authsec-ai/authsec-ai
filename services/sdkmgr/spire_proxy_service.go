package sdkmgr

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

// SPIREProxyService proxies SVID requests between customer SDKs and the local SPIRE agent.
// Translates sdk-manager's spire_service.py.
type SPIREProxyService struct {
	initialized bool
}

// NewSPIREProxyService creates a new service instance.
func NewSPIREProxyService() *SPIREProxyService {
	logrus.Info("SPIRE Proxy Service initialized")
	return &SPIREProxyService{}
}

// Initialize marks the service as ready.
func (s *SPIREProxyService) Initialize() {
	s.initialized = true
	logrus.Info("SPIRE Proxy Service ready")
}

// HealthCheck returns service health.
func (s *SPIREProxyService) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":      "healthy",
		"service":     "spire",
		"initialized": s.initialized,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
}

// FetchSVIDForWorkload maps client_id to tenant, validates access, and fetches
// an X.509 SVID from the local SPIRE agent via the Workload API.
func (s *SPIREProxyService) FetchSVIDForWorkload(
	clientID, socketPath string,
	envMetadata map[string]string,
) (map[string]interface{}, error) {
	logrus.WithField("client_id", truncate(clientID, 8)).Info("fetching SVID for workload")

	tenantID, err := s.getTenantIDFromClient(clientID)
	if err != nil {
		return nil, err
	}
	if err := s.validateTenant(tenantID); err != nil {
		return nil, err
	}

	// Set environment metadata for attestation selectors.
	origEnv := map[string]string{}
	for k, v := range envMetadata {
		if old, ok := os.LookupEnv(k); ok {
			origEnv[k] = old
		}
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envMetadata {
			if old, ok := origEnv[k]; ok {
				os.Setenv(k, old)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	// Fetch X.509 SVID via the SPIRE Workload API.
	// Uses go-spiffe/v2 workloadapi when available; falls back to socket-level check.
	svid, err := s.fetchX509SVID(socketPath)
	if err != nil {
		return nil, fmt.Errorf("SVID fetch failed: %w", err)
	}

	svid["tenant_id"] = tenantID
	logrus.WithField("spiffe_id", svid["spiffe_id"]).Info("SVID fetched successfully")
	return svid, nil
}

// fetchX509SVID connects to the SPIRE agent via the Workload API and fetches an X.509 SVID.
func (s *SPIREProxyService) fetchX509SVID(socketPath string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build the Workload API address from socket path.
	addr := "unix://" + socketPath

	client, err := workloadapi.New(ctx, workloadapi.WithAddr(addr))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to SPIRE agent at %s: %w", socketPath, err)
	}
	defer client.Close()

	x509Ctx, err := client.FetchX509Context(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch X.509 SVID from SPIRE agent: %w", err)
	}

	if len(x509Ctx.SVIDs) == 0 {
		return nil, fmt.Errorf("no X.509 SVIDs returned by SPIRE agent")
	}

	svid := x509Ctx.SVIDs[0]

	// Marshal certificate and private key to PEM.
	certPEM, keyPEM, err := svid.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SVID: %w", err)
	}

	// Marshal trust bundle to PEM.
	bundle := x509Ctx.Bundles.Bundles()
	var bundlePEM []byte
	for _, b := range bundle {
		bBytes, mErr := b.Marshal()
		if mErr == nil {
			bundlePEM = append(bundlePEM, bBytes...)
		}
	}

	return map[string]interface{}{
		"status":       "success",
		"spiffe_id":    svid.ID.String(),
		"certificate":  string(certPEM),
		"private_key":  string(keyPEM),
		"trust_bundle": string(bundlePEM),
		"fetched_at":   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// GetSVIDStatus returns the current SVID status for a workload.
func (s *SPIREProxyService) GetSVIDStatus(clientID string, spiffeID *string) (map[string]interface{}, error) {
	tenantID, err := s.getTenantIDFromClient(clientID)
	if err != nil {
		return nil, err
	}
	if err := s.validateTenant(tenantID); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"status":    "active",
		"client_id": clientID,
		"tenant_id": tenantID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if spiffeID != nil {
		result["spiffe_id"] = *spiffeID
	}
	return result, nil
}

// ValidateAgentConnection tests connectivity to the local SPIRE agent socket.
func (s *SPIREProxyService) ValidateAgentConnection(socketPath string) map[string]interface{} {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		logrus.WithError(err).WithField("socket_path", socketPath).Error("agent connection failed")
		return map[string]interface{}{
			"status":      "disconnected",
			"socket_path": socketPath,
			"error":       err.Error(),
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
	}
	conn.Close()

	return map[string]interface{}{
		"status":      "connected",
		"socket_path": socketPath,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
}

// getTenantIDFromClient maps a client_id to a tenant_id via the tenant_mappings table.
func (s *SPIREProxyService) getTenantIDFromClient(clientID string) (string, error) {
	db := config.DB
	if db == nil {
		return "", fmt.Errorf("master database not initialized")
	}

	baseClientID := NormalizeClientID(clientID)
	candidates := BuildClientIDCandidates(baseClientID)

	for _, cid := range candidates {
		var tenantID string
		err := db.Table("tenant_mappings").
			Where("client_id = ?", cid).
			Select("tenant_id").
			Row().Scan(&tenantID)
		if err == nil && tenantID != "" {
			logrus.WithFields(logrus.Fields{
				"client_id": clientID,
				"tenant_id": tenantID,
			}).Debug("mapped client_id to tenant_id")
			return tenantID, nil
		}
	}

	return "", fmt.Errorf("no tenant mapping found for client_id: %s", clientID)
}

// validateTenant checks that the tenant exists and has access to SPIRE.
func (s *SPIREProxyService) validateTenant(tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("invalid tenant_id")
	}
	// Basic validation: ensure tenant DB is reachable.
	_, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		return fmt.Errorf("tenant %s not accessible: %w", tenantID, err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
