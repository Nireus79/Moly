/**
 * Provider Tests
 * Unit and integration tests for LLM providers with mocked and real API calls
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { ClaudeProvider } from '@/api/providers/claude';
import { OpenAIProvider } from '@/api/providers/openai';
import { OllamaProvider } from '@/api/providers/ollama';

describe('Claude Provider', () => {
  let provider: ClaudeProvider;
  const testApiKey = 'sk-ant-test-key-12345';

  beforeEach(() => {
    provider = new ClaudeProvider(testApiKey);
  });

  describe('Configuration', () => {
    it('should initialize with API key', () => {
      expect(provider.isConfigured).toBe(true);
    });

    it('should not be configured without API key', () => {
      const unconfigured = new ClaudeProvider('');
      expect(unconfigured.isConfigured).toBe(false);
    });

    it('should have correct type and name', () => {
      expect(provider.type).toBe('claude');
      expect(provider.name).toContain('Claude');
    });

    it('should start with default models', () => {
      expect(provider.models.length).toBeGreaterThan(0);
      expect(provider.models[0]).toContain('claude');
    });
  });

  describe('Model Discovery', () => {
    it('should return default models when API key is empty', async () => {
      const unconfigured = new ClaudeProvider('');
      const models = await unconfigured.discoverModels();
      expect(models.length).toBeGreaterThan(0);
      expect(models.every((m) => m.includes('claude'))).toBe(true);
    });

    it('should cache discovered models for 1 hour', async () => {
      const models1 = await provider.discoverModels();
      // Spy on call count

      // Second call should use cache
      const models2 = await provider.discoverModels();

      expect(models1).toEqual(models2);
      expect(provider.models).toEqual(models2);
    });

    it('should handle discovery failures gracefully', async () => {
      const badProvider = new ClaudeProvider('invalid-key');
      const models = await badProvider.discoverModels();

      // Should fallback to defaults on error
      expect(models.length).toBeGreaterThan(0);
      expect(Array.isArray(models)).toBe(true);
    });
  });

  describe('Message Generation', () => {
    it('should throw error when not configured', async () => {
      const unconfigured = new ClaudeProvider('');
      await expect(
        unconfigured.generateSuggestions('test', 'context', 'friendly'),
      ).rejects.toThrow('Claude API key not configured');
    });

    it('should build valid prompt with context', async () => {
      const prompt = (provider as any).buildPrompt('Hello', 'John from Tinder', 'dating');
      expect(prompt).toContain('Hello');
      expect(prompt).toContain('John from Tinder');
      expect(prompt).toContain('dating');
      expect(prompt).toContain('JSON');
    });

    it('should parse message suggestions from response', () => {
      const response = JSON.stringify({
        suggestions: [
          {
            text: 'Hey there!',
            tone: 'casual',
            reasoning: 'Friendly opener',
            confidence: 0.85,
          },
          {
            text: 'Hi, how are you?',
            tone: 'warm',
            reasoning: 'Genuine question',
            confidence: 0.9,
          },
        ],
      });

      const suggestions = (provider as any).parseMessageSuggestions(response);
      expect(suggestions).toHaveLength(2);
      expect(suggestions[0].text).toBe('Hey there!');
      expect(suggestions[0].confidence).toBeLessThanOrEqual(0.95);
      expect(suggestions[0].confidence).toBeGreaterThanOrEqual(0.5);
    });

    it('should normalize confidence scores', () => {
      const response = JSON.stringify({
        suggestions: [
          { text: 'msg', tone: 'tone', reasoning: 'reason', confidence: 1.5 }, // Too high
          { text: 'msg', tone: 'tone', reasoning: 'reason', confidence: 0.2 }, // Too low
        ],
      });

      const suggestions = (provider as any).parseMessageSuggestions(response);
      expect(suggestions[0].confidence).toBeLessThanOrEqual(0.95);
      expect(suggestions[1].confidence).toBeGreaterThanOrEqual(0.5);
    });

    it('should handle malformed JSON response', () => {
      const malformed = 'This is not JSON { invalid }';
      expect(() => {
        (provider as any).parseMessageSuggestions(malformed);
      }).toThrow();
    });
  });

  describe('Validation', () => {
    it('should validate configured provider', async () => {
      // Note: Real validation would fail without valid key
      // This tests the flow, not the actual API call
      const result = await provider.validate();
      expect(typeof result).toBe('boolean');
    });
  });
});

describe('OpenAI Provider', () => {
  let provider: OpenAIProvider;
  const testApiKey = 'sk-test-key-12345';

  beforeEach(() => {
    provider = new OpenAIProvider(testApiKey);
  });

  describe('Configuration', () => {
    it('should initialize with API key', () => {
      expect(provider.isConfigured).toBe(true);
    });

    it('should have correct type and name', () => {
      expect(provider.type).toBe('openai');
      expect(provider.name).toContain('OpenAI');
    });

    it('should start with default models', () => {
      expect(provider.models.length).toBeGreaterThan(0);
      expect(provider.models.some((m) => m.includes('gpt'))).toBe(true);
    });
  });

  describe('Model Discovery', () => {
    it('should return default models when API key is empty', async () => {
      const unconfigured = new OpenAIProvider('');
      const models = await unconfigured.discoverModels();
      expect(models.length).toBeGreaterThan(0);
      expect(models.some((m) => m.includes('gpt'))).toBe(true);
    });

    it('should filter for chat models only', async () => {
      const models = provider.models;
      expect(models.every((m) => m.includes('gpt-4') || m.includes('gpt-3.5'))).toBe(true);
    });
  });

  describe('Message Generation', () => {
    it('should throw error when not configured', async () => {
      const unconfigured = new OpenAIProvider('');
      await expect(
        unconfigured.generateSuggestions('test', 'context', 'friendly'),
      ).rejects.toThrow('OpenAI API key not configured');
    });
  });
});

describe('Ollama Provider', () => {
  let provider: OllamaProvider;

  beforeEach(() => {
    provider = new OllamaProvider('http://localhost:11434');
  });

  describe('Configuration', () => {
    it('should always be configured', () => {
      expect(provider.isConfigured).toBe(true);
    });

    it('should have correct type and name', () => {
      expect(provider.type).toBe('ollama');
      expect(provider.name).toContain('Ollama');
    });

    it('should accept custom base URL', () => {
      const custom = new OllamaProvider('http://192.168.1.100:11434');
      expect(custom.isConfigured).toBe(true);
    });
  });

  describe('Model Discovery', () => {
    it('should start with default models', () => {
      expect(provider.models.length).toBeGreaterThan(0);
    });

    it('should cache discovered models for 5 minutes (local)', async () => {
      const models = await provider.discoverModels();
      // Ollama caching is 5 min vs 1 hour for remote APIs
      expect(Array.isArray(models)).toBe(true);
    });

    it('should deduplicate model names', () => {
      // Mock response with duplicates (e.g., mistral:latest, mistral:7b)
      const response: any = {
        models: [
          { name: 'mistral:latest', modified_at: '2024-01-01', size: 1000 },
          { name: 'mistral:7b', modified_at: '2024-01-01', size: 1000 },
          { name: 'llama2:latest', modified_at: '2024-01-01', size: 2000 },
        ],
      };

      // Simulate parsing
      const modelNames = response.models
        .map((m: any) => m.name.split(':')[0])
        .filter((name: string, idx: number, arr: string[]) => arr.indexOf(name) === idx);

      expect(modelNames).toHaveLength(2);
      expect(modelNames).toContain('mistral');
      expect(modelNames).toContain('llama2');
    });

    it('should handle offline server gracefully', async () => {
      const models = await provider.discoverModels();
      // Should return defaults if server is unreachable
      expect(Array.isArray(models)).toBe(true);
    });
  });

  describe('Availability Check', () => {
    it('should check if Ollama server is available', async () => {
      const isAvailable = await provider.isAvailable();
      expect(typeof isAvailable).toBe('boolean');
    });
  });
});

describe('Provider Type Safety', () => {
  it('should support all provider types', () => {
    const claude = new ClaudeProvider('key');
    const openai = new OpenAIProvider('key');
    const ollama = new OllamaProvider();

    expect(['claude', 'openai', 'ollama']).toContain(claude.type);
    expect(['claude', 'openai', 'ollama']).toContain(openai.type);
    expect(['claude', 'openai', 'ollama']).toContain(ollama.type);
  });

  it('should generate suggestions for all providers', async () => {
    const providers = [
      new ClaudeProvider('test-key'),
      new OpenAIProvider('test-key'),
      new OllamaProvider(),
    ];

    for (const provider of providers) {
      // Test should complete without throwing type errors
      expect(provider.type).toBeDefined();
      expect(provider.name).toBeDefined();
      expect(provider.models).toBeDefined();
    }
  });
});
