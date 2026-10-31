import React, { useState } from 'react';
import type { Session } from '@/stores/authStore';
import { resetPassword, signIn, type AccountResult } from '@/api/accounts';
import { RecoveryCode } from './RecoveryCode';
import { AuthFrame } from './AuthFrame';
import { Icon } from './Icon';

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
    return (
      <AuthFrame title="Your new recovery code">
        <RecoveryCode code={step.result.recoveryCode} again bare onDone={() => onSession(step.name, step.result.session)} />
      </AuthFrame>
    );
  }

  const Notice = error ? <div className="notice danger" role="alert"><Icon name="alert" /><span>{error}</span></div> : null;

  if (step.kind === 'password') {
    const name = step.name;
    return (
      <AuthFrame title={`Welcome back, ${name}`} lead="Type your password to open your conversations.">
        <form className="login-form" onSubmit={(e) => { e.preventDefault(); void run(async () => onSession(name, await signIn(name, password))); }}>
          {Notice}
          <label className="form-label" htmlFor="profile-password">Password</label>
          <input id="profile-password" className="input" type="password" autoComplete="current-password" autoFocus value={password} onChange={(e) => setPassword(e.target.value)} disabled={busy} />
          <button type="submit" className="btn btn-primary login-submit" disabled={busy || !password}>{busy ? 'Signing in...' : 'Sign in'}</button>
        </form>
        <div className="auth-links">
          <button type="button" className="ai-link" onClick={() => go({ kind: 'reset', name })} disabled={busy}>I forgot my password</button>
          <button type="button" className="ai-link" onClick={() => go({ kind: 'choose' })} disabled={busy}>Another profile</button>
        </div>
      </AuthFrame>
    );
  }

  if (step.kind === 'reset') {
    const name = step.name;
    return (
      <AuthFrame title={`New password for ${name}`} lead="Type the recovery code you were shown when this profile was made, and choose a new password (at least 8 characters).">
        <form className="login-form" onSubmit={(e) => { e.preventDefault(); void run(async () => go({ kind: 'code', name, result: await resetPassword(name, code, password) })); }}>
          {Notice}
          <label className="form-label" htmlFor="profile-code">Recovery code</label>
          <input id="profile-code" className="input" type="text" autoComplete="off" autoFocus placeholder="XXXX-XXXX-XXXX-XXXX" value={code} onChange={(e) => setCode(e.target.value)} disabled={busy} />
          <label className="form-label" htmlFor="profile-newpass">New password</label>
          <input id="profile-newpass" className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={busy} />
          <button type="submit" className="btn btn-primary login-submit" disabled={busy || !code.trim() || password.length < 8}>{busy ? 'Working...' : 'Set the new password'}</button>
        </form>
        <div className="auth-links">
          <button type="button" className="ai-link" onClick={() => go({ kind: 'password', name })} disabled={busy}>Back</button>
        </div>
        <p className="login-footnote">Without the code the profile cannot be opened. Its conversations stay on this computer, encrypted.</p>
      </AuthFrame>
    );
  }

  return (
    <AuthFrame title="Who is using Moly?" lead="Choose your profile.">
      <div className="profile-list">
        {profiles.map((n) => (
          <button key={n} className="profile-choice" data-initial={n.trim().charAt(0).toUpperCase()} onClick={() => go({ kind: 'password', name: n })}>{n}</button>
        ))}
      </div>
    </AuthFrame>
  );
};

export default ProfileChooser;
