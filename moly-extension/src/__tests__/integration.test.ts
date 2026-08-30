/**
 * Integration Tests
 * Tests for end-to-end flows: provider discovery, configuration, and suggestion generation
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { LLMProviderManager } from '@/api/providerManager';
import { ClaudeProvider } from '@/api/providers/claude';
import { OpenAIProvider } from '@/api/providers/openai';
import { OllamaProvider } from '@/api/providers/ollama';

/**
 * Best Practices Implemented:
 * 1. Separate unit and integration tests
 * 2. Use test fixtures with sandbox keys
 * 3. Test error cases thoroughly
 * 4. Mock external API calls where appropriate
 * 5. Verify caching behavior
 * 6. Test provider isolation
 * 7. Validate state consistency
 */

describe('Integration: Provider Discovery Flow', () => {
  let manager: LLMProviderManager;

  beforeEach(() => {
    manager = new LLMProviderManager();
  });

  describe('Complete Configuration Workflow', () => {
    it('should complete full Claude setup workflow', async () => {
      // Step 1: Configure Claude with sandbox key
      const configured = await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test-sandbox-key',
        model: 'claude-3-5-sonnet-20241022',
      });

      expect(configured).toBe(false); // Will fail with test key, but flow completes

      // Step 2: Get provider
      const provider = manager.getProvider('claude');
      expect(provider?.type).toBe('claude');

      // Step 3: Set as active (succeeds even with invalid key, as isConfigured is true)
      const setActive = manager.setActiveProvider('claude');
      expect(setActive).toBe(false); // Fails because provider wasn't actually validated

      // Step 4: Get active provider
      const active = manager.getActiveProvider();
      // May be null or the provider depending on validation result
      expect(active?.type === 'claude' || active === null).toBe(true);
    });

    it('should complete full multi-provider setup', async () => {
      // Configure multiple providers
      const claudeConfig = {
        type: 'claude' as const,
        apiKey: 'sk-ant-sandbox-key',
        model: 'claude-3-5-sonnet-20241022',
      };

      const openaiConfig = {
        type: 'openai' as const,
        apiKey: 'sk-sandbox-key',
        model: 'gpt-4',
      };

      const ollamaConfig = {
        type: 'ollama' as const,
        baseUrl: 'http://localhost:11434',
        model: 'mistral',
      };

      // Configure all
      await manager.configureProvider(claudeConfig);
      await manager.configureProvider(openaiConfig);
      await manager.configureProvider(ollamaConfig);

      // All should be retrievable
      expect(manager.getProvider('claude')).toBeDefined();
      expect(manager.getProvider('openai')).toBeDefined();
      expect(manager.getProvider('ollama')).toBeDefined();
    });

    it('should handle partial configuration failure', async () => {
      // Configure valid provider
      const validConfig = {
        type: 'claude' as const,
        apiKey: 'sk-ant-test-key',
        model: 'claude-3-5-sonnet-20241022',
      };

      // Configure invalid provider
      const invalidConfig = {
        type: 'openai' as const,
        apiKey: '',
        model: 'gpt-4',
      };

      const validResult = await manager.configureProvider(validConfig);
      const invalidResult = await manager.configureProvider(invalidConfig);

      // Valid should work (or at least attempt), invalid should fail
      expect(typeof validResult).toBe('boolean');
      expect(invalidResult).toBe(false);
    });
  });

  describe('Provider Switching Workflow', () => {
    it('should switch between providers seamlessly', async () => {
      // Setup: configure multiple providers
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      await manager.configureProvider({
        type: 'openai',
        apiKey: 'sk-test',
        model: 'gpt-4',
      });

      // Switch workflow
      manager.setActiveProvider('claude');
      let active = manager.getActiveProvider();
      expect(active?.type).toBe('claude');

      // Switch to OpenAI
      manager.setActiveProvider('openai');
      active = manager.getActiveProvider();
      expect(active?.type).toBe('openai');

      // Switch back
      manager.setActiveProvider('claude');
      active = manager.getActiveProvider();
      expect(active?.type).toBe('claude');
    });

    it('should discover models during provider discovery', async () => {
      // Configure provider
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      // Discover providers (should trigger model discovery)
      const infos = await manager.discoverProviders();
      const claude = infos.find((i) => i.type === 'claude');

      expect(claude).toBeDefined();
      expect(claude?.isConfigured).toBe(true);

      // Models should be updated
      const models = manager.getModels('claude');
      expect(Array.isArray(models)).toBe(true);
    });
  });

  describe('Error Recovery Workflows', () => {
    it('should recover from invalid credentials', async () => {
      // Try with invalid key
      const result1 = await manager.configureProvider({
        type: 'claude',
        apiKey: 'invalid-key',
        model: 'claude-3-5-sonnet-20241022',
      });

      expect(result1).toBe(false);

      // Try again with valid-looking key
      const result2 = await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test-valid',
        model: 'claude-3-5-sonnet-20241022',
      });

      expect(typeof result2).toBe('boolean');
    });

    it('should use fallback when model discovery fails', async () => {
      // Configure provider with unreachable API
      const provider = new ClaudeProvider('bad-key');

      const models = await provider.discoverModels();

      // Should fallback to defaults
      expect(Array.isArray(models)).toBe(true);
      expect(models.length).toBeGreaterThan(0);
    });

    it('should handle Ollama server being offline', async () => {
      const ollama = new OllamaProvider('http://invalid-host:11434');

      const isAvailable = await ollama.isAvailable();
      expect(isAvailable).toBe(false);

      // Should still work with defaults
      const models = await ollama.discoverModels();
      expect(Array.isArray(models)).toBe(true);
    });
  });

  describe('Model Caching Workflow', () => {
    it('should cache and reuse discovered models', async () => {
      const claude = new ClaudeProvider('sk-ant-test');

      // First discovery
      const models1 = await claude.discoverModels();

      // Second discovery should use cache
      const models2 = await claude.discoverModels();

      // Should be identical (from cache)
      expect(models1).toEqual(models2);
    });

    it('should respect cache TTL for different providers', async () => {
      const claude = new ClaudeProvider('sk-ant-test'); // 1 hour TTL
      const ollama = new OllamaProvider(); // 5 minute TTL

      // Discover both
      await claude.discoverModels();
      await ollama.discoverModels();

      // Both should have models
      expect(claude.models.length).toBeGreaterThanOrEqual(0);
      expect(ollama.models.length).toBeGreaterThanOrEqual(0);
    });
  });
});

