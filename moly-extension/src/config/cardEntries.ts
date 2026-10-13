/**
 * The messages the card can show. They are part of the extension package: nothing is fetched, nothing about the person is
 * used to choose one. Add a sponsor by adding an entry with kind 'sponsor' (it is then labelled "Sponsored"); set active to
 * false to retire an entry. Rules (Chrome Web Store): clearly labelled, static, easy to hide, never disguised as a system
 * message, and only shown where no reply of Moly can be next to it.
 */

export type CardKind = 'moly' | 'sponsor';

export interface CardEntry {
  id: string;
  kind: CardKind;
  title: string;
  text: string;
  linkText?: string;
  url?: string; // https only
  active: boolean;
}

export const CARD_ENTRIES: CardEntry[] = [
  {
    id: 'about-moly',
    kind: 'moly',
    title: 'Moly is free and open source',
    text: 'It keeps your data on your computer. If it helps you, tell a friend, or follow the project.',
    linkText: 'github.com/Nireus79/Moly',
    url: 'https://github.com/Nireus79/Moly',
    active: true,
  },
];
