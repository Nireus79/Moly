/**
 * Background Service Worker for Moly Extension
 * Handles backend initialization and background tasks
 */

import { getBackendManager } from './api/backendManager';

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

// Handle messages from content script or popup
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  const backendManager = getBackendManager();

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

console.info('[Background] Service worker loaded');
