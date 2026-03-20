package services

// authmgr_service.go – in-process helpers extracted from the authmgr controller
// so that other packages (e.g. internal/hydra/models) can call authmgr logic
// directly without an HTTP round-trip.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	hydra "github.com/ory/hydra-client-go/v2"
)

// IssueOIDCJWT validates an OIDC token from Hydra and issues an authsec JWT.
// This is the core logic of the authmgr /oidcToken endpoint, extracted so it
// can be called in-process without going through the HTTP handler.
func IssueOIDCJWT(ctx context.Context, oidcToken string) (*sharedmodels.TokenResponse, error) {
	introspection, err := introspectOIDCToken(oidcToken)
	if err != nil || introspection == nil || introspection.Active == nil || !*introspection.Active {
		if err != nil {
			return nil, fmt.Errorf("invalid or inactive OIDC token: %w", err)
		}
		return nil, errors.New("invalid or inactive OIDC token")
	}

	required := []string{"provider", "provider_id", "user_id", "tenant_id", "email"}
	for _, field := range required {
		if v, _ := introspection.Ext[field]; v == nil || fmt.Sprintf("%v", v) == "" {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
	}

	safeStr := func(key string) string {
		v, _ := introspection.Ext[key]
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}

	provider := safeStr("provider")
	providerID := safeStr("provider_id")
	userID := safeStr("user_id")
	tenantID := safeStr("tenant_id")
	emailID := safeStr("email")

	clientID, projectID, err := authmgrLookupClient(ctx, tenantID, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tenant information: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id":   tenantID,
		"project_id":  projectID,
		"client_id":   clientID,
		"email_id":    emailID,
		"provider":    provider,
		"provider_id": providerID,
		"user_id":     userID,
		"token_type":  "oidc",
		"aud":         "authsec-api",
		"iat":         time.Now().Unix(),
		"exp":         time.Now().Add(24 * time.Hour).Unix(),
		"iss":         "authsec-ai/auth-manager",
		"jti":         uuid.New().String(),
	})

	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTDefSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &sharedmodels.TokenResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   24 * 60 * 60,
	}, nil
}

// introspectOIDCToken calls Hydra's introspect endpoint.
func introspectOIDCToken(token string) (*sharedmodels.Introspection, error) {
	hydraAdminURL := config.AppConfig.HydraAdminURL
	if hydraAdminURL == "" {
		return nil, errors.New("hydra admin URL not configured")
	}
	if strings.HasPrefix(hydraAdminURL, "http://") {
		hydraAdminURL = hydraAdminURL[7:]
	}
	cfg := hydra.NewConfiguration()
	cfg.Host = hydraAdminURL
	cfg.Scheme = "http"
	client := hydra.NewAPIClient(cfg)

	resp, httpResp, err := client.OAuth2API.IntrospectOAuth2Token(context.Background()).Token(token).Execute()
	if err != nil {
		return nil, fmt.Errorf("introspect: %w", err)
	}
	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("introspect status: %s", httpResp.Status)
	}
	return &sharedmodels.Introspection{
		Active:   &resp.Active,
		Scope:    *resp.Scope,
		ClientID: *resp.ClientId,
		Ext:      resp.Ext,
	}, nil
}

// authmgrLookupClient looks up client_id and project_id for an email within a tenant.
func authmgrLookupClient(ctx context.Context, tenantID, email string) (string, string, error) {
	if tenantID == "" || email == "" {
		return "", "", errors.New("tenantID and email required")
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return "", "", fmt.Errorf("parse tenantID: %w", err)
	}

	if config.DB != nil {
		var user sharedmodels.User
		if err := config.DB.WithContext(ctx).
			Select("client_id", "project_id").
			Where("tenant_id = ? AND email = ?", tid, email).
			First(&user).Error; err == nil {
			return user.ClientID.String(), user.ProjectID.String(), nil
		}
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		return "", "", fmt.Errorf("tenant db: %w", err)
	}
	var user sharedmodels.User
	if err := tenantDB.WithContext(ctx).
		Select("client_id", "project_id").
		Where("tenant_id = ? AND email = ?", tid, email).
		First(&user).Error; err != nil {
		return "", "", fmt.Errorf("client lookup: %w", err)
	}
	return user.ClientID.String(), user.ProjectID.String(), nil
}
