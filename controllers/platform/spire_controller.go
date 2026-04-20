// Package controllers — SpireController: SPIRE Headless workload identity platform.
// Ported from spire-headless microservice (registry, attestation, oidc, policy services).
package platform

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	entryv1 "github.com/spiffe/spire-api-sdk/proto/spire/api/server/entry/v1"
	typespb "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	"github.com/authsec-ai/authsec/config"
)

// ===== MODELS =====

// SpireWorkload is the GORM model for registered SPIFFE workloads.
type SpireWorkload struct {
	ID       uint   `json:"-" gorm:"primaryKey"`
	SpiffeID string `json:"spiffe_id" gorm:"uniqueIndex"`
	Owner    string `json:"owner"`
}

func (SpireWorkload) TableName() string { return "spire_workloads" }

// WorkloadEntry is the GORM model for workload_entries in tenant databases.
// Stores the full workload registration with selectors, parent_id, and TTL
// so that SPIRE agents can look up and attest workloads.
type WorkloadEntry struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID     uuid.UUID       `json:"tenant_id" gorm:"type:uuid;not null"`
	SpiffeID     string          `json:"spiffe_id" gorm:"type:varchar(512);uniqueIndex;not null"`
	ParentID     string          `json:"parent_id" gorm:"type:varchar(512);not null"`
	Selectors    json.RawMessage `json:"selectors" gorm:"type:jsonb;not null"`
	TTL          int             `json:"ttl" gorm:"default:3600"`
	Admin        bool            `json:"admin" gorm:"default:false"`
	Downstream   bool            `json:"downstream" gorm:"default:false"`
	SpireEntryID *string         `json:"spire_entry_id,omitempty" gorm:"type:varchar(255)"`
	CreatedAt    time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

func (WorkloadEntry) TableName() string { return "workload_entries" }

// SpireOIDCToken stores OIDC token metadata for revocation tracking.
type SpireOIDCToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	JWTID     string    `json:"jti" gorm:"uniqueIndex"`
	Subject   string    `json:"subject"`
	SPIFFEID  string    `json:"spiffe_id"`
	TokenType string    `json:"token_type"`
	Audience  string    `json:"audience"`
	Scope     string    `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
}

func (SpireOIDCToken) TableName() string { return "spire_oidc_tokens" }

// SpirePolicy represents a policy document.
type SpirePolicy struct {
	ID          uint              `json:"id" gorm:"primaryKey"`
	Name        string            `json:"name" gorm:"uniqueIndex"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Engine      string            `json:"engine"`
	Rules       []SpirePolicyRule `json:"rules" gorm:"foreignKey:PolicyID;constraint:OnDelete:CASCADE"`
	Metadata    SpirePolicyMeta   `json:"metadata" gorm:"embedded"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Active      bool              `json:"active"`
}

func (SpirePolicy) TableName() string { return "spire_policies" }

// SpirePolicyRule represents individual policy rules.
type SpirePolicyRule struct {
	ID         uint                   `json:"id" gorm:"primaryKey"`
	PolicyID   uint                   `json:"policy_id"`
	Name       string                 `json:"name"`
	Effect     string                 `json:"effect"`
	Priority   int                    `json:"priority"`
	Subjects   []SpirePolicySubject   `json:"subjects" gorm:"foreignKey:RuleID;constraint:OnDelete:CASCADE"`
	Resources  []SpirePolicyResource  `json:"resources" gorm:"foreignKey:RuleID;constraint:OnDelete:CASCADE"`
	Actions    []SpirePolicyAction    `json:"actions" gorm:"foreignKey:RuleID;constraint:OnDelete:CASCADE"`
	Conditions []SpirePolicyCondition `json:"conditions" gorm:"foreignKey:RuleID;constraint:OnDelete:CASCADE"`
	Attributes map[string]interface{} `json:"attributes" gorm:"serializer:json"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

func (SpirePolicyRule) TableName() string { return "spire_policy_rules" }

// SpirePolicySubject, SpirePolicyResource, SpirePolicyAction, SpirePolicyCondition.
type SpirePolicySubject struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	RuleID  uint   `json:"rule_id"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Pattern string `json:"pattern"`
}

func (SpirePolicySubject) TableName() string { return "spire_policy_subjects" }

type SpirePolicyResource struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	RuleID  uint   `json:"rule_id"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Pattern string `json:"pattern"`
}

func (SpirePolicyResource) TableName() string { return "spire_policy_resources" }

type SpirePolicyAction struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	RuleID uint   `json:"rule_id"`
	Type   string `json:"type"`
	Value  string `json:"value"`
}

func (SpirePolicyAction) TableName() string { return "spire_policy_actions" }

type SpirePolicyCondition struct {
	ID       uint                   `json:"id" gorm:"primaryKey"`
	RuleID   uint                   `json:"rule_id"`
	Type     string                 `json:"type"`
	Operator string                 `json:"operator"`
	Key      string                 `json:"key"`
	Value    string                 `json:"value"`
	Metadata map[string]interface{} `json:"metadata" gorm:"serializer:json"`
}

func (SpirePolicyCondition) TableName() string { return "spire_policy_conditions" }

// SpirePolicyMeta holds policy metadata.
type SpirePolicyMeta struct {
	Author      string            `json:"author"`
	Tags        []string          `json:"tags" gorm:"serializer:json"`
	Labels      map[string]string `json:"labels" gorm:"serializer:json"`
	Annotations map[string]string `json:"annotations" gorm:"serializer:json"`
}

// SpirePolicyEvaluation is the input for policy evaluation.
type SpirePolicyEvaluation struct {
	Subject   string                 `json:"subject" binding:"required"`
	Resource  string                 `json:"resource" binding:"required"`
	Action    string                 `json:"action" binding:"required"`
	Context   map[string]interface{} `json:"context"`
	Timestamp time.Time              `json:"timestamp"`
}

