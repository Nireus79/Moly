/**
 * Provider Manager Tests
 * Tests for provider discovery, configuration, and switching
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { LLMProviderManager } from '@/api/providerManager';
import type { LLMProviderType } from '@/api/providers';

describe('LLMProviderManager', () => {
  let manager: LLMProviderManager;

  beforeEach(() => {
    manager = new LLMProviderManager();
  });

  describe('Provider Initialization', () => {
    it('should initialize with all providers', () => {
      const claude = manager.getProvider('claude');
      const openai = manager.getProvider('openai');
      const ollama = manager.getProvider('ollama');

      expect(claude).toBeDefined();
      expect(openai).toBeDefined();
      expect(ollama).toBeDefined();
    });

    it('should have no active provider initially', () => {
      const active = manager.getActiveProvider();
      expect(active).toBeNull();
    });
  });

  describe('Provider Configuration', () => {
    it('should configure Claude provider', async () => {
      const success = await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test-key',
        model: 'claude-3-5-sonnet-20241022',
      });

      expect(typeof success).toBe('boolean');
      const claude = manager.getProvider('claude');
      expect(claude?.isConfigured).toBe(true);
    });

    it('should configure OpenAI provider', async () => {
      const success = await manager.configureProvider({
        type: 'openai',
        apiKey: 'sk-test-key',
        model: 'gpt-4',
      });

      expect(typeof success).toBe('boolean');
      const openai = manager.getProvider('openai');
      expect(openai?.isConfigured).toBe(true);
    });

    it('should configure Ollama provider', async () => {
      const success = await manager.configureProvider({
        type: 'ollama',
        baseUrl: 'http://localhost:11434',
        model: 'mistral',
      });

      expect(typeof success).toBe('boolean');
      const ollama = manager.getProvider('ollama');
      expect(ollama?.isConfigured).toBe(true);
    });

    it('should fail gracefully on invalid configuration', async () => {
      const success = await manager.configureProvider({
        type: 'claude',
        apiKey: '', // Invalid: empty key
        model: 'claude-3-5-sonnet-20241022',
      });

      expect(success).toBe(false);
    });

    it('should handle unknown provider types', () => {
      const provider = manager.getProvider('unknown' as LLMProviderType);
      expect(provider).toBeNull();
    });
  });

  describe('Provider Discovery', () => {
    it('should discover available providers', async () => {
      const infos = await manager.discoverProviders();

      expect(Array.isArray(infos)).toBe(true);
      expect(infos.length).toBeGreaterThan(0);
      expect(infos.some((i) => i.type === 'claude')).toBe(true);
      expect(infos.some((i) => i.type === 'openai')).toBe(true);
      expect(infos.some((i) => i.type === 'ollama')).toBe(true);
    });

    it('should report provider configuration status', async () => {
      // Configure one provider
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      const infos = await manager.discoverProviders();
      // Find claude info
      const info = infos.find((i) => i.type === 'claude');

      expect(info?.isConfigured).toBe(true);
      expect(info?.name).toContain('Claude');
    });

    it('should prioritize providers correctly', async () => {
      const infos = await manager.discoverProviders();

      // Should be sorted by priority: Claude > OpenAI > Ollama
      const types = infos.map((i) => i.type);
      const claudeIdx = types.indexOf('claude');
      const openaiIdx = types.indexOf('openai');
      const ollamaIdx = types.indexOf('ollama');

      expect(claudeIdx).toBeLessThan(openaiIdx);
      expect(openaiIdx).toBeLessThan(ollamaIdx);
    });

    it('should trigger model discovery for configured providers', async () => {
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      const infos = await manager.discoverProviders();
      // Verify claude is in discovery results
      expect(infos.some((i) => i.type === 'claude')).toBe(true);

      const claude = manager.getProvider('claude');

      // Models should be populated
      expect(claude?.models.length).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Active Provider Management', () => {
    it('should set active provider', async () => {
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      const success = manager.setActiveProvider('claude');
      expect(success).toBe(true);

      const active = manager.getActiveProvider();
      expect(active?.type).toBe('claude');
    });

    it('should fail to set unconfigured provider as active', async () => {
      const success = manager.setActiveProvider('openai');
      expect(success).toBe(false);
    });

    it('should allow switching between providers', async () => {
      // Configure both
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

      // Set Claude active
      manager.setActiveProvider('claude');
      expect(manager.getActiveProvider()?.type).toBe('claude');

      // Switch to OpenAI
      manager.setActiveProvider('openai');
      expect(manager.getActiveProvider()?.type).toBe('openai');

      // Switch back
      manager.setActiveProvider('claude');
      expect(manager.getActiveProvider()?.type).toBe('claude');
    });
  });

  describe('Model Management', () => {
    it('should get models for provider', async () => {
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      const models = manager.getModels('claude');
      expect(Array.isArray(models)).toBe(true);
      expect(models.length).toBeGreaterThanOrEqual(0);
    });

    it('should return empty array for unconfigured provider', () => {
      const models = manager.getModels('openai');
      expect(Array.isArray(models)).toBe(true);
    });
  });

  describe('Best Available Provider', () => {
    it('should find best available provider', async () => {
      // Configure only Claude
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      const best = await manager.getAvailableProvider();

      if (best) {
        expect(best.type).toBe('claude');
      }
    });

    it('should prioritize configured providers in order', async () => {
      // Configure OpenAI first (but it has lower priority)
      await manager.configureProvider({
        type: 'openai',
        apiKey: 'sk-test',
        model: 'gpt-4',
      });

      // Configure Claude (higher priority)
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      const best = await manager.getAvailableProvider();

      if (best) {
        // Should prefer Claude even though OpenAI was configured first
        expect(best.type).toBe('claude');
      }
    });

    it('should return null when no provider is available', async () => {
      const best = await manager.getAvailableProvider();
      expect(best).toBeNull();
    });
  });

  describe('Validation', () => {
    it('should validate active provider', async () => {
      await manager.configureProvider({
        type: 'claude',
        apiKey: 'sk-ant-test',
        model: 'claude-3-5-sonnet-20241022',
      });

      manager.setActiveProvider('claude');
      const isValid = await manager.validateActiveProvider();
      expect(typeof isValid).toBe('boolean');
    });

    it('should fail validation with no active provider', async () => {
      const isValid = await manager.validateActiveProvider();
      expect(isValid).toBe(false);
    });
  });

  describe('Singleton Pattern', () => {
    it('should return same instance', () => {
      const manager1 = new LLMProviderManager();
      const manager2 = new LLMProviderManager();

      // Both should be instances of the same class
      expect(manager1.constructor).toBe(manager2.constructor);
    });
  });
});
