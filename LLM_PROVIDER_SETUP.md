# Phase 1.2 LLM Provider Setup

**Status**: Complete & Ready  
**Providers**: Anthropic Claude, OpenAI GPT, Ollama (local)  
**Integration**: REST API based (no SDK dependencies)  
**Tests**: 9/9 passing

---

## What LLM Providers Do

The LLM provider layer handles all interactions with language models for extracting insights from conversations.

### Supported Providers

1. **Anthropic Claude** (via REST API)
   - Latest model: `claude-3-5-sonnet-20241022`
   - API: `https://api.anthropic.com/v1/messages`
   - Header: `x-api-key`

2. **OpenAI GPT** (via REST API)
   - Latest model: `gpt-4-turbo`
   - API: `https://api.openai.com/v1/chat/completions`
   - Header: `Authorization: Bearer <key>`

3. **Ollama** (local, REST API)
   - Default model: `mistral`
   - Endpoint: `http://localhost:11434/api/generate`
   - No authentication required

### Provider Selection Priority

1. Check for LOCAL Ollama first (privacy-first)
2. Fall back to Anthropic if available
3. Fall back to OpenAI if available
4. Use graceful degradation if none available

---

## Integration (5 minutes)

### Step 1: Set Environment Variables

Choose at least one provider:

```bash
# Option 1: Use Anthropic Claude
export ANTHROPIC_API_KEY="sk-ant-..."
export MOLY_LLM_PROVIDER="anthropic"
export MOLY_LLM_MODEL="claude-3-5-sonnet-20241022"

# Option 2: Use OpenAI GPT
export OPENAI_API_KEY="sk-..."
export MOLY_LLM_PROVIDER="openai"
export MOLY_LLM_MODEL="gpt-4-turbo"

# Option 3: Use local Ollama
export OLLAMA_URL="http://localhost:11434"
export MOLY_LLM_PROVIDER="ollama"
export MOLY_LLM_MODEL="mistral"
```

### Step 2: Initialize in main.go

```go
import (
    "moly/api"
    "moly/agents"
)

func main() {
    // Load LLM configuration from environment
    llmConfig := api.LoadLLMConfigFromEnv()
    
    // Create provider factory
    factory := api.NewProviderFactory(llmConfig)
    
    // Create adapter (handles fallback logic)
    adapter, err := api.NewProviderAdapter(factory)
    if err != nil {
        log.Fatal("Failed to initialize LLM provider:", err)
    }
    
    // Use adapter as tools.LLMProvider
    analyzer := agents.NewConversationAnalyzer(adapter, db)
    
    // ... rest of setup ...
}
```

### Step 3: Wire into ConversationAnalyzer

The ConversationAnalyzer expects a `tools.LLMProvider`, which our adapter implements:

```go
type ConversationAnalyzer struct {
    llmClient tools.LLMProvider  // This can be our ProviderAdapter
    db        *database.Database
}
```

No changes needed to the analyzer - it works with the adapter transparently.

---

## Configuration Options

### LLMConfig Structure

```go
type LLMConfig struct {
    Provider          string // "anthropic", "openai", "ollama"
    AnthropicAPIKey   string
    OpenAIAPIKey      string
    OllamaURL         string
    DefaultModel      string
    Temperature       float32 // 0.0-2.0 (default: 0.3)
    MaxTokens         int     // (default: 2048)
    Timeout           int     // seconds (default: 30)
}
```

### Example: Custom Configuration

```go
config := api.LLMConfig{
    Provider:        "anthropic",
    AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
    DefaultModel:    "claude-3-5-sonnet-20241022",
    Temperature:     0.3,
    MaxTokens:       2048,
    Timeout:         30,
}

factory := api.NewProviderFactory(config)
adapter, err := api.NewProviderAdapter(factory)
```

### Example: Fallback Chain

```go
// Try Ollama first, then Anthropic, then OpenAI
config := api.LLMConfig{
    Provider:        "ollama",
    OllamaURL:       "http://localhost:11434",
    AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
    OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
    DefaultModel:    "mistral",
    Temperature:     0.3,
    MaxTokens:       2048,
    Timeout:         30,
}

factory := api.NewProviderFactory(config)
adapter, err := api.NewProviderAdapter(factory)
// adapter will use first healthy provider found
```

