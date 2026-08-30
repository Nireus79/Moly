/**
 * Message Detection System
 * Uses MutationObserver to detect new messages on any website
 */

import type { DetectedMessage } from '@/types';
import { detectPlatform } from './platformDetector';
import {
  extractText,
  extractSenderName,
  isIncomingMessage,
  extractTimestamp,
  isValidMessageElement,
  isNewMessage,
} from './messageExtractor';

interface DetectionConfig {
  debounceMs: number;
  checkIntervalMs: number;
  maxMessagesPerBatch: number;
}

const DEFAULT_CONFIG: DetectionConfig = {
  debounceMs: 200,
  checkIntervalMs: 1000,
  maxMessagesPerBatch: 5,
};

export class MessageDetector {
  private observer: MutationObserver | null = null;
  private processedMessages: Set<string> = new Set();
  private debounceTimer: NodeJS.Timeout | null = null;
  private config: DetectionConfig;
  private onMessageDetected: (message: DetectedMessage) => void;

  constructor(
    onMessageDetected: (message: DetectedMessage) => void,
    config: Partial<DetectionConfig> = {},
  ) {
    this.onMessageDetected = onMessageDetected;
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Start monitoring for new messages
   */
  public start(): void {
    if (this.observer) {
      console.warn('MessageDetector already started');
      return;
    }

    console.log('Starting message detection...');

    this.observer = new MutationObserver(() => {
      this.debouncedCheckForMessages();
    });

    // Observe entire document body for changes
    this.observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: false,
      characterData: false,
    });

    // Also check periodically in case mutation observer misses something
    setInterval(() => this.checkForMessages(), this.config.checkIntervalMs);
  }

  /**
   * Stop monitoring for messages
   */
  public stop(): void {
    if (this.observer) {
      this.observer.disconnect();
      this.observer = null;
    }
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
    }
    console.log('Message detection stopped');
  }

  /**
   * Debounced check for new messages
   */
  private debouncedCheckForMessages(): void {
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
    }
    this.debounceTimer = setTimeout(
      () => this.checkForMessages(),
      this.config.debounceMs,
    );
  }

  /**
   * Scan page for new messages
   */
  private checkForMessages(): void {
    try {
      // Look for common message container selectors
      const messageSelectors = [
        '[role="article"]', // Aria role for messages
        '.message',
        '.msg',
        '.chat-message',
        '.message-item',
        '[data-message-id]',
        '[data-msg-id]',
        '[data-qa*="message"]',
        '.bubble', // For messaging apps
      ];

      const detectedThisBatch: DetectedMessage[] = [];

      for (const selector of messageSelectors) {
        const elements = document.querySelectorAll(selector);

        for (const element of Array.from(elements)) {
          if (detectedThisBatch.length >= this.config.maxMessagesPerBatch) {
            break;
          }

          const message = this.extractMessageFromElement(element);
          if (message && !this.processedMessages.has(this.hashMessage(message))) {
            detectedThisBatch.push(message);
            this.processedMessages.add(this.hashMessage(message));

            // Keep processed messages set from growing too large
            if (this.processedMessages.size > 10000) {
              this.processedMessages.clear();
            }
          }
        }

        if (detectedThisBatch.length >= this.config.maxMessagesPerBatch) {
          break;
        }
      }

      // Process detected messages in reverse order (newest first)
      for (const message of detectedThisBatch.reverse()) {
        this.onMessageDetected(message);
      }
    } catch (error) {
      console.error('Error checking for messages:', error);
    }
  }

  /**
   * Extract message data from DOM element
   */
  private extractMessageFromElement(element: Element): DetectedMessage | null {
    try {
      // Validate element
      if (!isValidMessageElement(element)) {
        return null;
      }

      // Get text
      const text = extractText(element);
      if (!text) return null;

      // Get sender
      const sender = extractSenderName(element);

      // Get timestamp
      const timestamp = extractTimestamp(element);

      // Check if new and incoming
      if (!isNewMessage(timestamp)) {
        return null; // Only detect recently received messages
      }

      if (!isIncomingMessage(element)) {
        return null; // Only incoming messages
      }

      const platform = detectPlatform();

      const message: DetectedMessage = {
        sender,
        text,
        timestamp,
        platform: platform.platform,
        url: window.location.href,
        profileId: element.getAttribute('data-sender-id') || element.getAttribute('data-user-id') || undefined,
        context: platform.messageContext === 'dating'
          ? 'dating'
          : platform.messageContext === 'professional'
            ? 'formal'
            : 'friendly',
      };

      return message;
    } catch (error) {
      console.debug('Error extracting message from element:', error);
      return null;
    }
  }

  /**
   * Create hash of message for deduplication
   */
  private hashMessage(message: DetectedMessage): string {
    return `${message.sender}:${message.text}:${message.timestamp}`;
  }

  /**
   * Get currently processed message count
   */
  public getProcessedCount(): number {
    return this.processedMessages.size;
  }

  /**
   * Clear processing history (for testing)
   */
  public clearHistory(): void {
    this.processedMessages.clear();
  }
}
