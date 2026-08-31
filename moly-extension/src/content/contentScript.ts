/**
 * Content Script for Moly Extension
 * Injected into all webpages to detect messages
 */

import { MessageDetector } from './messageDetector';
import { detectPlatform } from './platformDetector';
import type { DetectedMessage } from '@/types';

console.log('Moly content script loaded');

let messageDetector: MessageDetector | null = null;

/**
 * Handle detected messages
 */
function handleMessageDetected(message: DetectedMessage): void {
  console.log('Message detected:', message.sender, '-', message.text.substring(0, 50));

  // Send to background script (which will broadcast to sidebar and store)
  chrome.runtime.sendMessage(
    {
      type: 'NEW_MESSAGE_DETECTED',
      message,
    },
    () => {
      if (chrome.runtime.lastError) {
        console.debug('Could not send to background:', chrome.runtime.lastError);
      } else {
        console.debug('Message sent to background');
      }
    },
  );
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
 * Listen for messages from sidebar/background
 */
chrome.runtime.onMessage.addListener((request, _sender, sendResponse) => {
  if (request.type === 'GET_PLATFORM') {
    const platform = detectPlatform();
    sendResponse({ platform: platform.platform });
  } else if (request.type === 'STOP_DETECTION') {
    if (messageDetector) {
      messageDetector.stop();
    }
    sendResponse({ stopped: true });
  } else if (request.type === 'START_DETECTION') {
    if (!messageDetector) {
      initializeMessageDetection();
    } else {
      messageDetector.start();
    }
    sendResponse({ started: true });
  } else if (request.type === 'GET_DETECTION_STATUS') {
    sendResponse({
      isRunning: messageDetector !== null,
      processedCount: messageDetector?.getProcessedCount() || 0,
    });
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
