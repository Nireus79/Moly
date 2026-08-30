import { create } from 'zustand';
import type { LLMConfig } from '@/types';

interface ConfigStore {
  llmConfig: LLMConfig | null;
  isInitialized: boolean;

  setLLMConfig: (config: LLMConfig) => void;
  getLLMConfig: () => LLMConfig | null;
  setInitialized: (initialized: boolean) => void;
  clearConfig: () => void;
}

export const useConfigStore = create<ConfigStore>((set, get) => ({
  llmConfig: null,
  isInitialized: false,

  setLLMConfig: (config) => {
    set({ llmConfig: config, isInitialized: true });
    chrome.storage.local.set({ llmConfig: config });
  },

  getLLMConfig: () => get().llmConfig,

  setInitialized: (initialized) => set({ isInitialized: initialized }),

  clearConfig: () => {
    set({ llmConfig: null, isInitialized: false });
    chrome.storage.local.remove('llmConfig');
  },
}));

// Load config from storage on app start
export const initializeConfig = async () => {
  const result = await chrome.storage.local.get('llmConfig');
  if (result.llmConfig) {
    useConfigStore.setState({ llmConfig: result.llmConfig, isInitialized: true });
  }
};
