import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  backendIsReady,
  getKeystoreStatus,
  recheckKeystore,
  setupKeystore,
  unlockKeystore,
  type KeystoreStatus,
} from '@/api/keystoreAPI';
import './ai/ai-provider.css';

type Phase = { kind: 'checking' } | { kind: 'open' } | { kind: 'opening' } | { kind: 'screen'; status: KeystoreStatus };

const MIN_PASSPHRASE = 8;

/**
 * Sits between "the Moly app runs" and "the app is usable". When the app has no database key yet (no system keychain on
 * this computer) it asks the person to protect the key with a passphrase (recommended) or, knowingly, a key file; when
 * the key is sealed with a passphrase it asks for it. Anything else (a normal start with a keychain, a development
 * backend) passes straight through.
 */
export const KeystoreGate: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [phase, setPhase] = useState<Phase>({ kind: 'checking' });
  const alive = useRef(true);

  const waitUntilOpen = useCallback(async () => {
    setPhase({ kind: 'opening' });
    for (let i = 0; i < 180 && alive.current; i++) {
      if (await backendIsReady()) {
        if (alive.current) setPhase({ kind: 'open' });
        return;
      }
      await new Promise((r) => setTimeout(r, 1000));
    }
    if (alive.current) setPhase({ kind: 'open' }); // let the next screens show the real problem
  }, []);

  const apply = useCallback(
    (status: KeystoreStatus) => {
      if (!alive.current) return;
      if (status.state === 'ready') void waitUntilOpen();
      else setPhase({ kind: 'screen', status });
    },
    [waitUntilOpen],
  );

  useEffect(() => {
    alive.current = true;
    getKeystoreStatus().then((r) => {
      // Not there (development backend, or the app is already past this step): nothing to ask.
      if (!alive.current) return;
      if (!r.ok || !r.data || r.data.state === 'ready') {
        if (r.ok && r.data?.state === 'ready') void waitUntilOpen();
        else setPhase({ kind: 'open' });
      } else setPhase({ kind: 'screen', status: r.data });
    });
    return () => {
      alive.current = false;
    };
  }, [waitUntilOpen]);

  if (phase.kind === 'open') return <>{children}</>;
  if (phase.kind === 'checking' || phase.kind === 'opening') {
    return (
      <div className="ai-card" role="status">
        <h2>{phase.kind === 'opening' ? 'Opening your data...' : 'Starting Moly...'}</h2>
        <p className="ai-muted">This can take a few seconds.</p>
      </div>
    );
  }
  return <KeystoreScreen status={phase.status} onStatus={apply} />;
};

