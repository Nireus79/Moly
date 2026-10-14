package tools

import (
	"context"
	"errors"
)

// ErrNoProvider is returned while a user has no usable AI provider (not chosen yet, or the key is locked).
var ErrNoProvider = errors.New("no AI provider is available: choose one in Moly's settings (or unlock the saved API key)")

// ErrNoUser is returned by a provider that belongs to nobody. Code that has no user gets this and nothing else: there is no
// shared provider to fall back to.
var ErrNoUser = errors.New("no user: a model call needs the user whose provider it uses")

// LLMSource is the one place that hands out AI providers, and it hands out one per USER. Everything that calls a model gets
// its provider from here, through For(userID), when the user is known; a component built for one user can only ever use that
// user's provider and key. Unbound() is for the parts of the program that exist before any user does: it fails every call.
type LLMSource interface {
	// For returns the provider of the user. It looks the user's settings up at every call, so a key that is changed, locked or
	// removed takes effect at once.
	For(userID string) LLMProvider
	// Unbound returns a provider that fails every call with ErrNoUser.
	Unbound() LLMProvider
}

type unboundLLM struct{}

func (unboundLLM) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	return nil, ErrNoUser
}

// UnboundLLM is the provider of nobody.
func UnboundLLM() LLMProvider { return unboundLLM{} }

// staticSource gives the same provider to every user. For tests, which script one model, and for the operator override.
type staticSource struct{ p LLMProvider }

// StaticSource returns an LLMSource that gives p to every user.
func StaticSource(p LLMProvider) LLMSource { return staticSource{p} }

func (s staticSource) For(userID string) LLMProvider { return s.p }
func (s staticSource) Unbound() LLMProvider          { return s.p }
