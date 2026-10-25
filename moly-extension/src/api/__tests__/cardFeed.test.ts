import { describe, it, expect, beforeEach, vi } from 'vitest';
import { loadCards, parseFeed, FEED_KEY } from '../cardFeed';

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
