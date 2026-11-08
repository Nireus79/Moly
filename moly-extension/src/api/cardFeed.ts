/**
 * The card messages can be updated without a new extension version: a data file on the project's website. It is data only
 * (never code), fetched at most once a day with a plain request that carries no cookies, no referrer and nothing about the
 * person. Everything is checked; whatever does not pass is ignored, and the built-in messages show when nothing valid
 * is available. The text is shown as text (never as markup) and only https links are kept.
 */

import type { Card, CardEntry, CardKind, CardLayout, CardLink, CardLogo, CardMark } from '@/config/cardEntries';
import { cardFromEntry } from '@/config/cardEntries';
import { CARDS_URL } from '@/config/links';

// A new key: what is kept here is now the cards (the current format), not the entries of the first one.
export const FEED_KEY = 'moly_card_feed_v2';
export const FEED_MAX_AGE_MS = 24 * 3600 * 1000;
const MARKS: CardMark[] = ['hermes', 'socrates'];
const KINDS: CardKind[] = ['moly', 'sponsor'];
const LAYOUTS: CardLayout[] = ['credit-row', 'logo-center', 'logo-text'];
export const MAX_IMAGE_BYTES = 24 * 1024;
export const MAX_IMAGE_WIDTH = 1200;
export const MAX_IMAGE_HEIGHT = 400;
const PNG_URI = /^data:image\/png;base64,([A-Za-z0-9+/]+={0,2})$/;
const DAY = /^\d{4}-\d{2}-\d{2}$/;

const str = (v: unknown, max: number): string | null => (typeof v === 'string' && v.trim() !== '' && v.length <= max ? v : null);

function validLink(v: any): CardLink | null {
  const text = str(v?.text, 30);
  const url = str(v?.url, 300);
  if (!text || !url) return null;
  try {
    const u = new URL(url);
    if (u.protocol !== 'https:' || u.username || u.password || u.port) return null;
  } catch {
    return null;
  }
  return { text, url, primary: v.primary === true };
}

const validLinks = (v: unknown): CardLink[] =>
  (Array.isArray(v) ? v : []).slice(0, 4).map(validLink).filter((l: CardLink | null): l is CardLink => l !== null);

function validEntry(v: any): CardEntry | null {
  const id = str(v?.id, 64);
  const title = str(v?.title, 80);
  const text = v?.text === undefined || v?.text === '' ? '' : str(v.text, 280); // optional, but bounded when given
  if (!id || !title || text === null || !KINDS.includes(v.kind) || typeof v.active !== 'boolean') return null;
  const marks = Array.isArray(v.marks) ? (v.marks.filter((m: unknown) => MARKS.includes(m as CardMark)) as CardMark[]) : [];
  return { id, kind: v.kind, title, ...(text ? { text } : {}), marks, links: validLinks(v.links), active: v.active };
}

/** Entries of the first format ({"version": 1, "entries": [...]}), or null when the file is not a feed. */
export function parseFeed(json: unknown): CardEntry[] | null {
  const j = json as { version?: unknown; entries?: unknown };
  if (!j || j.version !== 1 || !Array.isArray(j.entries) || j.entries.length > 10) return null;
  return j.entries.map(validEntry).filter((e): e is CardEntry => e !== null);
}

/** Standard base64 to bytes, by hand (so that it does not depend on the web view's atob); null when it is not well formed. */
function base64Bytes(s: string): number[] | null {
  if (s.length % 4 !== 0) return null;
  const out: number[] = [];
  const val = (c: string) => 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'.indexOf(c);
  for (let i = 0; i < s.length; i += 4) {
    const q = [s[i], s[i + 1], s[i + 2], s[i + 3]];
    const pad = q.filter((c) => c === '=').length;
    if (pad > 2 || (pad > 0 && i + 4 < s.length) || q.slice(0, 4 - pad).some((c) => val(c) < 0)) return null;
    const n = (val(q[0]) << 18) | (val(q[1]) << 12) | ((pad < 2 ? val(q[2]) : 0) << 6) | (pad < 1 ? val(q[3]) : 0);
    out.push((n >> 16) & 255);
    if (pad < 2) out.push((n >> 8) & 255);
    if (pad < 1) out.push(n & 255);
  }
  return out;
}

