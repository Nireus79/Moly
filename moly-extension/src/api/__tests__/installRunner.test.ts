import { describe, it, expect, vi, afterEach } from 'vitest';
import { downloadAndOpen, ensureDownloadsPermission, openDownload, type RunState } from '../installRunner';

const URL = 'https://github.com/Nireus79/Moly/releases/download/v0.1.2/moly-setup-0.1.2.exe';
const g = globalThis as any;
const savedChrome = g.chrome;
afterEach(() => {
  g.chrome = savedChrome;
});
const instant = async () => {};

function chromeWith(items: Array<{ state: string; bytesReceived: number; totalBytes: number }>, openImpl: () => any = async () => {}) {
  let i = 0;
  const downloads = {
    download: vi.fn(async () => 7),
    search: vi.fn(async () => [items[Math.min(i++, items.length - 1)]]),
    open: vi.fn(openImpl),
  };
  g.chrome = { downloads, permissions: { contains: vi.fn(async () => false), request: vi.fn(async () => true) } };
  return downloads;
}

describe('permission', () => {
  it('is asked for once, when it is not held', async () => {
    chromeWith([]);
    expect(await ensureDownloadsPermission()).toBe(true);
    expect(g.chrome.permissions.request).toHaveBeenCalledWith({ permissions: ['downloads', 'downloads.open'] });
  });
  it('is not asked for again when it is held, and a refusal is a no', async () => {
    chromeWith([]);
    g.chrome.permissions.contains = vi.fn(async () => true);
    expect(await ensureDownloadsPermission()).toBe(true);
    expect(g.chrome.permissions.request).not.toHaveBeenCalled();
    g.chrome.permissions.contains = vi.fn(async () => false);
    g.chrome.permissions.request = vi.fn(async () => false);
    expect(await ensureDownloadsPermission()).toBe(false);
  });
});

describe('download and open', () => {
  it('downloads, shows progress, then opens the file', async () => {
    const d = chromeWith([{ state: 'in_progress', bytesReceived: 50, totalBytes: 100 }, { state: 'complete', bytesReceived: 100, totalBytes: 100 }]);
    const seen: RunState[] = [];
    await downloadAndOpen(URL, 'moly-setup-0.1.2.exe', true, (s) => seen.push(s), instant);
    expect(d.download).toHaveBeenCalledWith({ url: URL, filename: 'moly-setup-0.1.2.exe', conflictAction: 'uniquify' });
    expect(seen.some((s) => s.phase === 'downloading' && s.percent === 50)).toBe(true);
    expect(seen[seen.length - 1]).toEqual({ phase: 'opened' });
    expect(d.open).toHaveBeenCalledWith(7);
  });

  it('when the browser will not open it without a click, the next click does', async () => {
    const d = chromeWith([{ state: 'complete', bytesReceived: 1, totalBytes: 1 }], async () => { throw new Error('needs a user gesture'); });
    const seen: RunState[] = [];
    await downloadAndOpen(URL, 'a.exe', true, (s) => seen.push(s), instant);
    expect(seen[seen.length - 1]).toEqual({ phase: 'ready', id: 7 });
    d.open.mockImplementation(async () => {});
    expect(await openDownload(7)).toBe(true);
  });

  it('an interrupted download is reported', async () => {
    chromeWith([{ state: 'interrupted', bytesReceived: 1, totalBytes: 10 }]);
    const seen: RunState[] = [];
    await downloadAndOpen(URL, 'a.exe', true, (s) => seen.push(s), instant);
    expect(seen[seen.length - 1].phase).toBe('failed');
  });

  it('without the permission, an ordinary download link is used', async () => {
    const d = chromeWith([]);
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    const seen: RunState[] = [];
    await downloadAndOpen(URL, 'a.exe', false, (s) => seen.push(s), instant);
    expect(d.download).not.toHaveBeenCalled();
    expect(click).toHaveBeenCalled();
    expect(seen[seen.length - 1]).toEqual({ phase: 'saved' });
    click.mockRestore();
  });

  it('never downloads a file from anywhere but the Moly releases', async () => {
    const d = chromeWith([]);
    const seen: RunState[] = [];
    await downloadAndOpen('https://evil.example/a.exe', 'a.exe', true, (s) => seen.push(s), instant);
    expect(d.download).not.toHaveBeenCalled();
    expect(seen[0].phase).toBe('failed');
  });
});
