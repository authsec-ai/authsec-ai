package vault

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

// Client implements VaultClient interface
type Client struct {
	client *api.Client
}

// NewClient returns a VaultClient backed by HashiCorp Vault.
func NewClient(addr, token string) (VaultClient, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(token)
	return &Client{client: client}, nil
}

func (v *Client) WriteSecret(path string, data map[string]interface{}) error {
	// KV v2 requires data wrapped in a "data" field
	payload := map[string]interface{}{"data": data}
	_, err := v.client.Logical().Write(path, payload)
	if err != nil {
		return fmt.Errorf("vault write error: %w", err)
	}
	return nil
}

func (v *Client) ReadSecret(path string) (map[string]interface{}, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("vault read error: %w", err)
	}
	if secret == nil {
		return nil, fmt.Errorf("no secret found at path: %s", path)
	}
	// KV v2 wraps the actual secret within data.data
	if rawData, ok := secret.Data["data"]; ok {
		if nested, ok := rawData.(map[string]interface{}); ok {
			return nested, nil
		}
	}
	// Fallback: KV v1
	return secret.Data, nil
}

func (v *Client) DeleteSecret(path string) error {
	_, err := v.client.Logical().Delete(path)
	return err
}
