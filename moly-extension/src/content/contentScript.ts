/**
 * Content Script for Moly Extension
 * Injected into all webpages to detect messages and manage sidebar
 */

import { MessageDetector } from './messageDetector';
import { detectPlatform } from './platformDetector';
import type { DetectedMessage } from '@/types';

console.log('Moly content script loaded');

let messageDetector: MessageDetector | null = null;
let sidebarFrame: HTMLIFrameElement | null = null;
let sidebarContainer: HTMLDivElement | null = null;
let isPinned = false;

/**
 * Handle detected messages
 */
function handleMessageDetected(message: DetectedMessage): void {
  console.log('Message detected:', message.sender, '-', message.text.substring(0, 50));

  chrome.runtime.sendMessage(
    {
      type: 'NEW_MESSAGE_DETECTED',
      message,
    },
    () => {
      if (chrome.runtime.lastError) {
        console.debug('Could not send to background:', chrome.runtime.lastError);
      }
    },
  );

  // Update sidebar if it exists
  if (sidebarFrame?.contentWindow) {
    try {
      sidebarFrame.contentWindow.postMessage(
        { type: 'UPDATE_DETECTED_MESSAGE', message },
        '*'
      );
    } catch (e) {
      console.debug('Could not post to sidebar:', e);
    }
  }
}

/**
 * Initialize message detection
 */
function initializeMessageDetection(): void {
  const platform = detectPlatform();
  console.log('Initializing message detection for:', platform.platform);

  messageDetector = new MessageDetector(handleMessageDetected);
  messageDetector.start();
}

/**
 * Inject sidebar into page
 */
function injectSidebar(): void {
  try {
    if (sidebarContainer) return;

    // Create container
    sidebarContainer = document.createElement('div');
    sidebarContainer.id = 'moly-sidebar-container';
    sidebarContainer.style.cssText = `
      position: fixed;
      right: 0;
      top: 0;
      width: 20%;
      height: 100vh;
      z-index: 2147483647;
      background: white;
      box-shadow: -2px 0 8px rgba(0, 0, 0, 0.1);
      transition: transform 0.3s ease;
      transform: translateX(0);
      display: none;
    `;

    // Create iframe
    const sidebarUrl = chrome.runtime.getURL('injected-sidebar.html');
    sidebarFrame = document.createElement('iframe');
    sidebarFrame.src = sidebarUrl;
    sidebarFrame.style.cssText = `
      width: 100%;
      height: 100%;
      border: none;
      background: white;
    `;

    sidebarContainer.appendChild(sidebarFrame);
    document.body.appendChild(sidebarContainer);

    // Handle sidebar messages
    window.addEventListener('message', (event) => {
      if (event.source !== sidebarFrame?.contentWindow) return;

      if (event.data.type === 'MOLY_CLOSE_SIDEBAR') {
        if (!isPinned) {
          hideSidebar();
        }
      } else if (event.data.type === 'MOLY_OPEN_CHAT') {
        openFullChat();
      }
    });

    // Handle mouse events for retract
    if (sidebarContainer) {
      sidebarContainer.addEventListener('mouseleave', () => {
        if (!isPinned) {
          retractSidebar();
        }
      });

      sidebarContainer.addEventListener('mouseenter', () => {
        expandSidebar();
      });
    }

    console.log('Moly sidebar injected');
  } catch (error) {
    console.error('Error injecting sidebar:', error);
  }
}

/**
 * Show sidebar
 */
function showSidebar(): void {
  if (!sidebarContainer) {
    injectSidebar();
  }
  if (sidebarContainer) {
    sidebarContainer.style.display = 'block';
    expandSidebar();
  }
}

/**
 * Hide sidebar
 */
function hideSidebar(): void {
  if (sidebarContainer) {
    sidebarContainer.style.display = 'none';
  }
}

/**
 * Expand sidebar (full width)
 */
function expandSidebar(): void {
  if (sidebarContainer) {
    sidebarContainer.style.transform = 'translateX(0)';
  }
}

/**
 * Retract sidebar (slide out)
 */
function retractSidebar(): void {
  if (sidebarContainer) {
    sidebarContainer.style.transform = 'translateX(100%)';
  }
}

/**
 * Open full chat window
 */
function openFullChat(): void {
  chrome.runtime.sendMessage({ type: 'OPEN_SIDEPANEL' }).catch(() => {
    console.debug('Could not send OPEN_SIDEPANEL message');
  });
}

/**
 * Listen for messages from background
 */
chrome.runtime.onMessage.addListener((request, _sender, sendResponse) => {
  try {
    if (request.type === 'GET_PLATFORM') {
      const platform = detectPlatform();
      sendResponse({ platform: platform.platform });
    } else if (request.type === 'GET_DETECTION_STATUS') {
      sendResponse({
        isRunning: messageDetector !== null,
        processedCount: messageDetector?.getProcessedCount() || 0,
      });
    } else if (request.type === 'SHOW_MOLY_SIDEBAR') {
      console.log('Showing Moly sidebar');
      showSidebar();
      sendResponse({ success: true });
    }
  } catch (error) {
    console.error('Error handling message:', error);
    sendResponse({ error: error instanceof Error ? error.message : 'Unknown error' });
  }
});

// Listen for storage changes (pin state)
chrome.storage.onChanged.addListener((changes) => {
  if (changes.molyPinned) {
    isPinned = changes.molyPinned.newValue || false;
    console.log('Pin state changed:', isPinned);
  }
});

// Start detection when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initializeMessageDetection);
} else {
  initializeMessageDetection();
}

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
  if (messageDetector) {
    messageDetector.stop();
  }
});
