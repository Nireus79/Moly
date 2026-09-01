/**
 * Service Worker for Moly Extension
 */

import { getProviderManager } from '@/api/providerManager';
import type { ExtensionSettings } from '@/stores/settingsStore';

console.log('[Moly] Background service worker loaded');

chrome.runtime.onInstalled.addListener(() => {
  console.log('[Moly] Extension installed');
});

// Handle extension icon click - inject sidebar directly
chrome.action.onClicked.addListener(async () => {
  console.log('[Moly] Icon clicked');

  try {
    const tabs = await chrome.tabs.query({ active: true, currentWindow: true });

    if (!tabs[0]?.id) {
      console.error('[Moly] No active tab');
      return;
    }

    const tabId = tabs[0].id;
    const sidebarUrl = chrome.runtime.getURL('sidebar/sidebar.html');

    console.log('[Moly] Sidebar URL:', sidebarUrl);

    // Try to inject sidebar into the page
    try {
      await chrome.scripting.executeScript({
        target: { tabId },
        function: injectSidebar,
        args: [sidebarUrl],
        world: 'MAIN',
      });
      console.log('[Moly] Sidebar injected');
    } catch (injectError) {
      // Try to show a message in the page when sidebar injection fails
      console.log('[Moly] Cannot inject on restricted page, trying message:', injectError);
      try {
        await chrome.scripting.executeScript({
          target: { tabId },
          function: showRestrictedPageMessage,
          world: 'ISOLATED',
        });
      } catch (messageError) {
        console.log('[Moly] Could not show message on restricted page:', messageError);
      }
    }
  } catch (error) {
    console.error('[Moly] Error:', error);
  }
});

// Show message on restricted pages
function showRestrictedPageMessage() {
  const messageDiv = document.createElement('div');
  messageDiv.id = 'moly-restricted-message';
  messageDiv.style.cssText = `
    position: fixed !important;
    top: 20px !important;
    right: 20px !important;
    padding: 16px !important;
    background: #fef3c7 !important;
    border: 1px solid #f59e0b !important;
    border-radius: 8px !important;
    color: #92400e !important;
    font-family: system-ui, -apple-system, sans-serif !important;
    font-size: 14px !important;
    font-weight: 500 !important;
    z-index: 2147483647 !important;
    max-width: 300px !important;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1) !important;
  `;
  messageDiv.textContent = 'Moly works on real websites only. This page is restricted.';
  document.documentElement.appendChild(messageDiv);

  // Auto-hide after 5 seconds
  setTimeout(() => {
    messageDiv.remove();
  }, 5000);
}

// Function to inject sidebar (runs in page context)
function injectSidebar(sidebarUrl: string) {
  console.log('[inject] Creating sidebar with URL:', sidebarUrl);

  // Check if already injected
  const existing = document.getElementById('moly-sidebar-container');
  if (existing) {
    const isHidden = existing.style.display === 'none';
    existing.style.display = isHidden ? 'block' : 'none';
    console.log('[inject] Toggled sidebar to:', existing.style.display);
    return;
  }

  // Create container
  const container = document.createElement('div');
  container.id = 'moly-sidebar-container';
  container.style.cssText = `
    position: fixed !important;
    right: 0 !important;
    top: 0 !important;
    width: 400px !important;
    height: 100vh !important;
    background: white !important;
    border-left: 1px solid #e5e7eb !important;
    box-shadow: -2px 0 8px rgba(0,0,0,0.1) !important;
    z-index: 2147483647 !important;
    margin: 0 !important;
    padding: 0 !important;
    box-sizing: border-box !important;
  `;

  // Create iframe
  const iframe = document.createElement('iframe');
  iframe.src = sidebarUrl;
  iframe.style.cssText = `
    width: 100% !important;
    height: 100% !important;
    border: none !important;
    margin: 0 !important;
    padding: 0 !important;
  `;

  iframe.onload = () => {
    console.log('[inject] iframe loaded successfully');
  };

  iframe.onerror = (error) => {
    console.error('[inject] iframe failed to load:', error);
  };

  container.appendChild(iframe);
  document.documentElement.appendChild(container);

  console.log('[inject] Sidebar container created and appended');
}

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