const KeystoreScreen: React.FC<{ status: KeystoreStatus; onStatus: (s: KeystoreStatus) => void }> = ({ status, onStatus }) => {
  const [passphrase, setPassphrase] = useState('');
  const [again, setAgain] = useState('');
  const [insecure, setInsecure] = useState(false);
  const [useFile, setUseFile] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const run = async (fn: () => ReturnType<typeof unlockKeystore>) => {
    setBusy(true);
    setError('');
    const r = await fn();
    setBusy(false);
    if (r.ok && r.data) {
      setPassphrase('');
      setAgain('');
      onStatus(r.data);
    } else setError(r.error || 'That did not work.');
  };

  if (status.state === 'locked') {
    return (
      <form className="ai-card" onSubmit={(e) => { e.preventDefault(); if (passphrase) void run(() => unlockKeystore(passphrase)); }}>
        <h2>Unlock Moly</h2>
        <p>Your data is protected by a passphrase. Enter it to open it.</p>
        <label className="ai-field" htmlFor="ks-unlock">Passphrase</label>
        <input id="ks-unlock" className="ai-input" type="password" autoComplete="current-password" autoFocus value={passphrase} onChange={(e) => setPassphrase(e.target.value)} disabled={busy} />
        {error && <div className="ai-error">{error}</div>}
        <div className="ai-actions">
          <button className="ai-btn primary" type="submit" disabled={busy || !passphrase}>{busy ? 'Checking...' : 'Unlock'}</button>
        </div>
        <p className="ai-muted">Moly cannot recover a forgotten passphrase: the data stays encrypted. If you exported your data earlier, you can import it into a new profile.</p>
      </form>
    );
  }

  if (status.state === 'needs_setup') {
    const tooShort = passphrase.length < MIN_PASSPHRASE;
    const mismatch = passphrase !== again;
    return (
      <form
        className="ai-card"
        onSubmit={(e) => {
          e.preventDefault();
          if (useFile) void run(() => setupKeystore('file', undefined, insecure));
          else if (!tooShort && !mismatch) void run(() => setupKeystore('passphrase', passphrase));
        }}
      >
        <h2>Protect your data</h2>
        <p>
          Moly keeps your conversations in an encrypted database. Its key is normally kept in your system keychain, but this
          computer has none. Choose how to protect the key:
        </p>
        {!useFile ? (
          <>
            <p><strong>A passphrase (recommended).</strong> You type it each time Moly starts. Nobody who copies your files can open them.</p>
            <label className="ai-field" htmlFor="ks-pass">Passphrase (at least {MIN_PASSPHRASE} characters)</label>
            <input id="ks-pass" className="ai-input" type="password" autoComplete="new-password" autoFocus value={passphrase} onChange={(e) => setPassphrase(e.target.value)} disabled={busy} />
            <label className="ai-field" htmlFor="ks-again">Once more</label>
            <input id="ks-again" className="ai-input" type="password" autoComplete="new-password" value={again} onChange={(e) => setAgain(e.target.value)} disabled={busy} />
            {passphrase && tooShort && <div className="ai-muted">Use at least {MIN_PASSPHRASE} characters.</div>}
            {again && mismatch && <div className="ai-error">The two passphrases are different.</div>}
            <p className="ai-muted">Moly cannot recover a forgotten passphrase. Keep it somewhere safe.</p>
          </>
        ) : (
          <>
            <p><strong>A key file (less secure).</strong> The key is saved in a file in your Moly folder. Anyone or any program that can read your files, including backups, can open your data.</p>
            <label className="ai-check">
              <input type="checkbox" checked={insecure} onChange={(e) => setInsecure(e.target.checked)} /> I understand and accept this
            </label>
          </>
        )}
        {error && <div className="ai-error">{error}</div>}
        <div className="ai-actions">
          <button className="ai-btn primary" type="submit" disabled={busy || (useFile ? !insecure : tooShort || mismatch)}>
            {busy ? 'Saving...' : useFile ? 'Use a key file' : 'Protect with this passphrase'}
          </button>
          <button type="button" className="ai-link" onClick={() => { setUseFile(!useFile); setError(''); }}>
            {useFile ? 'Use a passphrase instead' : 'Use a key file instead'}
          </button>
        </div>
        {status.keychainAvailable && (
          <p className="ai-muted">
            A keychain is available now. <button type="button" className="ai-link" onClick={() => void run(() => setupKeystore('keychain'))}>Use it instead</button>
          </p>
        )}
      </form>
    );
  }

  const title =
    status.state === 'keychain_unavailable' ? 'Your keychain is not reachable' : status.state === 'unknown_key' ? 'Moly cannot find the key to your data' : 'Moly cannot open its settings';
  return (
    <div className="ai-card" role="alert">
      <h2>{title}</h2>
      <p>{status.reason || 'Something is wrong with the database key.'}</p>
      {error && <div className="ai-error">{error}</div>}
      {status.state !== 'broken' && (
        <div className="ai-actions">
          <button className="ai-btn primary" type="button" disabled={busy} onClick={() => void run(recheckKeystore)}>{busy ? 'Checking...' : 'Check again'}</button>
        </div>
      )}
    </div>
  );
};

export default KeystoreGate;
