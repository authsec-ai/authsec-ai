package handlers

// contains checks if a string exists in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type AuthenticationResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Method   string `json:"method"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
}
