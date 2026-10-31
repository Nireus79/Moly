import React, { useState } from 'react';
import type { Session } from '@/stores/authStore';
import { resetPassword, signIn, type AccountResult } from '@/api/accounts';
import { RecoveryCode } from './RecoveryCode';

type Step = { kind: 'choose' } | { kind: 'password'; name: string } | { kind: 'reset'; name: string } | { kind: 'code'; name: string; result: AccountResult };

/**
 * Shown on a computer every time Moly starts: who is using Moly? Every profile asks for its password; a forgotten one is reset with its recovery code.
 */
export const ProfileChooser: React.FC<{
  profiles: string[];
  preferred?: string;
  onSession: (name: string, session: Session) => void;
}> = ({ profiles, preferred, onSession }) => {
  const [step, setStep] = useState<Step>(preferred && profiles.includes(preferred) ? { kind: 'password', name: preferred } : { kind: 'choose' });
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const go = (next: Step) => {
    setStep(next);
    setPassword('');
    setCode('');
    setError('');
  };

  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    setError('');
    try {
      await fn();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Something went wrong.');
    } finally {
      setBusy(false);
    }
  };

  if (step.kind === 'code') {
    return <RecoveryCode code={step.result.recoveryCode} again onDone={() => onSession(step.name, step.result.session)} />;
  }

  if (step.kind === 'password') {
    const name = step.name;
    return (
      <form className="ai-card" onSubmit={(e) => { e.preventDefault(); void run(async () => onSession(name, await signIn(name, password))); }}>
        <h2>Welcome back, {name}</h2>
        <label className="form-label" htmlFor="profile-password">Password</label>
        <input id="profile-password" className="input" type="password" autoComplete="current-password" autoFocus value={password} onChange={(e) => setPassword(e.target.value)} disabled={busy} />
        {error && <p className="notice danger" role="alert">{error}</p>}
        <div className="ai-actions">
          <button type="submit" className="ai-btn primary" disabled={busy || !password}>{busy ? 'Signing in...' : 'Sign in'}</button>
          <button type="button" className="ai-btn" onClick={() => go({ kind: 'reset', name })} disabled={busy}>I forgot my password</button>
          <button type="button" className="ai-btn" onClick={() => go({ kind: 'choose' })} disabled={busy}>Another profile</button>
        </div>
      </form>
    );
  }

  if (step.kind === 'reset') {
    const name = step.name;
    return (
      <form className="ai-card" onSubmit={(e) => { e.preventDefault(); void run(async () => go({ kind: 'code', name, result: await resetPassword(name, code, password) })); }}>
        <h2>Set a new password for {name}</h2>
        <p>Type the recovery code you were shown when this profile was made, and choose a new password (at least 8 characters).</p>
        <label className="form-label" htmlFor="profile-code">Recovery code</label>
        <input id="profile-code" className="input" type="text" autoComplete="off" autoFocus placeholder="XXXX-XXXX-XXXX-XXXX" value={code} onChange={(e) => setCode(e.target.value)} disabled={busy} />
        <label className="form-label" htmlFor="profile-newpass">New password</label>
        <input id="profile-newpass" className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={busy} />
        {error && <p className="notice danger" role="alert">{error}</p>}
        <div className="ai-actions">
          <button type="submit" className="ai-btn primary" disabled={busy || !code.trim() || password.length < 8}>{busy ? 'Working...' : 'Set the new password'}</button>
          <button type="button" className="ai-btn" onClick={() => go({ kind: 'password', name })} disabled={busy}>Back</button>
        </div>
        <p className="form-hint">Without the code the profile cannot be opened. Its conversations stay on this computer, encrypted.</p>
      </form>
    );
  }

  return (
    <div className="ai-card">
      <h2>Who is using Moly?</h2>
      <div className="ai-actions" style={{ flexDirection: 'column', alignItems: 'stretch' }}>
        {profiles.map((n) => (
          <button key={n} className="ai-btn" onClick={() => go({ kind: 'password', name: n })}>{n}</button>
        ))}
      </div>
    </div>
  );
};

export default ProfileChooser;
