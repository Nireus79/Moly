/**
 * The card messages can be updated without a new extension version: a data file on the project's website. It is data only
 * (never code), fetched at most once a day with a plain request that carries no cookies, no referrer and nothing about the
 * person. Everything is checked; whatever does not pass is ignored, and the built-in messages show when nothing valid
 * is available. The text is shown as text (never as markup) and only https links are kept.
 */

import type { CardEntry, CardKind, CardLink, CardMark } from '@/config/cardEntries';
import { CARDS_URL } from '@/config/links';

export const FEED_KEY = 'moly_card_feed';
export const FEED_MAX_AGE_MS = 24 * 3600 * 1000;
const MARKS: CardMark[] = ['hermes', 'socrates'];
const KINDS: CardKind[] = ['moly', 'sponsor'];

const str = (v: unknown, max: number): string | null => (typeof v === 'string' && v.trim() !== '' && v.length <= max ? v : null);

function validLink(v: any): CardLink | null {
  const text = str(v?.text, 30);
  const url = str(v?.url, 300);
  if (!text || !url) return null;
  try {
    if (new URL(url).protocol !== 'https:') return null;
  } catch {
    return null;
  }
  return { text, url, primary: v.primary === true };
}

function validEntry(v: any): CardEntry | null {
  const id = str(v?.id, 64);
  const title = str(v?.title, 80);
  const text = str(v?.text, 280);
  if (!id || !title || !text || !KINDS.includes(v.kind) || typeof v.active !== 'boolean') return null;
  const marks = Array.isArray(v.marks) ? (v.marks.filter((m: unknown) => MARKS.includes(m as CardMark)) as CardMark[]) : [];
  const links = (Array.isArray(v.links) ? v.links : []).slice(0, 3).map(validLink).filter((l: CardLink | null): l is CardLink => l !== null);
  return { id, kind: v.kind, title, text, marks, links, active: v.active };
}

/** Entries of a feed file ({"version": 1, "entries": [...]}), or null when the file is not a feed. */
export function parseFeed(json: unknown): CardEntry[] | null {
  const j = json as { version?: unknown; entries?: unknown };
  if (!j || j.version !== 1 || !Array.isArray(j.entries) || j.entries.length > 10) return null;
  return j.entries.map(validEntry).filter((e): e is CardEntry => e !== null);
}

async function readCache(): Promise<{ at: number; entries: CardEntry[] } | null> {
  try {
    const got = await chrome.storage.local.get(FEED_KEY);
    const c = got?.[FEED_KEY];
    return c && typeof c.at === 'number' && Array.isArray(c.entries) ? c : null;
  } catch {
    return null;
  }
}

/** The current messages from the website (cached for a day), or null: then the built-in ones are used. */
export async function loadCards(get: typeof fetch = fetch, now: number = Date.now()): Promise<CardEntry[] | null> {
  const cached = await readCache();
  if (cached && now - cached.at < FEED_MAX_AGE_MS) return cached.entries;
  try {
    const ctl = new AbortController();
    const timer = setTimeout(() => ctl.abort(), 5000);
    const res = await get(CARDS_URL, { credentials: 'omit', cache: 'no-store', referrerPolicy: 'no-referrer', signal: ctl.signal });
    clearTimeout(timer);
    if (!res.ok) return cached?.entries ?? null;
    const entries = parseFeed(await res.json());
    if (!entries) return cached?.entries ?? null;
    try {
      await chrome.storage.local.set({ [FEED_KEY]: { at: now, entries } });
    } catch {
      /* no storage: the file is fetched again next time */
    }
    return entries;
  } catch {
    return cached?.entries ?? null;
  }
}
