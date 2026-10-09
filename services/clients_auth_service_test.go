package services

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestNewClientsAuthServiceRequiresSecret(t *testing.T) {
	t.Setenv("JWT_SDK_SECRET", "")
	require.PanicsWithValue(t,
		"CRITICAL: JWT_SDK_SECRET environment variable is not set. Cannot authenticate requests.",
		func() { NewClientsAuthService() })
}

func TestClientsAuthServiceUsesEnvironmentSecret(t *testing.T) {
	key := make([]byte, 48)
	_, err := rand.Read(key)
	require.NoError(t, err)
	secret := base64.StdEncoding.EncodeToString(key)
	t.Setenv("JWT_SDK_SECRET", secret)
	service := NewClientsAuthService()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, ClientsTokenClaims{UserID: "test-user"})
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	claims, err := service.ValidateToken(signed)
	require.NoError(t, err)
	require.Equal(t, "test-user", claims.UserID)

	signed, err = token.SignedString([]byte("fake-other-key-for-tests-only"))
	require.NoError(t, err)
	_, err = service.ValidateToken(signed)
	require.Error(t, err)
}
