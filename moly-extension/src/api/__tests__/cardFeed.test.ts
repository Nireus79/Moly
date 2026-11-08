import { describe, it, expect, beforeEach, vi } from 'vitest';
import { loadCards, parseCards, parseFeed, pngSize, validCard, FEED_KEY } from '../cardFeed';

const good = { version: 1, entries: [{ id: 'a', kind: 'sponsor', title: 'T', text: 'Hello', marks: ['hermes', 'evil'], links: [{ text: 'Go', url: 'https://example.org' }, { text: 'Bad', url: 'javascript:alert(1)' }, { text: 'Http', url: 'http://example.org' }], active: true }] };
const reply = (body: unknown, ok = true) => (async () => ({ ok, json: async () => body })) as unknown as typeof fetch;

let store: Record<string, unknown>;
beforeEach(() => {
  store = {};
  (globalThis as any).chrome = { storage: { local: { get: async (k: string) => ({ [k]: store[k] }), set: async (o: Record<string, unknown>) => Object.assign(store, o) } } };
});

describe('parseFeed', () => {
  it('keeps valid entries, drops unknown marks and every link that is not https', () => {
    const [e] = parseFeed(good)!;
    expect(e.marks).toEqual(['hermes']);
    expect(e.links!.map((l) => l.url)).toEqual(['https://example.org']);
  });
  it('accepts an entry without a message but not one with an oversized message', () => {
    const { text: _t, ...bare } = good.entries[0] as any;
    expect(parseFeed({ version: 1, entries: [bare] })!.length).toBe(1);
    expect(parseFeed({ version: 1, entries: [{ ...bare, text: 'x'.repeat(281) }] })).toEqual([]);
  });

  it('rejects files that are not a version 1 feed, and entries with missing or oversized text', () => {
    expect(parseFeed({ version: 2, entries: [] })).toBeNull();
    expect(parseFeed('x')).toBeNull();
    expect(parseFeed({ version: 1, entries: [{ ...good.entries[0], text: 'x'.repeat(281) }, { ...good.entries[0], kind: 'ad' }, { ...good.entries[0], active: 'yes' }] })).toEqual([]);
  });
});

describe('loadCards', () => {
  it('fetches, caches for a day and sends no cookies or referrer', async () => {
    const get = vi.fn(reply(good));
    expect((await loadCards(get as any, 1000))!).toHaveLength(1);
    expect(get.mock.calls[0][1]).toMatchObject({ credentials: 'omit', referrerPolicy: 'no-referrer' });
    await loadCards(get as any, 2000);
    expect(get).toHaveBeenCalledTimes(1);
    await loadCards(get as any, 1000 + 25 * 3600 * 1000);
    expect(get).toHaveBeenCalledTimes(2);
  });
  it('falls back to the cache, then to null, when the file cannot be used', async () => {
    store[FEED_KEY] = { at: 0, entries: parseFeed(good) };
    expect(await loadCards(reply({}, false), 99 * 3600 * 1000)).toHaveLength(1);
    expect(await loadCards((async () => { throw new Error('offline'); }) as any, 99 * 3600 * 1000)).toHaveLength(1);
    store = {};
    expect(await loadCards(reply({ version: 9 }), 1)).toBeNull();
  });
});

// ---- the cards of the current format (docs/CARD_FORMAT.md)
function png(w: number, h: number, extra = 0): string {
  const be = (n: number) => String.fromCharCode((n >>> 24) & 255, (n >>> 16) & 255, (n >>> 8) & 255, n & 255);
  const bin = '\x89PNG\r\n\x1a\n' + be(13) + 'IHDR' + be(w) + be(h) + '\x08\x06\x00\x00\x00' + be(0) + 'x'.repeat(extra);
  return 'data:image/png;base64,' + Buffer.from(bin, 'latin1').toString('base64');
}
const buttons = [{ text: 'Visit', url: 'https://example.org/x', primary: true }, { text: 'Http', url: 'http://example.org/' }, { text: 'Js', url: 'javascript:alert(1)' }, { text: 'Port', url: 'https://example.org:81/' }, { text: 'User', url: 'https://u:p@example.org/' }];
const logoCard = (over: Record<string, unknown> = {}) => ({ id: 'a', layout: 'logo-center', active: true, logo: { image: png(200, 80), alt: 'Acme' }, buttons, ...over });

