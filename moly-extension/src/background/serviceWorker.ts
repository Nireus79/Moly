/**
 * Service Worker for Moly Extension
 */

import { getProviderManager } from '@/api/providerManager';
import type { ExtensionSettings } from '@/stores/settingsStore';
import { cleanupMoly, getInstalledModels } from '@/api/installerLauncher';

console.log('[Moly] Background service worker loaded');
console.log('[Moly] CORS proxy auto-starts via systemd service - should be running on localhost:11435');

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

  chrome.tabs.sendMessage(tab.id, { action: 'toggle-sidebar' }).catch(() => {
    console.log('[Moly] Content script not available on this page');
  });
});

// Note: Sidebar injection is handled by content.js
// Content script automatically injects sidebar from desktop app on all pages

function launchDesktopApp(sendResponse: Function) {
  console.log('[Moly] Attempting to connect to native host: com.moly.native_host');
  try {
    const port = chrome.runtime.connectNative('com.moly.native_host');
    console.log('[Moly] Native port connected');

    port.onMessage.addListener((response) => {
      console.log('[Moly] Native host response:', response);
      port.disconnect();
      sendResponse(response);
    });

    port.onDisconnect.addListener(() => {
      console.log('[Moly] Native host disconnected');
      if (chrome.runtime.lastError) {
        console.error('[Moly] Native host error:', chrome.runtime.lastError);
        console.error('[Moly] Error details:', JSON.stringify(chrome.runtime.lastError));
        sendResponse({
          success: false,
          error: chrome.runtime.lastError?.message || 'Failed to connect to native host',
        });
      }
    });

    console.log('[Moly] Sending launch-app message');
    port.postMessage({ action: 'launch-app' });
  } catch (error) {
    console.error('[Moly] Exception launching app:', error);
    sendResponse({
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    });
  }
}

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('[Moly] Background received message:', request.action || request.type);

  if (request.action === 'launch-app') {
    launchDesktopApp(sendResponse);
    return true;
  }

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
