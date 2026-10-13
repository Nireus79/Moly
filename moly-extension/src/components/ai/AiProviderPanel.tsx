import React, { useEffect, useState } from 'react';
import {
  deleteLLMKey,
  getLLMStatus,
  listOllamaModels,
  lockLLM,
  saveLLMSettings,
  testLLMSettings,
  type LLMStatus,
  type ProviderId,
} from '@/api/llmSettingsAPI';
import {
  CLAUDE_MODELS,
  OPENAI_MODELS,
  PROVIDER_COMPANY,
  PROVIDER_NAMES,
  buildSaveBody,
  buildTestBody,
  emptyForm,
  isCloud,
  passphraseStrength,
  validateForm,
  type SetupForm,
} from '@/api/llmSetupLogic';
import './ai-provider.css';

interface Props {
  /** First run: shows the welcome text and the notice that Moly is not a crisis service. */
  firstRun?: boolean;
  /** Called after the settings were saved and the provider is usable. */
  onDone?: (status: LLMStatus) => void;
  /** Initial status, if the caller already loaded it. */
  initialStatus?: LLMStatus | null;
}

const STATE_TEXT = { ready: 'Ready', locked: 'Locked', unconfigured: 'Not set up' } as const;

export const AiProviderPanel: React.FC<Props> = ({ firstRun = false, onDone, initialStatus = null }) => {
  const [status, setStatus] = useState<LLMStatus | null>(initialStatus);
  const [form, setForm] = useState<SetupForm>(emptyForm('ollama'));
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null);

  const update = (patch: Partial<SetupForm>) => setForm((f) => ({ ...f, ...patch }));

  const applyStatus = (st: LLMStatus, provider?: ProviderId) => {
    setStatus(st);
    const p = (provider || (st.provider as ProviderId) || 'ollama') as ProviderId;
    const key = isCloud(p) ? st.keys?.[p] : undefined;
    setForm((f) => ({
      ...f,
      provider: p,
      model: p === (st.provider as ProviderId) ? st.model || '' : '',
      endpoint: st.ollamaEndpoint || f.endpoint,
      hasStoredKey: !!key?.stored,
      storedKeySealed: !!key?.sealed,
      apiKey: '',
      passphrase: '',
      passphrase2: '',
    }));
  };

  useEffect(() => {
    let alive = true;
    getLLMStatus().then((r) => {
      if (alive && r.ok && r.data) applyStatus(r.data);
    });
    return () => {
      alive = false;
    };
  }, []);

  useEffect(() => {
    if (form.provider !== 'ollama') return;
    let alive = true;
    listOllamaModels(form.endpoint).then((r) => {
      if (!alive) return;
      setOllamaModels(r.models);
      if (r.models.length > 0 && !r.models.includes(form.model) && !form.model) update({ model: r.models[0] });
    });
    return () => {
      alive = false;
    };
  }, [form.provider, form.endpoint]);

  const choose = (provider: ProviderId) => {
    setMessage(null);
    if (status) applyStatus(status, provider);
    else update({ provider, model: '' });
    update({ provider });
  };

  const problems = validateForm(form);

  const runTest = async () => {
    setBusy(true);
    setMessage(null);
    const r = await testLLMSettings(buildTestBody(form));
    setMessage(r.ok ? { kind: 'ok', text: 'The connection works.' } : { kind: 'error', text: r.error || 'The test failed.' });
    setBusy(false);
  };

  const save = async () => {
    if (problems.length) return;
    setBusy(true);
    setMessage(null);
    const r = await saveLLMSettings(buildSaveBody(form));
    setBusy(false);
    if (!r.ok || !r.data) {
      setMessage({ kind: 'error', text: r.error || 'Saving failed.' });
      return;
    }
    applyStatus(r.data);
    setMessage({ kind: 'ok', text: 'Saved.' });
    if (r.data.state === 'ready') onDone?.(r.data);
  };

  const removeKey = async () => {
    if (!isCloud(form.provider)) return;
    if (!window.confirm('Remove the saved API key?')) return;
    setBusy(true);
    const r = await deleteLLMKey(form.provider);
    setBusy(false);
    if (r.ok && r.data) {
      applyStatus(r.data, form.provider);
      setMessage({ kind: 'ok', text: 'The key was removed.' });
    } else setMessage({ kind: 'error', text: r.error || 'Could not remove the key.' });
  };

  const lockNow = async () => {
    const r = await lockLLM();
    if (r.ok && r.data) {
      setStatus(r.data);
      setMessage({ kind: 'ok', text: 'Locked. You will be asked for the passphrase again.' });
    }
  };

  if (status?.fixedByEnvironment) {
    return (
      <div className="ai-card">
        <h2>AI provider</h2>
        <p className="ai-muted">The provider is fixed by the LLM_PROVIDER setting of this installation (development). Active: {status.active}.</p>
      </div>
    );
  }

  const cloud = isCloud(form.provider);
  const models = form.provider === 'claude' ? CLAUDE_MODELS : form.provider === 'openai' ? OPENAI_MODELS : [];
  const strength = passphraseStrength(form.passphrase);

  return (
    <div className="ai-card">
      {firstRun ? (
        <>
          <h2>Welcome to Moly</h2>
          <p>Moly asks the question that matters before you send a message. First, choose where its thinking happens.</p>
          <div className="ai-notice">Moly is a writing coach. It is not a crisis, medical or legal service, and it gives no emergency advice.</div>
        </>
      ) : (
        <h2>
          AI provider {status && <span className={`ai-state ${status.state}`}>{STATE_TEXT[status.state]}</span>}
        </h2>
      )}

      <div className="ai-tabs" role="tablist">
        {(['ollama', 'claude', 'openai'] as ProviderId[]).map((p) => (
          <button key={p} role="tab" aria-selected={form.provider === p} className={`ai-tab ${form.provider === p ? 'active' : ''}`} onClick={() => choose(p)} disabled={busy}>
            {PROVIDER_NAMES[p]}
          </button>
        ))}
      </div>

      {form.provider === 'ollama' && (
        <>
          <p className="ai-muted">Everything stays on this computer. Needs Ollama installed and a model downloaded.</p>
          <label className="ai-field" htmlFor="ai-endpoint">Ollama address</label>
          <input id="ai-endpoint" className="ai-input" value={form.endpoint} onChange={(e) => update({ endpoint: e.target.value })} disabled={busy} />
          <label className="ai-field" htmlFor="ai-ollama-model">Model</label>
          {ollamaModels.length > 0 ? (
            <select id="ai-ollama-model" className="ai-input" value={form.model} onChange={(e) => update({ model: e.target.value })} disabled={busy}>
              {ollamaModels.map((m) => (
                <option key={m} value={m}>{m}</option>
              ))}
            </select>
          ) : (
            <p className="ai-error">No models found. Start Ollama and download one (for example: ollama pull mistral), then reopen this page.</p>
          )}
        </>
      )}

      {cloud && (
        <>
          <p className="ai-muted">
            Uses your own {PROVIDER_COMPANY[form.provider as 'claude' | 'openai']} account. You pay the provider for what you use; Moly never sees your key on any server.
          </p>
          {form.hasStoredKey && (
            <div className="ai-notice">
              A key is saved ({form.storedKeySealed ? 'protected by your passphrase' : 'not protected by a passphrase'}).
              Enter a new one below to replace it{!form.storedKeySealed ? ', or choose a passphrase to protect the saved one' : ''}.{' '}
              <button className="ai-link" onClick={removeKey} disabled={busy}>Remove the key</button>
            </div>
          )}
          <label className="ai-field" htmlFor="ai-key">API key</label>
          <input id="ai-key" className="ai-input" type="password" autoComplete="off" spellCheck={false} placeholder={form.hasStoredKey ? '(saved: leave empty to keep it)' : 'sk-...'} value={form.apiKey} onChange={(e) => update({ apiKey: e.target.value.trim() })} disabled={busy} />
          <label className="ai-field" htmlFor="ai-model">Model</label>
          <input id="ai-model" className="ai-input" list="ai-models" value={form.model} placeholder="default" onChange={(e) => update({ model: e.target.value })} disabled={busy} />
          <datalist id="ai-models">
            {models.map((m) => (
              <option key={m.id} value={m.id}>{m.label}</option>
            ))}
          </datalist>

          <label className="ai-check">
            <input type="checkbox" checked={form.usePassphrase} onChange={(e) => update({ usePassphrase: e.target.checked })} disabled={busy} />
            <span><strong>Protect the key with a passphrase (recommended).</strong> Without it, a program running on your computer that can read your keychain could read the key. With it, the key can only be opened with the passphrase, which Moly never stores. You will enter it once each time Moly starts.</span>
          </label>
          {form.usePassphrase && (!form.hasStoredKey || !form.storedKeySealed || form.apiKey) && (
            <>
              <label className="ai-field" htmlFor="ai-pass">Passphrase</label>
              <input id="ai-pass" className="ai-input" type="password" autoComplete="new-password" value={form.passphrase} onChange={(e) => update({ passphrase: e.target.value })} disabled={busy} />
              <p className="ai-muted">{strength === 'short' ? 'Too short: at least 8 characters.' : strength === 'ok' ? 'Acceptable. A longer phrase is stronger.' : strength === 'good' ? 'Good.' : ' '}</p>
              <label className="ai-field" htmlFor="ai-pass2">Repeat the passphrase</label>
              <input id="ai-pass2" className="ai-input" type="password" autoComplete="new-password" value={form.passphrase2} onChange={(e) => update({ passphrase2: e.target.value })} disabled={busy} />
              <div className="ai-warning">If you forget the passphrase it cannot be recovered. You would enter the API key again.</div>
            </>
          )}
          {!form.usePassphrase && <div className="ai-warning">The key will be saved without a passphrase. It is still encrypted on disk, but not against malware that can use your keychain.</div>}

          <label className="ai-check">
            <input type="checkbox" checked={form.consent} onChange={(e) => update({ consent: e.target.checked })} disabled={busy} />
            <span>I understand that the text of my messages is sent to {PROVIDER_COMPANY[form.provider as 'claude' | 'openai']} to write the replies, under {PROVIDER_COMPANY[form.provider as 'claude' | 'openai']}'s own terms.</span>
          </label>
          <p className="ai-muted">Tip: create a separate API key for Moly and set a spending limit at {PROVIDER_COMPANY[form.provider as 'claude' | 'openai']}.</p>
        </>
      )}

      {problems.length > 0 && (form.apiKey || form.passphrase || form.consent) && (
        <div className="ai-error">{problems.join(' ')}</div>
      )}
      {message && <div className={message.kind === 'ok' ? 'ai-ok' : 'ai-error'}>{message.text}</div>}

      <div className="ai-actions">
        <button className="ai-btn secondary" onClick={runTest} disabled={busy || (cloud ? !form.apiKey : problems.length > 0)} title={cloud && !form.apiKey ? 'Enter a key to test it' : undefined}>
          {busy ? 'Working...' : 'Test connection'}
        </button>
        <button className="ai-btn primary" onClick={save} disabled={busy || problems.length > 0}>
          Save
        </button>
        {status?.state === 'ready' && status.keys?.[status.provider]?.sealed && (
          <button className="ai-btn secondary" onClick={lockNow} disabled={busy}>Lock now</button>
        )}
      </div>
    </div>
  );
};

export default AiProviderPanel;
