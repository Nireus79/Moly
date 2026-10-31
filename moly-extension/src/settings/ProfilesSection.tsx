import React, { useEffect, useState } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { useAuth } from '@/hooks/useAuth';
import { appLacksProfiles, createProfile, lastProfile, rememberProfile, type AccountResult } from '@/api/accounts';
import { RecoveryCode } from '@/components/RecoveryCode';
import { InstallRunner } from '@/components/InstallRunner';

/**
 * Profiles on this computer (not on the phone: a phone has one owner). Every profile has a name and a password; another person
 * can make a profile here. Each profile has its own conversations, people, settings and AI key.
 */
export const ProfilesSection: React.FC = () => {
  const local = useAuthStore((s) => s.session?.local === true);
  const { logout } = useAuth();
  const [step, setStep] = useState<'idle' | 'form' | 'code'>('idle');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [made, setMade] = useState<AccountResult | null>(null);
  const [oldApp, setOldApp] = useState(false);

  useEffect(() => {
    void appLacksProfiles().then(setOldApp);
  }, [step]);

  const who = local ? 'You' : lastProfile() || 'a profile with a password';

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return setError('Choose a name.');
    if (password.length < 8) return setError('The password needs at least 8 characters.');
    if (password !== confirm) return setError('The two passwords are not the same.');
    setBusy(true);
    setError('');
    try {
      setMade(await createProfile(name, password));
      setStep('code');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'The profile could not be made.');
    } finally {
      setBusy(false);
    }
  };

  // The new profile becomes the one in use. The old one is left first, so nothing of it stays on the screen or in the browser.
  const enter = () => {
    if (!made) return;
    const store = useAuthStore.getState();
    store.setSession(null);
    rememberProfile(name.trim());
    store.setSession(made.session);
  };

  const switchProfile = async () => {
    rememberProfile('');
    await logout();
  };

  if (step === 'code' && made) return <RecoveryCode code={made.recoveryCode} onDone={enter} />;

  return (
    <section className="settings-section">
      <h2>Profiles</h2>
      <p className="section-description">
        You are using: <strong>{who}</strong>. Each profile keeps its own conversations, people, settings and AI key on this computer. Moly asks for a password every time it
        starts; make another profile if someone else uses Moly here.
      </p>
      {oldApp ? (
        <>
          <p className="notice danger" role="alert">The Moly app on this computer is older than this extension and cannot make profiles. Install the newest Moly app, then come back here.</p>
          <p className="section-description">Press the button: it downloads the installer for this computer and opens it (on Linux and Mac without an installer, it shows one line to paste in a terminal). Follow the installer, then reopen this page. Your conversations stay.</p>
          <div className="button-row"><InstallRunner label="Download and install the newest Moly app" /></div>
        </>
      ) : step === 'form' ? (
        <form onSubmit={submit}>
          <div className="form-group">
            <label className="form-label" htmlFor="new-profile-name">Name</label>
            <input id="new-profile-name" className="input" type="text" autoComplete="off" value={name} onChange={(e) => setName(e.target.value)} disabled={busy} />
          </div>
          <div className="form-group">
            <label className="form-label" htmlFor="new-profile-password">Password</label>
            <input id="new-profile-password" className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={busy} />
            <p className="form-hint">At least 8 characters. Moly shows a recovery code next; keep it, it is the only way back in after a forgotten password.</p>
          </div>
          <div className="form-group">
            <label className="form-label" htmlFor="new-profile-confirm">Password again</label>
            <input id="new-profile-confirm" className="input" type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} disabled={busy} />
          </div>
          {error && <p className="notice danger" role="alert">{error}</p>}
          <div className="button-row">
            <button type="submit" className="btn btn-primary" disabled={busy}>{busy ? 'Making the profile...' : 'Make the profile'}</button>
            <button type="button" className="btn" onClick={() => { setStep('idle'); setError(''); }} disabled={busy}>Cancel</button>
          </div>
        </form>
      ) : (
        <div className="button-row">
          <button className="btn" onClick={() => { setName(''); setPassword(''); setConfirm(''); setError(''); setStep('form'); }}>Make another profile</button>
          <button className="btn" onClick={switchProfile}>Sign out</button>
        </div>
      )}
    </section>
  );
};

export default ProfilesSection;
