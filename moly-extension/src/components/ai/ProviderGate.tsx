import React, { useCallback, useEffect, useState } from 'react';
import { getLLMStatus, type LLMStatus } from '@/api/llmSettingsAPI';
import { AiProviderPanel } from './AiProviderPanel';
import { UnlockPrompt } from './UnlockPrompt';

/**
 * Shows the children once an AI provider is usable. Before that: the first-run setup (nothing chosen yet) or the
 * passphrase prompt (a protected key, Moly just started). If the backend cannot be asked, the children are shown so the
 * normal "backend not running" screen can explain.
 */
export const ProviderGate: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [status, setStatus] = useState<LLMStatus | null>(null);
  const [asked, setAsked] = useState(false);
  const [reconfigure, setReconfigure] = useState(false);

  const refresh = useCallback(async () => {
    const r = await getLLMStatus();
    setStatus(r.ok && r.data ? r.data : null);
    setAsked(true);
  }, []);

  useEffect(() => {
    refresh();
    const timer = setInterval(refresh, 15000); // the backend may have restarted and locked the key again
    window.addEventListener('focus', refresh);
    return () => {
      clearInterval(timer);
      window.removeEventListener('focus', refresh);
    };
  }, [refresh]);

  if (!asked) return null;
  if (!status || status.fixedByEnvironment) return <>{children}</>;

  if (reconfigure || status.state === 'unconfigured') {
    return (
      <AiProviderPanel
        firstRun={status.state === 'unconfigured' && !status.provider}
        initialStatus={status}
        onDone={(st) => {
          setReconfigure(false);
          setStatus(st);
        }}
      />
    );
  }
  if (status.state === 'locked') {
    return <UnlockPrompt status={status} onUnlocked={setStatus} onChangeProvider={() => setReconfigure(true)} />;
  }
  return <>{children}</>;
};

export default ProviderGate;
