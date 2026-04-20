package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the JWT claims structure
type JWTClaims struct {
	UserID      string   `json:"user_id,omitempty"`
	TenantID    string   `json:"tenant_id,omitempty"`
	Role        string   `json:"role,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// HasRole checks if the claims contain a specific role
func (c *JWTClaims) HasRole(role string) bool {
	if c.Role == role {
		return true
	}
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the claims contain a specific permission string
func (c *JWTClaims) HasPermission(perm string) bool {
	for _, p := range c.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// CanDelegate returns true if the caller has delegation rights
func (c *JWTClaims) CanDelegate() bool {
	return c.HasRole("admin") || c.HasPermission("jwt:delegate")
}

// JWTValidator handles JWT validation
type JWTValidator struct {
	publicKey  *rsa.PublicKey
	hmacSecret []byte
	isHMACMode bool
}

// NewJWTValidator creates a new JWT validator.
// Supports both RSA public keys (PEM format) and HMAC secrets.
func NewJWTValidator(keyOrSecret string) (*JWTValidator, error) {
	if keyOrSecret == "" {
		return nil, fmt.Errorf("JWT key or secret is required")
	}

	if strings.HasPrefix(strings.TrimSpace(keyOrSecret), "-----BEGIN") {
		block, _ := pem.Decode([]byte(keyOrSecret))
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block")
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}

		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA public key")
		}

		return &JWTValidator{publicKey: rsaPub, isHMACMode: false}, nil
	}

	return &JWTValidator{hmacSecret: []byte(keyOrSecret), isHMACMode: true}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (v *JWTValidator) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if v.isHMACMode {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v (expected HMAC)", token.Header["alg"])
			}
			return v.hmacSecret, nil
		}
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v (expected RSA)", token.Header["alg"])
		}
		return v.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
