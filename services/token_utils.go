package services

import (
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/golang-jwt/jwt/v5"
)

// GenerateOIDCServiceToken generates a service-to-service JWT token for ICP authentication
func GenerateOIDCServiceToken() (string, error) {
	cfg := config.GetConfig()

	// Create simple claims for service-to-service auth
	claims := jwt.MapClaims{
		"user_id": "user-flow-service",
		"role":    "service",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // Token valid for 24 hours
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign with JWT_DEF_SECRET (same secret ICP uses for validation)
	tokenString, err := token.SignedString([]byte(cfg.JWTDefSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign service token: %w", err)
	}

	return tokenString, nil
}
