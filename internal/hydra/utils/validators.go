package hydrautils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail validates email format and length
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if len(email) > 320 {
		return fmt.Errorf("email exceeds maximum length of 320 characters")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// SanitizeString removes potentially dangerous characters and enforces length limits
func SanitizeString(input string, maxLength int) (string, error) {
	if input == "" {
		return "", nil
	}

	cleaned := strings.TrimSpace(input)
	cleaned = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, cleaned)

	if !utf8.ValidString(cleaned) {
		return "", fmt.Errorf("invalid UTF-8 encoding in input")
	}

	if utf8.RuneCountInString(cleaned) > maxLength {
		return "", fmt.Errorf("input exceeds maximum length of %d characters", maxLength)
	}
	return cleaned, nil
}

// ValidateName validates a user name field
func ValidateName(name string) (string, error) {
	sanitized, err := SanitizeString(name, 255)
	if err != nil {
		return "", fmt.Errorf("invalid name: %w", err)
	}
	if sanitized == "" {
		return "", fmt.Errorf("name cannot be empty")
	}
	return sanitized, nil
}

// ValidateUUID validates a UUID string
func ValidateUUID(uuidStr string, fieldName string) (uuid.UUID, error) {
	if uuidStr == "" {
		return uuid.Nil, fmt.Errorf("%s is required", fieldName)
	}
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s: must be a valid UUID", fieldName)
	}
	if parsed == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%s cannot be nil UUID", fieldName)
	}
	return parsed, nil
}

// ValidateSAMLAttribute validates and sanitizes a SAML attribute value
func ValidateSAMLAttribute(value interface{}, attributeName string, maxLength int) (string, error) {
	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("SAML attribute %s must be a string", attributeName)
	}
	sanitized, err := SanitizeString(strValue, maxLength)
	if err != nil {
		return "", fmt.Errorf("invalid SAML attribute %s: %w", attributeName, err)
	}
	return sanitized, nil
}

// ValidateSAMLEmail validates email from SAML assertion
func ValidateSAMLEmail(email interface{}) (string, error) {
	strEmail, ok := email.(string)
	if !ok {
		return "", fmt.Errorf("SAML email must be a string")
	}
	if err := ValidateEmail(strEmail); err != nil {
		return "", fmt.Errorf("invalid SAML email: %w", err)
	}
	return strings.ToLower(strings.TrimSpace(strEmail)), nil
}
