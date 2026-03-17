package sdkmgr

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
)

// DecodeJWTPayload extracts the payload from a JWT without signature verification.
// This is used to read user claims from external OAuth tokens before verification
// is performed via the auth-manager service.
func DecodeJWTPayload(jwtToken string) (map[string]interface{}, error) {
	parts := strings.Split(jwtToken, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidJWT
	}

	payload := parts[1]
	// Add base64 padding if needed.
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		logrus.WithError(err).Error("failed to base64-decode JWT payload")
		return nil, err
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		logrus.WithError(err).Error("failed to unmarshal JWT payload")
		return nil, err
	}

	return claims, nil
}

// ErrInvalidJWT is returned when the token does not have 3 dot-separated parts.
var ErrInvalidJWT = &sdkmgrError{msg: "invalid JWT: expected 3 parts"}

type sdkmgrError struct {
	msg string
}

func (e *sdkmgrError) Error() string { return e.msg }
