package sdkmgr

import (
	"strings"
)

const mainClientSuffix = "-main-client"

// BuildClientIDCandidates returns all plausible client_id variants for a raw
// input string. The SDK may send UUIDs in different formats:
//   - Plain UUID:                xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
//   - With suffix:              xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx-main-client
//   - Underscore UUID:          xxxxxxxx_xxxx_xxxx_xxxx_xxxxxxxxxxxx
//   - Mixed suffix:             xxxxxxxx_xxxx_xxxx_xxxx_xxxxxxxxxxxx_main-client
//   - Wrapped in quotes:        "xxxxxxxx-..."
//
// This function normalizes and produces a de-duplicated candidate list so that
// session/tenant lookups work regardless of the form the SDK sends.
//
// Translates: mcp_auth_service.py _build_client_id_candidates (lines 876-922).
func BuildClientIDCandidates(clientID string) []string {
	raw := strings.TrimSpace(clientID)
	raw = strings.Trim(raw, `"'`)
	if raw == "" {
		return nil
	}

	// Fix common suffix typo.
	raw = strings.Replace(raw, "_main-client", mainClientSuffix, 1)

	var base string
	if strings.HasSuffix(raw, mainClientSuffix) {
		base = raw[:len(raw)-len(mainClientSuffix)]
	} else {
		base = raw
	}

	baseVariants := []string{base}

	// UUID-like underscore form (5 segments) → hyphen form.
	if strings.Contains(base, "_") && strings.Count(base, "_") == 4 {
		baseVariants = append(baseVariants, strings.ReplaceAll(base, "_", "-"))
	}

	// Inverse: hyphen → underscore for environments that stored underscored IDs.
	if strings.Contains(base, "-") {
		baseVariants = append(baseVariants, strings.ReplaceAll(base, "-", "_"))
	}

	// Determine canonical base: prefer hyphenated UUID form.
	canonicalBase := base
	if len(baseVariants) > 1 && strings.Contains(base, "_") && strings.Count(base, "_") == 4 {
		canonicalBase = baseVariants[1] // the hyphen-normalized variant
	}

	// Build candidate list (order matters: most canonical first).
	var candidates []string
	seen := make(map[string]struct{})

	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		candidates = append(candidates, v)
	}

	add(canonicalBase + mainClientSuffix)
	add(raw)
	add(base)

	for _, b := range baseVariants {
		add(b)
		add(b + mainClientSuffix)
	}

	return candidates
}

// NormalizeClientID returns the canonical form of a client ID:
// {uuid-with-hyphens}-main-client.
func NormalizeClientID(clientID string) string {
	candidates := BuildClientIDCandidates(clientID)
	if len(candidates) == 0 {
		return clientID
	}
	return candidates[0]
}
