import React, { useEffect, useState } from 'react';
import { isHidden, isSafeUrl, labelFor, pickEntry, setHidden, HIDDEN_KEY } from '@/api/sponsorCard';
import './sponsor-card.css';

/**
 * A small, labelled card with a message from Moly or from a sponsor. It is static (no network, no scripts), the person can hide
 * it with one click (and bring it back in Settings), and it is shown only where there is no conversation yet, so it can never
 * stand next to one of Moly's replies.
 */
export const SponsorCard: React.FC = () => {
  const [hidden, setHiddenState] = useState<boolean | null>(null);
  const entry = pickEntry();

  useEffect(() => {
    let alive = true;
    isHidden().then((h) => alive && setHiddenState(h));
    const onChange = (changes: Record<string, { newValue?: unknown }>, area: string) => {
      if (area === 'local' && changes[HIDDEN_KEY]) setHiddenState(changes[HIDDEN_KEY].newValue === true);
    };
    chrome.storage?.onChanged?.addListener?.(onChange);
    return () => {
      alive = false;
      chrome.storage?.onChanged?.removeListener?.(onChange);
    };
  }, []);

  if (hidden !== false || !entry) return null;

  return (
    <aside className="moly-card" aria-label={labelFor(entry.kind)}>
      <div className="moly-card-head">
        <span className="moly-card-label">{labelFor(entry.kind)}</span>
        <button className="moly-card-hide" aria-label="Hide this card" title="Hide this card (you can show it again in Settings)" onClick={() => { setHiddenState(true); void setHidden(true); }}>
          Hide
        </button>
      </div>
      <strong className="moly-card-title">{entry.title}</strong>
      <p className="moly-card-text">{entry.text}</p>
      {isSafeUrl(entry.url) && (
        <a className="moly-card-link" href={entry.url} target="_blank" rel="noopener noreferrer sponsored">
          {entry.linkText || entry.url}
        </a>
      )}
    </aside>
  );
};

export default SponsorCard;
