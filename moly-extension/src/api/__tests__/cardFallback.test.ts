import { describe, it, expect } from 'vitest';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { CARD_ENTRIES } from '@/config/cardEntries';
import { parseCards, parseFeed } from '../cardFeed';

// The built-in card and the card of site/cards.json say the same: two copies, one text. The file also holds the entry of the first format, for the apps already downloaded. (The source zip for the stores holds only this folder, so the check
// is skipped when the website folder is not next to it.)
const file = resolve(__dirname, '../../../../site/cards.json');

describe('built-in card against site/cards.json', () => {
  it.skipIf(!existsSync(file))('are the same card', () => {
    const json = JSON.parse(readFileSync(file, 'utf8'));
    expect(Array.isArray(json.cards) && json.cards.length > 0).toBe(true); // a card of the current format, not only the first format
    const feed = parseCards(json)!;
    expect(feed).not.toBeNull();
    const norm = (cs: typeof feed) => cs.map((c) => ({ id: c.id, layout: c.layout, credit: c.credit, text: c.text, logo: c.logo, buttons: c.buttons.map((l) => ({ text: l.text, url: l.url, primary: l.primary === true })), active: c.active }));
    expect(norm(CARD_ENTRIES)).toEqual(norm(feed));
  });
  it.skipIf(!existsSync(file))('the file keeps an entry of the first format, so that the apps already downloaded show a card', () => {
    const entries = parseFeed(JSON.parse(readFileSync(file, 'utf8')))!;
    expect(entries.length).toBeGreaterThan(0);
    expect(entries[0].text).toBeTruthy(); // the older apps refuse a card without a message
  });
});
