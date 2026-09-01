/**
 * Service Worker for Moly Extension
 */

import { getProviderManager } from '@/api/providerManager';
import type { ExtensionSettings } from '@/stores/settingsStore';

console.log('[Moly] Background service worker loaded');

chrome.runtime.onInstalled.addListener(() => {
  console.log('[Moly] Extension installed');
});

// Test: Show notification when icon clicked (to verify background script is running)
chrome.action.onClicked.addListener(async () => {
  console.log('[Moly] Icon clicked - attempting to open sidebar');

  try {
    const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
    console.log('[Moly] Active tabs found:', tabs.length);

    if (tabs[0]?.id) {
      console.log('[Moly] Opening sidePanel on tab', tabs[0].id);
      await chrome.sidePanel.open({ tabId: tabs[0].id });
      console.log('[Moly] sidePanel opened successfully');
    } else {
      console.log('[Moly] No active tab found');
    }
  } catch (error) {
    console.error('[Moly] Error opening sidePanel:', error);
    // Fallback: open as window
    try {
      const sidebarUrl = chrome.runtime.getURL('sidebar/sidebar.html');
      await chrome.windows.create({
        url: sidebarUrl,
        type: 'popup',
        width: 450,
        height: 800,
      });
      console.log('[Moly] Opened as window instead');
    } catch (winError) {
      console.error('[Moly] Failed to open window:', winError);
    }
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
