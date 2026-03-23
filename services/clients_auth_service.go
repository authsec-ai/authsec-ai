package services

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ClientsTokenClaims holds JWT claims used by the clients service.
type ClientsTokenClaims struct {
	TenantID  string   `json:"tenant_id" mapstructure:"tenant_id"`
	ProjectID string   `json:"project_id" mapstructure:"project_id"`
	ClientID  string   `json:"client_id" mapstructure:"client_id"`
	EmailID   string   `json:"email_id" mapstructure:"email_id"`
	UserID    string   `json:"user_id" mapstructure:"user_id"`
	Scopes    []string `json:"scopes" mapstructure:"scopes"`
	Roles     []string `json:"roles" mapstructure:"roles"`
	Groups    []string `json:"groups" mapstructure:"groups"`
	Resources []string `json:"resources" mapstructure:"resources"`
	jwt.RegisteredClaims
}

// ClientsAuthService handles JWT token validation and role checking for the clients service.
type ClientsAuthService struct {
	jwtSecret []byte
}

// NewClientsAuthService creates a new auth service instance
func NewClientsAuthService() *ClientsAuthService {
	secret := os.Getenv("JWT_SDK_SECRET")
	if secret == "" {
		secret = "7f9b2a3c8e6d4f1b9a0c3e7d2f5b8a1c9e3d6f2a4b7c8e0d1f9a2b3c"
	}
	return &ClientsAuthService{
		jwtSecret: []byte(secret),
	}
}

// ValidateToken validates a JWT token and returns the claims
func (as *ClientsAuthService) ValidateToken(tokenString string) (*ClientsTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ClientsTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(*ClientsTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// ValidateOwnership validates if the user ID from token matches the owner ID
func (as *ClientsAuthService) ValidateOwnership(claims *ClientsTokenClaims, ownerID uuid.UUID) error {
	claimsUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID in token claims: %w", err)
	}

	if claimsUserID != ownerID {
		return fmt.Errorf("access denied: user does not own this resource")
	}

	return nil
}

// HasRole checks if the user has a specific role
func (as *ClientsAuthService) HasRole(claims *ClientsTokenClaims, requiredRole string) bool {
	for _, role := range claims.Roles {
		if role == requiredRole {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the required roles
func (as *ClientsAuthService) HasAnyRole(claims *ClientsTokenClaims, requiredRoles []string) bool {
	for _, userRole := range claims.Roles {
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}
	return false
}
