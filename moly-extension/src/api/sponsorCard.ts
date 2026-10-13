/** Logic of the card: which entry, how it is labelled, whether the person hid it. Nothing here reads the person's data. */

import { CARD_ENTRIES, type CardEntry, type CardKind } from '@/config/cardEntries';

export const HIDDEN_KEY = 'moly_card_hidden';

export const labelFor = (kind: CardKind): string => (kind === 'sponsor' ? 'Sponsored' : 'Moly message');

/** Only https links are ever shown as links. */
export function isSafeUrl(url: string | undefined): url is string {
  if (!url) return false;
  try {
    return new URL(url).protocol === 'https:';
  } catch {
    return false;
  }
}

/** One of the active entries, chosen by the day only (so the card changes slowly and does not depend on anything else). */
export function pickEntry(entries: CardEntry[] = CARD_ENTRIES, now: Date = new Date()): CardEntry | null {
  const active = entries.filter((e) => e.active);
  if (active.length === 0) return null;
  const day = Math.floor(now.getTime() / 86_400_000);
  return active[day % active.length];
}

/**
 * Hide lasts only while the panel is open: the choice lives in sessionStorage, which the browser discards when the panel
 * closes, so the card is back the next time Moly opens. (Reloading an open panel keeps it hidden.)
 */
export function isHidden(): boolean {
  try {
    return sessionStorage.getItem(HIDDEN_KEY) === '1';
  } catch {
    return false;
  }
}

export function setHidden(hidden: boolean): void {
  try {
    if (hidden) sessionStorage.setItem(HIDDEN_KEY, '1');
    else sessionStorage.removeItem(HIDDEN_KEY);
  } catch {
    /* storage unavailable: the card is hidden until the panel is rendered again */
  }
}

/** Earlier versions kept the choice for good in the extension storage; that is removed so the card returns. */
export async function clearLegacyHidden(): Promise<void> {
  try {
    await chrome.storage.local.remove(HIDDEN_KEY);
  } catch {
    /* nothing to clean */
  }
}
