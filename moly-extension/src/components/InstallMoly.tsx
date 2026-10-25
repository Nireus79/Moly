import React, { useState } from 'react';
import { detectOS, installInfo, problemText, type HostProblem } from '@/api/hostErrors';
import { installAction, type InstallAction } from '@/api/installerDownload';
import { DOWNLOAD_URL, LOG_HINT, SUPPORT_EMAIL } from '@/config/links';
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
  const [action, setAction] = useState<InstallAction | null>(null);
  const [copied, setCopied] = useState(false);

  /** The one install button: finds the right file for this system. The browser cannot run it, so it says what to do next. */
  const startInstall = async () => {
    const a = await installAction(detectOS());
    if (a.kind === 'download') {
      const link = document.createElement('a');
      link.href = a.url;
      link.download = a.fileName;
      document.body.appendChild(link);
      link.click();
      link.remove();
    } else if (a.kind === 'page') {
      window.open(DOWNLOAD_URL, '_blank', 'noopener,noreferrer');
    }
    setAction(a);
  };
  const copyCommand = async (command: string) => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };
  const installHelp = action && (
    <div className="ai-notice" role="status">
      {action.kind === 'download' && (
        <>Downloading {action.fileName}. When it has finished, open it from your browser's download list and follow the installer. This screen continues by itself.</>
      )}
      {action.kind === 'command' && (
        <>
          Open a terminal, paste this line and press Enter. This screen continues by itself.
          <br />
          <code style={{ wordBreak: 'break-all' }}>{action.command}</code>{' '}
          <button className="ai-link" onClick={() => copyCommand(action.command)}>{copied ? 'Copied' : 'Copy'}</button>
        </>
      )}
      {action.kind === 'page' && <>The download page is open in a new tab. Install the app there, then come back.</>}
    </div>
  );

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
            <button className="ai-btn primary" onClick={startInstall}>
              Download latest version
            </button>
            <button className="ai-btn secondary" onClick={onCheckAgain} disabled={checking}>
              {checking ? 'Checking...' : 'Check again'}
            </button>
          </div>
          {installHelp}
        </>
      ) : (
        <>
          <h2>{text.title}</h2>
          <p>{text.body}</p>
          <div className="ai-actions">
            {problem === 'not-allowed' && (
              <button className="ai-btn primary" onClick={startInstall}>
                Download latest version
              </button>
            )}
            <button className="ai-btn secondary" onClick={onCheckAgain} disabled={checking}>
              {checking ? 'Checking...' : 'Check again'}
            </button>
          </div>
          {installHelp}
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
