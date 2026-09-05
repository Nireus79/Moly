/**
 * Service Worker for Moly Extension
 */

import { getProviderManager } from '@/api/providerManager';
import type { ExtensionSettings } from '@/stores/settingsStore';

console.log('[Moly] Background service worker loaded');
console.log('[Moly] Make sure Go backend is running: moly-go/moly runs on 127.0.0.1:11436');

chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    console.log('[Moly] Extension installed');
  } else if (details.reason === 'update') {
    console.log('[Moly] Extension updated');
  }
});

// Handle extension icon click - inject sidebar on current page
chrome.action.onClicked.addListener(async () => {
  console.log('[Moly] Icon clicked - injecting sidebar');

  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id) return;

  chrome.tabs.sendMessage(tab.id, { type: 'TOGGLE_MOLY_SIDEBAR' }).catch(() => {
    console.log('[Moly] Content script not available on this page');
  });
});

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('[Moly] Background received message:', request.action || request.type);

  if (request.type === 'GENERATE_SUGGESTIONS') {
    generateSuggestions(request.data)
      .then((result) => {
        sendResponse({
          success: true,
          suggestions: result.suggestions,
          provider: result.provider,
        });
      })
      .catch((error) => {
        sendResponse({
          success: false,
          error: error instanceof Error ? error.message : 'Unknown error',
        });
      });
    return true;
  }
  return false;
});

interface SuggestionsResult {
  suggestions: string[];
  provider: string;
}

async function generateSuggestions(data: any): Promise<SuggestionsResult> {
  const settings = await getSettings();
  if (!settings) {
    throw new Error('No settings found. Please configure a provider in Settings.');
  }

  const manager = getProviderManager();
  const activeProviderType = settings.activeProvider;

  // Try active provider first
  try {
    const providerConfig = settings.providers[activeProviderType];
    if (providerConfig?.enabled) {
      const configured = await manager.configureProvider({
        type: activeProviderType,
        apiKey: providerConfig.apiKey,
        baseUrl: providerConfig.baseUrl,
        model: providerConfig.model,
      });

      if (configured) {
        const provider = manager.getActiveProvider();
        if (provider) {
          console.log(`[Moly] Using ${activeProviderType} provider`);
          const suggestions = await provider.generateSuggestions(
            data.userMessage || '',
            data.context || 'Unknown',
            data.communicationContext || 'friendly',
          );
          return {
            suggestions: suggestions.map((s) => s.text),
            provider: `${activeProviderType.charAt(0).toUpperCase() + activeProviderType.slice(1)}${activeProviderType === 'ollama' ? ' (Local)' : ' (Cloud)'}`,
          };
        }
      }
    }
  } catch (activeError) {
    console.warn(`[Moly] ${activeProviderType} failed, trying fallback providers:`, activeError);

    // If active provider fails (e.g., Ollama CORS issue), try fallback providers
    const fallbackProviders: Array<'claude' | 'openai'> = ['claude', 'openai'];

    for (const fallbackType of fallbackProviders) {
      try {
        const fallbackConfig = settings.providers[fallbackType];
        if (fallbackConfig?.enabled && fallbackConfig.apiKey) {
          console.log(`[Moly] Falling back to ${fallbackType}`);

          const configured = await manager.configureProvider({
            type: fallbackType,
            apiKey: fallbackConfig.apiKey,
            baseUrl: fallbackConfig.baseUrl,
            model: fallbackConfig.model,
          });

          if (configured) {
            const provider = manager.getActiveProvider();
            if (provider) {
              const suggestions = await provider.generateSuggestions(
                data.userMessage || '',
                data.context || 'Unknown',
                data.communicationContext || 'friendly',
              );
              console.log(`[Moly] Successfully used fallback ${fallbackType}`);
              return {
                suggestions: suggestions.map((s) => s.text),
                provider: `${fallbackType.charAt(0).toUpperCase() + fallbackType.slice(1)} (Cloud - Fallback)`,
              };
            }
          }
        }
      } catch (fallbackError) {
        console.warn(`[Moly] ${fallbackType} fallback also failed:`, fallbackError);
      }
    }

    // All providers failed
    throw new Error(`All providers failed. ${activeProviderType} error: ${activeError}`);
  }

  throw new Error('No enabled provider available.');
}

async function getSettings(): Promise<ExtensionSettings | null> {
  return new Promise((resolve) => {
    chrome.storage.local.get('settings', (result) => {
      resolve(result.settings || null);
    });
  });
}

console.log('[Moly] Background service worker ready');
