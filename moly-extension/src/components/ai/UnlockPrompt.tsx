import React, { useState } from 'react';
import { unlockLLM, type LLMStatus } from '@/api/llmSettingsAPI';
import './ai-provider.css';

interface Props {
  status: LLMStatus;
  onUnlocked: (status: LLMStatus) => void;
  onChangeProvider: () => void;
}

/** Shown when the saved API key is protected by a passphrase and Moly has just started. */
export const UnlockPrompt: React.FC<Props> = ({ status, onUnlocked, onChangeProvider }) => {
  const [passphrase, setPassphrase] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!passphrase) return;
    setBusy(true);
    setError('');
    const r = await unlockLLM(passphrase);
    setBusy(false);
    if (r.ok && r.data) {
      setPassphrase('');
      onUnlocked(r.data);
    } else {
      setError(r.error === 'wrong passphrase' ? 'That passphrase is not right.' : r.error || 'Could not unlock.');
    }
  };

  const provider = status.provider === 'openai' ? 'OpenAI' : 'Claude';
  return (
    <form className="ai-card" onSubmit={submit}>
      <h2>Unlock Moly</h2>
      <p>Your {provider} key is protected by a passphrase. Enter it to start.</p>
      <label className="ai-field" htmlFor="ai-unlock">Passphrase</label>
      <input id="ai-unlock" className="ai-input" type="password" autoComplete="current-password" autoFocus value={passphrase} onChange={(e) => setPassphrase(e.target.value)} disabled={busy} />
      {error && <div className="ai-error">{error}</div>}
      <div className="ai-actions">
        <button className="ai-btn primary" type="submit" disabled={busy || !passphrase}>{busy ? 'Checking...' : 'Unlock'}</button>
      </div>
      <p className="ai-muted">
        Forgot it? <button type="button" className="ai-link" onClick={onChangeProvider}>Set up a provider again</button>. Your messages and contacts are not affected.
      </p>
    </form>
  );
};

export default UnlockPrompt;
