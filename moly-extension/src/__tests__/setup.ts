/**
 * Test Setup
 * Global mocks and configuration for Vitest
 */

import { vi, beforeEach } from 'vitest';

// Mock chrome API
const chromeMock = {
  storage: {
    local: {
      set: vi.fn(async () => {}),
      get: vi.fn(async () => ({})),
      remove: vi.fn(async () => {}),
    },
  },
  runtime: {
    sendMessage: vi.fn(async () => ({})),
    onMessage: {
      addListener: vi.fn(),
      removeListener: vi.fn(),
    },
  },
  sidePanel: {
    open: vi.fn(),
  },
};

// Set up global chrome mock
Object.defineProperty(window, 'chrome', {
  value: chromeMock,
  writable: true,
});

// Mock fetch for API tests
global.fetch = vi.fn();

// Reset mocks before each test
beforeEach(() => {
  vi.clearAllMocks();
});
