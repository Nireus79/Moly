import { describe, it, expect, vi, afterEach } from 'vitest';
import React from 'react';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';

const fetchExport = vi.fn();
const importExport = vi.fn();
vi.mock('@/api/dataTransfer', () => ({ fetchExport: (...a: unknown[]) => fetchExport(...a), importExport: (...a: unknown[]) => importExport(...a) }));

import { PhoneDataSection } from '../PhoneDataSection';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement | null = null;
async function render(node: React.ReactElement) {
  host = document.createElement('div');
  document.body.appendChild(host);
  await act(async () => createRoot(host!).render(node));
  return host;
}
const click = async (el: Element | null) => { await act(async () => { (el as HTMLElement).click(); }); };
const button = (root: HTMLElement, label: string) => Array.from(root.querySelectorAll('button')).find((b) => b.textContent?.includes(label)) ?? null;

const win = window as unknown as { molyPhone?: unknown };
const bridge = (extra: Record<string, unknown>) => ({ readClipboard: async () => '', takeSharedText: async () => '', ...extra });

afterEach(() => { host?.remove(); host = null; delete win.molyPhone; fetchExport.mockReset(); importExport.mockReset(); });

describe('Your data (phone app)', () => {
  it('shows nothing outside the phone app', async () => {
    const el = await render(<PhoneDataSection />);
    expect(el.innerHTML).toBe('');
  });

  it('saves a copy through the phone\'s save picker', async () => {
    const saveFile = vi.fn(async (_name: string, _text: string) => true);
    win.molyPhone = bridge({ saveFile, pickFile: async () => null });
    fetchExport.mockResolvedValue({ ok: true, text: '{"format":"moly-export-1"}' });
    const el = await render(<PhoneDataSection />);
    await click(button(el, 'Save a copy'));
    expect(saveFile).toHaveBeenCalledTimes(1);
    expect(saveFile.mock.calls[0][0]).toMatch(/^moly-copy-\d{4}-\d{2}-\d{2}\.json$/);
    expect(saveFile.mock.calls[0][1]).toBe('{"format":"moly-export-1"}');
    expect(el.textContent).toContain('The copy was saved.');
  });

  it('says nothing was saved when the person cancels the picker', async () => {
    win.molyPhone = bridge({ saveFile: async () => false, pickFile: async () => null });
    fetchExport.mockResolvedValue({ ok: true, text: '{}' });
    const el = await render(<PhoneDataSection />);
    await click(button(el, 'Save a copy'));
    expect(el.textContent).toContain('Nothing was saved.');
  });

  it('reports an export that failed, without opening the picker', async () => {
    const saveFile = vi.fn(async () => true);
    win.molyPhone = bridge({ saveFile, pickFile: async () => null });
    fetchExport.mockResolvedValue({ ok: false, message: 'The Moly app is not reachable.' });
    const el = await render(<PhoneDataSection />);
    await click(button(el, 'Save a copy'));
    expect(saveFile).not.toHaveBeenCalled();
    expect(el.textContent).toContain('Could not save a copy: The Moly app is not reachable.');
  });

  it('restores the chosen file, and does nothing when none is chosen', async () => {
    const pickFile = vi.fn<[], Promise<string | null>>().mockResolvedValueOnce('{"format":"moly-export-1"}').mockResolvedValueOnce(null);
    win.molyPhone = bridge({ saveFile: async () => true, pickFile });
    importExport.mockResolvedValue({ ok: true, message: 'Restored 3 conversations.' });
    const el = await render(<PhoneDataSection />);
    await click(button(el, 'Restore from a copy'));
    expect(importExport).toHaveBeenCalledWith('{"format":"moly-export-1"}');
    expect(el.textContent).toContain('Restored 3 conversations.');
    await click(button(el, 'Restore from a copy'));
    expect(importExport).toHaveBeenCalledTimes(1);
    expect(el.textContent).toContain('Nothing was restored.');
  });

  it('words the restore result for a phone, not for a browser panel', async () => {
    win.molyPhone = bridge({ saveFile: async () => true, pickFile: async () => '{"format":"moly-export-1"}' });
    importExport.mockResolvedValue({ ok: true, message: 'Restored 11 items. Close and reopen the Moly panel to see them.' });
    const el = await render(<PhoneDataSection />);
    await click(button(el, 'Restore from a copy'));
    expect(el.textContent).toContain('Restored 11 items. Your conversations are under Previous conversations.');
    expect(el.textContent).not.toContain('panel');
  });

  it('tells why a restore failed', async () => {
    win.molyPhone = bridge({ saveFile: async () => true, pickFile: async () => '{}' });
    importExport.mockResolvedValue({ ok: false, message: 'This file is not a Moly export.' });
    const el = await render(<PhoneDataSection />);
    await click(button(el, 'Restore from a copy'));
    expect(el.textContent).toContain('Could not restore: This file is not a Moly export.');
  });
});
