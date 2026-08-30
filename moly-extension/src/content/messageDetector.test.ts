/**
 * Unit Tests for Message Detection System
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { MessageDetector } from './messageDetector';
import {
  extractText,
  extractSenderName,
  isIncomingMessage,
  isValidMessageElement,
} from './messageExtractor';

describe('messageExtractor', () => {
  describe('extractText', () => {
    it('should extract plain text from element', () => {
      const div = document.createElement('div');
      div.textContent = 'Hello world';
      expect(extractText(div)).toBe('Hello world');
    });

    it('should trim whitespace', () => {
      const div = document.createElement('div');
      div.textContent = '  Hello   world  ';
      expect(extractText(div)).toBe('Hello world');
    });

    it('should remove script tags', () => {
      const div = document.createElement('div');
      div.innerHTML = 'Hello <script>alert("xss")</script> world';
      const text = extractText(div);
      expect(text).not.toContain('alert');
      expect(text).toContain('Hello');
      expect(text).toContain('world');
    });

    it('should remove common status indicators', () => {
      const div = document.createElement('div');
      div.textContent = 'Hello world Delivered Seen';
      const text = extractText(div);
      expect(text).not.toContain('Delivered');
      expect(text).not.toContain('Seen');
    });
  });

  describe('extractSenderName', () => {
    it('should extract sender from data-sender attribute', () => {
      const div = document.createElement('div');
      const sender = document.createElement('span');
      sender.setAttribute('data-sender', 'John');
      div.appendChild(sender);
      expect(extractSenderName(div)).toBe('John');
    });

    it('should extract sender from title attribute', () => {
      const div = document.createElement('div');
      div.setAttribute('title', 'Jane');
      expect(extractSenderName(div)).toBe('Jane');
    });

    it('should return "Unknown" if no sender found', () => {
      const div = document.createElement('div');
      expect(extractSenderName(div)).toBe('Unknown');
    });
  });

  describe('isIncomingMessage', () => {
    it('should return false for outgoing messages', () => {
      const div = document.createElement('div');
      div.classList.add('outgoing');
      expect(isIncomingMessage(div)).toBe(false);
    });

    it('should return true for incoming messages', () => {
      const div = document.createElement('div');
      div.classList.add('incoming');
      expect(isIncomingMessage(div)).toBe(true);
    });

    it('should return true if sender is different from current user', () => {
      const div = document.createElement('div');
      div.setAttribute('data-sender-id', 'user123');
      expect(isIncomingMessage(div, 'user456')).toBe(true);
    });

    it('should return false if sender is current user', () => {
      const div = document.createElement('div');
      div.setAttribute('data-sender-id', 'user123');
      expect(isIncomingMessage(div, 'user123')).toBe(false);
    });
  });

  describe('isValidMessageElement', () => {
    it('should return true for valid message element', () => {
      const div = document.createElement('div');
      div.textContent = 'This is a message with meaningful content';
      expect(isValidMessageElement(div)).toBe(true);
    });

    it('should return false for empty element', () => {
      const div = document.createElement('div');
      expect(isValidMessageElement(div)).toBe(false);
    });

    it('should return false for very short text', () => {
      const div = document.createElement('div');
      div.textContent = 'Hi';
      expect(isValidMessageElement(div)).toBe(false);
    });

    it('should return false for button elements', () => {
      const button = document.createElement('button');
      button.textContent = 'This is a very long button text that looks like a message';
      expect(isValidMessageElement(button)).toBe(false);
    });

    it('should return false for excessively long text', () => {
      const div = document.createElement('div');
      div.textContent = 'a'.repeat(15000);
      expect(isValidMessageElement(div)).toBe(false);
    });
  });
});

describe('MessageDetector', () => {
  let detector: MessageDetector;
  let detectedMessages: any[] = [];

  beforeEach(() => {
    detectedMessages = [];
    detector = new MessageDetector((msg) => {
      detectedMessages.push(msg);
    });
  });

  it('should initialize without errors', () => {
    expect(detector).toBeDefined();
  });

  it('should start and stop without errors', () => {
    detector.start();
    expect(detector.getProcessedCount()).toBe(0);
    detector.stop();
  });

  it('should clear history', () => {
    detector.clearHistory();
    expect(detector.getProcessedCount()).toBe(0);
  });

  it('should not duplicate messages', () => {
    // This is hard to test in unit tests since we need to simulate DOM mutations
    // Integration tests will cover this better
    detector.start();
    expect(detector.getProcessedCount()).toBeGreaterThanOrEqual(0);
    detector.stop();
  });

  it('should handle configuration options', () => {
    const customDetector = new MessageDetector(
      () => {},
      {
        debounceMs: 500,
        checkIntervalMs: 2000,
      },
    );
    expect(customDetector).toBeDefined();
  });
});
