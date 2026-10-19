/**
 * The messages the card can show. They are part of the extension package: nothing is fetched, nothing about the person is
 * used to choose one. Add a sponsor by adding an entry with kind 'sponsor' (it is then labelled "Sponsored"); set active to
 * false to retire an entry. Rules (Chrome Web Store): clearly labelled, static, easy to hide, never disguised as a system
 * message, and in its own area, never between Moly's replies.
 */

import { HERMES_URL, REPO_URL, SOCRATES_URL, SPONSOR_URL } from './links';

export type CardKind = 'moly' | 'sponsor';

/** Marks that can stand at the top of a card (drawn from styles/marks.css). */
export type CardMark = 'hermes' | 'socrates';

export interface CardLink {
  text: string;
  url: string; // https only
  primary?: boolean;
}

export interface CardEntry {
  id: string;
  kind: CardKind;
  title: string;
  text: string;
  marks?: CardMark[];
  links?: CardLink[];
  active: boolean;
}

export const CARD_ENTRIES: CardEntry[] = [
  {
    id: 'about-moly',
    kind: 'moly',
    title: 'Μώλυ was developed by Hermes',
    text: 'Using solutions inherited from Socrates AI. Please sponsor.',
    marks: ['hermes'],
    links: [
      { text: 'Sponsor', url: SPONSOR_URL, primary: true },
      { text: 'GitHub', url: REPO_URL },
      { text: 'Hermes Software', url: HERMES_URL },
      { text: 'Socrates AI', url: SOCRATES_URL },
    ],
    active: true,
  },
];
