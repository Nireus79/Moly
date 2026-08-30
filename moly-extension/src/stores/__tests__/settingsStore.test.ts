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

      expect(store.settings).toBeDefined();
      expect(store.settings?.activeProvider).toBe('claude');
      expect(store.settings?.providers).toBeDefined();
    });

    it('should have all three providers in default settings', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      expect(store.settings?.providers['claude']).toBeDefined();
      expect(store.settings?.providers['openai']).toBeDefined();
      expect(store.settings?.providers['ollama']).toBeDefined();
    });

    it('should initialize providers as disabled by default', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      expect(store.settings?.providers['claude']?.enabled).toBe(false);
      expect(store.settings?.providers['openai']?.enabled).toBe(false);
      expect(store.settings?.providers['ollama']?.enabled).toBe(false);
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

      expect(store.settings?.providers['claude']?.apiKey).toBe('sk-ant-test-123');
      expect(store.settings?.providers['claude']?.model).toBe('claude-3-opus-20250219');
      expect(store.settings?.providers['claude']?.enabled).toBe(true);
    });

    it('should update OpenAI provider config', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('openai', {
        apiKey: 'sk-openai-test-456',
        model: 'gpt-4-turbo',
        enabled: true,
      });

      expect(store.settings?.providers['openai']?.apiKey).toBe('sk-openai-test-456');
      expect(store.settings?.providers['openai']?.model).toBe('gpt-4-turbo');
      expect(store.settings?.providers['openai']?.enabled).toBe(true);
    });

    it('should update Ollama provider config', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('ollama', {
        baseUrl: 'http://192.168.1.100:11434',
        model: 'neural-chat',
        enabled: true,
      });

      expect(store.settings?.providers['ollama']?.baseUrl).toBe('http://192.168.1.100:11434');
      expect(store.settings?.providers['ollama']?.model).toBe('neural-chat');
      expect(store.settings?.providers['ollama']?.enabled).toBe(true);
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
      expect(store.settings?.providers['claude']?.enabled).toBe(true);
      expect(store.settings?.providers['openai']?.enabled).toBe(true);
    });

    it('should maintain provider type consistency', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      expect(store.settings?.providers['claude']?.type).toBe('claude');
    });

    it('should mark settings as configured after update', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      expect(store.settings?.isConfigured).toBe(false);

      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      expect(store.settings?.isConfigured).toBe(true);
    });
  });

  describe('Active Provider Management', () => {
    it('should set active provider', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.setActiveProvider('openai');

      expect(store.settings?.activeProvider).toBe('openai');
    });

    it('should switch between providers', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.setActiveProvider('claude');
      expect(store.settings?.activeProvider).toBe('claude');

      await store.setActiveProvider('ollama');
      expect(store.settings?.activeProvider).toBe('ollama');

      await store.setActiveProvider('openai');
      expect(store.settings?.activeProvider).toBe('openai');
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

      expect(store.settings?.defaultContext).toBe('formal');
    });

    it('should set chat mode preference', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.setChatMode('socratic');

      expect(store.settings?.chatMode).toBe('socratic');
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
      expect(store.settings?.providers['claude']?.enabled).toBe(true);
      expect(store.settings?.defaultContext).toBe('friendly');
    });
  });

  describe('Persistence', () => {
    it('should persist settings to chrome storage', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      const stored = testStorage['settings'];
      expect(stored?.providers['claude']?.apiKey).toBe('sk-ant-test');
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

      expect(store.settings?.activeProvider).toBe('openai');
      expect(store.settings?.providers['openai']?.apiKey).toBe('sk-test-123');
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

      expect(store.settings?.lastChecked).toBeGreaterThanOrEqual(before);
      expect(store.settings?.lastChecked).toBeLessThanOrEqual(after);
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

      expect(store.settings?.isConfigured).toBe(false);
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
      expect(store.isLoading).toBe(false);
    });

    it('should clear error after successful operation', async () => {
      const store = useSettingsStore.getState();
      await store.loadSettings();

      // First, cause an error
      // Clear error state
      expect(store.error).toBe('Test error');

      // Perform successful operation
      await store.updateProvider('claude', {
        apiKey: 'sk-ant-test',
        enabled: true,
      });

      // Error should be cleared
      expect(store.error).toBeNull();
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

      const config = store.settings?.providers['claude'];
      expect(config?.type).toBe('claude');
      expect(config?.type).not.toBe('openai');
      expect(config?.type).not.toBe('ollama');
    });
  });
});
