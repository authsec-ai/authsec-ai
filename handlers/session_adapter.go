package handlers

import (
	"fmt"

	session "github.com/authsec-ai/authsec/internal/session"
	"github.com/go-webauthn/webauthn/webauthn"
)

type sessionManagerAdapter struct {
	manager session.SessionManagerInterface
}

// NewSessionManagerAdapter adapts an internal session manager to the handler interface.
func NewSessionManagerAdapter(manager session.SessionManagerInterface) SessionManagerInterface {
	return &sessionManagerAdapter{manager: manager}
}

func (s *sessionManagerAdapter) Save(key string, data interface{}) error {
	switch v := data.(type) {
	case *webauthn.SessionData:
		return s.manager.Save(key, v)
	case interface{ ToWebAuthnSessionData() *webauthn.SessionData }:
		return s.manager.Save(key, v.ToWebAuthnSessionData())
	default:
		return fmt.Errorf("unsupported session data type %T", data)
	}
}

func (s *sessionManagerAdapter) Get(key string) (interface{}, bool) {
	data, found := s.manager.Get(key)
	if !found || data == nil {
		return nil, false
	}

	// Return the original webauthn.SessionData instead of converting
	// This ensures type assertions in handlers work correctly
	return data, true
}

func (s *sessionManagerAdapter) Delete(key string) {
	s.manager.Delete(key)
}
