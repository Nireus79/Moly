import { describe, it, expect, vi, afterEach } from 'vitest';
import React from 'react';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';
import { ReportSection } from '../ReportSection';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement | null = null;
async function render(node: React.ReactElement) {
  host = document.createElement('div');
  document.body.appendChild(host);
  await act(async () => createRoot(host!).render(node));
  return host;
}
const click = async (el: Element | null) => { await act(async () => { (el as HTMLElement).click(); }); };
const button = (root: HTMLElement, label: string) => Array.from(root.querySelectorAll('button')).find((b) => b.textContent === label) ?? null;

const win = window as unknown as { molyPhone?: unknown };
const bridge = (extra: Record<string, unknown>) => ({ readClipboard: async () => '', takeSharedText: async () => '', ...extra });

afterEach(() => { host?.remove(); host = null; delete win.molyPhone; });

describe('Report a problem (phone app)', () => {
  it('shows nothing outside the phone app', async () => {
    const el = await render(<ReportSection />);
    expect(el.innerHTML).toBe('');
  });

  it('shows the report before it can be shared, and shares exactly that text', async () => {
    const shareReport = vi.fn(async () => {});
    win.molyPhone = bridge({ buildReport: async () => 'Moly 1.0\nengine started 09:12', shareReport });
    const el = await render(<ReportSection />);
    expect(button(el, 'Send report')).toBeNull();
    await click(button(el, 'Prepare report'));
    const box = el.querySelector('textarea[aria-label="The report"]') as HTMLTextAreaElement;
    expect(box.value).toContain('engine started 09:12');
    expect(box.readOnly).toBe(true);
    await click(button(el, 'Send report'));
    expect(shareReport).toHaveBeenCalledWith('Moly 1.0\nengine started 09:12');
    await click(button(el, 'Close'));
    expect(el.querySelector('textarea')).toBeNull();
  });

  it('adds the person\'s own note, and only what they typed, in front of the report', async () => {
    const shareReport = vi.fn(async () => {});
    win.molyPhone = bridge({ buildReport: async () => 'Moly 1.0', shareReport });
    const el = await render(<ReportSection />);
    await click(button(el, 'Prepare report'));
    const note = el.querySelector('textarea[aria-label="Your note"]') as HTMLTextAreaElement;
    expect(note.readOnly).toBe(false);
    expect(note.value).toBe('');
    await act(async () => {
      const set = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')!.set!;
      set.call(note, 'The reply said something unkind.');
      note.dispatchEvent(new Event('input', { bubbles: true }));
    });
    await click(button(el, 'Send report'));
    expect(shareReport).toHaveBeenCalledWith('The reply said something unkind.\n\n---\nMoly 1.0');
  });

  it('says so when the report cannot be prepared', async () => {
    win.molyPhone = bridge({ buildReport: async () => { throw new Error('no'); }, shareReport: async () => {} });
    const el = await render(<ReportSection />);
    await click(button(el, 'Prepare report'));
    expect(el.textContent).toContain('Could not prepare the report.');
    expect(el.querySelector('textarea')).toBeNull();
  });
});
