/**
 * Test Setup
 * Global mocks and configuration for Vitest
 */

import { vi, beforeEach } from 'vitest';

// Simple in-memory storage for testing
const mockStorage: Record<string, any> = {};

// Mock chrome API
const chromeMock = {
  storage: {
    local: {
      set: vi.fn(async (data: Record<string, any>) => {
        Object.assign(mockStorage, data);
      }),
      get: vi.fn(async (keys?: string | string[] | null) => {
        if (!keys) return { ...mockStorage };
        if (typeof keys === 'string') return { [keys]: mockStorage[keys] };
        return keys.reduce((acc: Record<string, any>, key: string) => {
          acc[key] = mockStorage[key];
          return acc;
        }, {});
      }),
      remove: vi.fn(async (keys: string | string[]) => {
        if (typeof keys === 'string') {
          delete mockStorage[keys];
        } else {
          keys.forEach(key => delete mockStorage[key]);
        }
      }),
      clear: vi.fn(async () => {
        Object.keys(mockStorage).forEach(key => delete mockStorage[key]);
      }),
    },
  },
  runtime: {
    sendMessage: vi.fn(async () => ({})),
    onMessage: {
      addListener: vi.fn(),
      removeListener: vi.fn(),
    },
    openOptionsPage: vi.fn(async () => {}),
  },
  sidePanel: {
    open: vi.fn(async () => {}),
  },
  tabs: {
    query: vi.fn(async () => [{ id: 1 }]),
  },
};

// Set up global chrome mock
Object.defineProperty(window, 'chrome', {
  value: chromeMock,
  writable: true,
});

// Mock fetch for API tests
global.fetch = vi.fn();

// Reset mocks and storage before each test
beforeEach(() => {
  vi.clearAllMocks();
  Object.keys(mockStorage).forEach(key => delete mockStorage[key]);
});
