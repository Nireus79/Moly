import { describe, it, expect, beforeEach, vi } from 'vitest';
import { getBackendManager } from '../backendManager';

// The app answers /api/status with 200 when ready, 423 while it waits for the person to unlock or set up the database key,
// and 503 while it starts. Only the first two mean "the app is there".
function answer(status: number, body: object = { version: '2.1' }) {
  global.fetch = vi.fn().mockResolvedValue({ ok: status >= 200 && status < 300, status, statusText: '', json: async () => body }) as any;
}

describe('BackendManager.checkHealth', () => {
  beforeEach(async () => {
    await chrome.storage.local.set({ detectedBackendUrl: 'http://127.0.0.1:11436' });
    (getBackendManager() as any).detectedBackendUrl = 'http://127.0.0.1:11436';
  });

  it('counts a ready app as running', async () => {
    answer(200);
    expect((await getBackendManager().checkHealth()).running).toBe(true);
  });

  it('counts an app that waits for the database key (423) as running, so the unlock screen can be shown', async () => {
    answer(423, { status: 'locked', version: '2.1' });
    const s = await getBackendManager().checkHealth();
    expect(s.running).toBe(true);
    expect(s.version).toBe('2.1');
  });

  it('does not count an app that is still starting (503)', async () => {
    answer(503, { status: 'starting' });
    expect((await getBackendManager().checkHealth()).running).toBe(false);
  });
});
