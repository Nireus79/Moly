import React, { useCallback, useEffect, useRef, useState } from 'react';
import { getBackendManager, type BackendStatus as BackendStatusType } from '@/api/backendManager';
import { classifyHostError } from '@/api/hostErrors';
import { InstallMoly } from './InstallMoly';
import BackendStatus from '@/sidebar/components/BackendStatus';
import './ai/ai-provider.css';

type Phase = { kind: 'checking' } | { kind: 'running' } | { kind: 'down'; status: BackendStatusType };

/**
 * Starts the Moly app through the native host and shows the children once the backend answers. If the app is missing or
 * cannot start, it explains what to do instead of showing developer instructions (those remain, folded away, for an
 * unpacked development build). While the children are shown it keeps watching: a stopped backend brings this screen back.
 */
export const BackendGate: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [phase, setPhase] = useState<Phase>({ kind: 'checking' });
  const [checking, setChecking] = useState(false);
  const [isDevBuild, setIsDevBuild] = useState(false);
  const alive = useRef(true);

  const apply = useCallback((status: BackendStatusType) => {
    if (!alive.current) return;
    setPhase(status.running ? { kind: 'running' } : { kind: 'down', status });
  }, []);

  useEffect(() => {
    alive.current = true;
    const manager = getBackendManager();
    manager.onStatusChange(apply);
    manager.initialize().then(apply);
    chrome.management
      ?.getSelf?.()
      .then((self) => alive.current && setIsDevBuild(self.installType === 'development'))
      .catch(() => undefined);
    return () => {
      alive.current = false;
    };
  }, [apply]);

  // While the app is missing, look again every few seconds, so the screen moves on by itself once the install has finished.
  const missing = phase.kind === 'down' && classifyHostError(getBackendManager().getLastHostError() || phase.status.error) === 'not-installed';
  useEffect(() => {
    if (!missing) return;
    const timer = setInterval(async () => {
      const status = await getBackendManager().retry();
      apply(status);
    }, 4000);
    return () => clearInterval(timer);
  }, [missing, apply]);

  const checkAgain = async () => {
    setChecking(true);
    const status = await getBackendManager().retry();
    setChecking(false);
    apply(status);
  };

  if (phase.kind === 'running') return <>{children}</>;
  if (phase.kind === 'checking') {
    return (
      <div className="ai-card" role="status">
        <h2>Starting Moly...</h2>
        <p className="ai-muted">This can take a few seconds the first time.</p>
      </div>
    );
  }

  const detail = getBackendManager().getLastHostError() || phase.status.error;
  let problem = classifyHostError(detail);
  if (problem === 'none') problem = 'unknown';
  return (
    <>
      <InstallMoly problem={problem} detail={detail} checking={checking} onCheckAgain={checkAgain} />
      {isDevBuild && (
        <details className="ai-card">
          <summary>For developers: start the services by hand</summary>
          <BackendStatus />
        </details>
      )}
    </>
  );
};

export default BackendGate;
