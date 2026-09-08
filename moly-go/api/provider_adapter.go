package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/tools"
)

// ProviderAdapter adapts the new LLMProvider interface to work with tools.LLMProvider
type ProviderAdapter struct {
	provider LLMProvider
	factory  *ProviderFactory
}

// NewProviderAdapter creates a new adapter from the provider factory
func NewProviderAdapter(factory *ProviderFactory) (*ProviderAdapter, error) {
	// Get a healthy provider first
	provider, err := factory.GetHealthyProvider(context.Background())
	if err != nil {
		return nil, fmt.Errorf("no healthy provider available: %w", err)
	}

	log.Printf("[ProviderAdapter] Using provider: %s (%s)\n", provider.GetProvider(), provider.GetModel())

	return &ProviderAdapter{
		provider: provider,
		factory:  factory,
	}, nil
}

// Call implements tools.LLMProvider interface
func (pa *ProviderAdapter) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now()

	// Use the adapter's current provider
	response, err := pa.provider.GenerateCompletion(ctx, req.UserPrompt, req.SystemPrompt)
	if err != nil {
		// On error, try to get another healthy provider
		log.Printf("[ProviderAdapter] Provider %s failed, trying fallback: %v\n", pa.provider.GetProvider(), err)

		fallback, fallbackErr := pa.factory.GetHealthyProvider(ctx)
		if fallbackErr != nil {
			return nil, fmt.Errorf("primary provider failed and no fallback available: %w", err)
		}

		// Use fallback provider
		pa.provider = fallback
		response, err = pa.provider.GenerateCompletion(ctx, req.UserPrompt, req.SystemPrompt)
		if err != nil {
			return nil, fmt.Errorf("fallback provider also failed: %w", err)
		}
	}

	duration := time.Since(start)

	return &tools.LLMResponse{
		Content:          response,
		Model:            pa.provider.GetModel(),
		ProcessingTimeMs: duration.Milliseconds(),
		TokensUsed:       0, // Not tracked in this implementation
		CostUSD:          0, // Not tracked in this implementation
		StopReason:       "end_turn",
		ThinkingContent:  "",
	}, nil
}

// GetProvider returns the current provider being used
func (pa *ProviderAdapter) GetProvider() LLMProvider {
	return pa.provider
}

// GetFactory returns the underlying provider factory
func (pa *ProviderAdapter) GetFactory() *ProviderFactory {
	return pa.factory
}

// SwitchProvider manually switches to a different provider
func (pa *ProviderAdapter) SwitchProvider(providerName string) error {
	provider, err := pa.factory.GetProvider(providerName)
	if err != nil {
		return err
	}

	pa.provider = provider
	log.Printf("[ProviderAdapter] Switched to provider: %s\n", providerName)
	return nil
}

// GetHealthyProvider attempts to find a healthy provider (including current)
func (pa *ProviderAdapter) GetHealthyProvider(ctx context.Context) error {
	provider, err := pa.factory.GetHealthyProvider(ctx)
	if err != nil {
		return err
	}

	pa.provider = provider
	return nil
}