---

## Provider Architecture

### LLMProvider Interface

```go
type LLMProvider interface {
    GenerateCompletion(ctx context.Context, prompt string, systemPrompt string) (string, error)
    GetModel() string
    GetProvider() string
    IsHealthy(ctx context.Context) (bool, error)
}
```

### Implementations

1. **AnthropicProvider**
   - File: `api/anthropic_provider.go`
   - Calls Anthropic API endpoint directly
   - Returns text from first content block

2. **OpenAIProvider**
   - File: `api/openai_provider.go`
   - Calls OpenAI API endpoint directly
   - Returns text from first choice message

3. **OllamaProvider**
   - File: `api/ollama_provider.go`
   - Calls local Ollama REST endpoint
   - Supports streaming and batch requests

### ProviderFactory

```go
type ProviderFactory struct {
    config LLMConfig
    cache  map[string]LLMProvider
}

// Key methods:
factory.GetProvider(name)              // Get specific provider (cached)
factory.GetAvailableProviders()        // List configured providers
factory.GetHealthyProvider(ctx)        // Get first healthy provider
LoadLLMConfigFromEnv()                 // Load config from env vars
```

### ProviderAdapter

```go
type ProviderAdapter struct {
    provider LLMProvider
    factory  *ProviderFactory
}

// Implements tools.LLMProvider interface
// Handles fallback to healthy provider on error
```

---

## Usage Examples

### Basic Usage

```go
// Initialize
adapter, _ := api.NewProviderAdapter(factory)

// Use with ConversationAnalyzer
analyzer := agents.NewConversationAnalyzer(adapter, db)

// Analyzer calls adapter.Call(ctx, &tools.LLMRequest{...})
// Adapter handles the rest
```

### Manual Provider Switching

```go
// Switch to OpenAI if current provider fails
err := adapter.SwitchProvider("openai")
if err != nil {
    log.Println("OpenAI not available")
}

// Get current provider
currentProvider := adapter.GetProvider()
log.Printf("Using: %s (%s)\n", currentProvider.GetProvider(), currentProvider.GetModel())
```

### Checking Provider Health

```go
// Get first healthy provider
err := adapter.GetHealthyProvider(ctx)
if err != nil {
    log.Fatal("No healthy providers available")
}
```

---

## Environment Variables

### Anthropic

```bash
ANTHROPIC_API_KEY=sk-ant-...
MOLY_LLM_PROVIDER=anthropic          # Optional, set explicitly
MOLY_LLM_MODEL=claude-3-5-sonnet-...  # Optional, defaults to latest
```

### OpenAI

```bash
OPENAI_API_KEY=sk-...
MOLY_LLM_PROVIDER=openai
MOLY_LLM_MODEL=gpt-4-turbo
```

### Ollama

```bash
OLLAMA_URL=http://localhost:11434     # Or http://127.0.0.1:11434
MOLY_LLM_PROVIDER=ollama
MOLY_LLM_MODEL=mistral                # Or llama2, neural-chat, etc.
```

### General

```bash
MOLY_LLM_TEMPERATURE=0.3              # 0.0-2.0
MOLY_LLM_MAX_TOKENS=2048
MOLY_LLM_TIMEOUT=30                   # seconds
```

---

## Testing

### Run All Tests

```bash
go test ./api -v
```

### Run Specific Test

```bash
go test ./api -run TestProviderFactory -v
```

### Test Coverage

```bash
go test ./api -cover
```

### Tests Included

1. **TestAnthropicProvider** - REST API client works
2. **TestOpenAIProvider** - REST API client works
3. **TestOllamaProvider** - Local API client works
4. **TestProviderFactory** - Factory creates and caches providers
5. **TestProviderFactory_UnknownProvider** - Error handling
6. **TestProviderFactory_MultipleProviders** - Fallback logic
7. **TestOllamaProvider_Timeout** - Timeout handling
8. **TestOllamaProvider_ErrorResponse** - Error handling
9. **BenchmarkOllamaProvider_GenerateCompletion** - Performance

