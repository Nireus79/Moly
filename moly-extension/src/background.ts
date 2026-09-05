/**
 * Background Service Worker for Moly Extension
 * Handles backend initialization and background tasks
 */

import { backendManager } from './api/backendManager';

// Initialize backend when extension loads
chrome.runtime.onInstalled.addListener(async () => {
  console.info('[Background] Extension installed, initializing backend...');
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
  const status = await backendManager.initialize();
  
  if (!status.running) {
    console.warn('[Background] Backend not available:', status.error);
  }
});

// Handle messages from content script or popup
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
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
});

// Listen for alarms (periodic tasks)
chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name === 'backend_healthcheck') {
    const status = await backendManager.getStatus();
    console.debug('[Background] Backend health check:', status.running ? 'OK' : 'FAIL');
  }
});

// Set up periodic health checks (every 5 minutes)
chrome.alarms.create('backend_healthcheck', { periodInMinutes: 5 });

console.info('[Background] Service worker loaded');
