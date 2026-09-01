/**
 * Service Worker for Moly Extension
 */

import { getProviderManager } from '@/api/providerManager';
import type { ExtensionSettings } from '@/stores/settingsStore';

console.log('[Moly] Background service worker loaded');

chrome.runtime.onInstalled.addListener(() => {
  console.log('[Moly] Extension installed');
});

// Handle extension icon click - toggle injected sidebar
chrome.action.onClicked.addListener(async () => {
  console.log('[Moly] Icon clicked');

  try {
    const tabs = await chrome.tabs.query({ active: true, currentWindow: true });

    if (tabs[0]?.id) {
      // Send message to content script to toggle sidebar
      chrome.tabs.sendMessage(tabs[0].id, { type: 'TOGGLE_MOLY_SIDEBAR' }).catch((error) => {
        console.log('[Moly] Could not reach content script:', error.message);
      });
    }
  } catch (error) {
    console.error('[Moly] Error:', error);
  }
});

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('[Moly] Background received message:', request.type);

  if (request.type === 'GENERATE_SUGGESTIONS') {
    generateSuggestions(request.data)
      .then((suggestions) => {
        sendResponse({ success: true, suggestions });
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

async function generateSuggestions(data: any): Promise<string[]> {
  try {
    const settings = await getSettings();
    if (!settings) {
      throw new Error('No settings found. Please configure a provider in Settings.');
    }

    const activeProviderType = settings.activeProvider;
    const providerConfig = settings.providers[activeProviderType];

    if (!providerConfig?.enabled) {
      throw new Error(`Provider ${activeProviderType} is not enabled.`);
    }

    const manager = getProviderManager();
    const configured = await manager.configureProvider({
      type: activeProviderType,
      apiKey: providerConfig.apiKey,
      baseUrl: providerConfig.baseUrl,
      model: providerConfig.model,
    });

    if (!configured) {
      throw new Error(`Failed to configure ${activeProviderType} provider.`);
    }

    const provider = manager.getActiveProvider();
    if (!provider) {
      throw new Error('No active provider available.');
    }

    const suggestions = await provider.generateSuggestions(
      data.userMessage || '',
      data.context || 'Unknown',
      data.communicationContext || 'friendly',
    );

    return suggestions;
  } catch (error) {
    console.error('[Moly] Error generating suggestions:', error);
    throw error;
  }
}

async function getSettings(): Promise<ExtensionSettings | null> {
  return new Promise((resolve) => {
    chrome.storage.local.get('settings', (result) => {
      resolve(result.settings || null);
    });
  });
}

console.log('[Moly] Background service worker ready');
