import React, { useEffect, useState } from 'react';
import { useAuthStore, getStoredSession, type Session } from '@/stores/authStore';
import { LoginScreen } from './components';
import Sidebar from './Sidebar';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { ProviderGate } from '@/components/ai/ProviderGate';
import { BackendGate } from '@/components/BackendGate';
import { KeystoreGate } from '@/components/KeystoreGate';
import { getConnection } from '@/api/backendAuth';
import { startLocalSession } from '@/api/localProfile';
import { localSessionNeedsRenewal, renewLocalSession } from '@/api/sessionRenewal';
import '@/components/ai/ai-provider.css';
import './sidebar.css';

/**
 * Under the gates: with the installed app (a backend connection exists) the person never meets a login screen; a local
 * profile starts by itself. Without it (a development backend started by hand) the login screen stays.
 */
const SessionArea: React.FC<{ isAuthenticated: boolean; onLoginSuccess: (userId: string, token: string) => void }> = ({ isAuthenticated, onLoginSuccess }) => {
  const setSession = useAuthStore((state) => state.setSession);
  const installed = getConnection() !== null;
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  // A local session close to its end is renewed quietly while the panel is open.
  useEffect(() => {
    if (isAuthenticated && localSessionNeedsRenewal()) void renewLocalSession();
  }, [isAuthenticated]);

  useEffect(() => {
    if (!installed || isAuthenticated) return;
    let alive = true;
    setFailed(false);
    startLocalSession().then((session) => {
      if (!alive) return;
      if (session) setSession(session);
      else setFailed(true);
    });
    return () => {
      alive = false;
    };
  }, [installed, isAuthenticated, attempt, setSession]);

  // The AI provider belongs to the signed-in user, so it is asked for only once there is a session (the local profile or a login).
  if (isAuthenticated) {
    return (
      <ProviderGate>
        <Sidebar />
      </ProviderGate>
    );
  }
  if (!installed) return <LoginScreen onLoginSuccess={onLoginSuccess} />;
  if (failed) {
    return (
      <div className="ai-card" role="alert">
        <h2>Could not open your profile</h2>
        <p>Moly could not start your local profile. Press the button to try again.</p>
        <div className="ai-actions">
          <button className="ai-btn primary" onClick={() => setAttempt((n) => n + 1)}>Try again</button>
        </div>
      </div>
    );
  }
  return (
    <div className="ai-card" role="status">
      <h2>Opening Moly...</h2>
    </div>
  );
};

/**
 *.1 Sidebar App - Auth-gated chat interface
 * Shows LoginScreen if not authenticated, full Sidebar (with chat + settings) if authenticated
 * Shows the install / start screen (BackendGate) while the Moly app is not running
 */
export const SidebarApp: React.FC = () => {
  const session = useAuthStore((state) => state.session);
  const setSession = useAuthStore((state) => state.setSession);
  const [authLoading, setAuthLoading] = useState(true);

  // Initialize auth from storage on mount
  useEffect(() => {
    // A valid stored session: the local profile, or an account session of this open panel (see authStore)
    const saved = getStoredSession();
    if (saved) setSession(saved);
    setAuthLoading(false);
  }, [setSession]);

  // Show loading while checking auth
  if (authLoading) {
    return (
      <div className="sidebar-app" style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      }}>
        <div style={{ color: 'white', textAlign: 'center' }}>
          <div style={{ fontSize: '24px', marginBottom: '16px' }}>Loading Moly...</div>
        </div>
      </div>
    );
  }

  const isAuthenticated = session !== null && session.expiresAt > Date.now();

  const handleLoginSuccess = (userId: string, token: string) => {
    // Set session in authStore
    const newSession: Session = {
      sessionId: token,
      userId: userId,
      expiresAt: Date.now() + (24 * 60 * 60 * 1000), // 24 hours
    };
    setSession(newSession);
  };

  return (
    <div className="sidebar-app">
      <ErrorBoundary>
        <BackendGate>
          <KeystoreGate>
            <SessionArea isAuthenticated={isAuthenticated} onLoginSuccess={handleLoginSuccess} />
          </KeystoreGate>
        </BackendGate>
      </ErrorBoundary>
    </div>
  );
};
