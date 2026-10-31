import React, { useState } from 'react';
import { detectOS, installInfo, problemText, uninstallInfo, type HostProblem } from '@/api/hostErrors';
import { INSTALL_COMMAND } from '@/api/installerDownload';
import { InstallRunner } from './InstallRunner';
import { DOWNLOAD_URL, LOG_HINT, RELEASES_URL, SUPPORT_EMAIL, UNINSTALL_URL } from '@/config/links';
import './ai/ai-provider.css';

interface Props {
  problem: HostProblem;
  /** The browser's own message, shown under "Technical details". */
  detail?: string;
  checking: boolean;
  onCheckAgain: () => void;
}

/** Always visible under the button: where the app comes from, the line to paste (Linux and Mac) and how to remove it again. */
const ByHand: React.FC = () => {
  const [copied, setCopied] = useState(false);
  const os = detectOS();
  const remove = uninstallInfo(os);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(INSTALL_COMMAND);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };
  return (
    <>
      <h3>Install it yourself</h3>
      <p>
        Download from <a href={DOWNLOAD_URL} target="_blank" rel="noopener noreferrer">the download page</a> or{' '}
        <a href={RELEASES_URL} target="_blank" rel="noopener noreferrer">GitHub releases</a>.
      </p>
      {(os === 'linux' || os === 'macos') && (
        <p>
          Or open a terminal, paste this line and press Enter:
          <br />
          <code style={{ wordBreak: 'break-all' }}>{INSTALL_COMMAND}</code>{' '}
          <button className="ai-link" onClick={copy}>{copied ? 'Copied' : 'Copy'}</button>
        </p>
      )}
      <details>
        <summary>{remove.heading}</summary>
        <ul>
          {remove.steps.map((s) => (
            <li key={s}>{s}</li>
          ))}
        </ul>
        <p className="ai-muted"><a href={UNINSTALL_URL} target="_blank" rel="noopener noreferrer">More about removing Moly</a></p>
      </details>
    </>
  );
};

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
            {/* Whatever went wrong, installing the app again is a way forward the person can take without a terminal. */}
            <InstallRunner label={problem === 'not-allowed' || problem === 'app-mismatch' ? 'Download latest version' : 'Reinstall the Moly app'} />
            <button className="ai-btn secondary" onClick={onCheckAgain} disabled={checking}>
              {checking ? 'Checking...' : 'Check again'}
            </button>
          </div>
          <p className="ai-muted">The log file is at {LOG_HINT}.</p>
        </>
      )}
      <ByHand />
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
