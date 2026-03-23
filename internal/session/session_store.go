package session

import (
	"log"
	"sync"

	"github.com/go-webauthn/webauthn/webauthn"
)

// SessionManagerInterface defines the interface for session management
type SessionManagerInterface interface {
	Save(key string, data *webauthn.SessionData) error
	Get(key string) (*webauthn.SessionData, bool)
	Delete(key string)
	ListKeys() []string
}

type SessionManager struct {
	store map[string]*webauthn.SessionData
	mu    sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		store: make(map[string]*webauthn.SessionData),
	}
}

func (s *SessionManager) Save(key string, data *webauthn.SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = data
	return nil
}

func (s *SessionManager) Get(key string) (*webauthn.SessionData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, exists := s.store[key]
	if !exists {
		availableKeys := make([]string, 0, len(s.store))
		for k := range s.store {
			availableKeys = append(availableKeys, k)
		}
		log.Printf("SessionManager: session not found for key=%s, available keys: %v", key, availableKeys)
	}
	return data, exists
}

func (s *SessionManager) ListKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.store))
	for key := range s.store {
		keys = append(keys, key)
	}
	return keys
}

func (s *SessionManager) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, key)
}
