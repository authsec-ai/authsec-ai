package config

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
)

var VaultClient *api.Client

func InitVault(cfg *Config) error {
	config := api.DefaultConfig()

	vaultAddr := cfg.VaultAddr
	if vaultAddr != "" {
		config.Address = vaultAddr
	} else {
		// Default to localhost for development
		config.Address = "http://localhost:8200"
	}

	var err error
	VaultClient, err = api.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}

	// Set token if provided
	vaultToken := cfg.VaultToken
	if vaultToken != "" {
		VaultClient.SetToken(vaultToken)
		log.Printf("Vault token set: %s", maskToken(vaultToken))
	} else {
		log.Println("Warning: No Vault token provided")
	}

	// Test the connection to ensure Vault is actually running
	// Skip test if no token is provided (development mode)
	if cfg.VaultToken != "" {
		if err := testVaultConnection(cfg); err != nil {
			log.Printf("Warning: Vault connection test failed: %v", err)
			log.Println("Continuing without Vault - some features may not work")
			return nil
		}
	} else {
		log.Println("Skipping Vault connection test - no token provided (development mode)")
	}

	log.Println("Vault client initialized and connected successfully")
	return nil
}

// testVaultConnection verifies that Vault is running and accessible
func testVaultConnection(cfg *Config) error {
	if VaultClient == nil {
		return fmt.Errorf("vault client is nil")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	log.Printf("Testing connection to Vault at: %s", VaultClient.Address())

	// Try to get Vault health status
	resp, err := VaultClient.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if resp == nil {
		return fmt.Errorf("vault health response is nil")
	}

	log.Printf("Vault health status: initialized=%t, sealed=%t", resp.Initialized, resp.Sealed)

	// Check if Vault is sealed
	if resp.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	// Additional check: try to authenticate if token is provided
	if cfg.VaultToken != "" {
		// Try to lookup self to verify token is valid
		tokenInfo, err := VaultClient.Auth().Token().LookupSelfWithContext(ctx)
		if err != nil {
			return fmt.Errorf("vault token validation failed: %w", err)
		}

		if tokenInfo != nil && tokenInfo.Data != nil {
			if policies, ok := tokenInfo.Data["policies"].([]interface{}); ok {
				log.Printf("Token validated successfully. Policies: %v", policies)
			}
		}
	}

	return nil
}

// maskToken returns a masked version of the token for logging
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// GetSecretData extracts data from KV v2 secret response
func GetSecretData(secret *api.Secret) (map[string]interface{}, error) {
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret is nil or empty")
	}

	// For KV v2, the actual data is nested under "data" key
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("secret data is not in expected KV v2 format")
	}

	return data, nil
}

