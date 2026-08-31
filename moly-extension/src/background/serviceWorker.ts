/**
 * Service Worker for Moly Extension
 * Handles background tasks and message routing
 */

import { getProviderManager } from '@/api/providerManager';
import type { ExtensionSettings } from '@/stores/settingsStore';

chrome.runtime.onInstalled.addListener(() => {
  console.log('Moly extension installed');
});

// Handle extension icon click (show injected sidebar)
chrome.action.onClicked.addListener(async () => {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tabs[0]?.id) {
    chrome.tabs.sendMessage(tabs[0].id, { type: 'SHOW_MOLY_SIDEBAR' }).catch(() => {
      console.log('Content script not loaded on this tab');
    });
  }
});

// Listen for messages from content script and sidebar
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('Background received message:', request.type);

  if (request.type === 'OPEN_SIDEPANEL') {
    openSidePanel()
      .then(() => sendResponse({ success: true }))
      .catch((error) => sendResponse({ success: false, error: error instanceof Error ? error.message : 'Unknown error' }));
    return true;
  } else if (request.type === 'NEW_MESSAGE_DETECTED') {
    handleMessageDetected(request.message);
    sendResponse({ status: 'ok' });
  } else if (request.type === 'GENERATE_SUGGESTIONS') {
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
    return true; // Keep channel open for async response
  } else if (request.type === 'SAVE_CONTACT') {
    saveContact(request.contact)
      .then((success) => {
        sendResponse({ success });
      })
      .catch((error) => {
        sendResponse({ success: false, error: error instanceof Error ? error.message : 'Unknown error' });
      });
    return true;
  }
  return false;
});

function handleMessageDetected(message: any): void {
  console.log('Processing detected message from:', message.sender);

  // Store in local storage for sidebar to pick up via storage listener
  chrome.storage.local.set({
    lastDetectedMessage: message,
    lastDetectedAt: Date.now(),
  });

  // Also notify all listeners via runtime messaging (for real-time updates)
  chrome.runtime.sendMessage(
    {
      type: 'DETECTED_MESSAGE_BROADCAST',
      message,
    },
    () => {
      if (chrome.runtime.lastError) {
        console.debug('Broadcast error (expected if no listeners):', chrome.runtime.lastError.message);
      }
    },
  );
}

async function generateSuggestions(data: any): Promise<any[]> {
  try {
    const settings = await getSettings();
    if (!settings) {
      throw new Error('No settings found. Please configure a provider in Settings.');
    }

    const activeProviderType = settings.activeProvider;
    const providerConfig = settings.providers[activeProviderType];

    if (!providerConfig?.enabled) {
      throw new Error(`Provider ${activeProviderType} is not enabled. Please configure it in Settings.`);
    }

    // Configure the active provider
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

    console.log(`Generating suggestions with ${activeProviderType} for context:`, data.context);

    const suggestions = await provider.generateSuggestions(
      data.userMessage || '',
      data.context || 'Unknown recipient',
      data.communicationContext || 'friendly',
    );

    console.log(`Generated ${suggestions.length} suggestions`);
    return suggestions;
  } catch (error) {
    console.error('Error generating suggestions:', error);
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

async function openSidePanel(): Promise<void> {
  try {
    // Try using sidePanel API (Chrome 114+)
    if (chrome.sidePanel) {
      const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
      const tabId = tabs[0]?.id;
      if (tabId) {
        await chrome.sidePanel.open({ tabId });
        console.log('Sidebar opened using sidePanel API');
        return;
      }
    }
  } catch (error) {
    console.debug('SidePanel API not available or failed:', error);
  }

  // Fallback: open sidebar in new tab (Brave, older Chrome, sidePanel failed)
  const sidebarUrl = chrome.runtime.getURL('sidebar/sidebar.html');
  await chrome.tabs.create({ url: sidebarUrl });
  console.log('Sidebar opened in new tab (fallback)');
}

async function saveContact(contact: any): Promise<boolean> {
  return new Promise((resolve, reject) => {
    try {
      chrome.storage.local.get('contacts', (result) => {
        const contacts = (result.contacts || []) as any[];
        const updated = [...contacts, { ...contact, id: Date.now().toString() }];
        chrome.storage.local.set({ contacts: updated }, () => {
          resolve(true);
        });
      });
    } catch (error) {
      reject(error);
    }
  });
}