// SpirePolicyResult is the output of policy evaluation.
type SpirePolicyResult struct {
	Decision    string                 `json:"decision"`
	Reason      string                 `json:"reason"`
	MatchedRule *SpirePolicyRule       `json:"matched_rule,omitempty"`
	Context     map[string]interface{} `json:"context"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
	RequestID   string                 `json:"request_id"`
}

// SpireAuditLog stores policy audit entries.
type SpireAuditLog struct {
	ID        uint                   `json:"id" gorm:"primaryKey"`
	RequestID string                 `json:"request_id"`
	Subject   string                 `json:"subject"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Decision  string                 `json:"decision"`
	Reason    string                 `json:"reason"`
	PolicyID  *uint                  `json:"policy_id,omitempty"`
	RuleID    *uint                  `json:"rule_id,omitempty"`
	Context   map[string]interface{} `json:"context" gorm:"serializer:json"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Timestamp time.Time              `json:"timestamp"`
}

func (SpireAuditLog) TableName() string { return "spire_audit_logs" }

// SpireRoleBinding represents RBAC role bindings.
type SpireRoleBinding struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Subject   string    `json:"subject"`
	Role      string    `json:"role"`
	Resource  string    `json:"resource"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SpireRoleBinding) TableName() string { return "spire_role_bindings" }

// SpireAllModels lists all GORM models for AutoMigrate.
var SpireAllModels = []interface{}{
	&SpireWorkload{},
	&SpireOIDCToken{},
	&SpirePolicy{},
	&SpirePolicyRule{},
	&SpirePolicySubject{},
	&SpirePolicyResource{},
	&SpirePolicyAction{},
	&SpirePolicyCondition{},
	&SpireAuditLog{},
	&SpireRoleBinding{},
}

// ===== OIDC TYPES =====

type spireOIDCConfig struct {
	IssuerURL   string
	TokenExpiry time.Duration
}

type spireOIDCProvider struct {
	cfg        *spireOIDCConfig
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string
	db         *gorm.DB
}

type spireTokenClaims struct {
	Subject   string                 `json:"sub"`
	Issuer    string                 `json:"iss"`
	Audience  []string               `json:"aud"`
	ExpiresAt int64                  `json:"exp"`
	IssuedAt  int64                  `json:"iat"`
	NotBefore int64                  `json:"nbf"`
	JWTID     string                 `json:"jti"`
	SPIFFEID  string                 `json:"spiffe_id,omitempty"`
	Claims    map[string]interface{} `json:"claims,omitempty"`
	jwt.RegisteredClaims
}

type spireJWTSVIDClaims struct {
	Subject   string   `json:"sub"`
	Audience  []string `json:"aud"`
	Issuer    string   `json:"iss"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	NotBefore int64    `json:"nbf"`
	JWTID     string   `json:"jti"`
	SPIFFEID  string   `json:"spiffe_id"`
	jwt.RegisteredClaims
}

type spireCloudTokenRequest struct {
	Provider     string `json:"provider" binding:"required"`
	Audience     string `json:"audience" binding:"required"`
	Scope        string `json:"scope,omitempty"`
	RoleARN      string `json:"role_arn,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
	ServiceEmail string `json:"service_email,omitempty"`
}

type spireCloudTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

type spireTokenExchangeResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int64  `json:"expires_in"`
	Scope           string `json:"scope,omitempty"`
}

// ===== CONTROLLER =====

// SpireController is the merged SPIRE headless platform controller.
type SpireController struct {
	db           *gorm.DB
	entryClient  entryv1.EntryClient
	grpcConn     *grpc.ClientConn
	oidcProvider *spireOIDCProvider
	policyEngine string
	trustDomain  string
}

// sharedSpireController is the singleton used by in-process callers (e.g. clients controller).
var sharedSpireController *SpireController

// SetSharedSpireController stores the singleton so other controllers can create entries in-process.
func SetSharedSpireController(sc *SpireController) { sharedSpireController = sc }

