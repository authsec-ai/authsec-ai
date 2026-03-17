package platform

import (
	"net/http"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/services"
	"github.com/authsec-ai/authsec/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TokenController handles token management operations
type TokenController struct {
	tokenService *services.TokenService
}

// NewTokenController creates a new token controller
func NewTokenController(tokenService *services.TokenService) *TokenController {
	return &TokenController{
		tokenService: tokenService,
	}
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RevokeTokenRequest represents a token revocation request
type RevokeTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// @Summary Refresh access token
// @Description Generates a new access token using a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} services.TokenPair "New access token"
// @Failure 400 {object} utils.ErrorResponse "Invalid request"
// @Failure 401 {object} utils.ErrorResponse "Invalid or expired refresh token"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /auth/refresh [post]
func (tc *TokenController) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondBadRequest(c, err)
		return
	}
	
	// Generate new access token
	tokenPair, err := tc.tokenService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, utils.ErrMsgSessionExpired, err, map[string]interface{}{
			"operation": "refresh_token",
		})
		return
	}
	
	c.JSON(http.StatusOK, tokenPair)
}

// @Summary Revoke refresh token
// @Description Revokes a specific refresh token, invalidating it for future use
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RevokeTokenRequest true "Revoke token request"
// @Success 200 {object} map[string]string "Token revoked successfully"
// @Failure 400 {object} utils.ErrorResponse "Invalid request"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /auth/revoke [post]
func (tc *TokenController) RevokeToken(c *gin.Context) {
	var req RevokeTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondBadRequest(c, err)
		return
	}
	
	err := tc.tokenService.RevokeToken(req.RefreshToken)
	if err != nil {
		utils.RespondInternalError(c, err)
		return
	}
	
	utils.LogSecurityEvent("token_revoked", map[string]interface{}{
		"ip": c.ClientIP(),
	}, "info")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Token revoked successfully",
	})
}

// @Summary Logout (revoke all tokens)
// @Description Revokes all refresh tokens for the authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Logged out successfully"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /auth/logout [post]
func (tc *TokenController) Logout(c *gin.Context) {
	// Get user ID from JWT claims (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		utils.RespondUnauthorized(c, nil)
		return
	}
	
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		utils.RespondUnauthorized(c, err)
		return
	}
	
	// Revoke all user tokens
	err = tc.tokenService.RevokeUserTokens(userID)
	if err != nil {
		utils.RespondInternalError(c, err)
		return
	}
	
	utils.LogSecurityEvent("user_logout", map[string]interface{}{
		"user_id": userID.String(),
		"ip":      c.ClientIP(),
	}, "info")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// @Summary Blacklist access token (emergency revocation)
// @Description Immediately blacklists an access token (for security incidents only)
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Token blacklisted successfully"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /auth/blacklist [post]
func (tc *TokenController) BlacklistToken(c *gin.Context) {
	// Get token from Authorization header
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		utils.RespondUnauthorized(c, nil)
		return
	}
	
	// Remove "Bearer " prefix
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}
	
	// Blacklist for remaining token lifetime (max 15 minutes for access tokens)
	err := tc.tokenService.BlacklistAccessToken(tokenString, 15*60)
	if err != nil {
		utils.RespondInternalError(c, err)
		return
	}
	
	utils.LogSecurityEvent("access_token_blacklisted", map[string]interface{}{
		"ip": c.ClientIP(),
	}, "high")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Token blacklisted successfully",
	})
}

// InitializeTokenService initializes the token service with Redis connection
func InitializeTokenService() *services.TokenService {
	// Get Redis client from config
	redisClient := config.GetRedisClient()
	if redisClient == nil {
		panic("Redis client not initialized - token service requires Redis")
	}
	
	// Get JWT secret from environment
	jwtSecret := config.GetEnv("JWT_DEF_SECRET", "")
	if jwtSecret == "" {
		panic("JWT_DEF_SECRET not set - token service requires JWT secret")
	}
	
	return services.NewTokenService(redisClient, jwtSecret)
}
