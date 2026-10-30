import React, { useEffect, useState } from 'react';
import { detectOS } from '@/api/hostErrors';
import { installAction, type InstallAction } from '@/api/installerDownload';
import { downloadAndOpen, ensureDownloadsPermission, openDownload, type RunState } from '@/api/installRunner';
import { DOWNLOAD_URL } from '@/config/links';
import './ai/ai-provider.css';

/**
 * The one button that installs or updates the Moly app on this system: it finds the installer of the newest release, downloads it and opens it.
 * The newest release is looked up when the button appears, not when it is pressed, because the browser allows the permission prompt only
 * right after a click and a lookup first would use up that moment. Where the system has no installer to open, the button shows the command or
 * the download page instead.
 */
export const InstallRunner: React.FC<{ label: string; secondary?: boolean }> = ({ label, secondary }) => {
  const [action, setAction] = useState<InstallAction | null>(null);
  const [run, setRun] = useState<RunState>({ phase: 'idle' });
  const [copied, setCopied] = useState(false);
  const [shown, setShown] = useState(false);

  useEffect(() => {
    let alive = true;
    installAction(detectOS()).then((a) => alive && setAction(a));
    return () => {
      alive = false;
    };
  }, []);

  const press = async () => {
    if (!action) return;
    if (action.kind === 'download') {
      const permitted = await ensureDownloadsPermission(); // first thing in the click
      await downloadAndOpen(action.url, action.fileName, permitted, setRun);
    } else if (action.kind === 'page') {
      window.open(DOWNLOAD_URL, '_blank', 'noopener,noreferrer');
      setShown(true);
    } else {
      setShown(true);
    }
  };

  const openNow = async () => {
    if (run.phase !== 'ready') return;
    setRun((await openDownload(run.id)) ? { phase: 'opened' } : { phase: 'saved' });
  };

  const copy = async (command: string) => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };

  const cls = secondary ? 'ai-btn secondary' : 'ai-btn primary';
  const busy = run.phase === 'downloading';
  return (
    <>
      {run.phase === 'ready' ? (
        <button className="ai-btn primary" onClick={openNow}>Open the installer</button>
      ) : (
        <button className={cls} onClick={press} disabled={!action || busy}>
          {busy ? `Downloading${run.percent !== null ? ` ${run.percent}%` : '...'}` : label}
        </button>
      )}
      <div aria-live="polite">
        {run.phase === 'ready' && <div className="ai-notice" role="status">The installer is downloaded. Press "Open the installer"; this screen continues by itself when Moly is ready.</div>}
        {run.phase === 'opened' && <div className="ai-notice" role="status">The installer is open. Follow it; this screen continues by itself when Moly is ready.</div>}
        {run.phase === 'saved' && action?.kind === 'download' && (
          <div className="ai-notice" role="status">Downloaded {action.fileName}. Open it from your browser's download list and follow the installer. This screen continues by itself.</div>
        )}
        {run.phase === 'failed' && <div className="ai-notice" role="alert">{run.message}</div>}
        {shown && action?.kind === 'command' && (
          <div className="ai-notice" role="status">
            Open a terminal, paste this line and press Enter. This screen continues by itself.
            <br />
            <code style={{ wordBreak: 'break-all' }}>{action.command}</code>{' '}
            <button className="ai-link" onClick={() => copy(action.command)}>{copied ? 'Copied' : 'Copy'}</button>
          </div>
        )}
        {shown && action?.kind === 'page' && <div className="ai-notice" role="status">The download page is open in a new tab. Install the app there, then come back.</div>}
      </div>
    </>
  );
};

export default InstallRunner;