// RegisterAgentWorkload creates a SPIRE workload entry for an AI agent.
// It writes to both the master DB (spire_workloads) and the tenant DB (workload_entries),
// and optionally creates a SPIRE entry via gRPC if the server is connected.
// Returns the generated full SPIFFE ID.
func RegisterAgentWorkload(tenantID, clientID, agentType, platform string, selectors map[string]string) (string, error) {
	sc := sharedSpireController
	if sc == nil {
		return "", fmt.Errorf("SPIRE controller not initialized")
	}

	spiffeID := fmt.Sprintf("/tenants/%s/agents/%s/%s", tenantID, agentType, clientID)
	fullSpiffeID := fmt.Sprintf("spiffe://%s%s", sc.trustDomain, spiffeID)
	parentID := fmt.Sprintf("spiffe://%s/tenants/%s/agent", sc.trustDomain, tenantID)

	// Build SPIRE selectors from the user-supplied key-value pairs
	var spireSelectors []*typespb.Selector
	for key, value := range selectors {
		// Split key like "k8s:ns" into type="k8s" value="ns:<user-value>"
		// or "k8s:pod-label:app" into type="k8s" value="pod-label:app:<user-value>"
		parts := strings.SplitN(key, ":", 2)
		selectorType := parts[0]
		selectorKey := ""
		if len(parts) > 1 {
			selectorKey = parts[1]
		}
		spireSelectors = append(spireSelectors, &typespb.Selector{
			Type:  selectorType,
			Value: fmt.Sprintf("%s:%s", selectorKey, value),
		})
	}

	// Fallback: if no selectors provided, use a default owner-based selector
	if len(spireSelectors) == 0 {
		spireSelectors = []*typespb.Selector{
			{Type: "k8s", Value: fmt.Sprintf("pod-label:owner:%s", clientID)},
		}
	}

	// Build selectors JSON for the workload_entries record
	selectorMap := map[string]string{
		"authsec:client_id":  clientID,
		"authsec:agent_type": agentType,
		"authsec:tenant_id":  tenantID,
	}
	for k, v := range selectors {
		selectorMap[k] = v
	}
	selectorsJSON, err := json.Marshal(selectorMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal selectors: %w", err)
	}

	// Save workload record to master DB (spire_workloads)
	w := SpireWorkload{
		SpiffeID: spiffeID,
		Owner:    clientID,
	}
	if err := sc.db.Create(&w).Error; err != nil {
		return "", fmt.Errorf("failed to save workload record: %w", err)
	}

	// Save workload entry to tenant DB (workload_entries)
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return "", fmt.Errorf("invalid tenant_id: %w", err)
	}

	tenantDB, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		log.Printf("[SPIRE] Warning: failed to connect to tenant DB for workload entry: %v", err)
		// Continue — master record and gRPC entry are still valuable
	} else {
		entry := WorkloadEntry{
			ID:        uuid.New(),
			TenantID:  tenantUUID,
			SpiffeID:  fullSpiffeID,
			ParentID:  parentID,
			Selectors: selectorsJSON,
			TTL:       3600,
		}
		if err := tenantDB.Create(&entry).Error; err != nil {
			log.Printf("[SPIRE] Warning: failed to save workload entry to tenant DB: %v", err)
			// Continue — don't fail the whole registration
		} else {
			log.Printf("[SPIRE] Workload entry saved to tenant DB: id=%s spiffe_id=%s", entry.ID, entry.SpiffeID)
		}
	}

	// Create SPIRE entry via gRPC (if server is connected)
	if sc.entryClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		entry := &typespb.Entry{
			SpiffeId:    &typespb.SPIFFEID{TrustDomain: sc.trustDomain, Path: spiffeID},
			ParentId:    &typespb.SPIFFEID{TrustDomain: sc.trustDomain, Path: fmt.Sprintf("/tenants/%s/agent", tenantID)},
			Selectors:   spireSelectors,
			X509SvidTtl: 3600,
			StoreSvid:   true,
		}
		_, err := sc.entryClient.BatchCreateEntry(ctx, &entryv1.BatchCreateEntryRequest{
			Entries: []*typespb.Entry{entry},
		})
		if err != nil {
			// Rollback DB records
			sc.db.Delete(&w)
			log.Printf("[SPIRE] SPIRE gRPC entry creation failed, rolled back DB records: %v", err)
			return "", fmt.Errorf("SPIRE entry creation failed: %w", err)
		}
	} else {
		log.Printf("[SPIRE] Warning: SPIRE gRPC entryClient is nil — workload entry saved to DB but not registered with SPIRE server. Set SPIRE_SERVER_ADDR to enable.")
	}

	log.Printf("[SPIRE] Agent workload registered: spiffe_id=%s tenant=%s client=%s", fullSpiffeID, tenantID, clientID)
	return fullSpiffeID, nil
}

// NewSpireController creates and initialises the SPIRE controller.
func NewSpireController() *SpireController {
	sc := &SpireController{
		db:           config.DB,
		policyEngine: spireGetenv("POLICY_ENGINE", "hybrid"),
		trustDomain:  config.AppConfig.SpiffeTrustDomain,
	}
	if sc.trustDomain == "" {
		sc.trustDomain = spireGetenv("SPIRE_TRUST_DOMAIN", "example.org")
	}

	// Initialize SPIRE entry client (optional — degrades gracefully)
	spireAddr := spireGetenv("SPIRE_SERVER_ADDR", "spire-server:8081")
	spiffeSocket := "/run/spire/sockets/workload_api.sock"
	ctx := context.Background()

	var conn *grpc.ClientConn
	var err error
	if _, statErr := os.Stat(spiffeSocket); statErr == nil {
		source, srcErr := workloadapi.NewX509Source(ctx)
		if srcErr == nil {
			tlsCfg := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
			conn, err = grpc.NewClient(spireAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
			if err != nil {
				log.Printf("[spire] mTLS gRPC connect failed: %v — falling back to insecure", err)
			} else {
				log.Printf("[spire] Using SPIFFE mTLS for SPIRE server connection")
			}
		}
	}
	if conn == nil {
		conn, err = grpc.NewClient(spireAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("[spire] Warning: failed to create SPIRE gRPC client: %v", err)
		} else {
			log.Printf("[spire] Using insecure gRPC for SPIRE server connection")
		}
	}
	if conn != nil {
		sc.grpcConn = conn
		sc.entryClient = entryv1.NewEntryClient(conn)
	}

	// Initialize OIDC provider
	issuerURL := config.AppConfig.SpiffeOIDCIssuer
	if issuerURL == "" {
		issuerURL = spireGetenv("OIDC_ISSUER_URL", "https://spire-headless.example.org")
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Printf("[spire] Warning: failed to generate RSA key: %v", err)
	} else {
		sc.oidcProvider = &spireOIDCProvider{
			cfg:        &spireOIDCConfig{IssuerURL: issuerURL, TokenExpiry: time.Hour},
			privateKey: privateKey,
			publicKey:  &privateKey.PublicKey,
			keyID:      uuid.New().String(),
			db:         config.DB,
		}
	}

	// Start SPIRE reconciliation loop in background
	if sc.entryClient != nil {
		go sc.reconcileEntries(context.Background())
	}

	// Load default policies
	sc.loadDefaultPolicies()

	return sc
}

// reconcileEntries periodically syncs workloads DB ↔ SPIRE server.
func (sc *SpireController) reconcileEntries(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if sc.db == nil || sc.entryClient == nil {
				continue
			}
			var workloads []SpireWorkload
			if err := sc.db.Find(&workloads).Error; err != nil {
				log.Printf("[spire] reconcile: DB error: %v", err)
				continue
			}
			resp, err := sc.entryClient.ListEntries(ctx, &entryv1.ListEntriesRequest{})
			if err != nil {
				log.Printf("[spire] reconcile: list entries error: %v", err)
				continue
			}
			spireMap := make(map[string]*typespb.Entry)
			for _, e := range resp.Entries {
				spireMap[e.SpiffeId.Path] = e
			}
			for _, w := range workloads {
				if _, exists := spireMap[w.SpiffeID]; !exists {
					_, err := sc.entryClient.BatchCreateEntry(ctx, &entryv1.BatchCreateEntryRequest{
						Entries: []*typespb.Entry{spireEntryFromWorkload(w, sc.trustDomain)},
					})
					if err != nil {
						log.Printf("[spire] reconcile: create entry %s: %v", w.SpiffeID, err)
					}
				}
			}
			for path, e := range spireMap {
				var w SpireWorkload
				if err := sc.db.Where("spiffe_id = ?", path).First(&w).Error; err != nil {
					_, err := sc.entryClient.BatchDeleteEntry(ctx, &entryv1.BatchDeleteEntryRequest{
						Ids: []string{e.Id},
					})
					if err != nil {
						log.Printf("[spire] reconcile: delete stale entry %s: %v", path, err)
					}
				}
			}
		}
	}
}

