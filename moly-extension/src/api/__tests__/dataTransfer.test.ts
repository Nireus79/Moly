import { describe, it, expect, beforeEach, vi } from 'vitest';
import { fetchExport, importExport } from '../dataTransfer';

const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status });
const signIn = () => localStorage.setItem('moly_session', JSON.stringify({ sessionId: 'tok', userId: 'u', expiresAt: Date.now() + 3600_000 }));

describe('dataTransfer', () => {
  beforeEach(() => {
    localStorage.clear();
    (window as any).fetch = vi.fn();
  });

  it('refuses without a session and without a request', async () => {
    expect((await fetchExport()).ok).toBe(false);
    expect((await importExport('{"format":"moly-export-1","tables":{}}')).ok).toBe(false);
    expect(window.fetch).not.toHaveBeenCalled();
  });

  it('returns the export text with the session', async () => {
    signIn();
    (window.fetch as any).mockResolvedValue(new Response('{"format":"moly-export-1"}', { status: 200 }));
    const r = await fetchExport();
    expect(r).toEqual({ ok: true, text: '{"format":"moly-export-1"}' });
    expect((window.fetch as any).mock.calls[0][1].headers.Authorization).toBe('Bearer tok');
  });

  it('does not send a file that is not an export', async () => {
    signIn();
    expect((await importExport('not json')).message).toContain('not readable');
    expect((await importExport('{"format":"other"}')).message).toContain('not a Moly export');
    expect(window.fetch).not.toHaveBeenCalled();
  });

  it('tells how many items were restored', async () => {
    signIn();
    (window.fetch as any).mockResolvedValue(json(200, { success: true, rows: 42 }));
    const r = await importExport('{"format":"moly-export-1","tables":{}}');
    expect(r.ok).toBe(true);
    expect(r.message).toContain('42');
  });

  it('explains a profile that already has data', async () => {
    signIn();
    (window.fetch as any).mockResolvedValue(json(409, { error: 'x' }));
    const r = await importExport('{"format":"moly-export-1","tables":{}}');
    expect(r.ok).toBe(false);
    expect(r.message).toContain('empty profile');
  });

  it('passes on the app\'s reason for refusing a file', async () => {
    signIn();
    (window.fetch as any).mockResolvedValue(json(400, { error: 'this is not a usable Moly export: table "x" is not a table of a user\'s data' }));
    const r = await importExport('{"format":"moly-export-1","tables":{}}');
    expect(r.ok).toBe(false);
    expect(r.message).toContain('not a usable Moly export');
  });
});
