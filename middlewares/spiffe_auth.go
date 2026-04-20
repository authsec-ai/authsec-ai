package middlewares

// SpiffeAuthMiddleware validates SPIFFE JWT-SVIDs issued by authsec-spire.
// Falls back to the standard AuthMiddleware if the token is not a SPIFFE JWT-SVID.
// When a valid SPIFFE token is accepted the following context keys are set:
//
//   - "claims"      – jwt.MapClaims of the verified token
//   - "auth_method" – "spiffe-jwt-svid"
//   - "spiffe_id"   – the sub claim (e.g. "spiffe://tenant-id/agent/...")
//
// Ported from external-service/middleware/spiffe_auth.go.

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SpiffeAuthMiddleware returns a gin.HandlerFunc that accepts either a
// standard auth-manager JWT or a SPIFFE JWT-SVID.
func SpiffeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := spiffeExtractBearer(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			return
		}

		// Parse unverified to check whether this is a SPIFFE JWT-SVID.
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		unverified, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			return
		}

		claims, ok := unverified.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		sub, _ := claims["sub"].(string)
		if !strings.HasPrefix(sub, "spiffe://") {
			// Not a SPIFFE token — delegate to standard auth middleware.
			AuthMiddleware()(c)
			return
		}

		// Determine tenant_id from token claims or issuer.
		tenantID, _ := claims["tenant_id"].(string)
		if tenantID == "" {
			iss, _ := claims["iss"].(string)
			tenantID = strings.TrimPrefix(iss, "spiffe://")
		}
		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Cannot determine tenant_id from SPIFFE JWT-SVID"})
			return
		}

		pubKey, err := spiffeGetPublicKey(tenantID)
		if err != nil {
			log.Printf("[SpiffeAuth] Failed to fetch JWKS for tenant %s: %v", tenantID, err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Failed to verify SPIFFE token"})
			return
		}

		verified, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return pubKey, nil
		})
		if err != nil {
			log.Printf("[SpiffeAuth] Token verification failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "SPIFFE token verification failed"})
			return
		}

		verifiedClaims, ok := verified.Claims.(jwt.MapClaims)
		if !ok || !verified.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid SPIFFE token"})
			return
		}

		// Map SPIFFE "permissions" claim to the format expected by auth-manager.
		if perms, ok := verifiedClaims["permissions"].([]interface{}); ok {
			parts := make([]string, 0, len(perms))
			for _, p := range perms {
				if s, ok := p.(string); ok {
					parts = append(parts, s)
				}
			}
			verifiedClaims["scope"] = strings.Join(parts, " ")
		}

		c.Set("claims", verifiedClaims)
		c.Set("auth_method", "spiffe-jwt-svid")
		c.Set("spiffe_id", sub)

		c.Next()
	}
}

// --- JWKS caching ---

var (
	spiffeJWKSCache   = make(map[string]*spiffeJWKSCacheEntry)
	spiffeJWKSCacheMu sync.RWMutex
)

type spiffeJWKSCacheEntry struct {
	key       *rsa.PublicKey
	fetchedAt time.Time
}

const spiffeJWKSCacheTTL = 5 * time.Minute

func spiffeGetPublicKey(tenantID string) (*rsa.PublicKey, error) {
	spiffeJWKSCacheMu.RLock()
	if entry, ok := spiffeJWKSCache[tenantID]; ok && time.Since(entry.fetchedAt) < spiffeJWKSCacheTTL {
		spiffeJWKSCacheMu.RUnlock()
		return entry.key, nil
	}
	spiffeJWKSCacheMu.RUnlock()

	spireURL := os.Getenv("ICP_SERVICE_URL")
	if spireURL == "" {
		spireURL = "http://localhost:7001"
	}

	url := fmt.Sprintf("%s/v1/jwt/bundle?tenant_id=%s", spireURL, tenantID)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JWKS endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
			Kid string `json:"kid"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return nil, fmt.Errorf("no keys in JWKS response for tenant %s", tenantID)
	}

	k := jwks.Keys[0]
	if k.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key type: %s", k.Kty)
	}

	pubKey, err := spiffeParseRSAPublicKey(k.N, k.E)
	if err != nil {
		return nil, fmt.Errorf("parse RSA key: %w", err)
	}

	spiffeJWKSCacheMu.Lock()
	spiffeJWKSCache[tenantID] = &spiffeJWKSCacheEntry{key: pubKey, fetchedAt: time.Now()}
	spiffeJWKSCacheMu.Unlock()

	log.Printf("[SpiffeAuth] Cached JWKS for tenant %s (kid=%s)", tenantID, k.Kid)
	return pubKey, nil
}

func spiffeParseRSAPublicKey(nBase64, eBase64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nBase64)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eBase64)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}

func spiffeExtractBearer(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