/** The size of an embedded PNG from its real header (not from its name), or null when it is not a PNG within the limits. */
export function pngSize(uri: string): { width: number; height: number; bytes: number } | null {
  const m = PNG_URI.exec(uri);
  if (!m || uri.length > (MAX_IMAGE_BYTES * 4) / 3 + 64) return null;
  const b = base64Bytes(m[1]);
  if (!b || b.length < 33 || b.length > MAX_IMAGE_BYTES) return null;
  const sig = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];
  if (!sig.every((v, i) => b[i] === v) || String.fromCharCode(b[12], b[13], b[14], b[15]) !== 'IHDR') return null;
  const n = (o: number) => ((b[o] << 24) | (b[o + 1] << 16) | (b[o + 2] << 8) | b[o + 3]) >>> 0;
  const width = n(16);
  const height = n(20);
  if (width < 1 || height < 1 || width > MAX_IMAGE_WIDTH || height > MAX_IMAGE_HEIGHT) return null;
  return { width, height, bytes: b.length };
}

function validLogo(v: any): CardLogo | null {
  if (v?.mark === 'hermes' && v.image === undefined) return { mark: 'hermes' };
  const alt = str(v?.alt, 100);
  if (v && v.mark === undefined && typeof v.image === 'string' && alt && pngSize(v.image)) return { image: v.image, alt, tint: v.tint === true };
  return null;
}

const day = (v: unknown): string | null | undefined => (v === undefined || v === '' ? undefined : typeof v === 'string' && DAY.test(v) && !Number.isNaN(Date.parse(v)) ? v : null);

/** One card of the current format, or null: a card that fails any check is dropped on its own. */
export function validCard(v: any): Card | null {
  const id = str(v?.id, 64);
  const layout = v?.layout as CardLayout;
  if (!id || !LAYOUTS.includes(layout) || typeof v.active !== 'boolean') return null;
  const credit = v.credit === undefined || v.credit === '' ? '' : str(v.credit, 40);
  const text = v.text === undefined || v.text === '' ? '' : str(v.text, 280);
  if (credit === null || text === null) return null;
  const from = day(v.from);
  const until = day(v.until);
  if (from === null || until === null || (from && until && until < from)) return null;
  let logo: CardLogo | undefined;
  if (v.logo !== undefined) {
    logo = validLogo(v.logo) ?? undefined;
    if (!logo) return null;
  }
  if (layout === 'credit-row' && !credit) return null;
  if (layout !== 'credit-row' && !logo) return null;
  if (layout === 'logo-text' && !text) return null;
  const card: Card = { id, layout, buttons: validLinks(v.buttons), active: v.active };
  if (layout === 'credit-row') card.credit = credit;
  if (layout !== 'logo-center' && text) card.text = text;
  if (logo) card.logo = logo;
  if (from) card.from = from;
  if (until) card.until = until;
  return card;
}

/** Everything a feed file holds: its cards and, failing those, its entries of the first format drawn as cards; null when it is not a feed. */
export function parseCards(json: unknown): Card[] | null {
  const j = json as { version?: unknown; cards?: unknown };
  const entries = parseFeed(json);
  if (!entries) return null;
  const cards = Array.isArray(j.cards) && j.cards.length <= 10 ? j.cards.map(validCard).filter((c): c is Card => c !== null) : [];
  return cards.length ? cards : entries.map(cardFromEntry);
}

async function readCache(): Promise<{ at: number; entries: Card[] } | null> {
  try {
    const got = await chrome.storage.local.get(FEED_KEY);
    const c = got?.[FEED_KEY];
    return c && typeof c.at === 'number' && Array.isArray(c.entries) ? c : null;
  } catch {
    return null;
  }
}

/** The current messages from the website (cached for a day), or null: then the built-in ones are used. */
export async function loadCards(get: typeof fetch = fetch, now: number = Date.now()): Promise<Card[] | null> {
  const cached = await readCache();
  if (cached && now - cached.at < FEED_MAX_AGE_MS) return cached.entries;
  try {
    const ctl = new AbortController();
    const timer = setTimeout(() => ctl.abort(), 5000);
    const res = await get(CARDS_URL, { credentials: 'omit', cache: 'no-store', referrerPolicy: 'no-referrer', signal: ctl.signal });
    clearTimeout(timer);
    if (!res.ok) return cached?.entries ?? null;
    const entries = parseCards(await res.json());
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
