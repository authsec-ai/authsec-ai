package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// KubernetesValidator validates Kubernetes PSAT (Projected Service Account Token) attestation
type KubernetesValidator struct {
	logger              *logrus.Entry
	httpClient          *http.Client
	k8sAPIHost          string
	useTokenReview      bool
	serviceAccountToken string
}

// KubernetesValidatorConfig holds configuration for Kubernetes validation
type KubernetesValidatorConfig struct {
	// UseTokenReview enables production-grade token validation via TokenReview API
	// If false, falls back to JWT parsing without signature verification (dev only)
	UseTokenReview bool

	// KubernetesAPIHost is the Kubernetes API server URL (default: from in-cluster config)
	KubernetesAPIHost string

	// ServiceAccountToken for authenticating to K8s API (default: from in-cluster)
	ServiceAccountToken string

	// CACertPath for TLS verification (default: from in-cluster)
	CACertPath string
}

// NewKubernetesValidator creates a new Kubernetes validator
func NewKubernetesValidator(logger *logrus.Entry, cfg *KubernetesValidatorConfig) (*KubernetesValidator, error) {
	if cfg == nil {
		cfg = &KubernetesValidatorConfig{}
	}

	// In production, always require TokenReview validation.
	// Set USE_TOKEN_REVIEW=true or ENVIRONMENT=production to enforce.
	env := os.Getenv("ENVIRONMENT")
	if env == "production" || env == "staging" {
		if !cfg.UseTokenReview {
			logger.Warn("Forcing UseTokenReview=true because ENVIRONMENT is set to " + env)
			cfg.UseTokenReview = true
		}
	} else if os.Getenv("USE_TOKEN_REVIEW") == "true" {
		cfg.UseTokenReview = true
	}

	if !cfg.UseTokenReview {
		logger.Warn("K8s token validation running in DEV MODE (no signature verification). " +
			"Set ENVIRONMENT=production or USE_TOKEN_REVIEW=true for production use.")
	}

	// Default to in-cluster configuration
	if cfg.KubernetesAPIHost == "" {
		cfg.KubernetesAPIHost = os.Getenv("KUBERNETES_SERVICE_HOST")
		if cfg.KubernetesAPIHost != "" {
			port := os.Getenv("KUBERNETES_SERVICE_PORT")
			if port == "" {
				port = "443"
			}
			cfg.KubernetesAPIHost = fmt.Sprintf("https://%s:%s", cfg.KubernetesAPIHost, port)
		}
	}

	if cfg.ServiceAccountToken == "" {
		tokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
		if tokenData, err := os.ReadFile(tokenPath); err == nil {
			cfg.ServiceAccountToken = string(tokenData)
		}
	}

	if cfg.CACertPath == "" {
		cfg.CACertPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	}

	// Create HTTP client with TLS configuration
	var httpClient *http.Client
	if cfg.UseTokenReview {
		tlsConfig := &tls.Config{}

		// Load CA certificate for TLS verification
		if cfg.CACertPath != "" {
			caCert, err := os.ReadFile(cfg.CACertPath)
			if err != nil {
				logger.WithFields(logrus.Fields{"ca_path": cfg.CACertPath}).WithError(err).Warn("Failed to load CA certificate, using insecure TLS")
				tlsConfig.InsecureSkipVerify = true
			} else {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				tlsConfig.RootCAs = caCertPool
			}
		}

		httpClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
	}

	return &KubernetesValidator{
		logger:              logger,
		httpClient:          httpClient,
		k8sAPIHost:          cfg.KubernetesAPIHost,
		useTokenReview:      cfg.UseTokenReview,
		serviceAccountToken: cfg.ServiceAccountToken,
	}, nil
}

