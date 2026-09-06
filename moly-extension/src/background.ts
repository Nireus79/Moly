/**
 * Background Service Worker for Moly Extension
 * Handles backend initialization, message routing, and suggestion generation
 */

import { getBackendManager } from './api/backendManager';
import { getProviderManager } from './api/providerManager';
import type { ExtensionSettings } from './stores/settingsStore';

// Handle extension icon click - toggle sidePanel
chrome.action.onClicked.addListener((tab) => {
  if (!tab.id) return;

  const tabId = tab.id;
  const isOpen = sidePanelOpen[tabId];

  if (isOpen) {
    // Close by setting empty path
    chrome.sidePanel.setOptions({ tabId, path: '' }).catch((error) => {
      console.warn('[Background] Failed to close sidePanel:', error);
    });
    sidePanelOpen[tabId] = false;
    console.info('[Background] sidePanel closed');
  } else {
    // Open sidePanel
    console.info('[Background] Icon clicked - opening sidePanel...');
    chrome.sidePanel.open({ tabId }).catch((error) => {
      console.error('[Background] Failed to open sidePanel:', error);
    });
    sidePanelOpen[tabId] = true;

    // Initialize backend in background
    const backendManager = getBackendManager();
    backendManager.initialize().then((status) => {
      if (!status.running) {
        console.warn('[Background] Backend not available - Moly will work with LLM providers only');
      } else {
        console.info('[Background] Backend ready');
      }
    });
  }
});

// Initialize backend when extension loads
chrome.runtime.onInstalled.addListener(async () => {
  console.info('[Background] Extension installed, initializing backend...');
  const backendManager = getBackendManager();
  const status = await backendManager.initialize();
  
  if (status.running) {
    console.info('[Background] Backend ready at', status.url);
  } else {
    console.error('[Background] Backend failed to start:', status.error);
  }
});

// Listen for extension startup
chrome.runtime.onStartup.addListener(async () => {
  console.info('[Background] Extension started, checking backend...');
  const backendManager = getBackendManager();
  const status = await backendManager.initialize();

  if (!status.running) {
    console.warn('[Background] Backend not available:', status.error);
  }
});

// Store setup wizard state for UI to access
let setupWizardState: { extensionId: string; setupCommand: string } | null = null;

// Track sidePanel open state for toggle
let sidePanelOpen: { [tabId: number]: boolean } = {};

// Handle messages from content script, popup, or sidebar
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('[Background] Message received:', request.type || request.action, 'from', sender.url);
  const backendManager = getBackendManager();

  // Handle suggestion generation from sidebar
  if (request.type === 'GENERATE_SUGGESTIONS') {
    console.log('[Background] Processing GENERATE_SUGGESTIONS request...');
    generateSuggestions(request.data)
      .then((result) => {
        console.log('[Background] Suggestions generated successfully:', result.suggestions.length);
        sendResponse({
          success: true,
          suggestions: result.suggestions,
          provider: result.provider,
        });
      })
      .catch((error) => {
        console.error('[Background] Error generating suggestions:', error);
        sendResponse({
          success: false,
          error: error instanceof Error ? error.message : 'Unknown error',
        });
      });
    return true; // Will respond asynchronously
  }

  if (request.action === 'check_backend') {
    backendManager.getStatus().then(status => {
      sendResponse(status);
    });
    return true; // Will respond asynchronously
  }

  if (request.action === 'get_backend_url') {
    sendResponse({ url: backendManager.getBackendUrl() });
    return false;
  }

  if (request.type === 'SHOW_SETUP_WIZARD') {
    setupWizardState = {
      extensionId: request.extensionId,
      setupCommand: request.setupCommand,
    };
    // Notify all listeners that setup wizard should be shown
    chrome.runtime.sendMessage({
      type: 'SETUP_WIZARD_STATE_CHANGED',
      state: setupWizardState,
    }).catch(() => {
      // Listeners might not be ready yet
    });
    sendResponse({ success: true });
    return true;
  }

  if (request.action === 'get_setup_wizard_state') {
    sendResponse(setupWizardState);
    return false;
  }

  if (request.action === 'dismiss_setup_wizard') {
    setupWizardState = null;
    sendResponse({ success: true });
    return false;
  }

  if (request.type === 'CLOSE_SIDEPANEL') {
    // Close sidePanel from sidebar close button using same method as icon click
    const tabId = sender.tab?.id;
    if (tabId) {
      console.log('[Background] Closing sidePanel from sidebar button, tabId:', tabId);
      // Use setOptions to close the panel (same as icon click handler)
      chrome.sidePanel.setOptions({ tabId, path: '' }).then(() => {
        sidePanelOpen[tabId] = false;
        console.log('[Background] sidePanel closed via sidebar close button');
        sendResponse({ success: true });
      }).catch((error) => {
        console.error('[Background] Failed to close sidePanel:', error);
        sendResponse({ success: false, error: error.message });
      });
    } else {
      sendResponse({ success: false, error: 'No tab ID' });
    }
    return true; // Respond asynchronously
  }

  return false;
});

