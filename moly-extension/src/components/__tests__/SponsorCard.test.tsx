import { describe, it, expect, afterEach, beforeEach } from 'vitest';
import React from 'react';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';
import { SponsorCard } from '../SponsorCard';
import { HIDDEN_KEY, isHidden, isSafeUrl, labelFor, pickEntry, setHidden } from '@/api/sponsorCard';
import { CARD_ENTRIES, type CardEntry } from '@/config/cardEntries';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement | null = null;
async function render() {
  host = document.createElement('div');
  document.body.appendChild(host);
  await act(async () => createRoot(host!).render(<SponsorCard />));
  await act(async () => { await Promise.resolve(); });
  return host;
}

beforeEach(async () => {
  await chrome.storage.local.clear?.();
});
afterEach(() => {
  host?.remove();
  host = null;
});

const entry = (over: Partial<CardEntry> = {}): CardEntry => ({ id: 'x', kind: 'moly', title: 't', text: 'x', active: true, ...over });

describe('card logic', () => {
  it('labels Moly messages and sponsors differently', () => {
    expect(labelFor('moly')).toBe('Moly message');
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

  it('remembers that the person hid it', async () => {
    expect(await isHidden()).toBe(false);
    await setHidden(true);
    expect(await isHidden()).toBe(true);
    await setHidden(false);
    expect(await isHidden()).toBe(false);
  });

  it('the shipped entries are all labelled, https only and free of scripts', () => {
    expect(CARD_ENTRIES.some((e) => e.active)).toBe(true);
    for (const e of CARD_ENTRIES) {
      if (e.url) expect(isSafeUrl(e.url)).toBe(true);
      expect(`${e.title} ${e.text}`).not.toMatch(/<script|javascript:/i);
    }
  });
});

describe('SponsorCard', () => {
  it('shows a labelled card with a safe link and a Hide button', async () => {
    const el = await render();
    expect(el.textContent).toContain('Moly message');
    expect(el.textContent).toContain('Moly is free and open source');
    const link = el.querySelector('a')!;
    expect(link.getAttribute('target')).toBe('_blank');
    expect(link.getAttribute('rel')).toContain('noopener');
    expect(el.querySelector('button[aria-label="Hide this card"]')).not.toBeNull();
  });

  it('Hide removes the card and keeps it away', async () => {
    const el = await render();
    await act(async () => el.querySelector<HTMLButtonElement>('button[aria-label="Hide this card"]')!.click());
    expect(el.querySelector('aside')).toBeNull();
    expect((await chrome.storage.local.get(HIDDEN_KEY))[HIDDEN_KEY]).toBe(true);
    host?.remove();
    const again = await render(); // a new panel later
    expect(again.querySelector('aside')).toBeNull();
  });

  it('shows nothing once hidden elsewhere, and again after it is switched back on', async () => {
    await setHidden(true);
    expect((await render()).querySelector('aside')).toBeNull();
    host?.remove();
    await setHidden(false);
    expect((await render()).querySelector('aside')).not.toBeNull();
  });
});
