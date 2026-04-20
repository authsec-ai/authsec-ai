package models

import "time"

// AttestationPolicy defines rules for workload attestation
type AttestationPolicy struct {
	ID              string                 `json:"id" db:"id"`
	TenantID        string                 `json:"tenant_id" db:"tenant_id"`
	Name            string                 `json:"name" db:"name"`
	Description     string                 `json:"description" db:"description"`
	AttestationType string                 `json:"attestation_type" db:"attestation_type"`
	SelectorRules   map[string]interface{} `json:"selector_rules" db:"selector_rules"`
	VaultRole       string                 `json:"vault_role" db:"vault_role"`
	TTL             int                    `json:"ttl" db:"ttl"`
	Priority        int                    `json:"priority" db:"priority"`
	Enabled         bool                   `json:"enabled" db:"enabled"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// MatchesSelectors checks if the policy matches given selectors
func (p *AttestationPolicy) MatchesSelectors(selectors map[string]string) bool {
	if !p.Enabled {
		return false
	}

	for key, expectedValue := range p.SelectorRules {
		actualValue, exists := selectors[key]
		if !exists {
			return false
		}

		expectedStr, ok := expectedValue.(string)
		if !ok {
			return false
		}

		// Wildcard matches any value
		if expectedStr == "*" {
			continue
		}

		if actualValue != expectedStr {
			return false
		}
	}

	return true
}
