package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var spiffeComponentRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateSpiffeID validates a complete SPIFFE ID string.
// Format: spiffe://<trust-domain>/<path>
func ValidateSpiffeID(spiffeID string) error {
	if !strings.HasPrefix(spiffeID, "spiffe://") {
		return fmt.Errorf("SPIFFE ID must start with spiffe://")
	}

	remainder := strings.TrimPrefix(spiffeID, "spiffe://")
	parts := strings.SplitN(remainder, "/", 2)

	if len(parts) < 1 || parts[0] == "" {
		return fmt.Errorf("SPIFFE ID must include a trust domain")
	}

	if err := ValidateSpiffeComponent(parts[0], "trust domain"); err != nil {
		return err
	}

	if len(parts) == 2 && parts[1] != "" {
		segments := strings.Split(parts[1], "/")
		for _, seg := range segments {
			if seg == "" {
				return fmt.Errorf("SPIFFE ID path contains empty segment")
			}
			if err := ValidateSpiffeComponent(seg, "path segment"); err != nil {
				return err
			}
		}
	}

	if len(spiffeID) > 2048 {
		return fmt.Errorf("SPIFFE ID exceeds maximum length of 2048 characters")
	}

	return nil
}

// ValidateSpiffeComponent validates a single component of a SPIFFE ID
func ValidateSpiffeComponent(component, label string) error {
	if component == "" {
		return fmt.Errorf("%s cannot be empty", label)
	}
	if len(component) > 255 {
		return fmt.Errorf("%s exceeds maximum length of 255 characters", label)
	}
	if !spiffeComponentRe.MatchString(component) {
		return fmt.Errorf("%s contains invalid characters: %q (allowed: alphanumeric, hyphens, underscores, dots)", label, component)
	}
	return nil
}

// ValidateUUID validates that a string is a valid UUID.
func ValidateUUID(value, label string) error {
	if value == "" {
		return fmt.Errorf("%s cannot be empty", label)
	}
	if !uuidRe.MatchString(value) {
		return fmt.Errorf("%s is not a valid UUID: %q", label, value)
	}
	return nil
}
