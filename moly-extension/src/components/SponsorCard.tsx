import React, { useEffect, useState } from 'react';
import type { CardEntry } from '@/config/cardEntries';
import { clearLegacyHidden, isHidden, isSafeUrl, labelFor, pickEntry, setHidden } from '@/api/sponsorCard';
import { isPhoneApp } from '@/utils/platform';
import './sponsor-card.css';

/**
 * A small, labelled card with a message from Moly or from a sponsor. It is static (no network, no scripts), the person can hide
 * it with one click (it returns the next time Moly opens), and it has its own area below the writing box, so it never stands between
 * Moly's replies.
 */
export const SponsorCard: React.FC<{ entries?: CardEntry[] }> = ({ entries }) => {
  const [hidden, setHiddenState] = useState<boolean>(() => isHidden());
  const entry = pickEntry(entries);

  useEffect(() => {
    void clearLegacyHidden();
  }, []);

  if (hidden || !entry) return null;

  const links = (entry.links ?? []).filter((l) => isSafeUrl(l.url));

  return (
    <aside className={isPhoneApp() ? 'moly-card moly-card-phone' : 'moly-card'} aria-label={labelFor(entry.kind)}>
      <div className="moly-card-head">
        <span className="moly-card-label">{labelFor(entry.kind)}</span>
      </div>
      <button className="moly-card-hide" aria-label="Hide this card" title="Hide this card until you open Moly again" onClick={() => { setHiddenState(true); setHidden(true); }}>
        Hide
      </button>
      {entry.marks && entry.marks.length > 0 && (
        <div className="moly-card-marks" aria-hidden="true">
          {entry.marks.map((m) => (
            <span key={m} className={`mark mark-${m}`} />
          ))}
        </div>
      )}
      <strong className="moly-card-title">{entry.title}</strong>
      <p className="moly-card-text">{entry.text}</p>
      {links.length > 0 && (
        <div className="moly-card-links">
          {links.map((l) => (
            <a key={l.url} className={`moly-card-link${l.primary ? ' primary' : ''}`} href={l.url} target="_blank" rel="noopener noreferrer sponsored">
              {l.text}
            </a>
          ))}
        </div>
      )}
    </aside>
  );
};

export default SponsorCard;