func spireEntryFromWorkload(w SpireWorkload, trustDomain string) *typespb.Entry {
	return &typespb.Entry{
		SpiffeId:    &typespb.SPIFFEID{TrustDomain: trustDomain, Path: w.SpiffeID},
		Selectors:   []*typespb.Selector{{Type: "k8s", Value: fmt.Sprintf("pod-label:owner:%s", w.Owner)}},
		X509SvidTtl: 3600,
		Downstream:  false,
		Admin:       false,
		StoreSvid:   true,
	}
}

// ===== REGISTRY HANDLERS =====

// RegisterWorkload registers a new SPIFFE workload.
func (sc *SpireController) RegisterWorkload(c *gin.Context) {
	var w SpireWorkload
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := sc.db.Create(&w).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if sc.entryClient != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		_, err := sc.entryClient.BatchCreateEntry(ctx, &entryv1.BatchCreateEntryRequest{
			Entries: []*typespb.Entry{spireEntryFromWorkload(w, sc.trustDomain)},
		})
		if err != nil {
			sc.db.Delete(&w)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("SPIRE entry creation failed: %v", err)})
			return
		}
	}
	c.JSON(http.StatusCreated, w)
}

// UpdateWorkload updates an existing workload.
func (sc *SpireController) UpdateWorkload(c *gin.Context) {
	spiffeID := c.Param("spiffe_id")
	var w SpireWorkload
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var existing SpireWorkload
	if err := sc.db.Where("spiffe_id = ?", spiffeID).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workload not found"})
		return
	}
	w.SpiffeID = spiffeID
	if err := sc.db.Model(&existing).Updates(w).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sc.entryClient != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		resp, err := sc.entryClient.ListEntries(ctx, &entryv1.ListEntriesRequest{
			Filter: &entryv1.ListEntriesRequest_Filter{
				BySpiffeId: &typespb.SPIFFEID{TrustDomain: sc.trustDomain, Path: spiffeID},
			},
		})
		if err == nil && len(resp.Entries) > 0 {
			entry := resp.Entries[0]
			entry.Selectors = []*typespb.Selector{{Type: "k8s", Value: fmt.Sprintf("pod-label:owner:%s", w.Owner)}}
			_, _ = sc.entryClient.BatchUpdateEntry(ctx, &entryv1.BatchUpdateEntryRequest{Entries: []*typespb.Entry{entry}})
		}
	}
	c.JSON(http.StatusOK, w)
}

// DeleteWorkload removes a workload.
func (sc *SpireController) DeleteWorkload(c *gin.Context) {
	spiffeID := c.Param("spiffe_id")
	var w SpireWorkload
	if err := sc.db.Where("spiffe_id = ?", spiffeID).First(&w).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workload not found"})
		return
	}
	if sc.entryClient != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		resp, err := sc.entryClient.ListEntries(ctx, &entryv1.ListEntriesRequest{
			Filter: &entryv1.ListEntriesRequest_Filter{
				BySpiffeId: &typespb.SPIFFEID{TrustDomain: sc.trustDomain, Path: spiffeID},
			},
		})
		if err == nil && len(resp.Entries) > 0 {
			_, _ = sc.entryClient.BatchDeleteEntry(ctx, &entryv1.BatchDeleteEntryRequest{
				Ids: []string{resp.Entries[0].Id},
			})
		}
	}
	if err := sc.db.Delete(&w).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Workload deleted"})
}

// ListWorkloads lists all registered workloads.
func (sc *SpireController) ListWorkloads(c *gin.Context) {
	var workloads []SpireWorkload
	if err := sc.db.Find(&workloads).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, workloads)
}

// ===== OIDC HANDLERS =====

// OIDCDiscovery serves the OIDC discovery document.
func (sc *SpireController) OIDCDiscovery(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"issuer":                                p.cfg.IssuerURL,
		"authorization_endpoint":                p.cfg.IssuerURL + "/oidc/auth",
		"token_endpoint":                        p.cfg.IssuerURL + "/oidc/token",
		"jwks_uri":                              p.cfg.IssuerURL + "/oidc/jwks",
		"introspection_endpoint":                p.cfg.IssuerURL + "/oidc/introspect",
		"revocation_endpoint":                   p.cfg.IssuerURL + "/oidc/revoke",
		"response_types_supported":              []string{"code", "token", "id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "spiffe"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "spiffe_id"},
		"grant_types_supported":                 []string{"authorization_code", "urn:ietf:params:oauth:grant-type:token-exchange"},
	})
}

// OIDCJWKSHandler serves the JSON Web Key Set.
func (sc *SpireController) OIDCJWKSHandler(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	type JWK struct {
		KeyType   string `json:"kty"`
		Algorithm string `json:"alg"`
		Use       string `json:"use"`
		KeyID     string `json:"kid"`
		Modulus   string `json:"n"`
		Exponent  string `json:"e"`
	}
	jwk := JWK{
		KeyType:   "RSA",
		Algorithm: "RS256",
		Use:       "sig",
		KeyID:     p.keyID,
		Modulus:   spireEncodeBase64URL(p.publicKey.N.Bytes()),
		Exponent:  spireEncodeBase64URL(big.NewInt(int64(p.publicKey.E)).Bytes()),
	}
	c.JSON(http.StatusOK, gin.H{"keys": []JWK{jwk}})
}

