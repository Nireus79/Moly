import { describe, it, expect, afterEach, vi } from 'vitest';

const win = window as unknown as { molyPhone?: unknown };

afterEach(() => { delete win.molyPhone; vi.resetModules(); });

describe('the engine port', () => {
  it('is the desktop port outside the phone app', async () => {
    vi.resetModules();
    const { getBackendManager } = await import('../backendManager');
    expect(getBackendManager().getBackendUrl()).toBe('http://127.0.0.1:11436');
  });

  it('is the phone install\'s own port in the phone app', async () => {
    win.molyPhone = { readClipboard: async () => '', takeSharedText: async () => '', enginePort: 23456 };
    vi.resetModules();
    const { getBackendManager } = await import('../backendManager');
    const { BACKEND_CONFIG } = await import('../backendConfig');
    expect(getBackendManager().getBackendUrl()).toBe('http://127.0.0.1:23456');
    expect(BACKEND_CONFIG.baseUrl).toBe('http://localhost:23456');
  });
});
