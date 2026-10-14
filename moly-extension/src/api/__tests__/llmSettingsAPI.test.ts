import { describe, it, expect, beforeEach, vi } from 'vitest';

vi.mock('../backendManager', () => ({ getBackendManager: () => ({ getBackendUrl: () => 'http://127.0.0.1:11436' }) }));
const ensure = vi.fn();
vi.mock('../backendAuth', () => ({ ensureBackendConnection: () => ensure() }));

import { getLLMStatus, saveLLMSettings, testLLMSettings, unlockLLM, NOT_SIGNED_IN } from '../llmSettingsAPI';
import { useAuthStore } from '@/stores/authStore';

const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status });

describe('llmSettingsAPI', () => {
  beforeEach(() => {
    ensure.mockReset();
    (window as any).fetch = vi.fn();
    useAuthStore.getState().setSession(null);
  });

  // The settings belong to the signed-in user: every request says whose.
  it('sends the session of the signed-in user with every settings call', async () => {
    useAuthStore.getState().setSession({ sessionId: 'session-of-user-a', userId: 'user_a', expiresAt: Date.now() + 3600_000 });
    (window.fetch as any).mockResolvedValue(json(200, { state: 'ready' }));
    await getLLMStatus();
    await saveLLMSettings({ provider: 'claude', apiKey: 'k' });
    await unlockLLM('p');
    const calls = (window.fetch as any).mock.calls;
    expect(calls).toHaveLength(3);
    for (const [, init] of calls) expect(init.headers['Authorization']).toBe('Bearer session-of-user-a');
  });

  // A missing user session is not a connection problem (it was reported as "the Moly app is not reachable").
  it('says the user is not signed in when the app answers 401 Invalid token, without trying to start the app', async () => {
    (window.fetch as any).mockResolvedValue(json(401, { error: 'Invalid token' }));
    const r = await saveLLMSettings({ provider: 'claude', apiKey: 'k' });
    expect(r).toEqual({ ok: false, error: NOT_SIGNED_IN });
    expect(ensure).not.toHaveBeenCalled();
    expect((window.fetch as any).mock.calls.length).toBe(1);
  });

  it('sends no session when nobody is signed in (the backend then refuses)', async () => {
    (window.fetch as any).mockResolvedValue(json(401, { error: 'Invalid token' }));
    await getLLMStatus();
    const [, init] = (window.fetch as any).mock.calls[0];
    expect(init.headers?.['Authorization']).toBeUndefined();
  });

  it('returns the data on success and sends JSON bodies to the right path', async () => {
    (window.fetch as any).mockResolvedValueOnce(json(200, { state: 'ready' }));
    const r = await saveLLMSettings({ provider: 'claude', apiKey: 'k', passphrase: 'p' });
    expect(r).toEqual({ ok: true, data: { state: 'ready' } });
    const [url, init] = (window.fetch as any).mock.calls[0];
    expect(url).toBe('http://127.0.0.1:11436/api/settings/llm');
    expect(init.method).toBe('PUT');
    expect(JSON.parse(init.body)).toMatchObject({ provider: 'claude', apiKey: 'k' });
  });

  it('starts the Moly app and retries once after a network error', async () => {
    (window.fetch as any).mockRejectedValueOnce(new TypeError('Failed to fetch')).mockResolvedValueOnce(json(200, { state: 'locked' }));
    ensure.mockResolvedValue({ ok: true, url: 'http://127.0.0.1:11436' });
    const r = await getLLMStatus();
    expect(r.ok).toBe(true);
    expect(ensure).toHaveBeenCalledTimes(1);
    expect((window.fetch as any).mock.calls.length).toBe(2);
  });

  it('retries once after a 401 (no token yet), not forever', async () => {
    (window.fetch as any).mockResolvedValue(json(401, { error: 'missing or wrong token' }));
    ensure.mockResolvedValue({ ok: true });
    const r = await getLLMStatus();
    expect(r.ok).toBe(false);
    expect(r.error).toBe('missing or wrong token');
    expect((window.fetch as any).mock.calls.length).toBe(2);
  });

  it('says the app is not reachable when it cannot be started', async () => {
    (window.fetch as any).mockRejectedValue(new TypeError('Failed to fetch'));
    ensure.mockResolvedValue({ ok: false, error: 'Specified native messaging host not found.' });
    const r = await getLLMStatus();
    expect(r).toEqual({ ok: false, error: 'The Moly app is not reachable.' });
    expect((window.fetch as any).mock.calls.length).toBe(1);
  });

  it('passes the backend error text on (wrong passphrase)', async () => {
    (window.fetch as any).mockResolvedValueOnce(json(403, { error: 'wrong passphrase' }));
    const r = await unlockLLM('nope');
    expect(r).toEqual({ ok: false, error: 'wrong passphrase' });
  });

  it('the test call reports ok false with the reason', async () => {
    (window.fetch as any).mockResolvedValueOnce(json(200, { ok: false, error: 'invalid x-api-key' }));
    expect(await testLLMSettings({ provider: 'claude', apiKey: 'k' })).toEqual({ ok: false, error: 'invalid x-api-key' });
  });
});
