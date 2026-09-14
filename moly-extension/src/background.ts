/**
 * Background Service Worker for Moly Extension (Manifest V3 Compatible)
 * Minimalist design: service workers cannot use top-level ES6 imports
 * All initialization and logic is handled by UI components
 */

// Track if side panel is open per tab
const sidePanelOpen: { [tabId: number]: boolean } = {};

// Handle extension icon click - toggle sidePanel
chrome.action.onClicked.addListener((tab) => {
  if (!tab.id) return;

  const tabId = tab.id;
  const isOpen = sidePanelOpen[tabId];

  if (isOpen) {
    chrome.sidePanel.setOptions({ tabId, path: '' }).catch((error) => {
      console.warn('[Background] Failed to close sidePanel:', error);
    });
    sidePanelOpen[tabId] = false;
    console.info('[Background] sidePanel closed');
  } else {
    chrome.sidePanel.open({ tabId }).catch((error) => {
      console.error('[Background] Failed to open sidePanel:', error);
    });
    sidePanelOpen[tabId] = true;
    console.info('[Background] sidePanel opened');
  }
});

// Handle messages from sidebar/popup
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('[Background] Message:', request.type || request.action);

  // Handle close sidePanel request
  if (request.type === 'CLOSE_SIDEPANEL') {
    const tabId = sender.tab?.id;
    if (tabId) {
      chrome.sidePanel.setOptions({ tabId, path: '' }).then(() => {
        sidePanelOpen[tabId] = false;
        sendResponse({ success: true });
      }).catch((error) => {
        sendResponse({ success: false, error: error.message });
      });
      return true; // Respond asynchronously
    }
  }

  // Acknowledge other messages
  sendResponse({ received: true });
  return false;
});

// Register for lifecycle events
chrome.runtime.onInstalled.addListener(() => {
  console.info('[Background] Extension installed/updated');
});

chrome.runtime.onStartup.addListener(() => {
  console.info('[Background] Extension started');
});

console.info('[Background] Service worker loaded');
