import React, { useEffect, useState } from 'react';
import { cardFromEntry, type Card, type CardEntry } from '@/config/cardEntries';
import { loadCards } from '@/api/cardFeed';
import { clearLegacyHidden, isHidden, isSafeUrl, labelFor, pickEntry, setHidden } from '@/api/sponsorCard';
import { isPhoneApp } from '@/utils/platform';
import './sponsor-card.css';

/**
 * A small card, the same in Moly and Socrates, always labelled "Sponsored". It is static (no scripts; what it shows is a small data file on the
 * project's website, see docs/CARD_FORMAT.md): a layout, a credit line or a message, a logo, link buttons. The person can hide it with one
 * click (it returns the next time Moly opens), and it has its own area below the writing box, so it never stands between Moly's replies.
 * The label, the Hide button and how links open belong to the app, never to the file.
 */
const toCard = (e: Card | CardEntry): Card => ('layout' in e ? e : cardFromEntry(e));

const Logo: React.FC<{ card: Card }> = ({ card }) => {
  const logo = card.logo;
  if (!logo) return null;
  if (logo.mark) {
    return (
      <span className="moly-card-marks" aria-hidden="true">
        <span className={`mark mark-${logo.mark}`} />
      </span>
    );
  }
  return <img className={`moly-card-logo${logo.tint ? ' tint' : ''}`} src={logo.image} alt={logo.alt ?? ''} />;
};

export const SponsorCard: React.FC<{ entries?: Array<Card | CardEntry> }> = ({ entries }) => {
  const [hidden, setHiddenState] = useState<boolean>(() => isHidden());
  const [feed, setFeed] = useState<Card[] | null>(null);
  const card = pickEntry(entries ? entries.map(toCard) : (feed ?? undefined));

  useEffect(() => {
    void clearLegacyHidden();
  }, []);

  // The cards may be updated from the project's website; the built-in one shows until (and unless) a valid file arrives.
  useEffect(() => {
    if (entries) return;
    let alive = true;
    loadCards().then((f) => alive && f && setFeed(f));
    return () => {
      alive = false;
    };
  }, [entries]);

  if (hidden || !card) return null;

  const links = (card.buttons ?? []).filter((l) => isSafeUrl(l.url));
  const buttons =
    links.length > 0 ? (
      <div className="moly-card-links">
        {links.map((l) => (
          <a key={l.url} className={`moly-card-link${l.primary ? ' primary' : ''}`} href={l.url} target="_blank" rel="noopener noreferrer sponsored">
            {l.text}
          </a>
        ))}
      </div>
    ) : null;
  const text = card.text ? <p className="moly-card-text">{card.text}</p> : null;
  let body: React.ReactNode;
  if (card.layout === 'logo-center') {
    body = (
      <div className="moly-card-body">
        <Logo card={card} />
        {buttons}
      </div>
    );
  } else if (card.layout === 'logo-text') {
    body = (
      <div className="moly-card-body">
        <div className="moly-card-media">
          <Logo card={card} />
          {text}
        </div>
        {buttons}
      </div>
    );
  } else {
    body = (
      <div className="moly-card-body">
        <div className="moly-card-credit">
          <strong className="moly-card-title">{card.credit}</strong>
          <Logo card={card} />
        </div>
        {buttons}
      </div>
    );
  }

  return (
    <aside className={`moly-card moly-card-${card.layout}${isPhoneApp() ? ' moly-card-phone' : ''}`} aria-label={labelFor()}>
      <div className="moly-card-head">
        <span className="moly-card-label">{labelFor()}</span>
      </div>
      <button className="moly-card-hide" aria-label="Hide this card" title="Hide this card until you open Moly again" onClick={() => { setHiddenState(true); setHidden(true); }}>
        Hide
      </button>
      {body}
      {card.layout === 'credit-row' ? text : null}
    </aside>
  );
};

export default SponsorCard;
