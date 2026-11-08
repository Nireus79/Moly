/**
 * The cards. The built-in one shows when the card file (site/cards.json, read by api/cardFeed.ts) is not available. Nothing about the person
 * is used to choose a card. Rules (Chrome Web Store): clearly labelled "Sponsored" on every card, static, easy to hide, never disguised as a
 * system message, and in its own area, never between Moly's replies.
 *
 * A card describes itself in the file as DATA, never as code: a layout from a short fixed list, a credit line or a message, a logo (the
 * built-in Hermes mark, or a small PNG embedded in the file itself, so nothing is ever fetched from a sponsor) and up to four link buttons.
 * What the file can never change, because the app draws it: the label, the Hide button and how links open. The format is in docs/CARD_FORMAT.md.
 * Entries of the first format (title, text, links, marks) are still read, for files written before the cards existed.
 */

import { HERMES_URL, PROFILE_URL, SPONSOR_URL } from './links';

export type CardKind = 'moly' | 'sponsor';

/** Marks of the first format that can stand at the top of a card (drawn from styles/marks.css). */
export type CardMark = 'hermes' | 'socrates';

export interface CardLink {
  text: string;
  url: string; // https only
  primary?: boolean;
}

/** An entry of the first format. */
export interface CardEntry {
  id: string;
  kind: CardKind;
  title: string;
  /** Optional. */
  text?: string;
  marks?: CardMark[];
  links?: CardLink[];
  active: boolean;
}

export type CardLayout = 'credit-row' | 'logo-center' | 'logo-text';

/** The picture of a card: the built-in Hermes mark, or a PNG embedded in the file (a data address). */
export interface CardLogo {
  mark?: 'hermes';
  image?: string;
  alt?: string;
  /** Black line art that turns light on the dark theme. */
  tint?: boolean;
}

export interface Card {
  id: string;
  layout: CardLayout;
  credit?: string;
  text?: string;
  logo?: CardLogo;
  buttons: CardLink[];
  /** First and last day shown (YYYY-MM-DD, UTC), both optional. */
  from?: string;
  until?: string;
  active: boolean;
}

/** A first-format entry, drawn as a credit row. */
export function cardFromEntry(e: CardEntry): Card {
  return {
    id: e.id,
    layout: 'credit-row',
    credit: e.title,
    text: e.text,
    logo: e.marks?.includes('hermes') ? { mark: 'hermes' } : undefined,
    buttons: e.links ?? [],
    active: e.active,
  };
}

export const CARD_ENTRIES: Card[] = [
  {
    id: 'created-by-hermes',
    layout: 'credit-row',
    credit: 'Created by',
    logo: { mark: 'hermes' },
    buttons: [
      { text: 'Sponsor', url: SPONSOR_URL, primary: true },
      { text: 'GitHub', url: PROFILE_URL },
      { text: 'Hermes', url: HERMES_URL },
    ],
    active: true,
  },
];
