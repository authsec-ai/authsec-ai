package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TokenService handles JWT token management including revocation and refresh
type TokenService struct {
	redisClient *redis.Client
	jwtSecret   string
}

// NewTokenService creates a new token service instance
func NewTokenService(redisClient *redis.Client, jwtSecret string) *TokenService {
	return &TokenService{
		redisClient: redisClient,
		jwtSecret:   jwtSecret,
	}
}

// RefreshToken represents a refresh token stored in Redis
type RefreshToken struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"` // Access token expiry in seconds
	ExpiresAt    time.Time `json:"expires_at"`
}

// GenerateTokenPair generates both access and refresh tokens
func (ts *TokenService) GenerateTokenPair(userID, tenantID uuid.UUID, email string, additionalClaims map[string]interface{}) (*TokenPair, error) {
	ctx := context.Background()

	// Generate access token (short-lived: 15 minutes)
	accessTokenExpiry := time.Now().Add(15 * time.Minute)
	accessToken, err := ts.generateAccessToken(userID, tenantID, email, accessTokenExpiry, additionalClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (long-lived: 7 days)
	refreshToken, err := generateSecureRefreshToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshTokenExpiry := time.Now().Add(7 * 24 * time.Hour)

	// Store refresh token in Redis with user and tenant info
	refreshTokenKey := fmt.Sprintf("refresh_token:%s", refreshToken)
	refreshTokenData := fmt.Sprintf("%s:%s:%d", userID.String(), tenantID.String(), refreshTokenExpiry.Unix())

	err = ts.redisClient.Set(ctx, refreshTokenKey, refreshTokenData, 7*24*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Store user's active refresh token for revocation tracking
	userTokenKey := fmt.Sprintf("user_tokens:%s", userID.String())
	err = ts.redisClient.SAdd(ctx, userTokenKey, refreshToken).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to track user token: %w", err)
	}

	// Set expiry on user token set
	ts.redisClient.Expire(ctx, userTokenKey, 7*24*time.Hour)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(time.Until(accessTokenExpiry).Seconds()),
		ExpiresAt:    accessTokenExpiry,
	}, nil
}

// generateAccessToken creates a JWT access token
func (ts *TokenService) generateAccessToken(userID, tenantID uuid.UUID, email string, expiresAt time.Time, additionalClaims map[string]interface{}) (string, error) {
	claims := jwt.MapClaims{
		"user_id":   userID.String(),
		"tenant_id": tenantID.String(),
		"email":     email,
		"exp":       expiresAt.Unix(),
		"iat":       time.Now().Unix(),
		"jti":       uuid.New().String(), // JWT ID for tracking
	}

	// Add any additional claims
	for key, value := range additionalClaims {
		claims[key] = value
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(ts.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// RefreshAccessToken generates a new access token using a valid refresh token
func (ts *TokenService) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	ctx := context.Background()

	// Check if refresh token exists and is valid
	refreshTokenKey := fmt.Sprintf("refresh_token:%s", refreshToken)
	tokenData, err := ts.redisClient.Get(ctx, refreshTokenKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("refresh token not found or expired")
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve refresh token: %w", err)
	}

	// Parse token data (format: userID:tenantID:expiresAt)
	var userIDStr, tenantIDStr string
	var expiresAtUnix int64
	_, err = fmt.Sscanf(tokenData, "%s:%s:%d", &userIDStr, &tenantIDStr, &expiresAtUnix)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token data")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID in token")
	}

	// Check if token has expired
	if time.Now().Unix() > expiresAtUnix {
		// Clean up expired token
		ts.redisClient.Del(ctx, refreshTokenKey)
		return nil, fmt.Errorf("refresh token has expired")
	}

	// Generate new access token (refresh token remains the same)
	accessTokenExpiry := time.Now().Add(15 * time.Minute)
	accessToken, err := ts.generateAccessToken(userID, tenantID, "", accessTokenExpiry, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(time.Until(accessTokenExpiry).Seconds()),
		ExpiresAt:    accessTokenExpiry,
	}, nil
}