describe('the embedded logo is checked by what it really is', () => {
  it('accepts a PNG within the limits and reads its size from the header', () => {
    expect(pngSize(png(200, 80))).toMatchObject({ width: 200, height: 80 });
    expect(pngSize(png(1200, 400))).not.toBeNull();
  });
  it('refuses what is too large, not a PNG, another type, http, or empty', () => {
    expect(pngSize(png(1201, 10))).toBeNull();
    expect(pngSize(png(10, 401))).toBeNull();
    expect(pngSize(png(10, 10, 24 * 1024))).toBeNull(); // over 24 KB
    expect(pngSize('data:image/png;base64,' + Buffer.from('GIF89a not a png at all, long enough to pass the length test').toString('base64'))).toBeNull();
    expect(pngSize('data:image/svg+xml;base64,' + Buffer.from('<svg onload="alert(1)"/>').toString('base64'))).toBeNull();
    expect(pngSize('http://example.org/x.png')).toBeNull();
    expect(pngSize('data:image/png;base64,')).toBeNull();
    expect(pngSize('data:image/png;base64,!!!notbase64!!!')).toBeNull();
  });
});

describe('validCard', () => {
  it('draws what each layout needs and nothing else', () => {
    expect(validCard({ id: 'a', layout: 'credit-row', credit: 'Created by', logo: { mark: 'hermes' }, buttons: [], active: true })).toMatchObject({ layout: 'credit-row', credit: 'Created by' });
    expect(validCard({ id: 'a', layout: 'credit-row', credit: 'Created by', buttons: [], active: true })).not.toBeNull(); // no logo is fine
    const centre = validCard(logoCard({ credit: 'ignored', text: 'ignored' }))!;
    expect(centre.credit).toBeUndefined();
    expect(centre.text).toBeUndefined(); // a centred logo is a logo and buttons only
    expect(centre.buttons.map((b) => b.url)).toEqual(['https://example.org/x']); // https only, no port, no user name
    expect(validCard(logoCard({ layout: 'logo-text', text: 'Hello' }))).toMatchObject({ text: 'Hello' });
  });
  it('drops a card that fails any check', () => {
    const bad: Array<Record<string, unknown>> = [
      logoCard({ layout: 'banner' }), logoCard({ active: 'yes' }), logoCard({ id: '' }), logoCard({ logo: undefined }),
      logoCard({ logo: { image: png(200, 80) } }), // no alt text
      logoCard({ logo: { image: png(2000, 80), alt: 'x' } }), logoCard({ logo: { mark: 'acme' } }), logoCard({ logo: { mark: 'hermes', image: png(2, 2), alt: 'x' } }),
      logoCard({ layout: 'logo-text' }), // no message
      logoCard({ layout: 'credit-row' }), // no credit
      logoCard({ layout: 'logo-text', text: 'x'.repeat(281) }),
      logoCard({ from: '15/12/2026' }), logoCard({ from: '2026-12-10', until: '2026-12-01' }),
    ];
    for (const c of bad) expect(validCard(c), JSON.stringify(c).slice(0, 80)).toBeNull();
  });
});

describe('parseCards', () => {
  it('one bad card does not take the others down, and the cards win over the entries of the first format', () => {
    const cards = parseCards({ version: 1, entries: good.entries, cards: [{ id: 'bad', layout: 'nope' }, 'not an object', logoCard()] })!;
    expect(cards.map((c) => c.id)).toEqual(['a']);
    expect(cards[0].layout).toBe('logo-center');
  });
  it('falls back to the entries of the first format, drawn as credit rows, when no card is valid', () => {
    const cards = parseCards({ version: 1, entries: good.entries, cards: [{ id: 'bad', layout: 'nope' }] })!;
    expect(cards.map((c) => [c.id, c.layout, c.credit])).toEqual([['a', 'credit-row', 'T']]);
    expect(parseCards(good)!.length).toBe(1); // a file with entries only
  });
  it('refuses what is not a version 1 feed, or has too many cards', () => {
    expect(parseCards({ version: 2, entries: [], cards: [] })).toBeNull();
    expect(parseCards({ version: 1, entries: [], cards: Array(11).fill(logoCard()) })).toEqual([]);
  });
});
