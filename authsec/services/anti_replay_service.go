package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

// AntiReplayService handles challenge-response and nonce validation to prevent replay attacks
type AntiReplayService struct {
	redisClient *redis.Client
	ctx         context.Context
}

// NewAntiReplayService creates a new anti-replay service
func NewAntiReplayService(redisClient *redis.Client) *AntiReplayService {
	return &AntiReplayService{
		redisClient: redisClient,
		ctx:         context.Background(),
	}
}

// GenerateChallenge creates a new challenge for authentication
func (ars *AntiReplayService) GenerateChallenge() (*models.AuthChallenge, error) {
	challenge := &models.AuthChallenge{
		Challenge: uuid.New().String(),
		ExpiresAt: time.Now().Add(5 * time.Minute), // 5 minute expiry
		CreatedAt: time.Now(),
	}

	// Store challenge in Redis with TTL
	key := fmt.Sprintf("auth:challenge:%s", challenge.Challenge)
	err := ars.redisClient.Set(ars.ctx, key, "pending", 5*time.Minute).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store challenge: %w", err)
	}

	return challenge, nil
}

// ValidateChallenge verifies that a challenge exists and hasn't been used
func (ars *AntiReplayService) ValidateChallenge(challengeID string) error {
	if challengeID == "" {
		return fmt.Errorf("challenge is required")
	}

	key := fmt.Sprintf("auth:challenge:%s", challengeID)

	// Check if challenge exists
	status, err := ars.redisClient.Get(ars.ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("invalid or expired challenge")
	}
	if err != nil {
		return fmt.Errorf("failed to validate challenge: %w", err)
	}

	// Check if already used
	if status == "used" {
		return fmt.Errorf("challenge already used - potential replay attack detected")
	}

	// Mark challenge as used
	err = ars.redisClient.Set(ars.ctx, key, "used", 5*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to mark challenge as used: %w", err)
	}

	return nil
}

// ValidateNonce checks if a nonce has been used before
func (ars *AntiReplayService) ValidateNonce(nonce string) error {
	if nonce == "" {
		return fmt.Errorf("nonce is required")
	}

	key := fmt.Sprintf("auth:nonce:%s", nonce)

	// Check if nonce exists (has been used)
	exists, err := ars.redisClient.Exists(ars.ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check nonce: %w", err)
	}

	if exists > 0 {
		return fmt.Errorf("nonce already used - potential replay attack detected")
	}

	// Store nonce with TTL (2 minutes - should cover acceptable clock skew)
	err = ars.redisClient.Set(ars.ctx, key, time.Now().Unix(), 2*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to store nonce: %w", err)
	}

	return nil
}

// ValidateTimestamp ensures the request timestamp is within acceptable range
func (ars *AntiReplayService) ValidateTimestamp(timestamp int64, maxSkewSeconds int64) error {
	if timestamp == 0 {
		return fmt.Errorf("timestamp is required")
	}

	now := time.Now().Unix()
	skew := math.Abs(float64(now - timestamp))

	if skew > float64(maxSkewSeconds) {
		return fmt.Errorf("request timestamp too old or too far in future (skew: %.0f seconds, max: %d)", skew, maxSkewSeconds)
	}

	return nil
}

// ValidateRequestSignature verifies HMAC signature of request
func (ars *AntiReplayService) ValidateRequestSignature(data string, signature string, secret string) error {
	if signature == "" {
		return fmt.Errorf("signature is required")
	}

	// Compute HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison to prevent timing attacks
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid signature - potential tampering detected")
	}

	return nil
}

// ValidateLoginRequest performs comprehensive validation for login requests
func (ars *AntiReplayService) ValidateLoginRequest(req *models.SecureLoginRequest) error {
	// 1. Validate timestamp (60 second window)
	if err := ars.ValidateTimestamp(req.Timestamp, 60); err != nil {
		return fmt.Errorf("timestamp validation failed: %w", err)
	}

	// 2. Validate nonce (prevent duplicate requests)
	if err := ars.ValidateNonce(req.Nonce); err != nil {
		return fmt.Errorf("nonce validation failed: %w", err)
	}

	// 3. Validate challenge if provided (for challenge-response flow)
	if req.Challenge != "" {
		if err := ars.ValidateChallenge(req.Challenge); err != nil {
			return fmt.Errorf("challenge validation failed: %w", err)
		}
	}

	return nil
}

// ComputeRequestSignature generates HMAC signature for a request
// This is a helper for clients to compute signatures
func ComputeRequestSignature(email, password, nonce string, timestamp int64, secret string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", email, password, nonce, timestamp)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
