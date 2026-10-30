import React, { useState } from 'react';
import { detectOS, installInfo, problemText, type HostProblem } from '@/api/hostErrors';
import { InstallRunner } from './InstallRunner';
import { LOG_HINT, SUPPORT_EMAIL } from '@/config/links';
import './ai/ai-provider.css';

interface Props {
  problem: HostProblem;
  /** The browser's own message, shown under "Technical details". */
  detail?: string;
  checking: boolean;
  onCheckAgain: () => void;
}

/** Shown when the Moly app is missing, not recognised, or could not start. */
export const InstallMoly: React.FC<Props> = ({ problem, detail, checking, onCheckAgain }) => {
  const [showDetail, setShowDetail] = useState(false);
  const missing = problem === 'not-installed';
  const info = installInfo(detectOS());
  const text = problemText(problem);
  return (
    <div className="ai-card" role="alert">
      {missing ? (
        <>
          <h2>{info.heading}</h2>
          <p>
            Moly needs a small app on your computer. It keeps everything on your computer, starts when you open Moly in your browser, and stops when you close the browser.
          </p>
          <ol>
            {info.steps.map((s) => (
              <li key={s}>{s}</li>
            ))}
          </ol>
          {info.note && <div className="ai-notice">{info.note}</div>}
          <div className="ai-actions">
            <InstallRunner label="Download latest version" />
            <button className="ai-btn secondary" onClick={onCheckAgain} disabled={checking}>
              {checking ? 'Checking...' : 'Check again'}
            </button>
          </div>
        </>
      ) : (
        <>
          <h2>{text.title}</h2>
          <p>{text.body}</p>
          <div className="ai-actions">
            {(problem === 'not-allowed' || problem === 'app-mismatch') && <InstallRunner label="Download latest version" />}
            <button className="ai-btn secondary" onClick={onCheckAgain} disabled={checking}>
              {checking ? 'Checking...' : 'Check again'}
            </button>
          </div>
          <p className="ai-muted">The log file is at {LOG_HINT}.</p>
        </>
      )}
      <p className="ai-muted">
        Need help? Write to <a href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a>.
        {detail && (
          <>
            {' '}
            <button className="ai-link" onClick={() => setShowDetail((v) => !v)}>
              {showDetail ? 'Hide' : 'Technical details'}
            </button>
          </>
        )}
      </p>
      {showDetail && detail && <div className="ai-notice" style={{ wordBreak: 'break-word' }}>{detail}</div>}
    </div>
  );
};

export default InstallMoly;
