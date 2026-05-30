// Package apikey manages the integration API key. The key is stored in the
// database (settings table) so it can be viewed, copied and regenerated from
// the dashboard. An in-memory cache avoids a DB hit on every request.
package apikey

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"wa-proxy/internal/repository"
)

// SettingKey is the settings-table key under which the API key is stored.
const SettingKey = "api_key"

// keyPrefix makes generated keys recognizable.
const keyPrefix = "wak_"

// Service provides cached access to the current API key.
type Service struct {
	repo *repository.Repository

	mu     sync.RWMutex
	cached string
	loaded bool
}

// New creates the service.
func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the current API key (cached). Empty string means key auth is
// effectively disabled (only Basic Auth works).
func (s *Service) Get() string {
	s.mu.RLock()
	if s.loaded {
		v := s.cached
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	v, _ := s.repo.GetSetting(SettingKey)
	s.mu.Lock()
	s.cached = v
	s.loaded = true
	s.mu.Unlock()
	return v
}

// Generate creates a new random key, persists it, and updates the cache.
func (s *Service) Generate() (string, error) {
	key, err := newKey()
	if err != nil {
		return "", err
	}
	if err := s.repo.SetSetting(SettingKey, key); err != nil {
		return "", err
	}
	s.mu.Lock()
	s.cached = key
	s.loaded = true
	s.mu.Unlock()
	return key, nil
}

// EnsureExists generates a key if none is stored yet, returning the current key.
func (s *Service) EnsureExists() (string, error) {
	if cur := s.Get(); cur != "" {
		return cur, nil
	}
	return s.Generate()
}

// Clear removes the stored API key (e.g. on WhatsApp logout) and clears cache.
func (s *Service) Clear() error {
	if err := s.repo.DeleteSetting(SettingKey); err != nil {
		return err
	}
	s.mu.Lock()
	s.cached = ""
	s.loaded = true
	s.mu.Unlock()
	return nil
}

// newKey returns a cryptographically random key like "wak_<40 hex chars>".
func newKey() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return keyPrefix + hex.EncodeToString(b), nil
}
