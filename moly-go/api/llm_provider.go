package api

import (
	"context"
	"fmt"
	"log"
	"os"
)

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	// GenerateCompletion generates a text completion
	GenerateCompletion(ctx context.Context, prompt string, systemPrompt string) (string, error)

	// GetModel returns the model being used
	GetModel() string

	// GetProvider returns the provider name
	GetProvider() string

	// IsHealthy checks if the provider is healthy
	IsHealthy(ctx context.Context) (bool, error)
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider        string // "anthropic", "openai", "ollama"
	AnthropicAPIKey string
	OpenAIAPIKey    string
	OllamaURL       string
	DefaultModel    string // "claude-3-5-sonnet", "gpt-4-turbo", "mistral"
	Temperature     float32
	MaxTokens       int
	Timeout         int // seconds
}

// ProviderFactory creates LLM providers
type ProviderFactory struct {
	config LLMConfig
	cache  map[string]LLMProvider
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(config LLMConfig) *ProviderFactory {
	return &ProviderFactory{
		config: config,
		cache:  make(map[string]LLMProvider),
	}
}

// GetProvider returns a provider by name, or the default
func (pf *ProviderFactory) GetProvider(providerName string) (LLMProvider, error) {
	if providerName == "" {
		providerName = pf.config.Provider
	}

	// Check cache first
	if provider, exists := pf.cache[providerName]; exists {
		return provider, nil
	}

	var provider LLMProvider
	var err error

	switch providerName {
	case "anthropic":
		provider, err = NewAnthropicProvider(pf.config)
	case "openai":
		provider, err = NewOpenAIProvider(pf.config)
	case "ollama":
		provider, err = NewOllamaProvider(pf.config)
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	if err != nil {
		return nil, err
	}

	// Cache it
	pf.cache[providerName] = provider
	log.Printf("[ProviderFactory] Cached provider: %s\n", providerName)

	return provider, nil
}

// GetAvailableProviders returns list of available providers
func (pf *ProviderFactory) GetAvailableProviders() []string {
	providers := []string{}

	if pf.config.AnthropicAPIKey != "" {
		providers = append(providers, "anthropic")
	}
	if pf.config.OpenAIAPIKey != "" {
		providers = append(providers, "openai")
	}
	if pf.config.OllamaURL != "" {
		providers = append(providers, "ollama")
	}

	return providers
}

// GetHealthyProvider returns the first healthy provider
func (pf *ProviderFactory) GetHealthyProvider(ctx context.Context) (LLMProvider, error) {
	providers := pf.GetAvailableProviders()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no LLM providers configured")
	}

	log.Printf("[ProviderFactory] Checking %d providers for health...\n", len(providers))

	for _, name := range providers {
		provider, err := pf.GetProvider(name)
		if err != nil {
			log.Printf("[ProviderFactory] Failed to get %s: %v\n", name, err)
			continue
		}

		healthy, err := provider.IsHealthy(ctx)
		if err != nil {
			log.Printf("[ProviderFactory] Health check failed for %s: %v\n", name, err)
			continue
		}

		if healthy {
			log.Printf("[ProviderFactory] Using provider: %s (%s)\n", name, provider.GetModel())
			return provider, nil
		}
	}

	return nil, fmt.Errorf("no healthy LLM providers available")
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadLLMConfigFromEnv() LLMConfig {
	return LLMConfig{
		Provider:        getEnv("MOLY_LLM_PROVIDER", "anthropic"),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
		OllamaURL:       getEnv("OLLAMA_URL", "http://localhost:11434"),
		DefaultModel:    getEnv("MOLY_LLM_MODEL", "claude-3-5-sonnet-20241022"),
		Temperature:     0.3, // For consistency in extraction
		MaxTokens:       2048,
		Timeout:         30,
	}
}

// Helper function
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
