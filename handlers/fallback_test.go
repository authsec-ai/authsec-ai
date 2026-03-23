package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sharedmodels "github.com/authsec-ai/sharedmodels"
)

// windowsHelloCredential constructs a CredentialContainer that mimics a Windows Hello registration payload.
func windowsHelloCredential(t *testing.T) (CredentialContainer, []byte, []byte) {
	t.Helper()

	rawID := bytes.Repeat([]byte{0xAB}, 32)
	rpHash := sha256.Sum256([]byte("app.authsec.dev"))
	aaguid := bytes.Repeat([]byte{0x10}, 16)
	xCoord := bytes.Repeat([]byte{0x01}, 32)
	yCoord := bytes.Repeat([]byte{0x02}, 32)

	coseKey := map[int]interface{}{
		1:  2,  // kty: EC2
		3:  -7, // alg: ES256
		-1: 1,  // curve: P-256
		-2: xCoord,
		-3: yCoord,
	}
	coseBytes, err := cbor.Marshal(coseKey)
	require.NoError(t, err)

	authDataLen := 32 + 1 + 4 + 16 + 2 + len(rawID) + len(coseBytes)
	authData := make([]byte, authDataLen)
	copy(authData[0:32], rpHash[:])
	authData[32] = 0x41 // User present + attested credential data flags
	binary.BigEndian.PutUint32(authData[33:37], 0)
	copy(authData[37:53], aaguid)
	binary.BigEndian.PutUint16(authData[53:55], uint16(len(rawID)))
	copy(authData[55:55+len(rawID)], rawID)
	copy(authData[55+len(rawID):], coseBytes)

	attestation := map[string]interface{}{
		"fmt":      "packed",
		"attStmt":  map[string]interface{}{},
		"authData": authData,
	}
	attBytes, err := cbor.Marshal(attestation)
	require.NoError(t, err)

	clientDataJSON := map[string]string{
		"type":      "webauthn.create",
		"challenge": base64.RawURLEncoding.EncodeToString([]byte("challenge-bytes")),
		"origin":    "https://app.authsec.dev",
	}
	clientDataBytes, err := json.Marshal(clientDataJSON)
	require.NoError(t, err)

	container := CredentialContainer{
		ID:    base64.RawURLEncoding.EncodeToString(rawID),
		RawID: base64.RawURLEncoding.EncodeToString(rawID),
		Type:  "public-key",
		Response: CredentialResponse{
			ClientDataJSON:    base64.RawURLEncoding.EncodeToString(clientDataBytes),
			AttestationObject: base64.RawURLEncoding.EncodeToString(attBytes),
		},
	}

	return container, rawID, coseBytes
}

func TestBuildFallbackCredentialFromContainer_WindowsHello(t *testing.T) {
	container, expectedID, expectedPubKey := windowsHelloCredential(t)

	cred, err := buildFallbackCredentialFromContainer(&container)
	require.NoError(t, err)
	require.NotNil(t, cred)

	require.Equal(t, expectedID, cred.ID)
	require.Equal(t, expectedPubKey, cred.PublicKey)
	require.Equal(t, "none", cred.AttestationType)
	require.Equal(t, 0, int(cred.Authenticator.SignCount))
	require.Len(t, cred.Authenticator.AAGUID, 16)
}

func TestFallbackCredential_AllowsLogin(t *testing.T) {
	container, expectedID, _ := windowsHelloCredential(t)

	cred, err := buildFallbackCredentialFromContainer(&container)
	require.NoError(t, err)

	user := &sharedmodels.User{
		ID:       uuid.New(),
		ClientID: uuid.New(),
		Email:    "windows.hello@example.com",
	}
	webUser := &WebAuthnUser{User: user}
	webUser.SetCredentials([]webauthn.Credential{*cred})

	cfg := &webauthn.Config{
		RPID:          "app.authsec.dev",
		RPDisplayName: "AuthSec Test",
		RPOrigins:     []string{"https://app.authsec.dev"},
	}
	wa, err := webauthn.New(cfg)
	require.NoError(t, err)

	options, sessionData, err := wa.BeginLogin(webUser)
	require.NoError(t, err)
	require.NotNil(t, sessionData)

	require.Len(t, options.Response.AllowedCredentials, 1)
	require.Equal(t, expectedID, []byte(options.Response.AllowedCredentials[0].CredentialID))
}
