package vault

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"

	spireerrors "github.com/authsec-ai/authsec/internal/spire/errors"
)

// Client wraps the Vault API client for PKI operations
type Client struct {
	client      *api.Client
	logger      *logrus.Entry
	pkiBasePath string
	caRole      string
}

// Config holds Vault PKI client configuration
type Config struct {
	Address     string
	Token       string
	Namespace   string
	RoleID      string
	SecretID    string
	AuthMethod  string // token, approle
	PKIBasePath string
	CARole      string
	MaxRetries  int
}

// CertificateRequest represents a certificate signing request
type CertificateRequest struct {
	CSR        string
	CommonName string
	TTL        string
	AltNames   []string
	IPSANs     []string
	URISANs    []string
}

// CertificateResponse represents a signed certificate response
type CertificateResponse struct {
	Certificate       string
	CAChain           []string
	SerialNumber      string
	SHA256Fingerprint string
	ExpirationTime    time.Time
}

// PKIRoleConfig represents PKI role configuration
type PKIRoleConfig struct {
	AllowedDomains  []string
	AllowedURISANs  []string
	AllowSubdomains bool
	AllowAnyName    bool
	AllowURISANs    bool
	AllowIPSANs     bool
	MaxTTL          string
	TTL             string
	KeyType         string
	KeyBits         int
	RequireCN       bool
}

// NewClient creates a new Vault PKI client
func NewClient(config *Config, logger *logrus.Entry) (*Client, error) {
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = config.Address
	vaultConfig.MaxRetries = config.MaxRetries

	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
		logger.WithField("namespace", config.Namespace).Info("Vault namespace set")
	}

	switch config.AuthMethod {
	case "token":
		if config.Token == "" {
			return nil, fmt.Errorf("vault token is required")
		}
		client.SetToken(config.Token)
	case "approle":
		if err := authenticateWithAppRole(client, config.RoleID, config.SecretID); err != nil {
			return nil, fmt.Errorf("failed to authenticate with AppRole: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", config.AuthMethod)
	}

	if _, err = client.Sys().Health(); err != nil {
		logger.WithError(err).Warn("Vault health check failed")
	}

	logger.WithFields(logrus.Fields{
		"address":     config.Address,
		"auth_method": config.AuthMethod,
	}).Info("Vault PKI client initialized")

	return &Client{
		client:      client,
		logger:      logger,
		pkiBasePath: config.PKIBasePath,
		caRole:      config.CARole,
	}, nil
}

// NewClientFromExisting creates a Vault PKI client using an existing *api.Client
func NewClientFromExisting(apiClient *api.Client, logger *logrus.Entry, pkiBasePath, caRole string) *Client {
	return &Client{
		client:      apiClient,
		logger:      logger,
		pkiBasePath: pkiBasePath,
		caRole:      caRole,
	}
}

func authenticateWithAppRole(client *api.Client, roleID, secretID string) error {
	data := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}
	resp, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		return err
	}
	if resp.Auth == nil {
		return fmt.Errorf("no auth info in response")
	}
	client.SetToken(resp.Auth.ClientToken)
	return nil
}

// IssueCertificate signs a CSR using Vault PKI
func (c *Client) IssueCertificate(ctx context.Context, tenantMount, role string, req *CertificateRequest) (*CertificateResponse, error) {
	var path string
	if c.pkiBasePath != "" {
		path = fmt.Sprintf("%s/%s/sign/%s", c.pkiBasePath, tenantMount, role)
	} else {
		path = fmt.Sprintf("%s/sign/%s", tenantMount, role)
	}

	data := map[string]interface{}{
		"csr":         req.CSR,
		"common_name": req.CommonName,
		"ttl":         req.TTL,
	}
	if len(req.AltNames) > 0 {
		data["alt_names"] = req.AltNames
	}
	if len(req.IPSANs) > 0 {
		data["ip_sans"] = req.IPSANs
	}
	if len(req.URISANs) > 0 {
		data["uri_sans"] = req.URISANs
	}

	c.logger.WithFields(logrus.Fields{"path": path, "common_name": req.CommonName}).Debug("Issuing certificate")

	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		c.logger.WithField("path", path).WithError(err).Error("Failed to issue certificate")
		return nil, spireerrors.NewInternalError("Failed to issue certificate from Vault", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, spireerrors.NewInternalError("Empty response from Vault", nil)
	}

	cert, ok := secret.Data["certificate"].(string)
	if !ok {
		return nil, spireerrors.NewInternalError("Invalid certificate in response", nil)
	}

	serialNumber, _ := secret.Data["serial_number"].(string)

	var sha256Fingerprint string
	if block, _ := pem.Decode([]byte(cert)); block != nil {
		if parsedCert, parseErr := x509.ParseCertificate(block.Bytes); parseErr == nil {
			hash := sha256.Sum256(parsedCert.Raw)
			sha256Fingerprint = hex.EncodeToString(hash[:])
		}
	}

	var caChain []string
	if chain, ok := secret.Data["ca_chain"].([]interface{}); ok {
		for _, ca := range chain {
			if caStr, ok := ca.(string); ok {
				caChain = append(caChain, caStr)
			}
		}
	}

	expiration := time.Now().Add(24 * time.Hour)
	if ttlSec, ok := secret.Data["expiration"].(float64); ok {
		expiration = time.Unix(int64(ttlSec), 0)
	}

	return &CertificateResponse{
		Certificate:       cert,
		CAChain:           caChain,
		SerialNumber:      serialNumber,
		SHA256Fingerprint: sha256Fingerprint,
		ExpirationTime:    expiration,
	}, nil
}

