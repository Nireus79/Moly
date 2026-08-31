/**
 * Settings Store
 * Manages extension settings including multi-provider LLM configuration
 */

import { create } from 'zustand';
import type { LLMProviderType, LLMProviderConfig } from '@/api/providers';

export interface ExtensionSettings {
  // LLM Provider Configuration
  activeProvider: LLMProviderType;
  providers: Record<LLMProviderType, LLMProviderConfig>;

  // UI Preferences
  defaultContext: 'formal' | 'friendly' | 'dating';
  chatMode: 'socratic' | 'direct';

  // Metadata
  isConfigured: boolean;
  lastChecked?: number;
}

interface SettingsStore {
  settings: ExtensionSettings | null;
  isLoading: boolean;
  error: string | null;

  loadSettings: () => Promise<void>;
  updateProvider: (type: LLMProviderType, config: Partial<LLMProviderConfig>) => Promise<void>;
  setActiveProvider: (type: LLMProviderType) => Promise<void>;
  setDefaultContext: (context: 'formal' | 'friendly' | 'dating') => Promise<void>;
  setChatMode: (mode: 'socratic' | 'direct') => Promise<void>;
  getSettings: () => ExtensionSettings | null;
  clearAllSettings: () => Promise<void>;
}

const DEFAULT_SETTINGS: ExtensionSettings = {
  activeProvider: 'claude',
  providers: {
    claude: {
      type: 'claude',
      model: 'claude-3-5-sonnet-20241022',
      enabled: false,
      apiKey: undefined,
    },
    openai: {
      type: 'openai',
      model: 'gpt-4-turbo',
      enabled: false,
      apiKey: undefined,
    },
    ollama: {
      type: 'ollama',
      model: 'mistral',
      baseUrl: 'http://localhost:11434',
      enabled: false,
    },
  },
  defaultContext: 'dating',
  chatMode: 'direct',
  isConfigured: false,
};

export const useSettingsStore = create<SettingsStore>((set, get) => ({
  settings: null,
  isLoading: false,
  error: null,

  loadSettings: async () => {
    set({ isLoading: true, error: null });
    try {
      const result = await chrome.storage.local.get('settings');
      const settings = result.settings || DEFAULT_SETTINGS;
      set({ settings, isLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to load settings';
      set({ error: message, isLoading: false });
    }
  },

  updateProvider: async (type: LLMProviderType, config: Partial<LLMProviderConfig>) => {
    try {
      const settings = get().settings || DEFAULT_SETTINGS;
      const updated: ExtensionSettings = {
        ...settings,
        providers: {
          ...settings.providers,
          [type]: {
            ...settings.providers[type],
            ...config,
            type, // Ensure type is always correct
          },
        },
        isConfigured: true,
        lastChecked: Date.now(),
      };
      await chrome.storage.local.set({ settings: updated });
      set({ settings: updated, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update provider';
      set({ error: message });
    }
  },

  setActiveProvider: async (type: LLMProviderType) => {
    try {
      const settings = get().settings || DEFAULT_SETTINGS;
      const updated: ExtensionSettings = {
        ...settings,
        activeProvider: type,
      };
      await chrome.storage.local.set({ settings: updated });
      set({ settings: updated, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to set active provider';
      set({ error: message });
    }
  },

  setDefaultContext: async (context: 'formal' | 'friendly' | 'dating') => {
    try {
      const settings = get().settings || DEFAULT_SETTINGS;
      const updated: ExtensionSettings = {
        ...settings,
        defaultContext: context,
      };
      await chrome.storage.local.set({ settings: updated });
      set({ settings: updated, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update context';
      set({ error: message });
    }
  },

  setChatMode: async (mode: 'socratic' | 'direct') => {
    try {
      const settings = get().settings || DEFAULT_SETTINGS;
      const updated: ExtensionSettings = {
        ...settings,
        chatMode: mode,
      };
      await chrome.storage.local.set({ settings: updated });
      set({ settings: updated, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update chat mode';
      set({ error: message });
    }
  },

  getSettings: () => get().settings,

  clearAllSettings: async () => {
    try {
      await chrome.storage.local.remove('settings');
      set({ settings: DEFAULT_SETTINGS, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to clear settings';
      set({ error: message });
    }
  },
}));

/**
 * Initialize settings on app start
 */
export const initializeSettings = async () => {
  await useSettingsStore.getState().loadSettings();
};
