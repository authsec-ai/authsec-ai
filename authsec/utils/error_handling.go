package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Generic error messages to prevent information disclosure
const (
	// Authentication errors
	ErrMsgAuthenticationFailed = "Authentication failed"
	ErrMsgInvalidCredentials   = "Invalid credentials"
	ErrMsgUnauthorized         = "Unauthorized access"
	ErrMsgSessionExpired       = "Session expired"
	ErrMsgAccountLocked        = "Account has been locked"
	
	// Authorization errors
	ErrMsgForbidden            = "Access forbidden"
	ErrMsgInsufficientPermission = "Insufficient permissions"
	
	// Validation errors
	ErrMsgInvalidInput         = "Invalid input provided"
	ErrMsgValidationFailed     = "Validation failed"
	ErrMsgMissingRequiredField = "Missing required field"
	
	// Resource errors
	ErrMsgResourceNotFound     = "Resource not found"
	ErrMsgResourceExists       = "Resource already exists"
	ErrMsgResourceConflict     = "Resource conflict"
	
	// Server errors
	ErrMsgInternalServerError  = "Internal server error"
	ErrMsgServiceUnavailable   = "Service temporarily unavailable"
	ErrMsgDatabaseError        = "Database operation failed"
	
	// Rate limiting
	ErrMsgTooManyRequests      = "Too many requests, please try again later"
	
	// General errors
	ErrMsgBadRequest           = "Bad request"
	ErrMsgOperationFailed      = "Operation failed"
)

// ErrorResponse represents a generic error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// RespondWithError sends a generic error response and logs detailed error internally
func RespondWithError(c *gin.Context, statusCode int, publicMessage string, internalError error, context map[string]interface{}) {
	// Log detailed error internally for debugging
	logger := logrus.WithFields(logrus.Fields{
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
		"status":     statusCode,
	})
	
	// Add custom context fields
	for key, value := range context {
		logger = logger.WithField(key, value)
	}
	
	if internalError != nil {
		logger.WithError(internalError).Error("Request failed")
	} else {
		logger.Error("Request failed")
	}
	
	// Send generic error to client
	c.JSON(statusCode, ErrorResponse{
		Error: publicMessage,
	})
}

// RespondWithValidationError sends validation error with sanitized field details
func RespondWithValidationError(c *gin.Context, fieldErrors map[string]string) {
	// Log validation failures
	logrus.WithFields(logrus.Fields{
		"path":         c.Request.URL.Path,
		"field_errors": fieldErrors,
		"ip":           c.ClientIP(),
	}).Warn("Validation failed")
	
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   ErrMsgValidationFailed,
		Details: fieldErrors,
	})
}

// Common error response helpers

// RespondUnauthorized sends a generic unauthorized error
func RespondUnauthorized(c *gin.Context, internalError error) {
	RespondWithError(c, http.StatusUnauthorized, ErrMsgAuthenticationFailed, internalError, nil)
}

// RespondForbidden sends a generic forbidden error
func RespondForbidden(c *gin.Context) {
	RespondWithError(c, http.StatusForbidden, ErrMsgForbidden, nil, nil)
}

// RespondNotFound sends a generic not found error
func RespondNotFound(c *gin.Context) {
	RespondWithError(c, http.StatusNotFound, ErrMsgResourceNotFound, nil, nil)
}

// RespondBadRequest sends a generic bad request error
func RespondBadRequest(c *gin.Context, internalError error) {
	RespondWithError(c, http.StatusBadRequest, ErrMsgInvalidInput, internalError, nil)
}

// RespondInternalError sends a generic internal server error
func RespondInternalError(c *gin.Context, internalError error) {
	RespondWithError(c, http.StatusInternalServerError, ErrMsgInternalServerError, internalError, nil)
}

// RespondDatabaseError sends a generic database error
func RespondDatabaseError(c *gin.Context, internalError error, context map[string]interface{}) {
	RespondWithError(c, http.StatusInternalServerError, ErrMsgDatabaseError, internalError, context)
}

// RespondRateLimited sends a rate limit exceeded error
func RespondRateLimited(c *gin.Context) {
	RespondWithError(c, http.StatusTooManyRequests, ErrMsgTooManyRequests, nil, nil)
}

// RespondConflict sends a resource conflict error
func RespondConflict(c *gin.Context, publicMessage string) {
	RespondWithError(c, http.StatusConflict, publicMessage, nil, nil)
}

// SanitizeError removes sensitive information from error messages
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}
	
	// Check for specific error types and return generic equivalents
	errMsg := err.Error()
	
	// Database connection errors
	if containsAny(errMsg, []string{"connection refused", "dial tcp", "no such host", "timeout"}) {
		return errors.New("database connection failed")
	}
	
	// SQL errors
	if containsAny(errMsg, []string{"syntax error", "column", "table", "constraint", "duplicate key"}) {
		return errors.New("database operation failed")
	}
	
	// Authentication errors
	if containsAny(errMsg, []string{"password", "credentials", "authentication", "unauthorized"}) {
		return errors.New("authentication failed")
	}
	
	// Vault/secrets errors
	if containsAny(errMsg, []string{"vault", "secret", "token", "key"}) {
		return errors.New("configuration service error")
	}
	
	// Network errors
	if containsAny(errMsg, []string{"network", "http", "TLS", "certificate"}) {
		return errors.New("network error occurred")
	}
	
	// Generic error
	return errors.New("operation failed")
}

// containsAny checks if a string contains any of the given substrings (case-insensitive)
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
	       (s == substr || len(s) > len(substr) && 
	       anyIndexOf(s, substr) >= 0)
}

func anyIndexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// LogSecurityEvent logs security-related events with proper context
func LogSecurityEvent(eventType string, context map[string]interface{}, severity string) {
	logger := logrus.WithFields(logrus.Fields{
		"event_type": eventType,
		"severity":   severity,
	})
	
	for key, value := range context {
		logger = logger.WithField(key, value)
	}
	
	switch severity {
	case "critical":
		logger.Error(fmt.Sprintf("SECURITY: %s", eventType))
	case "high":
		logger.Warn(fmt.Sprintf("SECURITY: %s", eventType))
	default:
		logger.Info(fmt.Sprintf("SECURITY: %s", eventType))
	}
}