// Validate validates Kubernetes attestation evidence
func (v *KubernetesValidator) Validate(ctx context.Context, evidence map[string]interface{}) (map[string]string, error) {
	// Extract PSAT token from evidence
	psatToken, ok := evidence["psat_token"].(string)
	if !ok || psatToken == "" {
		return nil, fmt.Errorf("psat_token required")
	}

	clusterName, _ := evidence["cluster_name"].(string)

	var claims map[string]interface{}
	var err error

	// Production: Validate token with Kubernetes TokenReview API
	if v.useTokenReview && v.k8sAPIHost != "" {
		claims, err = v.validateWithTokenReview(ctx, psatToken)
		if err != nil {
			return nil, fmt.Errorf("token review validation failed: %w", err)
		}
		v.logger.Info("Token validated via TokenReview API")
	} else {
		// Development: Parse JWT without signature verification
		claims, err = v.parseJWT(psatToken)
		if err != nil {
			return nil, fmt.Errorf("invalid token: %w", err)
		}
		v.logger.Warn("Token validated without signature verification (dev mode)")
	}

	// Extract selectors from token claims
	selectors := map[string]string{}

	if clusterName != "" {
		selectors["k8s:cluster"] = clusterName
	}

	// Extract namespace
	if ns, ok := claims["kubernetes.io/serviceaccount/namespace"].(string); ok {
		selectors["k8s:ns"] = ns
		selectors["k8s:namespace"] = ns // Backward compatibility
	}

	// Extract service account name
	if sa, ok := claims["kubernetes.io/serviceaccount/service-account.name"].(string); ok {
		selectors["k8s:sa"] = sa
		selectors["k8s:service-account"] = sa // Backward compatibility
	}

	// Extract pod name if present (from pod-bound tokens)
	if podName, ok := claims["kubernetes.io/pod/name"].(string); ok {
		selectors["k8s:pod-name"] = podName
	}

	// Extract pod UID if present
	if podUID, ok := claims["kubernetes.io/pod/uid"].(string); ok {
		selectors["k8s:pod-uid"] = podUID
	}

	// Extract node name if present
	if nodeName, ok := claims["kubernetes.io/pod/node-name"].(string); ok {
		selectors["k8s:node-name"] = nodeName
	}

	v.logger.WithFields(logrus.Fields{"namespace": selectors["k8s:ns"], "service_account": selectors["k8s:sa"], "use_token_review": v.useTokenReview}).Info("Kubernetes attestation validated")

	return selectors, nil
}

// validateWithTokenReview validates token using Kubernetes TokenReview API
func (v *KubernetesValidator) validateWithTokenReview(ctx context.Context, token string) (map[string]interface{}, error) {
	// Create TokenReview request
	tokenReview := map[string]interface{}{
		"apiVersion": "authentication.k8s.io/v1",
		"kind":       "TokenReview",
		"spec": map[string]interface{}{
			"token": token,
		},
	}

	reqBody, err := json.Marshal(tokenReview)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TokenReview request: %w", err)
	}

	// Call Kubernetes API
	url := fmt.Sprintf("%s/apis/authentication.k8s.io/v1/tokenreviews", v.k8sAPIHost)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Use service account token for authentication
	if v.serviceAccountToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", v.serviceAccountToken))
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TokenReview API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TokenReview API returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode TokenReview response: %w", err)
	}

	// Check if token is authenticated
	status, ok := response["status"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid TokenReview response: missing status")
	}

	authenticated, ok := status["authenticated"].(bool)
	if !ok || !authenticated {
		return nil, fmt.Errorf("token is not authenticated")
	}

	// Extract user information
	user, ok := status["user"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid TokenReview response: missing user info")
	}

	// Parse username to extract service account info
	// Username format: system:serviceaccount:<namespace>:<sa-name>
	username, _ := user["username"].(string)
	claims := make(map[string]interface{})

	if strings.HasPrefix(username, "system:serviceaccount:") {
		parts := strings.Split(username, ":")
		if len(parts) >= 4 {
			namespace := parts[2]
			saName := parts[3]
			claims["kubernetes.io/serviceaccount/namespace"] = namespace
			claims["kubernetes.io/serviceaccount/service-account.name"] = saName
		}
	}

	// Extract UID
	if uid, ok := user["uid"].(string); ok {
		claims["sub"] = uid
	}

	// Extract extra info (pod name, pod UID, node name from pod-bound tokens)
	if extra, ok := user["extra"].(map[string]interface{}); ok {
		if podName, ok := extra["authentication.kubernetes.io/pod-name"].([]interface{}); ok && len(podName) > 0 {
			if pn, ok := podName[0].(string); ok {
				claims["kubernetes.io/pod/name"] = pn
			}
		}
		if podUID, ok := extra["authentication.kubernetes.io/pod-uid"].([]interface{}); ok && len(podUID) > 0 {
			if pu, ok := podUID[0].(string); ok {
				claims["kubernetes.io/pod/uid"] = pu
			}
		}
		if nodeName, ok := extra["authentication.kubernetes.io/node-name"].([]interface{}); ok && len(nodeName) > 0 {
			if nn, ok := nodeName[0].(string); ok {
				claims["kubernetes.io/pod/node-name"] = nn
			}
		}
	}

	return claims, nil
}

// parseJWT parses a JWT token and returns claims
// This is a simplified implementation for MVP
// Production should use a proper JWT library with signature verification
func (v *KubernetesValidator) parseJWT(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode payload (second part)
	payload := parts[1]

	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT claims: %w", err)
	}

	return claims, nil
}
