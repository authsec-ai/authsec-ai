package utils

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

// Pre-compiled regex patterns for validation (avoids re-compilation on every call)
var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	uuidRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	domainRegex   = regexp.MustCompile(`^([a-z0-9-]+\.)*[a-z0-9-]+\.[a-z]{2,}$`)
	phoneRegex    = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	otpRegex      = regexp.MustCompile(`^\d{6}$`)
	clientIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	tenantIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)
)

// Input validation utility to prevent injection attacks and enforce data quality

// ValidateEmail validates email format using RFC 5322 standard
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	
	email = strings.TrimSpace(email)
	if len(email) > 254 {
		return fmt.Errorf("email exceeds maximum length of 254 characters")
	}
	
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format")
	}
	
	return nil
}

// ValidateName validates user names (first name, last name, display name)
func ValidateName(name string, fieldName string, required bool) error {
	name = strings.TrimSpace(name)
	
	if required && name == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	
	if name == "" && !required {
		return nil // Optional field, empty is OK
	}
	
	if len(name) < 1 {
		return fmt.Errorf("%s must be at least 1 character", fieldName)
	}
	
	if len(name) > 100 {
		return fmt.Errorf("%s exceeds maximum length of 100 characters", fieldName)
	}
	
	// Check for only printable characters and common name characters
	for _, r := range name {
		if !unicode.IsPrint(r) {
			return fmt.Errorf("%s contains invalid characters", fieldName)
		}
	}
	
	// Prevent script injection attempts
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "<script") || 
	   strings.Contains(lowerName, "javascript:") ||
	   strings.Contains(lowerName, "onerror=") ||
	   strings.Contains(lowerName, "onload=") {
		return fmt.Errorf("%s contains potentially malicious content", fieldName)
	}
	
	return nil
}

// ValidateUsername validates username format
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}
	
	username = strings.TrimSpace(username)
	
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	
	if len(username) > 50 {
		return fmt.Errorf("username exceeds maximum length of 50 characters")
	}
	
	// Username should only contain alphanumeric characters, underscores, hyphens, and dots
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, dots, underscores, and hyphens")
	}
	
	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}
	
	if len(password) < 10 {
		return fmt.Errorf("password must be at least 10 characters")
	}
	
	if len(password) > 128 {
		return fmt.Errorf("password exceeds maximum length of 128 characters")
	}
	
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}
	
	return nil
}

// ValidateUUID validates UUID format
func ValidateUUID(uuidStr string, fieldName string) error {
	if uuidStr == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	
	if !uuidRegex.MatchString(uuidStr) {
		return fmt.Errorf("%s must be a valid UUID", fieldName)
	}
	
	return nil
}

// ValidateURL validates URL format
func ValidateURL(urlStr string, fieldName string, required bool) error {
	urlStr = strings.TrimSpace(urlStr)
	
	if required && urlStr == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	
	if urlStr == "" && !required {
		return nil
	}
	
	if len(urlStr) > 2048 {
		return fmt.Errorf("%s exceeds maximum length of 2048 characters", fieldName)
	}
	
	// Basic URL validation
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return fmt.Errorf("%s must start with http:// or https://", fieldName)
	}
	
	// Prevent common injection attempts
	lowerURL := strings.ToLower(urlStr)
	if strings.Contains(lowerURL, "javascript:") || 
	   strings.Contains(lowerURL, "data:") ||
	   strings.Contains(lowerURL, "vbscript:") {
		return fmt.Errorf("%s contains potentially malicious scheme", fieldName)
	}
	
	return nil
}

// ValidateDomain validates domain name format
func ValidateDomain(domain string, fieldName string) error {
	if domain == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	
	domain = strings.TrimSpace(domain)
	domain = strings.ToLower(domain)
	
	if len(domain) > 253 {
		return fmt.Errorf("%s exceeds maximum length of 253 characters", fieldName)
	}
	
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("%s must be a valid domain name", fieldName)
	}
	
	// Check for invalid patterns
	if strings.Contains(domain, "..") || 
	   strings.HasPrefix(domain, "-") || 
	   strings.HasSuffix(domain, "-") ||
	   strings.HasPrefix(domain, ".") ||
	   strings.HasSuffix(domain, ".") {
		return fmt.Errorf("%s contains invalid domain format", fieldName)
	}
	
	return nil
}

// ValidatePhoneNumber validates phone number format (E.164 format recommended)
func ValidatePhoneNumber(phone string, fieldName string, required bool) error {
	phone = strings.TrimSpace(phone)
	
	if required && phone == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	
	if phone == "" && !required {
		return nil
	}
	
	// Remove common formatting characters
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("%s must be a valid phone number in E.164 format (e.g., +12345678900)", fieldName)
	}
	
	return nil
}

// SanitizeInput removes potentially dangerous characters for logging/display
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Remove common control characters except newline and tab
	var sanitized strings.Builder
	for _, r := range input {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			sanitized.WriteRune(r)
		}
	}
	
	return sanitized.String()
}

// ValidateOTPCode validates OTP/TOTP code format
func ValidateOTPCode(code string) error {
	if code == "" {
		return fmt.Errorf("OTP code is required")
	}
	
	code = strings.TrimSpace(code)
	
	if !otpRegex.MatchString(code) {
		return fmt.Errorf("OTP code must be 6 digits")
	}
	
	return nil
}

// ValidateClientID validates OAuth client ID format
func ValidateClientID(clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client ID is required")
	}
	
	clientID = strings.TrimSpace(clientID)
	
	if len(clientID) < 10 || len(clientID) > 255 {
		return fmt.Errorf("client ID must be between 10 and 255 characters")
	}
	
	if !clientIDRegex.MatchString(clientID) {
		return fmt.Errorf("client ID contains invalid characters")
	}
	
	return nil
}

// ValidateTenantID validates tenant ID (typically UUID or alphanumeric)
func ValidateTenantID(tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	
	tenantID = strings.TrimSpace(tenantID)
	
	// Try UUID format first
	if uuidRegex.MatchString(tenantID) {
		return nil
	}
	
	// Otherwise allow alphanumeric with hyphens/underscores
	if !tenantIDRegex.MatchString(tenantID) {
		return fmt.Errorf("tenant ID must be a valid UUID or alphanumeric identifier (3-50 characters)")
	}
	
	return nil
}
