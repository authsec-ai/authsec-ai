package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

// prepareCredentialRequest extracts the WebAuthn credential from a wrapper payload and
// replaces the request body with the credential JSON so go-webauthn can parse it.
func prepareCredentialRequest(c *gin.Context) (map[string]interface{}, error) {
	body, err := readAndCloneBody(c)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("invalid request JSON: %w", err)
	}

	credentialValue, ok := raw["credential"]
	if !ok {
		return nil, fmt.Errorf("credential is required")
	}

	credentialMap, ok := credentialValue.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("credential must be an object")
	}

	credentialJSON, err := json.Marshal(credentialMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credential: %w", err)
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(credentialJSON))
	c.Request.ContentLength = int64(len(credentialJSON))
	c.Request.Header.Set("Content-Type", "application/json")

	return raw, nil
}
