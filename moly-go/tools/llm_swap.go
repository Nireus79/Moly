package tools

import (
	"context"
	"errors"
	"sync"
)

// ErrNoProvider is returned while no AI provider is usable (not configured yet, or the key is locked).
var ErrNoProvider = errors.New("no AI provider is available: choose one in Moly's settings (or unlock the saved API key)")

// SwappableLLM lets the backend change the AI provider while it runs. Until Set is called every call fails with
// ErrNoProvider, so the server can start before the user has chosen a provider.
type SwappableLLM struct {
	mu    sync.RWMutex
	inner LLMProvider
	name  string
}

func NewSwappableLLM() *SwappableLLM { return &SwappableLLM{} }

// Set installs the provider (nil removes it, which is how locking works).
func (s *SwappableLLM) Set(p LLMProvider, name string) {
	s.mu.Lock()
	s.inner, s.name = p, name
	s.mu.Unlock()
}

func (s *SwappableLLM) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	s.mu.RLock()
	inner := s.inner
	s.mu.RUnlock()
	if inner == nil {
		return nil, ErrNoProvider
	}
	return inner.Call(ctx, req)
}

// ProviderName is the provider in use, or "none".
func (s *SwappableLLM) ProviderName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.inner == nil {
		return "none"
	}
	return s.name
}
