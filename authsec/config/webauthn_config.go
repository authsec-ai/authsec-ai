package config

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// NormalizeRPID ensures the RP ID is a registrable domain (no scheme, no port).
// If a full URL is provided, it extracts the hostname part.

// ValidateSubdomainOrigin checks if an origin matches the allowed subdomain pattern
func ValidateSubdomainOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	// Parse the origin URL
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// Must be HTTPS for production
	if u.Scheme != "https" {
		return false
	}

	// Check if it's a valid authsec.dev domain (app or staging)
	host := u.Host

	// Define allowed base domains
	allowedDomains := []string{"app.authsec.dev", "stage.authsec.dev", "dev.authsec.dev", "app.authsec.ai", "stage.authsec.ai", "dev.authsec.ai"}

	for _, domain := range allowedDomains {
		// Check if it's the base domain
		if host == domain {
			return true
		}

		// Check if it's a subdomain (*.domain)
		if strings.HasSuffix(host, "."+domain) {
			// Extract subdomain part
			subdomain := strings.TrimSuffix(host, "."+domain)

			// Basic subdomain validation (alphanumeric, hyphens, no dots)
			if len(subdomain) > 0 && len(subdomain) <= 63 {
				for _, r := range subdomain {
					if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
						(r >= '0' && r <= '9') || r == '-') {
						return false
					}
				}
				return true
			}
		}
	}

	return false
}

// CreateDynamicWebAuthnConfig creates a WebAuthn config with the provided origin
func CreateDynamicWebAuthnConfig(rpDisplayName, rpID, origin string) *webauthn.Config {
	rpID = NormalizeRPID(rpID)

	return &webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     []string{origin},
	}
}

var TOTPEncryptionKey []byte

func init() {
	keyHex := os.Getenv("TOTP_ENCRYPTION_key")
	if keyHex == "" {
		log.Println("[WARN] TOTP_ENCRYPTION_KEY not set, using default dev key.")
		keyHex = "6AB33320B8A8E177655F72CEDDAE56593D045BE5A47416FDE7C7CF983D5B80D6"
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		log.Fatalf("Invalid TOTP_ENCRYPTION_KEY format: %v", err)
	}
	if len(key) != 32 {
		log.Fatalf("TOTP_ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d", len(key))
	}

	TOTPEncryptionKey = key
}

// SetupWebAuthn configures and returns a WebAuthn instance
func SetupWebAuthn(rpName, rpID, origin string) *webauthn.WebAuthn {
	// Normalize RP ID (strip scheme, lowercase)
	rpID = NormalizeRPID(rpID)

	log.Printf("SetupWebAuthn: rpName=%s, rpID=%s, origin=%s", rpName, rpID, origin)

	// Parse origin to extract domain for RP ID matching
	originURL, err := url.Parse(origin)
	if err != nil {
		log.Printf("SetupWebAuthn: Failed to parse origin %s: %v", origin, err)
	} else {
		originHost := strings.ToLower(originURL.Host)
		// Strip port if present
		if idx := strings.Index(originHost, ":"); idx != -1 {
			originHost = originHost[:idx]
		}
		log.Printf("SetupWebAuthn: origin host=%s", originHost)

		// If origin contains app.authsec.dev, stage.authsec.dev, or dev.authsec.dev, adjust RP ID to match
		if strings.Contains(originHost, "app.authsec.dev") {
			rpID = "app.authsec.dev"
			log.Printf("SetupWebAuthn: Updated rpID to %s to match origin domain", rpID)
		} else if strings.Contains(originHost, "stage.authsec.dev") {
			rpID = "stage.authsec.dev"
			log.Printf("SetupWebAuthn: Updated rpID to %s to match origin domain", rpID)
		} else if strings.Contains(originHost, "dev.authsec.dev") {
			rpID = "dev.authsec.dev"
			log.Printf("SetupWebAuthn: Updated rpID to %s to match origin domain", rpID)
		} else if strings.Contains(originHost, "app.authsec.ai") {
			rpID = "app.authsec.ai"
			log.Printf("SetupWebAuthn: Updated rpID to %s to match origin domain", rpID)
		} else {
			// For custom domains, use the origin host as the rpID
			rpID = originHost
			log.Printf("SetupWebAuthn: Using custom domain as rpID: %s", rpID)
		}
	}

	// Build origins list - the dynamic validation will create per-request instances
	rpOrigins := []string{
		origin,                          // The configured origin from env
		fmt.Sprintf("https://%s", rpID), // Base domain
	}

	log.Printf("SetupWebAuthn: Final config - rpID=%s, rpOrigins=%v", rpID, rpOrigins)

	cfg := &webauthn.Config{
		RPDisplayName:         rpName,                       // Friendly name
		RPID:                  rpID,                         // e.g. "app.authsec.dev"
		RPOrigins:             rpOrigins,                    // ✅ now includes origin, base, wildcard
		AttestationPreference: protocol.PreferNoAttestation, // Allow "none" attestation format
		// Enable debug mode for better error reporting
		Debug:                true,
		EncodeUserIDAsString: false,
		// Additional configuration for better compatibility
		// Note: AuthenticatorSelection will be set during registration
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred, // Prefer user verification for biometrics
			// Don't specify AuthenticatorAttachment to allow both platform and cross-platform by default
			ResidentKey: protocol.ResidentKeyRequirementDiscouraged, // Don't require resident keys
		},
		// Timeout settings for better UX
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    false,
				Timeout:    time.Second * 60,
				TimeoutUVD: time.Second * 60,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    false,
				Timeout:    time.Second * 60,
				TimeoutUVD: time.Second * 60,
			},
		},
	}

	w, err := webauthn.New(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create webauthn instance: %v", err))
	}
	return w
}

// NormalizeRPID strips scheme/port and lowercases
func NormalizeRPID(raw string) string {
	// Remove scheme if present
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	// Remove any port suffix
	if strings.Contains(raw, ":") {
		raw = strings.Split(raw, ":")[0]
	}
	return strings.ToLower(raw)
}