// RevokeToken revokes a specific refresh token
func (ts *TokenService) RevokeToken(refreshToken string) error {
	ctx := context.Background()

	// Get token data to find user
	refreshTokenKey := fmt.Sprintf("refresh_token:%s", refreshToken)
	tokenData, err := ts.redisClient.Get(ctx, refreshTokenKey).Result()
	if err == redis.Nil {
		return nil // Token already doesn't exist
	} else if err != nil {
		return fmt.Errorf("failed to retrieve token: %w", err)
	}

	// Parse user ID
	var userIDStr string
	_, err = fmt.Sscanf(tokenData, "%s:", &userIDStr)
	if err == nil {
		// Remove from user's token set
		userTokenKey := fmt.Sprintf("user_tokens:%s", userIDStr)
		ts.redisClient.SRem(ctx, userTokenKey, refreshToken)
	}

	// Delete the refresh token
	return ts.redisClient.Del(ctx, refreshTokenKey).Err()
}

// RevokeUserTokens revokes all refresh tokens for a specific user
func (ts *TokenService) RevokeUserTokens(userID uuid.UUID) error {
	ctx := context.Background()

	userTokenKey := fmt.Sprintf("user_tokens:%s", userID.String())

	// Get all refresh tokens for this user
	tokens, err := ts.redisClient.SMembers(ctx, userTokenKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user tokens: %w", err)
	}

	// Revoke each token
	for _, token := range tokens {
		refreshTokenKey := fmt.Sprintf("refresh_token:%s", token)
		ts.redisClient.Del(ctx, refreshTokenKey)
	}

	// Clear user's token set
	return ts.redisClient.Del(ctx, userTokenKey).Err()
}

// BlacklistAccessToken adds an access token to a blacklist (for immediate revocation)
// This should only be used for emergency revocations (e.g., security incident)
// as it requires storing all active tokens
func (ts *TokenService) BlacklistAccessToken(tokenString string, expiryDuration time.Duration) error {
	ctx := context.Background()

	// Parse token to get JTI
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return fmt.Errorf("token missing JTI claim")
	}

	// Add to blacklist with expiry matching token expiry
	blacklistKey := fmt.Sprintf("blacklist:%s", jti)
	return ts.redisClient.Set(ctx, blacklistKey, "1", expiryDuration).Err()
}

// IsTokenBlacklisted checks if an access token has been blacklisted
func (ts *TokenService) IsTokenBlacklisted(tokenString string) (bool, error) {
	ctx := context.Background()

	// Parse token to get JTI
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return false, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, fmt.Errorf("invalid token claims")
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		// If no JTI, can't check blacklist
		return false, nil
	}

	// Check blacklist
	blacklistKey := fmt.Sprintf("blacklist:%s", jti)
	exists, err := ts.redisClient.Exists(ctx, blacklistKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return exists > 0, nil
}

// generateSecureRefreshToken generates a cryptographically secure random token
func generateSecureRefreshToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CleanupExpiredTokens removes expired tokens from user token sets (run periodically)
func (ts *TokenService) CleanupExpiredTokens() error {
	ctx := context.Background()

	// Find all user token keys
	keys, err := ts.redisClient.Keys(ctx, "user_tokens:*").Result()
	if err != nil {
		return fmt.Errorf("failed to find user token keys: %w", err)
	}

	for _, userTokenKey := range keys {
		// Get all tokens for this user
		tokens, err := ts.redisClient.SMembers(ctx, userTokenKey).Result()
		if err != nil {
			continue
		}

		// Check each token
		for _, token := range tokens {
			refreshTokenKey := fmt.Sprintf("refresh_token:%s", token)
			exists, err := ts.redisClient.Exists(ctx, refreshTokenKey).Result()
			if err != nil || exists == 0 {
				// Token doesn't exist or error - remove from set
				ts.redisClient.SRem(ctx, userTokenKey, token)
			}
		}
	}

	return nil
}
