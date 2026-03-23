package vault

// VaultClient interface defines the methods needed for vault operations
type VaultClient interface {
	WriteSecret(path string, data map[string]interface{}) error
	ReadSecret(path string) (map[string]interface{}, error)
	DeleteSecret(path string) error
}
