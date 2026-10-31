import { describe, it, expect } from 'vitest';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { CARD_ENTRIES } from '@/config/cardEntries';
import { parseFeed } from '../cardFeed';

// The built-in card and site/cards.json say the same: two copies, one text. (The source zip for the stores holds only this folder, so the check
// is skipped when the website folder is not next to it.)
const file = resolve(__dirname, '../../../../site/cards.json');

describe('built-in card against site/cards.json', () => {
  it.skipIf(!existsSync(file))('are the same entries', () => {
    const feed = parseFeed(JSON.parse(readFileSync(file, 'utf8')))!;
    expect(feed).not.toBeNull();
    const norm = (es: typeof feed) => es.map((e) => ({ id: e.id, kind: e.kind, title: e.title, text: e.text, marks: e.marks ?? [], links: (e.links ?? []).map((l) => ({ text: l.text, url: l.url, primary: l.primary === true })), active: e.active }));
    expect(norm(CARD_ENTRIES)).toEqual(norm(feed));
  });
});
