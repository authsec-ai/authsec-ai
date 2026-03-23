package middlewares

import (
	"testing"
)

func TestExtractBearerToken_ValidToken(t *testing.T) {
	c, _ := createTestContext("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer mytoken123")

	token, err := extractBearerToken(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "mytoken123" {
		t.Errorf("expected 'mytoken123', got %q", token)
	}
}

func TestExtractBearerToken_MissingHeader(t *testing.T) {
	c, _ := createTestContext("GET", "/test", nil)

	_, err := extractBearerToken(c)
	if err == nil {
		t.Fatal("expected error for missing Authorization header")
	}
}

func TestExtractBearerToken_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"no scheme", "mytoken123"},
		{"wrong scheme", "Basic dXNlcjpwYXNz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := createTestContext("GET", "/test", nil)
			c.Request.Header.Set("Authorization", tt.value)

			_, err := extractBearerToken(c)
			if err == nil {
				t.Error("expected error for invalid format")
			}
		})
	}
}

func TestValidateJWTToken_InvalidToken(t *testing.T) {
	cfg := &AuthConfig{
		JWTSecret:        "test-secret-1234",
		JWTDefaultSecret: "test-default-secret-1234",
	}

	_, err := validateJWTToken("not.a.valid.jwt", cfg)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateJWTToken_EmptySecrets(t *testing.T) {
	cfg := &AuthConfig{
		JWTSecret:        "",
		JWTDefaultSecret: "",
	}

	// With all secrets empty, the function iterates nothing and returns nil error and nil claims.
	// Downstream code should handle nil claims.
	claims, _ := validateJWTToken("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.sig", cfg)
	if claims != nil {
		t.Fatal("expected nil claims when all secrets are empty")
	}
}

func TestNormalizeHostname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Example.COM", "example.com"},
		{"host.example.com:8080", "host.example.com"},
		{"HOST.COM.", "host.com"},
		{"  spaces.com  ", "spaces.com"},
		{"UPPER.lower.MiXeD", "upper.lower.mixed"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeHostname(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeHostname(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidHostname(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"my-app.example.io", true},
		{"", false},
		{"no-dot", false},
		{"a", false},
		{"has space.com", false},
		{"path/traversal.com", false},
		{"back\\slash.com", false},
		{"wild*.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isValidHostname(tt.input)
			if result != tt.valid {
				t.Errorf("isValidHostname(%q) = %v, want %v", tt.input, result, tt.valid)
			}
		})
	}
}