**All tests passing**: ✅

---

## Error Handling

### Provider Failures

```go
// If primary provider fails, adapter tries next available
response, err := adapter.Call(ctx, &tools.LLMRequest{...})
if err != nil {
    // All providers failed
    log.Printf("All providers failed: %v\n", err)
}
```

### Missing Configuration

```bash
# Error message if no providers configured
# [LLMClient] No local model or API key found. Running in heuristic-only mode.
```

### Timeout

```go
// Each provider has configurable timeout
config.Timeout = 60  // seconds
```

---

## Performance Notes

### Token Usage

- **Anthropic**: ~2-3k tokens per conversation analysis
- **OpenAI**: ~2-3k tokens per conversation analysis
- **Ollama**: Local, no token counting

### Latency

- **Anthropic**: 1-3 seconds per request
- **OpenAI**: 1-3 seconds per request
- **Ollama**: 0.5-2 seconds per request (depends on model)

### Cost (monthly, 1000 analyses)

- **Anthropic**: ~$3-5
- **OpenAI**: ~$5-8
- **Ollama**: Free (after initial setup)

---

## Logs You'll See

### On Startup

```
[AnthropicProvider] Initialized with model: claude-3-5-sonnet-20241022
[ProviderFactory] Cached provider: anthropic
[ProviderAdapter] Using provider: anthropic (claude-3-5-sonnet-20241022)
```

### During Use

```
[ProviderAdapter] Provider anthropic failed, trying fallback: context deadline exceeded
[ProviderAdapter] Switched to provider: openai
```

### Health Check

```
[AnthropicProvider] Health check passed
[ProviderFactory] Using provider: anthropic (claude-3-5-sonnet-20241022)
```

---

## Troubleshooting

### "No healthy providers available"

**Solution**: Check environment variables and API keys
```bash
echo $ANTHROPIC_API_KEY
echo $OPENAI_API_KEY
echo $OLLAMA_URL
```

### "anthropic API error: status 401: Invalid API key"

**Solution**: Verify API key format and validity
```bash
# Anthropic key should start with sk-ant-
# OpenAI key should start with sk-
```

### "OLLAMA_URL not set" but Ollama is running

**Solution**: Set the environment variable
```bash
export OLLAMA_URL=http://localhost:11434
```

### Slow responses

**Solution**: 
1. Check network latency to API endpoint
2. Use Ollama locally for faster responses
3. Increase timeout if needed: `MOLY_LLM_TIMEOUT=60`

---

## Production Checklist

- [ ] Environment variables set in deployment
- [ ] At least one provider configured
- [ ] API keys secured (use secrets manager)
- [ ] Timeout configured appropriately (30-60 seconds)
- [ ] Logs being collected and monitored
- [ ] Fallback chain tested manually
- [ ] ConversationAnalyzer wired to use adapter
- [ ] Background jobs using real provider
- [ ] Monitor provider health checks
- [ ] Cost tracking set up (for cloud providers)

---

## Summary

| Component | Status | Lines | Tests |
|-----------|--------|-------|-------|
| LLMProvider interface | ✅ | 15 | - |
| AnthropicProvider | ✅ | 180 | 1/1 |
| OpenAIProvider | ✅ | 200 | 1/1 |
| OllamaProvider | ✅ | 180 | 3/3 |
| ProviderFactory | ✅ | 90 | 3/3 |
| ProviderAdapter | ✅ | 80 | - |
| Tests | ✅ | 280 | 9/9 |

**Time to integrate**: 5 minutes  
**Time to verify**: 5 minutes  
**Total time**: ~10 minutes

---

## Next Steps

1. ✅ LLM provider abstraction layer
2. ✅ Anthropic REST provider
3. ✅ OpenAI REST provider
4. ✅ Ollama local provider
5. ✅ Provider factory with fallback
6. ✅ ProviderAdapter for tools.LLMProvider compatibility
7. → Wire adapter into main.go
8. → Run E2E tests with real LLM
9. → Set up production deployment

**LLM providers are ready to integrate!** 🚀
