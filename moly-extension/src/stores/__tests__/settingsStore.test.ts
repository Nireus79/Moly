/**
 * Settings Store Tests
 * Tests for multi-provider settings persistence and state management
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useSettingsStore } from '@/stores/settingsStore';

// In-memory storage for tests
let testStorage: Record<string, any> = {};

describe('Settings Store - Multi-Provider', () => {
  beforeEach(() => {
    // Clear test storage
    testStorage = {};

    // Setup chrome.storage mock
    (window.chrome as any).storage.local.set = async (data: any) => {
      Object.assign(testStorage, data);
    };

    (window.chrome as any).storage.local.get = async (keys: any) => {
      if (typeof keys === 'string') {
        return { [keys]: testStorage[keys] };
      }
      const result: Record<string, any> = {};
      keys.forEach((key: string) => {
        if (key in testStorage) {
          result[key] = testStorage[key];
        }
      });
      return result;
    };

    (window.chrome as any).storage.local.remove = async (keys: any) => {
      const keyArray = Array.isArray(keys) ? keys : [keys];
      keyArray.forEach((key: string) => {
        delete testStorage[key];
      });
    };

    // Reset store
    useSettingsStore.setState({
      settings: null,
      isLoading: false,
      error: null,
    });
  });

  afterEach(() => {
    testStorage = {};
  });

  describe('Initialization', () => {
    it('should load default settings on first access', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      expect(useSettingsStore.getState().settings).toBeDefined();
      expect(useSettingsStore.getState().settings?.activeProvider).toBe('claude');
      expect(useSettingsStore.getState().settings?.providers).toBeDefined();
    });

    it('should have all three providers in default settings', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      expect(useSettingsStore.getState().settings?.providers['claude']).toBeDefined();
      expect(useSettingsStore.getState().settings?.providers['openai']).toBeDefined();
      expect(useSettingsStore.getState().settings?.providers['ollama']).toBeDefined();
    });

    it('should initialize providers as disabled by default', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      expect(useSettingsStore.getState().settings?.providers['claude']?.enabled).toBe(false);
      expect(useSettingsStore.getState().settings?.providers['openai']?.enabled).toBe(false);
      expect(useSettingsStore.getState().settings?.providers['ollama']?.enabled).toBe(false);
    });
  });

  describe('Provider Configuration', () => {
    it('should update Claude provider config', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test-123',
        model: 'claude-3-opus-20250219',
        enabled: true,
      });

      // API keys are kept by the Moly app, never in the browser
      expect(useSettingsStore.getState().settings?.providers['claude']?.apiKey).toBeUndefined();
      expect(useSettingsStore.getState().settings?.providers['claude']?.model).toBe('claude-3-opus-20250219');
      expect(useSettingsStore.getState().settings?.providers['claude']?.enabled).toBe(true);
    });

    it('should update OpenAI provider config', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('openai', {
        apiKey: 'sk-openai-test-456',
        model: 'gpt-4-turbo',
        enabled: true,
      });

      expect(useSettingsStore.getState().settings?.providers['openai']?.apiKey).toBeUndefined();
      expect(useSettingsStore.getState().settings?.providers['openai']?.model).toBe('gpt-4-turbo');
      expect(useSettingsStore.getState().settings?.providers['openai']?.enabled).toBe(true);
    });

    it('should update Ollama provider config', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('ollama', {
        baseUrl: 'http://192.168.1.100:11434',
        model: 'neural-chat',
        enabled: true,
      });

      expect(useSettingsStore.getState().settings?.providers['ollama']?.baseUrl).toBe('http://192.168.1.100:11434');
      expect(useSettingsStore.getState().settings?.providers['ollama']?.model).toBe('neural-chat');
      expect(useSettingsStore.getState().settings?.providers['ollama']?.enabled).toBe(true);
    });

    it('should preserve other provider settings when updating one', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      // Configure Claude
      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      // Configure OpenAI
      await store.updateProvider('openai', {
        apiKey: 'sk-openai-test',
        enabled: true,
      });

      // Both should still be configured
      expect(useSettingsStore.getState().settings?.providers['claude']?.enabled).toBe(true);
      expect(useSettingsStore.getState().settings?.providers['openai']?.enabled).toBe(true);
    });

    it('should maintain provider type consistency', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      expect(useSettingsStore.getState().settings?.providers['claude']?.type).toBe('claude');
    });

    it('should mark settings as configured after update', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      expect(useSettingsStore.getState().settings?.isConfigured).toBe(false);

      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      expect(useSettingsStore.getState().settings?.isConfigured).toBe(true);
    });
  });

  describe('Active Provider Management', () => {
    it('should set active provider', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.setActiveProvider('openai');

      expect(useSettingsStore.getState().settings?.activeProvider).toBe('openai');
    });

    it('should switch between providers', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.setActiveProvider('claude');
      expect(useSettingsStore.getState().settings?.activeProvider).toBe('claude');

      await store.setActiveProvider('ollama');
      expect(useSettingsStore.getState().settings?.activeProvider).toBe('ollama');

      await store.setActiveProvider('openai');
      expect(useSettingsStore.getState().settings?.activeProvider).toBe('openai');
    });

    it('should persist active provider to storage', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.setActiveProvider('openai');

      // Verify it was saved to storage
      const stored = testStorage['settings'];
      expect(stored?.activeProvider).toBe('openai');
    });
  });

  describe('Preferences Management', () => {
    it('should set default communication context', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.setDefaultContext('formal');

      expect(useSettingsStore.getState().settings?.defaultContext).toBe('formal');
    });

    it('should set chat mode preference', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.setChatMode('socratic');

      expect(useSettingsStore.getState().settings?.chatMode).toBe('socratic');
    });

    it('should preserve provider config when changing preferences', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      // Configure a provider
      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      // Change preference
      await store.setDefaultContext('friendly');

      // Provider should still be configured
      expect(useSettingsStore.getState().settings?.providers['claude']?.enabled).toBe(true);
      expect(useSettingsStore.getState().settings?.defaultContext).toBe('friendly');
    });
  });

  describe('Persistence', () => {
    it('should persist settings to chrome storage, but never an API key (the Moly app keeps keys)', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      const stored = testStorage['settings'];
      expect(stored?.providers['claude']?.enabled).toBe(true);
      expect(stored?.providers['claude']?.apiKey).toBeUndefined();
      expect(JSON.stringify(testStorage)).not.toContain('sk-ant-test');
    });

    it('should load settings from chrome storage', async () => {
      // Manually set storage
      testStorage['settings'] = {
        activeProvider: 'openai',
        providers: {
          openai: {
            type: 'openai',
            apiKey: 'sk-test-123',
            model: 'gpt-4',
            enabled: true,
          },
        },
      };

      const store = useSettingsStore.getState();
      await store.loadSettings();

      expect(useSettingsStore.getState().settings?.activeProvider).toBe('openai');
      // a key saved by an older version is erased when the settings are loaded
      expect(useSettingsStore.getState().settings?.providers['openai']?.apiKey).toBeUndefined();
      expect(JSON.stringify(testStorage)).not.toContain('sk-test-123');
    });

    it('should track last checked timestamp', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      const before = Date.now();
      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });
      const after = Date.now();

      expect(useSettingsStore.getState().settings?.lastChecked).toBeGreaterThanOrEqual(before);
      expect(useSettingsStore.getState().settings?.lastChecked).toBeLessThanOrEqual(after);
    });
  });

  describe('Clear Settings', () => {
    it('should clear all settings', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      // Configure providers
      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      // Clear
      await store.clearAllSettings();

      expect(useSettingsStore.getState().settings?.isConfigured).toBe(false);
      expect(testStorage['settings']).toBeUndefined();
    });
  });

  describe('Error Handling', () => {
    it('should handle storage errors gracefully', async () => {
      const store = useSettingsStore.getState();

      // Simulate storage error by breaking the mock
      const originalSet = (window.chrome as any).storage.local.set;
      (window.chrome as any).storage.local.set = async () => {
        throw new Error('Storage quota exceeded');
      };

      await store.loadSettings();

      // Restore original
      (window.chrome as any).storage.local.set = originalSet;

      // Should not crash
      expect(useSettingsStore.getState().isLoading).toBe(false);
    });

    it('should clear error after successful operation', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      // First, an error is set
      useSettingsStore.setState({ error: 'Test error' });
      expect(useSettingsStore.getState().error).toBe('Test error');

      // Perform successful operation
      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      // Error should be cleared
      expect(useSettingsStore.getState().error).toBeNull();
    });
  });

  describe('State Consistency', () => {
    it('should maintain consistent state across operations', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      // Perform multiple operations
      await store.updateProvider('claude', { apiKey: 'key1', enabled: true });
      await store.updateProvider('openai', { apiKey: 'key2', enabled: true });
      await store.setActiveProvider('openai');
      await store.setDefaultContext('formal');

      const state = store.getSettings();

      expect(state?.providers['claude']?.enabled).toBe(true);
      expect(state?.providers['openai']?.enabled).toBe(true);
      expect(state?.activeProvider).toBe('openai');
      expect(state?.defaultContext).toBe('formal');
    });

    it('should never have conflicting provider configs', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('claude', {
        type: 'claude',
        apiKey: 'key1',
        enabled: true,
      });

      const config = useSettingsStore.getState().settings?.providers['claude'];
      expect(config?.type).toBe('claude');
      expect(config?.type).not.toBe('openai');
      expect(config?.type).not.toBe('ollama');
    });
  });
});