// RevokeCertificate revokes a certificate
func (c *Client) RevokeCertificate(ctx context.Context, tenantMount, serialNumber string) error {
	var path string
	if c.pkiBasePath != "" {
		path = fmt.Sprintf("%s/%s/revoke", c.pkiBasePath, tenantMount)
	} else {
		path = fmt.Sprintf("%s/revoke", tenantMount)
	}

	data := map[string]interface{}{"serial_number": serialNumber}

	c.logger.WithFields(logrus.Fields{"path": path, "serial_number": serialNumber}).Info("Revoking certificate")

	_, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		c.logger.WithField("serial_number", serialNumber).WithError(err).Error("Failed to revoke certificate")
		return spireerrors.NewInternalError("Failed to revoke certificate in Vault", err)
	}
	return nil
}

// GetCABundle retrieves the CA certificate bundle
func (c *Client) GetCABundle(ctx context.Context, tenantMount string) (string, error) {
	var path string
	if c.pkiBasePath != "" {
		path = fmt.Sprintf("%s/%s/ca/pem", c.pkiBasePath, tenantMount)
	} else {
		path = fmt.Sprintf("%s/ca/pem", tenantMount)
	}

	c.logger.WithField("path", path).Debug("Getting CA bundle")

	req := c.client.NewRequest("GET", fmt.Sprintf("/v1/%s", path))
	resp, err := c.client.RawRequestWithContext(ctx, req)
	if err != nil {
		return "", spireerrors.NewInternalError("Failed to get CA bundle from Vault", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", spireerrors.NewNotFoundError("CA bundle not found", fmt.Errorf("vault returned status %d", resp.StatusCode))
	}

	pemBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", spireerrors.NewInternalError("Failed to read CA bundle", err)
	}
	return string(pemBytes), nil
}

// HealthCheck checks Vault health
func (c *Client) HealthCheck(ctx context.Context) error {
	health, err := c.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return err
	}
	if !health.Initialized {
		return fmt.Errorf("vault not initialized")
	}
	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}
	return nil
}

// EnablePKIEngine enables a PKI secrets engine at the specified path
func (c *Client) EnablePKIEngine(ctx context.Context, path string) error {
	c.logger.WithField("path", path).Info("Enabling PKI secrets engine")

	mountInput := &api.MountInput{
		Type:        "pki",
		Description: fmt.Sprintf("PKI engine for %s", path),
		Config:      api.MountConfigInput{MaxLeaseTTL: "87600h"},
	}

	if err := c.client.Sys().MountWithContext(ctx, path, mountInput); err != nil {
		c.logger.WithField("path", path).WithError(err).Error("Failed to enable PKI engine")
		return err
	}

	c.logger.WithField("path", path).Info("PKI engine enabled successfully")
	return nil
}

// GenerateRootCA generates a root CA certificate for a PKI mount
func (c *Client) GenerateRootCA(ctx context.Context, pkiMount, commonName, ttl string) (string, error) {
	path := fmt.Sprintf("%s/root/generate/internal", pkiMount)

	c.logger.WithFields(logrus.Fields{"path": path, "common_name": commonName}).Info("Generating root CA")

	data := map[string]interface{}{
		"common_name": commonName,
		"ttl":         ttl,
	}

	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		c.logger.WithField("path", path).WithError(err).Error("Failed to generate root CA")
		return "", err
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no data returned from root CA generation")
	}

	caCert, ok := secret.Data["certificate"].(string)
	if !ok {
		return "", fmt.Errorf("invalid CA certificate format")
	}

	c.logger.WithField("path", path).Info("Root CA generated successfully")
	return caCert, nil
}

// CreatePKIRole creates a role for certificate issuance
func (c *Client) CreatePKIRole(ctx context.Context, pkiMount, roleName string, config *PKIRoleConfig) error {
	path := fmt.Sprintf("%s/roles/%s", pkiMount, roleName)

	c.logger.WithFields(logrus.Fields{"path": path, "role": roleName}).Info("Creating PKI role")

	data := map[string]interface{}{
		"allowed_domains":   config.AllowedDomains,
		"allow_subdomains":  config.AllowSubdomains,
		"allow_any_name":    config.AllowAnyName,
		"allow_uri_sans":    config.AllowURISANs,
		"allow_ip_sans":     config.AllowIPSANs,
		"max_ttl":           config.MaxTTL,
		"ttl":               config.TTL,
		"key_type":          config.KeyType,
		"key_bits":          config.KeyBits,
		"require_cn":        config.RequireCN,
		"enforce_hostnames": false,
	}

	if len(config.AllowedURISANs) > 0 {
		data["allowed_uri_sans"] = config.AllowedURISANs
	}

	if _, err := c.client.Logical().WriteWithContext(ctx, path, data); err != nil {
		c.logger.WithField("path", path).WithError(err).Error("Failed to create PKI role")
		return err
	}

	c.logger.WithFields(logrus.Fields{"path": path, "role": roleName}).Info("PKI role created successfully")
	return nil
}

// ReadKVSecret reads a secret from KV v2 secrets engine
func (c *Client) ReadKVSecret(ctx context.Context, kvMount, secretPath string) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/data/%s", kvMount, secretPath)

	secret, err := c.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read KV secret at %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, nil
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected KV v2 response format at %s", path)
	}
	return data, nil
}

// WriteKVSecret writes a secret to KV v2 secrets engine
func (c *Client) WriteKVSecret(ctx context.Context, kvMount, secretPath string, data map[string]interface{}) error {
	path := fmt.Sprintf("%s/data/%s", kvMount, secretPath)

	wrappedData := map[string]interface{}{"data": data}

	if _, err := c.client.Logical().WriteWithContext(ctx, path, wrappedData); err != nil {
		return fmt.Errorf("failed to write KV secret at %s: %w", path, err)
	}
	return nil
}
