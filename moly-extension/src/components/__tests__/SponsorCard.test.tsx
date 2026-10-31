import { describe, it, expect, afterEach, beforeEach } from 'vitest';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';
import { SponsorCard } from '../SponsorCard';
import { HIDDEN_KEY, isHidden, isSafeUrl, labelFor, pickEntry, setHidden } from '@/api/sponsorCard';
import { CARD_ENTRIES, type CardEntry } from '@/config/cardEntries';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement | null = null;
async function render(entries?: CardEntry[]) {
  host = document.createElement('div');
  document.body.appendChild(host);
  await act(async () => createRoot(host!).render(<SponsorCard entries={entries} />));
  await act(async () => { await Promise.resolve(); });
  return host;
}

beforeEach(async () => {
  await chrome.storage.local.clear?.();
  sessionStorage.clear();
});
afterEach(() => {
  host?.remove();
  host = null;
});

const entry = (over: Partial<CardEntry> = {}): CardEntry => ({ id: 'x', kind: 'moly', title: 't', text: 'x', active: true, ...over });

describe('card logic', () => {
  it('labels every card "Sponsored", Moly\'s own included', () => {
    expect(labelFor('moly')).toBe('Sponsored');
    expect(labelFor('sponsor')).toBe('Sponsored');
  });

  it('shows only https links', () => {
    expect(isSafeUrl('https://example.org/x')).toBe(true);
    for (const bad of ['http://example.org', 'javascript:alert(1)', 'data:text/html,x', '', undefined, 'not a url']) expect(isSafeUrl(bad as any)).toBe(false);
  });

  it('chooses by the day only, among active entries', () => {
    const a = entry({ id: 'a' });
    const b = entry({ id: 'b' });
    const off = entry({ id: 'off', active: false });
    const day = (n: number) => new Date(n * 86_400_000);
    expect(pickEntry([off], day(5))).toBeNull();
    expect(pickEntry([a, off, b], day(10))!.id).toBe('a');
    expect(pickEntry([a, off, b], day(11))!.id).toBe('b');
    expect(pickEntry([a, b], day(10))).toBe(pickEntry([a, b], new Date(10 * 86_400_000 + 3_600_000))); // same day, same entry
  });

  it('hides only for the open panel: the choice is in sessionStorage, never in storage that outlives the panel', () => {
    expect(isHidden()).toBe(false);
    setHidden(true);
    expect(isHidden()).toBe(true);
    expect(sessionStorage.getItem(HIDDEN_KEY)).toBe('1');
    expect(localStorage.getItem(HIDDEN_KEY)).toBeNull();
    setHidden(false);
    expect(isHidden()).toBe(false);
  });

  it('the shipped entries are all labelled, https only and free of scripts', () => {
    expect(CARD_ENTRIES.some((e) => e.active)).toBe(true);
    for (const e of CARD_ENTRIES) {
      for (const l of e.links ?? []) expect(isSafeUrl(l.url)).toBe(true);
      expect(`${e.title} ${e.text}`).not.toMatch(/<script|javascript:/i);
    }
  });
});

describe('SponsorCard', () => {
  it('shows the Hermes card: marks, the message, a Sponsor button and safe links, labelled and hideable', async () => {
    const el = await render();
    expect(el.querySelector('.moly-card-label')?.textContent).toBe('Sponsored');
    expect(el.textContent).toContain('Μώλυ was developed by Hermes');
    expect(el.textContent).toContain('Using solutions inherited from Socrates AI. Please sponsor.');
    expect(el.querySelectorAll('.moly-card-marks .mark').length).toBe(1);
    const links = [...el.querySelectorAll<HTMLAnchorElement>('a.moly-card-link')];
    expect(links.map((a) => a.textContent)).toEqual(['Sponsor', 'GitHub', 'Hermes Software', 'Socrates AI']);
    expect(links[0].href).toBe('https://github.com/sponsors/Nireus79');
    expect(links[0].classList.contains('primary')).toBe(true);
    for (const a of links) {
      expect(a.getAttribute('target')).toBe('_blank');
      expect(a.getAttribute('rel')).toContain('noopener');
      expect(a.href.startsWith('https://')).toBe(true);
    }
    expect(el.querySelector('button[aria-label="Hide this card"]')).not.toBeNull();
  });

  it('labels a third party\'s card "Sponsored" too', async () => {
    const el = await render([entry({ id: 's', kind: 'sponsor', title: 'Acme coaching', text: 'Better meetings.' })]);
    expect(el.querySelector('.moly-card-label')?.textContent).toBe('Sponsored');
    expect(el.textContent).toContain('Acme coaching');
  });

  it('Hide removes the card for this panel and a reload keeps it hidden', async () => {
    const el = await render();
    await act(async () => el.querySelector<HTMLButtonElement>('button[aria-label="Hide this card"]')!.click());
    expect(el.querySelector('aside')).toBeNull();
    expect(sessionStorage.getItem(HIDDEN_KEY)).toBe('1');
    host?.remove();
    expect((await render()).querySelector('aside')).toBeNull(); // the same panel, reloaded
  });

  it('is back the next time Moly opens (a new panel has an empty sessionStorage), and nothing was saved for good', async () => {
    const el = await render();
    await act(async () => el.querySelector<HTMLButtonElement>('button[aria-label="Hide this card"]')!.click());
    host?.remove();
    sessionStorage.clear(); // what the browser does when the panel closes
    expect((await render()).querySelector('aside')).not.toBeNull();
    expect((await chrome.storage.local.get(HIDDEN_KEY))[HIDDEN_KEY]).toBeUndefined();
  });

  it('removes a hide choice that an older version saved for good, so the card returns', async () => {
    await chrome.storage.local.set({ [HIDDEN_KEY]: true });
    expect((await render()).querySelector('aside')).not.toBeNull();
    expect((await chrome.storage.local.get(HIDDEN_KEY))[HIDDEN_KEY]).toBeUndefined();
  });
});

describe('card on the phone', () => {
  it('fetches the website text there too and shows it', async () => {
    const feed = { version: 1, entries: [{ id: 'f', kind: 'moly', title: 'From the website', text: 'Fresh text', active: true }] };
    const realFetch = globalThis.fetch;
    globalThis.fetch = (async () => ({ ok: true, json: async () => feed })) as unknown as typeof fetch;
    (window as any).molyPhone = { readClipboard: async () => '' };
    try {
      const h = await render();
      await act(async () => { await new Promise((r) => setTimeout(r, 20)); });
      expect(h.textContent).toContain('Fresh text');
    } finally {
      delete (window as any).molyPhone;
      globalThis.fetch = realFetch;
    }
  });
});