// HealthCheck performs a quick health check on the Vault connection
func HealthCheck() error {
	if VaultClient == nil {
		return fmt.Errorf("vault client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := VaultClient.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if resp.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	return nil
}

// SecretInVault retrieves a secret from Vault for the specified tenant, project, and client
func SecretInVault(tenantID, projectID, clientID string) (string, error) {
	if VaultClient == nil {
		return "", fmt.Errorf("vault client not initialized")
	}

	secretPath := fmt.Sprintf("kv/data/secret/%s/%s/%s", tenantID, projectID, clientID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secret, err := VaultClient.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault at path %s: %w", secretPath, err)
	}

	data, err := GetSecretData(secret)
	if err != nil {
		return "", fmt.Errorf("failed to extract secret data: %w", err)
	}

	secretID, ok := data["secret_id"].(string)
	if !ok || secretID == "" {
		return "", fmt.Errorf("secret_id not found in Vault at path %s", secretPath)
	}

	return secretID, nil
}

// SaveSecretToVault saves a secret to Vault under the specified tenant and project
func SaveSecretToVault(tenantID, projectID, clientID string) (string, error) {
	if VaultClient == nil {
		log.Println("ERROR: Vault client not initialized - VaultClient is nil")
		return "", fmt.Errorf("vault client not initialized")
	}

	// Check if token is set
	token := VaultClient.Token()
	if token == "" {
		log.Println("ERROR: Vault token not set - VaultClient.Token() returned empty string")
		return "", fmt.Errorf("vault token not configured")
	}

	// Debug: Check if we can access Vault
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// First verify token is valid by doing a self-lookup
	tokenInfo, err := VaultClient.Auth().Token().LookupSelfWithContext(ctx)
	if err != nil {
		log.Printf("Error: Vault token lookup failed: %v", err)
		log.Printf("Vault Address: %s", VaultClient.Address())
		log.Printf("Token (masked): %s", maskToken(VaultClient.Token()))
		
		// Provide more specific error guidance
		if strings.Contains(err.Error(), "permission denied") {
			log.Printf("ERROR: Token does not have 'read' capability on 'auth/token/lookup-self'")
			log.Printf("ACTION REQUIRED: Update Vault policy and regenerate token using:")
			log.Printf("  1. vault policy write user-flow-secrets k8s/vault-policy.hcl")
			log.Printf("  2. Run: scripts/setup_vault_policy.sh")
			log.Printf("  3. Update Kubernetes secret with new token")
		} else if strings.Contains(err.Error(), "invalid token") {
			log.Printf("ERROR: Token is invalid, expired, or revoked")
			log.Printf("ACTION REQUIRED: Generate new token:")
			log.Printf("  1. Run: scripts/setup_vault_policy.sh")
			log.Printf("  2. Update Kubernetes secret: kubectl patch secret user-flow-secrets -n authsec -p '{\"data\":{\"vault-token\":\"<base64-token>\"}}'")
		}
		
		return "", fmt.Errorf("failed to lookup token: %w", err)
	}
	
	if tokenInfo != nil && tokenInfo.Data != nil {
		if policies, ok := tokenInfo.Data["policies"].([]interface{}); ok {
			log.Printf("Vault token validated. Policies: %v", policies)
			
			// Verify the token has the required policy
			hasRequiredPolicy := false
			for _, p := range policies {
				if pStr, ok := p.(string); ok && (pStr == "user-flow-secrets" || pStr == "root") {
					hasRequiredPolicy = true
					break
				}
			}
			
			if !hasRequiredPolicy {
				log.Printf("Warning: Token does not have 'user-flow-secrets' policy")
				log.Printf("Current policies: %v", policies)
				log.Printf("This may cause permission issues when saving secrets")
			}
		}
	}

	secretID := uuid.New().String()

	// Use KV v2 secret engine which is mounted at "kv/" in this Vault instance
	// Path format: kv/data/<path> where "data" is required for KV v2 API
	// Full path: kv/data/secret/{tenant_id}/{project_id}/{client_id}
	secretPath := fmt.Sprintf("kv/data/secret/%s/%s/%s", tenantID, projectID, clientID)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()

	// For KV v2 Logical API, data must be wrapped under "data" key
	requestData := map[string]interface{}{
		"data": map[string]interface{}{
			"secret_id": secretID,
		},
	}

	// Use Logical API (same as read function) instead of KVv2 API
	_, err = VaultClient.Logical().WriteWithContext(ctx2, secretPath, requestData)
	if err != nil {
		log.Printf("Error writing secret to Vault at path %s: %v", secretPath, err)
		return "", fmt.Errorf("failed to write secret to Vault at path %s: %w", secretPath, err)
	}

	log.Printf("Successfully saved secret %s to Vault at path %s", secretID, secretPath)
	return secretID, nil
}

// SaveProviderSecretToVault stores provider credentials under a deterministic path.
func SaveProviderSecretToVault(tenantID, providerName string, data map[string]interface{}) error {
	if VaultClient == nil {
		return fmt.Errorf("vault client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("kv/data/oidc/providers/%s/%s", tenantID, providerName)
	payload := map[string]interface{}{"data": data}

	if _, err := VaultClient.Logical().WriteWithContext(ctx, path, payload); err != nil {
		return fmt.Errorf("failed to write provider secret to Vault at %s: %w", path, err)
	}
	return nil
}

// DeleteProviderSecretFromVault removes provider credentials from Vault.
func DeleteProviderSecretFromVault(tenantID, providerName string) error {
	if VaultClient == nil {
		return fmt.Errorf("vault client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metadataPath := fmt.Sprintf("kv/metadata/oidc/providers/%s/%s", tenantID, providerName)
	if _, err := VaultClient.Logical().DeleteWithContext(ctx, metadataPath); err != nil {
		return fmt.Errorf("failed to delete provider secret metadata from Vault at %s: %w", metadataPath, err)
	}
	return nil
}

// GetProviderSecretFromVault retrieves provider credentials from Vault.
func GetProviderSecretFromVault(tenantID, providerName string) (map[string]interface{}, error) {
	if VaultClient == nil {
		return nil, fmt.Errorf("vault client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	path := fmt.Sprintf("kv/data/oidc/providers/%s/%s", tenantID, providerName)
	secret, err := VaultClient.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read provider secret from Vault at %s: %w", path, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("secret not found")
	}
	data, err := GetSecretData(secret)
	if err != nil {
		return nil, err
	}
	return data, nil
}
