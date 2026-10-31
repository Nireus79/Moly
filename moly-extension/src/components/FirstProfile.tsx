import React, { useState } from 'react';
import type { Session } from '@/stores/authStore';
import { createProfile, type AccountResult } from '@/api/accounts';
import { RecoveryCode } from './RecoveryCode';

/**
 * The first screen of Moly on a computer that has no profile yet: make yours, with a password. Conversations that an earlier version kept
 * on this computer become this profile's (the app does that when the first profile is made). The recovery code is shown once.
 */
export const FirstProfile: React.FC<{ onSession: (name: string, session: Session) => void }> = ({ onSession }) => {
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [made, setMade] = useState<AccountResult | null>(null);

  if (made) return <RecoveryCode code={made.recoveryCode} onDone={() => onSession(name.trim(), made.session)} />;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return setError('Choose a name.');
    if (password.length < 8) return setError('The password needs at least 8 characters.');
    if (password !== confirm) return setError('The two passwords are not the same.');
    setBusy(true);
    setError('');
    try {
      setMade(await createProfile(name, password));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'The profile could not be made.');
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className="ai-card" onSubmit={submit}>
      <h2>Create your profile</h2>
      <p>Make your profile. Your conversations, the people you talk about and your AI key stay on this computer, behind this password. Anyone else who uses this computer makes a profile of their own.</p>
      <label className="form-label" htmlFor="first-name">Your name</label>
      <input id="first-name" className="input" type="text" autoComplete="username" autoFocus value={name} onChange={(e) => setName(e.target.value)} disabled={busy} />
      <label className="form-label" htmlFor="first-password">Password (at least 8 characters)</label>
      <input id="first-password" className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={busy} />
      <label className="form-label" htmlFor="first-confirm">Password again</label>
      <input id="first-confirm" className="input" type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} disabled={busy} />
      {error && <p className="notice danger" role="alert">{error}</p>}
      <div className="ai-actions">
        <button type="submit" className="ai-btn primary" disabled={busy}>{busy ? 'Making your profile...' : 'Make my profile'}</button>
      </div>
      <p className="form-hint">Next you get a recovery code, shown once. It is the only way back in if you forget the password.</p>
    </form>
  );
};

export default FirstProfile;