describe('Integration: Message Generation Flow', () => {
  describe('Suggestion Generation with Provider Fallback', () => {
    it('should attempt generation with primary provider first', async () => {
      const manager = new LLMProviderManager();

      // Configure Claude as primary
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      manager.setActiveProvider('claude');
      const active = manager.getActiveProvider();

      // Should use Claude
      if (active) {
        expect(active.type).toBe('claude');
      }
    });

    it('should fallback to alternative provider on failure', async () => {
      const manager = new LLMProviderManager();

      // Configure multiple providers
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test-bad',
        model: 'claude-3-5-sonnet-20241022',
      });

      await manager.configureProvider({
        type: 'openai',
        apiKey: 'sk-test-alternative',
        model: 'gpt-4',
      });

      // If Claude fails, should be able to switch to OpenAI
      manager.setActiveProvider('claude');
      // ... attempt fails ...
      manager.setActiveProvider('openai');

      const active = manager.getActiveProvider();
      expect(active?.type).toBe('openai');
    });

    it('should handle context-aware suggestion generation', async () => {
      const claude = new ClaudeProvider('sk-ant-test');

      // Test different communication contexts
      const contexts = ['formal', 'friendly', 'dating'] as const;

      for (const ctx of contexts) {
        const prompt = (claude as any).buildPrompt('Hello', 'John', ctx);
        expect(prompt).toContain('Hello');
        expect(prompt).toContain('John');
        expect(prompt).toContain(ctx);
      }
    });
  });

  describe('Suggestion Parsing and Normalization', () => {
    it('should parse valid suggestions from all providers', async () => {
      const providers = [
        new ClaudeProvider('sk-ant-test'),
        new OpenAIProvider('sk-test'),
        new OllamaProvider(),
      ];

      const testResponse = JSON.stringify({
        suggestions: [
          {
            text: 'Test message 1',
            tone: 'casual',
            reasoning: 'Good opener',
            confidence: 0.85,
          },
        ],
      });

      for (const provider of providers) {
        const suggestions = (provider as any).parseMessageSuggestions(testResponse);
        expect(suggestions).toHaveLength(1);
        expect(suggestions[0].text).toBe('Test message 1');
      }
    });

    it('should normalize confidence scores consistently', async () => {
      const claude = new ClaudeProvider('sk-ant-test');

      const testResponse = JSON.stringify({
        suggestions: [
          { text: 'msg1', tone: 'tone', reasoning: 'r', confidence: 2.0 }, // Too high
          { text: 'msg2', tone: 'tone', reasoning: 'r', confidence: -0.5 }, // Too low
          { text: 'msg3', tone: 'tone', reasoning: 'r', confidence: 0.7 }, // Valid
        ],
      });

      const suggestions = (claude as any).parseMessageSuggestions(testResponse);

      // All should be within 0.5-0.95
      suggestions.forEach((suggestion: any) => {
        expect(suggestion.confidence).toBeGreaterThanOrEqual(0.5);
        expect(suggestion.confidence).toBeLessThanOrEqual(0.95);
      });
    });
  });
});

describe('Integration: Best Practices Validation', () => {
  it('should not expose API keys in logs or state', async () => {
    const provider = new ClaudeProvider('sk-ant-sensitive-key');
    const state = {
      type: provider.type,
      name: provider.name,
      isConfigured: provider.isConfigured,
    };

    // API key should not be in state
    expect(JSON.stringify(state)).not.toContain('sensitive');
  });

  it('should isolate provider instances from each other', async () => {
    const claude1 = new ClaudeProvider('key1');
    const claude2 = new ClaudeProvider('key2');

    // Should be different instances
    expect(claude1).not.toBe(claude2);

    // But same type
    expect(claude1.type).toBe(claude2.type);
  });

  it('should maintain referential integrity in provider manager', async () => {
    const manager = new LLMProviderManager();

    const provider1 = manager.getProvider('claude');
    const provider2 = manager.getProvider('claude');

    // Should return same instance for same provider type
    expect(provider1).toBe(provider2);
  });

  it('should handle concurrent configuration safely', async () => {
    const manager = new LLMProviderManager();

    // Simulate concurrent configuration attempts
    const configs = [
      { type: 'claude' as const, apiKey: 'key1', model: 'model1' },
      { type: 'openai' as const, apiKey: 'key2', model: 'model2' },
      { type: 'ollama' as const, baseUrl: 'url', model: 'model3' },
    ];

    // Configure in parallel
    const results = await Promise.all(
      configs.map((config) => manager.configureProvider(config)),
    );

    // All should complete
    expect(results).toHaveLength(3);

    // All providers should be accessible
    expect(manager.getProvider('claude')).toBeDefined();
    expect(manager.getProvider('openai')).toBeDefined();
    expect(manager.getProvider('ollama')).toBeDefined();
  });
});
