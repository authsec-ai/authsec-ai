package sdkmgr

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ServicesService handles external service credential retrieval via session validation.
// Translates sdk-manager's services_service.py — session-based credential lookup.
type ServicesService struct {
	SessionStore *OAuthSessionStore
}

// NewServicesService creates a new service instance.
func NewServicesService(store *OAuthSessionStore) *ServicesService {
	return &ServicesService{SessionStore: store}
}

// HealthCheck returns a simple health response.
func (s *ServicesService) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":    "healthy",
		"service":   "services-service",
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

// sessionInfo holds validated session data.
type sessionInfo struct {
	TenantID    string
	AccessToken string
	UserInfo    map[string]interface{}
}

// getSessionInfo validates the session and returns tenant_id + access_token.
func (s *ServicesService) getSessionInfo(sessionID string) (*sessionInfo, error) {
	session := s.SessionStore.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}
	if !session.IsTokenValid() {
		return nil, fmt.Errorf("session token expired")
	}

	info := &sessionInfo{}
	if session.TenantID != nil {
		info.TenantID = *session.TenantID
	}
	if session.AccessToken != nil {
		info.AccessToken = *session.AccessToken
	}
	info.UserInfo = session.GetUserInfoMap()
	return info, nil
}

// getServiceID looks up the service ID from the tenant database by name.
func (s *ServicesService) getServiceID(serviceName, tenantID string) (string, error) {
	db, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to connect to tenant DB: %w", err)
	}

	var result struct {
		ID string
	}
	err = db.Table("services").
		Where("created_by = ? AND name = ?", tenantID, serviceName).
		Order("created_at DESC").
		Select("id").
		First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("service '%s' not found for tenant %s", serviceName, tenantID)
		}
		return "", fmt.Errorf("failed to query service: %w", err)
	}
	return result.ID, nil
}

// GetServiceCredentials validates a session and retrieves external service credentials.
// In-process replacement for the Python HTTP call to /exsvc/services/{id}/credentials.
func (s *ServicesService) GetServiceCredentials(sessionID, serviceName string) (map[string]interface{}, error) {
	info, err := s.getSessionInfo(sessionID)
	if err != nil {
		return nil, err
	}
	if info.TenantID == "" {
		return nil, fmt.Errorf("no tenant_id in session")
	}
	if info.AccessToken == "" {
		return nil, fmt.Errorf("no access token in session")
	}

	serviceID, err := s.getServiceID(serviceName, info.TenantID)
	if err != nil {
		return nil, err
	}

	return s.fetchCredentials(serviceID, info.TenantID)
}

// fetchCredentials retrieves credentials for a service from the tenant DB.
func (s *ServicesService) fetchCredentials(serviceID, tenantID string) (map[string]interface{}, error) {
	db, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant DB: %w", err)
	}

	var svc struct {
		ID          string  `gorm:"column:id"`
		Name        string  `gorm:"column:name"`
		ServiceType string  `gorm:"column:service_type"`
		AuthType    string  `gorm:"column:auth_type"`
		URL         *string `gorm:"column:url"`
		Credentials *string `gorm:"column:credentials"`
		Metadata    *string `gorm:"column:metadata"`
	}

	err = db.Table("external_services").
		Where("id = ?", serviceID).
		First(&svc).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("external service %s not found", serviceID)
		}
		return nil, fmt.Errorf("failed to query external service: %w", err)
	}

	// Parse JSON fields.
	var credentials interface{}
	if svc.Credentials != nil {
		if jsonErr := json.Unmarshal([]byte(*svc.Credentials), &credentials); jsonErr != nil {
			logrus.WithError(jsonErr).Warn("failed to parse credentials JSON, returning raw")
			credentials = *svc.Credentials
		}
	}
	var metadata interface{}
	if svc.Metadata != nil {
		if jsonErr := json.Unmarshal([]byte(*svc.Metadata), &metadata); jsonErr != nil {
			metadata = map[string]interface{}{}
		}
	} else {
		metadata = map[string]interface{}{}
	}

	url := ""
	if svc.URL != nil {
		url = *svc.URL
	}

	return map[string]interface{}{
		"service_id":   svc.ID,
		"service_name": svc.Name,
		"service_type": svc.ServiceType,
		"auth_type":    svc.AuthType,
		"url":          url,
		"credentials":  credentials,
		"metadata":     metadata,
		"retrieved_at": time.Now().Format(time.RFC3339),
	}, nil
}

// GetServiceUserDetails validates a session and returns decoded JWT user details.
func (s *ServicesService) GetServiceUserDetails(sessionID string) map[string]interface{} {
	info, err := s.getSessionInfo(sessionID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if info.AccessToken == "" {
		return map[string]interface{}{"error": "no access token in session"}
	}

	claims, err := DecodeJWTPayload(info.AccessToken)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("error decoding JWT: %s", err.Error())}
	}
	return claims
}