// Listen for alarms (periodic tasks)
chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name === 'backend_healthcheck') {
    const backendManager = getBackendManager();
    const status = await backendManager.getStatus();
    console.debug('[Background] Backend health check:', status.running ? 'OK' : 'FAIL');
  }
});

// Set up periodic health checks (every 5 minutes)
chrome.alarms.create('backend_healthcheck', { periodInMinutes: 5 });

// Helper function to get settings from storage
async function getSettings(): Promise<ExtensionSettings | null> {
  return new Promise((resolve) => {
    chrome.storage.local.get('settings', (result) => {
      resolve(result.settings || null);
    });
  });
}

// Helper interface for suggestions result
interface SuggestionsResult {
  suggestions: string[];
  provider: string;
}

// Generate suggestions from LLM providers
async function generateSuggestions(data: any): Promise<SuggestionsResult> {
  const settings = await getSettings();
  if (!settings) {
    throw new Error('No settings found. Please configure a provider in Settings.');
  }

  if (!data.userMessage || !data.userMessage.trim()) {
    throw new Error('Please enter a message first.');
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
          console.log(`[Background] Using ${activeProviderType} provider`);
          const suggestions = await provider.generateSuggestions(
            data.userMessage || '',
            data.context || 'Unknown',
            data.communicationContext || 'friendly',
          );

          if (!suggestions || !Array.isArray(suggestions)) {
            throw new Error(`${activeProviderType} returned invalid suggestions format`);
          }

          if (suggestions.length === 0) {
            throw new Error(`${activeProviderType} generated no suggestions`);
          }

          return {
            suggestions: suggestions.map((s: any) => {
              if (!s || typeof s !== 'object') {
                console.warn('[Background] Invalid suggestion object:', s);
                return 'No suggestion available';
              }
              return s.text || 'No suggestion available';
            }),
            provider: `${activeProviderType.charAt(0).toUpperCase() + activeProviderType.slice(1)}${activeProviderType === 'ollama' ? ' (Local)' : ' (Cloud)'}`,
          };
        }
      }
    }
  } catch (activeError) {
    console.warn(`[Background] ${activeProviderType} failed, trying fallback providers:`, activeError);

    // If active provider fails (e.g., Ollama CORS issue), try fallback providers
    const fallbackProviders: Array<'claude' | 'openai'> = ['claude', 'openai'];

    for (const fallbackType of fallbackProviders) {
      try {
        const fallbackConfig = settings.providers[fallbackType];
        if (fallbackConfig?.enabled && fallbackConfig.apiKey) {
          console.log(`[Background] Falling back to ${fallbackType}`);

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

              if (!suggestions || !Array.isArray(suggestions)) {
                throw new Error(`${fallbackType} returned invalid suggestions format`);
              }

              if (suggestions.length === 0) {
                throw new Error(`${fallbackType} generated no suggestions`);
              }

              console.log(`[Background] Successfully used fallback ${fallbackType}`);
              return {
                suggestions: suggestions.map((s: any) => {
                  if (!s || typeof s !== 'object') {
                    console.warn('[Background] Invalid suggestion object:', s);
                    return 'No suggestion available';
                  }
                  return s.text || 'No suggestion available';
                }),
                provider: `${fallbackType.charAt(0).toUpperCase() + fallbackType.slice(1)} (Cloud - Fallback)`,
              };
            }
          }
        }
      } catch (fallbackError) {
        console.warn(`[Background] ${fallbackType} fallback also failed:`, fallbackError);
      }
    }

    // All providers failed
    throw new Error(`All providers failed. ${activeProviderType} error: ${activeError}`);
  }

  throw new Error('No enabled provider available.');
}

console.info('[Background] Service worker loaded');
