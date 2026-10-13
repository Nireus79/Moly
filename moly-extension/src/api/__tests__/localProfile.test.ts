import { describe, it, expect, beforeEach, vi } from 'vitest';

vi.mock('../backendManager', () => ({ getBackendManager: () => ({ getBackendUrl: () => 'http://127.0.0.1:11436' }) }));
import { startLocalSession } from '../localProfile';

describe('startLocalSession', () => {
  beforeEach(() => {
    (window as any).fetch = vi.fn();
  });

  it('turns the backend answer into a local session', async () => {
    (window.fetch as any).mockResolvedValue(new Response(JSON.stringify({ userId: 'user_local', token: 'abc', expiresIn: 2592000, local: true }), { status: 200 }));
    const before = Date.now();
    const s = await startLocalSession();
    expect(s).toMatchObject({ sessionId: 'abc', userId: 'user_local', local: true });
    expect(s!.expiresAt - before).toBeGreaterThan(29 * 86400 * 1000);
    const [url, init] = (window.fetch as any).mock.calls[0];
    expect(url).toBe('http://127.0.0.1:11436/api/auth/local');
    expect(init.method).toBe('POST');
  });

  it('gives null when the backend refuses, has no such route (development backend) or cannot be reached', async () => {
    (window.fetch as any).mockResolvedValueOnce(new Response('{}', { status: 404 }));
    expect(await startLocalSession()).toBeNull();
    (window.fetch as any).mockResolvedValueOnce(new Response(JSON.stringify({ userId: 'u' }), { status: 200 }));
    expect(await startLocalSession()).toBeNull();
    (window.fetch as any).mockRejectedValueOnce(new TypeError('Failed to fetch'));
    expect(await startLocalSession()).toBeNull();
  });
});