// OIDCTokenExchange handles RFC 8693 token exchange.
func (sc *SpireController) OIDCTokenExchange(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	var req struct {
		GrantType        string `json:"grant_type" binding:"required"`
		SubjectToken     string `json:"subject_token" binding:"required"`
		SubjectTokenType string `json:"subject_token_type" binding:"required"`
		Audience         string `json:"audience,omitempty"`
		Scope            string `json:"scope,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}
	if req.GrantType != "urn:ietf:params:oauth:grant-type:token-exchange" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}
	claims, err := p.validateToken(req.SubjectToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "Invalid subject token"})
		return
	}
	audience := req.Audience
	if audience == "" {
		audience = p.cfg.IssuerURL
	}
	newToken, err := p.createToken(claims.Subject, claims.SPIFFEID, []string{audience}, req.Scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, spireTokenExchangeResponse{
		AccessToken:     newToken,
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       "Bearer",
		ExpiresIn:       int64(p.cfg.TokenExpiry.Seconds()),
		Scope:           req.Scope,
	})
}

// OIDCIntrospect handles RFC 7662 token introspection.
func (sc *SpireController) OIDCIntrospect(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	token := c.PostForm("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	claims, err := p.validateToken(token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	var oidcToken SpireOIDCToken
	if err := p.db.Where("jti = ?", claims.JWTID).First(&oidcToken).Error; err == nil && oidcToken.Revoked {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"active":    true,
		"sub":       claims.Subject,
		"iss":       claims.Issuer,
		"aud":       claims.Audience,
		"exp":       claims.ExpiresAt,
		"iat":       claims.IssuedAt,
		"jti":       claims.JWTID,
		"spiffe_id": claims.SPIFFEID,
	})
}

// OIDCRevoke handles RFC 7009 token revocation.
func (sc *SpireController) OIDCRevoke(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	token := c.PostForm("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	claims, err := p.validateToken(token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	p.db.Model(&SpireOIDCToken{}).Where("jti = ?", claims.JWTID).Update("revoked", true)
	c.JSON(http.StatusOK, gin.H{})
}

// OIDCExchangeSPIFFE converts an X.509 SVID to an OIDC token.
func (sc *SpireController) OIDCExchangeSPIFFE(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	ctx := context.Background()
	spiffeSocket := "/run/spire/sockets/workload_api.sock"
	if _, err := os.Stat(spiffeSocket); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "spiffe_unavailable"})
		return
	}
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	defer source.Close()
	svid, err := source.GetX509SVID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	spiffeID := svid.ID.String()
	subject := svid.ID.Path()
	token, err := p.createToken(subject, spiffeID, []string{p.cfg.IssuerURL}, "openid spiffe")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, spireTokenExchangeResponse{
		AccessToken:     token,
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       "Bearer",
		ExpiresIn:       int64(p.cfg.TokenExpiry.Seconds()),
		Scope:           "openid spiffe",
	})
}

// OIDCIssueJWTSVID issues a JWT-SVID for the requesting workload.
func (sc *SpireController) OIDCIssueJWTSVID(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	spiffeID := sc.extractSPIFFEIDFromContext(c)
	if spiffeID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	audience := c.Query("audience")
	if audience == "" {
		audience = p.cfg.IssuerURL
	}
	jwtSVID, err := p.createJWTSVID(spiffeID, []string{audience})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, spireTokenExchangeResponse{
		AccessToken:     jwtSVID,
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       "Bearer",
		ExpiresIn:       int64(p.cfg.TokenExpiry.Seconds()),
	})
}

// OIDCExchangeCloud exchanges a JWT-SVID for a cloud provider token.
func (sc *SpireController) OIDCExchangeCloud(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	var req spireCloudTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}
	jwtSVID := sc.extractBearerToken(c)
	if jwtSVID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	claims, err := p.validateJWTSVID(jwtSVID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant"})
		return
	}
	var resp *spireCloudTokenResponse
	switch req.Provider {
	case "aws":
		resp, err = p.exchangeAWSToken(claims, &req)
	case "azure":
		resp, err = p.exchangeAzureToken(claims, &req)
	case "gcp":
		resp, err = p.exchangeGCPToken(claims, &req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_provider"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// OIDCExchangeAWS exchanges a JWT-SVID for AWS STS credentials.
func (sc *SpireController) OIDCExchangeAWS(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	var req struct {
		RoleARN  string `json:"role_arn" binding:"required"`
		Audience string `json:"audience,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}
	if req.Audience == "" {
		req.Audience = "sts.amazonaws.com"
	}
	jwtSVID := sc.extractBearerToken(c)
	if jwtSVID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	claims, err := p.validateJWTSVID(jwtSVID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant"})
		return
	}
	resp, err := p.exchangeAWSToken(claims, &spireCloudTokenRequest{Provider: "aws", Audience: req.Audience, RoleARN: req.RoleARN})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// OIDCExchangeAzure exchanges a JWT-SVID for an Azure AD token.
func (sc *SpireController) OIDCExchangeAzure(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	var req struct {
		TenantID   string `json:"tenant_id" binding:"required"`
		ResourceID string `json:"resource_id,omitempty"`
		Scope      string `json:"scope,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}
	if req.ResourceID == "" {
		req.ResourceID = "https://management.azure.com/"
	}
	audience := "https://login.microsoftonline.com/" + req.TenantID + "/v2.0"
	jwtSVID := sc.extractBearerToken(c)
	if jwtSVID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	claims, err := p.validateJWTSVID(jwtSVID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant"})
		return
	}
	resp, err := p.exchangeAzureToken(claims, &spireCloudTokenRequest{Provider: "azure", Audience: audience, ResourceID: req.ResourceID, Scope: req.Scope})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// OIDCExchangeGCP exchanges a JWT-SVID for a GCP access token.
func (sc *SpireController) OIDCExchangeGCP(c *gin.Context) {
	p := sc.oidcProvider
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC provider not initialized"})
		return
	}
	var req struct {
		ProjectID    string `json:"project_id" binding:"required"`
		ServiceEmail string `json:"service_email" binding:"required"`
		Scope        string `json:"scope,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}
	if req.Scope == "" {
		req.Scope = "https://www.googleapis.com/auth/cloud-platform"
	}
	jwtSVID := sc.extractBearerToken(c)
	if jwtSVID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	claims, err := p.validateJWTSVID(jwtSVID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant"})
		return
	}
	resp, err := p.exchangeGCPToken(claims, &spireCloudTokenRequest{Provider: "gcp", Audience: "https://sts.googleapis.com/", ServiceEmail: req.ServiceEmail, Scope: req.Scope})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ===== POLICY HANDLERS =====

// CreatePolicy creates a new policy.
func (sc *SpireController) CreatePolicy(c *gin.Context) {
	var policy SpirePolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := sc.validatePolicy(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	policy.Version = "1.0"
	policy.Active = true
	policy.Engine = sc.policyEngine
	if err := sc.db.Create(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sc.spireAuditLog(c, "policy.create", policy.Name, "allow", "Policy created", nil)
	c.JSON(http.StatusCreated, policy)
}

// ListPolicies lists policies with optional filters.
func (sc *SpireController) ListPolicies(c *gin.Context) {
	var policies []SpirePolicy
	q := sc.db.Preload("Rules.Subjects").Preload("Rules.Resources").Preload("Rules.Actions").Preload("Rules.Conditions")
	if name := c.Query("name"); name != "" {
		q = q.Where("name ILIKE ?", "%"+name+"%")
	}
	if engine := c.Query("engine"); engine != "" {
		q = q.Where("engine = ?", engine)
	}
	if active := c.Query("active"); active != "" {
		if v, err := strconv.ParseBool(active); err == nil {
			q = q.Where("active = ?", v)
		}
	}
	if err := q.Find(&policies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, policies)
}

// GetPolicy retrieves a policy by ID.
func (sc *SpireController) GetPolicy(c *gin.Context) {
	var policy SpirePolicy
	if err := sc.db.Preload("Rules.Subjects").Preload("Rules.Resources").Preload("Rules.Actions").Preload("Rules.Conditions").
		First(&policy, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}
	c.JSON(http.StatusOK, policy)
}

// UpdatePolicy updates an existing policy.
func (sc *SpireController) UpdatePolicy(c *gin.Context) {
	var policy SpirePolicy
	if err := sc.db.First(&policy, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := sc.validatePolicy(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sc.db.Save(&policy)
	sc.spireAuditLog(c, "policy.update", policy.Name, "allow", "Policy updated", nil)
	c.JSON(http.StatusOK, policy)
}

// DeletePolicy removes a policy.
func (sc *SpireController) DeletePolicy(c *gin.Context) {
	if err := sc.db.Delete(&SpirePolicy{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sc.spireAuditLog(c, "policy.delete", c.Param("id"), "allow", "Policy deleted", nil)
	c.JSON(http.StatusOK, gin.H{"message": "Policy deleted"})
}

// EvaluatePolicy evaluates a policy request.
func (sc *SpireController) EvaluatePolicy(c *gin.Context) {
	var eval SpirePolicyEvaluation
	if err := c.ShouldBindJSON(&eval); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	requestID := uuid.New().String()
	eval.Timestamp = time.Now()
	result := sc.evaluatePolicy(&eval, requestID)
	sc.spireAuditLog(c, eval.Action, eval.Resource, result.Decision, result.Reason, &requestID)
	c.JSON(http.StatusOK, result)
}

// BatchEvaluatePolicy evaluates multiple policy requests.
func (sc *SpireController) BatchEvaluatePolicy(c *gin.Context) {
	var evals []SpirePolicyEvaluation
	if err := c.ShouldBindJSON(&evals); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var results []SpirePolicyResult
	for _, eval := range evals {
		requestID := uuid.New().String()
		eval.Timestamp = time.Now()
		result := sc.evaluatePolicy(&eval, requestID)
		results = append(results, *result)
		sc.spireAuditLog(c, eval.Action, eval.Resource, result.Decision, result.Reason, &requestID)
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

// TestPolicy tests a policy without saving it.
func (sc *SpireController) TestPolicy(c *gin.Context) {
	var req struct {
		Policy     SpirePolicy           `json:"policy"`
		Evaluation SpirePolicyEvaluation `json:"evaluation"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	requestID := uuid.New().String()
	result := &SpirePolicyResult{
		Decision:    "deny",
		Reason:      "No matching rule",
		Context:     req.Evaluation.Context,
		EvaluatedAt: time.Now(),
		RequestID:   requestID,
	}
	if decision := sc.evaluatePolicyRules(&req.Policy, &req.Evaluation, result); decision != "" {
		result.Decision = decision
	}
	c.JSON(http.StatusOK, result)
}

// BindRole creates a role binding.
func (sc *SpireController) BindRole(c *gin.Context) {
	var binding SpireRoleBinding
	if err := c.ShouldBindJSON(&binding); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := sc.db.Create(&binding).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sc.spireAuditLog(c, "role.bind", binding.Resource, "allow", fmt.Sprintf("Role %s bound to %s", binding.Role, binding.Subject), nil)
	c.JSON(http.StatusCreated, binding)
}

// UnbindRole removes a role binding.
func (sc *SpireController) UnbindRole(c *gin.Context) {
	subject := c.Query("subject")
	role := c.Query("role")
	if err := sc.db.Where("subject = ? AND role = ?", subject, role).Delete(&SpireRoleBinding{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sc.spireAuditLog(c, "role.unbind", "", "allow", fmt.Sprintf("Role %s unbound from %s", role, subject), nil)
	c.JSON(http.StatusOK, gin.H{"message": "Role binding removed"})
}

// ListRoleBindings lists role bindings.
func (sc *SpireController) ListRoleBindings(c *gin.Context) {
	var bindings []SpireRoleBinding
	q := sc.db.Model(&SpireRoleBinding{})
	if s := c.Query("subject"); s != "" {
		q = q.Where("subject ILIKE ?", "%"+s+"%")
	}
	if r := c.Query("role"); r != "" {
		q = q.Where("role = ?", r)
	}
	if err := q.Find(&bindings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bindings)
}

// GetAuditLogs returns paginated audit logs.
func (sc *SpireController) GetAuditLogs(c *gin.Context) {
	var logs []SpireAuditLog
	q := sc.db.Model(&SpireAuditLog{}).Order("timestamp DESC")
	if s := c.Query("subject"); s != "" {
		q = q.Where("subject ILIKE ?", "%"+s+"%")
	}
	if a := c.Query("action"); a != "" {
		q = q.Where("action = ?", a)
	}
	if d := c.Query("decision"); d != "" {
		q = q.Where("decision = ?", d)
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err := q.Offset((page - 1) * limit).Limit(limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "page": page, "limit": limit})
}

// ExportAuditLogs exports all audit logs as JSON.
func (sc *SpireController) ExportAuditLogs(c *gin.Context) {
	var logs []SpireAuditLog
	sc.db.Find(&logs)
	c.Header("Content-Disposition", "attachment; filename=spire-audit-logs.json")
	c.JSON(http.StatusOK, logs)
}

// ===== POLICY ENGINE INTERNALS =====

func (sc *SpireController) evaluatePolicy(eval *SpirePolicyEvaluation, requestID string) *SpirePolicyResult {
	result := &SpirePolicyResult{
		Decision:    "deny",
		Reason:      "No matching policy found",
		Context:     eval.Context,
		EvaluatedAt: time.Now(),
		RequestID:   requestID,
	}
	var policies []SpirePolicy
	sc.db.Preload("Rules.Subjects").Preload("Rules.Resources").Preload("Rules.Actions").Preload("Rules.Conditions").
		Where("active = ?", true).Order("created_at ASC").Find(&policies)
	for _, policy := range policies {
		if decision := sc.evaluatePolicyRules(&policy, eval, result); decision != "" {
			result.Decision = decision
			break
		}
	}
	return result
}

func (sc *SpireController) evaluatePolicyRules(policy *SpirePolicy, eval *SpirePolicyEvaluation, result *SpirePolicyResult) string {
	for _, rule := range policy.Rules {
		r := rule
		if sc.matchRule(&r, eval) {
			result.MatchedRule = &r
			result.Reason = fmt.Sprintf("Matched rule '%s' in policy '%s'", rule.Name, policy.Name)
			return rule.Effect
		}
	}
	return ""
}

func (sc *SpireController) matchRule(rule *SpirePolicyRule, eval *SpirePolicyEvaluation) bool {
	if !sc.matchSubjects(rule.Subjects, eval.Subject) {
		return false
	}
	if !sc.matchResources(rule.Resources, eval.Resource) {
		return false
	}
	if !sc.matchActions(rule.Actions, eval.Action) {
		return false
	}
	return sc.matchConditions(rule.Conditions, eval)
}

func (sc *SpireController) matchSubjects(subjects []SpirePolicySubject, subject string) bool {
	if len(subjects) == 0 {
		return true
	}
	for _, s := range subjects {
		if s.Value == subject || s.Value == "*" || spireMatchPattern(s.Pattern, subject) {
			return true
		}
	}
	return false
}

func (sc *SpireController) matchResources(resources []SpirePolicyResource, resource string) bool {
	if len(resources) == 0 {
		return true
	}
	for _, r := range resources {
		if r.Value == resource || r.Value == "*" || spireMatchPattern(r.Pattern, resource) {
			return true
		}
	}
	return false
}

func (sc *SpireController) matchActions(actions []SpirePolicyAction, action string) bool {
	if len(actions) == 0 {
		return true
	}
	for _, a := range actions {
		if a.Value == action || a.Value == "*" {
			return true
		}
	}
	return false
}

func (sc *SpireController) matchConditions(conditions []SpirePolicyCondition, eval *SpirePolicyEvaluation) bool {
	for _, cond := range conditions {
		if !sc.evaluateCondition(&cond, eval) {
			return false
		}
	}
	return true
}

func (sc *SpireController) evaluateCondition(cond *SpirePolicyCondition, eval *SpirePolicyEvaluation) bool {
	switch cond.Type {
	case "time":
		switch cond.Operator {
		case "after":
			if t, err := time.Parse(time.RFC3339, cond.Value); err == nil {
				return eval.Timestamp.After(t)
			}
		case "before":
			if t, err := time.Parse(time.RFC3339, cond.Value); err == nil {
				return eval.Timestamp.Before(t)
			}
		}
		return true
	case "attribute":
		val, exists := eval.Context[cond.Key]
		if !exists {
			return false
		}
		strVal := fmt.Sprintf("%v", val)
		switch cond.Operator {
		case "eq":
			return strVal == cond.Value
		case "ne":
			return strVal != cond.Value
		case "regex":
			if m, err := regexp.MatchString(cond.Value, strVal); err == nil {
				return m
			}
		case "in":
			for _, v := range strings.Split(cond.Value, ",") {
				if strings.TrimSpace(v) == strVal {
					return true
				}
			}
		}
		return false
	}
	return true
}

func (sc *SpireController) validatePolicy(policy *SpirePolicy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if len(policy.Rules) == 0 {
		return fmt.Errorf("policy must have at least one rule")
	}
	for _, rule := range policy.Rules {
		if rule.Effect != "allow" && rule.Effect != "deny" {
			return fmt.Errorf("rule effect must be 'allow' or 'deny'")
		}
	}
	return nil
}

func (sc *SpireController) spireAuditLog(c *gin.Context, action, resource, decision, reason string, requestID *string) {
	rid := uuid.New().String()
	if requestID != nil {
		rid = *requestID
	}
	subject := sc.extractSPIFFEIDFromContext(c)
	if subject == "" {
		subject = "unknown"
	}
	sc.db.Create(&SpireAuditLog{
		RequestID: rid, Subject: subject, Resource: resource,
		Action: action, Decision: decision, Reason: reason,
		Context:   map[string]interface{}{},
		IPAddress: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"),
		Timestamp: time.Now(),
	})
}

func (sc *SpireController) loadDefaultPolicies() {
	if sc.db == nil {
		return
	}
	defaults := []SpirePolicy{
		{
			Name: "default-workload-access", Description: "Default policy for workload access",
			Engine: sc.policyEngine, Active: true,
			Rules: []SpirePolicyRule{
				{
					Name: "allow-workload-registry-access", Effect: "allow", Priority: 100,
					Subjects:  []SpirePolicySubject{{Type: "spiffe_id", Pattern: "spiffe://example.org/workload/.*"}},
					Resources: []SpirePolicyResource{{Type: "service", Value: "registry"}, {Type: "service", Value: "attestation"}},
					Actions:   []SpirePolicyAction{{Type: "http", Value: "read"}, {Type: "http", Value: "write"}},
				},
			},
		},
		{
			Name: "admin-access", Description: "Administrative access policy",
			Engine: sc.policyEngine, Active: true,
			Rules: []SpirePolicyRule{
				{
					Name: "allow-admin-all-access", Effect: "allow", Priority: 200,
					Subjects:  []SpirePolicySubject{{Type: "spiffe_id", Value: "spiffe://example.org/admin"}, {Type: "role", Value: "admin"}},
					Resources: []SpirePolicyResource{{Type: "service", Value: "*"}},
					Actions:   []SpirePolicyAction{{Type: "http", Value: "*"}},
				},
			},
		},
	}
	for _, p := range defaults {
		var existing SpirePolicy
		if sc.db.Where("name = ?", p.Name).First(&existing).Error != nil {
			sc.db.Create(&p)
			log.Printf("[spire] Created default policy: %s", p.Name)
		}
	}
}

// ===== OIDC PROVIDER INTERNALS =====

func (p *spireOIDCProvider) createToken(subject, spiffeID string, audience []string, scope string) (string, error) {
	now := time.Now()
	jti := uuid.New().String()
	claims := spireTokenClaims{
		Subject:   subject,
		Issuer:    p.cfg.IssuerURL,
		Audience:  audience,
		ExpiresAt: now.Add(p.cfg.TokenExpiry).Unix(),
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		JWTID:     jti,
		SPIFFEID:  spiffeID,
		Claims:    map[string]interface{}{"scope": scope},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = p.keyID
	tokenString, err := token.SignedString(p.privateKey)
	if err != nil {
		return "", err
	}
	p.db.Create(&SpireOIDCToken{
		JWTID: jti, Subject: subject, SPIFFEID: spiffeID,
		TokenType: "Bearer", Audience: audience[0], Scope: scope,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0), CreatedAt: now, Revoked: false,
	})
	return tokenString, nil
}

func (p *spireOIDCProvider) validateToken(tokenString string) (*spireTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &spireTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return p.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*spireTokenClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func (p *spireOIDCProvider) createJWTSVID(spiffeID string, audience []string) (string, error) {
	now := time.Now()
	jti := uuid.New().String()
	claims := spireJWTSVIDClaims{
		Subject: spiffeID, Audience: audience, Issuer: p.cfg.IssuerURL,
		ExpiresAt: now.Add(p.cfg.TokenExpiry).Unix(), IssuedAt: now.Unix(), NotBefore: now.Unix(),
		JWTID: jti, SPIFFEID: spiffeID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = p.keyID
	token.Header["typ"] = "JWT"
	tokenString, err := token.SignedString(p.privateKey)
	if err != nil {
		return "", err
	}
	p.db.Create(&SpireOIDCToken{
		JWTID: jti, Subject: spiffeID, SPIFFEID: spiffeID,
		TokenType: "JWT-SVID", Audience: audience[0],
		ExpiresAt: time.Unix(claims.ExpiresAt, 0), CreatedAt: now, Revoked: false,
	})
	return tokenString, nil
}

func (p *spireOIDCProvider) validateJWTSVID(tokenString string) (*spireJWTSVIDClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &spireJWTSVIDClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return p.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*spireJWTSVIDClaims); ok && token.Valid {
		var record SpireOIDCToken
		if p.db.Where("jti = ?", claims.JWTID).First(&record).Error == nil && record.Revoked {
			return nil, fmt.Errorf("JWT-SVID has been revoked")
		}
		return claims, nil
	}
	return nil, fmt.Errorf("invalid JWT-SVID")
}

func (p *spireOIDCProvider) exchangeAWSToken(claims *spireJWTSVIDClaims, req *spireCloudTokenRequest) (*spireCloudTokenResponse, error) {
	if req.RoleARN == "" {
		return nil, fmt.Errorf("role_arn is required for AWS token exchange")
	}
	// TODO: Implement STS AssumeRoleWithWebIdentity using the validated JWT-SVID.
	return nil, fmt.Errorf("AWS cloud token exchange is not yet implemented — configure STS AssumeRoleWithWebIdentity integration")
}

func (p *spireOIDCProvider) exchangeAzureToken(claims *spireJWTSVIDClaims, req *spireCloudTokenRequest) (*spireCloudTokenResponse, error) {
	// TODO: Implement Azure AD confidential client token exchange using the validated JWT-SVID.
	return nil, fmt.Errorf("Azure cloud token exchange is not yet implemented — configure Azure AD token endpoint integration")
}

func (p *spireOIDCProvider) exchangeGCPToken(claims *spireJWTSVIDClaims, req *spireCloudTokenRequest) (*spireCloudTokenResponse, error) {
	// TODO: Implement GCP STS token exchange using the validated JWT-SVID.
	return nil, fmt.Errorf("GCP cloud token exchange is not yet implemented — configure GCP STS endpoint integration")
}

// ===== HELPERS =====

func (sc *SpireController) extractSPIFFEIDFromContext(c *gin.Context) string {
	if id := c.GetHeader("X-SPIFFE-ID"); id != "" {
		return id
	}
	if sc.oidcProvider != nil {
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			if claims, err := sc.oidcProvider.validateJWTSVID(auth[7:]); err == nil {
				return claims.SPIFFEID
			}
		}
	}
	return ""
}

func (sc *SpireController) extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

func spireMatchPattern(pattern, value string) bool {
	if pattern == "" {
		return false
	}
	if m, err := regexp.MatchString(pattern, value); err == nil {
		return m
	}
	return false
}

func spireEncodeBase64URL(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func spireGetenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
