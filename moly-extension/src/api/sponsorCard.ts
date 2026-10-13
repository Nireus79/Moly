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

export async function isHidden(): Promise<boolean> {
  try {
    const stored = await chrome.storage.local.get(HIDDEN_KEY);
    return stored?.[HIDDEN_KEY] === true;
  } catch {
    return false;
  }
}

export async function setHidden(hidden: boolean): Promise<void> {
  try {
    await chrome.storage.local.set({ [HIDDEN_KEY]: hidden });
  } catch {
    /* storage unavailable: the card simply stays as it is */
  }
}
