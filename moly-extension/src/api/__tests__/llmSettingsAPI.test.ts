import { describe, it, expect, beforeEach, vi } from 'vitest';

vi.mock('../backendManager', () => ({ getBackendManager: () => ({ getBackendUrl: () => 'http://127.0.0.1:11436' }) }));
const ensure = vi.fn();
vi.mock('../backendAuth', () => ({ ensureBackendConnection: () => ensure() }));

import { getLLMStatus, saveLLMSettings, testLLMSettings, unlockLLM } from '../llmSettingsAPI';

const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status });

describe('llmSettingsAPI', () => {
  beforeEach(() => {
    ensure.mockReset();
    (window as any).fetch = vi.fn();
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
