package shared

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContextStringValue normalizes a value stored in the Gin context to a trimmed string.
func ContextStringValue(c *gin.Context, key string) string {
	value, exists := c.Get(key)
	if !exists || value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case uuid.UUID:
		if v == uuid.Nil {
			return ""
		}
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

// RequireTenantID retrieves tenant_id from context, returning an error when missing.
func RequireTenantID(c *gin.Context) (string, error) {
	tenantID := ContextStringValue(c, "tenant_id")
	if tenantID == "" {
		return "", fmt.Errorf("tenant not found")
	}
	return tenantID, nil
}

// StringPtr returns a pointer to the given string value.
func StringPtr(s string) *string {
	return &s
}

// SCIMContentType sets the proper SCIM content-type response header.
func SCIMContentType(c *gin.Context) {
	c.Header("Content-Type", "application/scim+json; charset=utf-8")
}

// FlexibleBool accepts JSON booleans, quoted booleans ("true"/"false"), and 0/1 integers.
type FlexibleBool bool

func (fb *FlexibleBool) UnmarshalJSON(data []byte) error {
	if fb == nil {
		return fmt.Errorf("flexibleBool: nil receiver")
	}
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*fb = FlexibleBool(b)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("flexibleBool: empty string")
		}
		parsed, err := strconv.ParseBool(strings.ToLower(s))
		if err != nil {
			return fmt.Errorf("flexibleBool: %w", err)
		}
		*fb = FlexibleBool(parsed)
		return nil
	}
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		switch i {
		case 0:
			*fb = FlexibleBool(false)
			return nil
		case 1:
			*fb = FlexibleBool(true)
			return nil
		default:
			return fmt.Errorf("flexibleBool: unsupported numeric value %d", i)
		}
	}
	return fmt.Errorf("flexibleBool: expected boolean-compatible value")
}

// Bool returns the underlying bool value.
func (fb FlexibleBool) Bool() bool { return bool(fb) }
