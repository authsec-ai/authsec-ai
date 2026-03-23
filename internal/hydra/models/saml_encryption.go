package hydramodels

import (
	"encoding/base64"
	"fmt"

	"github.com/authsec-ai/authsec/config"
)

// encryptPrivateKeyWithVault encrypts a private key using Vault transit engine
func encryptPrivateKeyWithVault(tenantID, privateKeyPEM string) (string, error) {
	if config.VaultClient == nil {
		return "", fmt.Errorf("Vault client not initialized")
	}

	encodedKey := base64.StdEncoding.EncodeToString([]byte(privateKeyPEM))

	data := map[string]interface{}{
		"plaintext": encodedKey,
		"context":   base64.StdEncoding.EncodeToString([]byte(tenantID)),
	}

	secret, err := config.VaultClient.Logical().Write("transit/encrypt/saml-sp-keys", data)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt with Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("Vault encryption returned nil")
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return "", fmt.Errorf("ciphertext not found in Vault response")
	}

	return "vault:" + ciphertext, nil
}

// decryptPrivateKeyWithVault decrypts a Vault-encrypted private key
func decryptPrivateKeyWithVault(tenantID, ciphertext string) (string, error) {
	if config.VaultClient == nil {
		return "", fmt.Errorf("Vault client not initialized")
	}

	if len(ciphertext) > 6 && ciphertext[:6] == "vault:" {
		ciphertext = ciphertext[6:]
	}

	data := map[string]interface{}{
		"ciphertext": ciphertext,
		"context":    base64.StdEncoding.EncodeToString([]byte(tenantID)),
	}

	secret, err := config.VaultClient.Logical().Write("transit/decrypt/saml-sp-keys", data)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt with Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("Vault decryption returned nil")
	}

	encodedPlaintext, ok := secret.Data["plaintext"].(string)
	if !ok {
		return "", fmt.Errorf("plaintext not found in Vault response")
	}

	plaintext, err := base64.StdEncoding.DecodeString(encodedPlaintext)
	if err != nil {
		return "", fmt.Errorf("failed to decode plaintext: %w", err)
	}

	return string(plaintext), nil
}
