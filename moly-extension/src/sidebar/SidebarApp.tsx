import React, { useEffect, useState } from 'react';
import { useAuthStore, restoreSharedSession } from '@/stores/authStore';
import Sidebar from './Sidebar';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { ProviderGate } from '@/components/ai/ProviderGate';
import { BackendGate } from '@/components/BackendGate';
import { KeystoreGate } from '@/components/KeystoreGate';
import { ensureBackendConnection, getConnection } from '@/api/backendAuth';
import { startLocalSession } from '@/api/localProfile';
import { appLacksProfiles, lastProfile, listProfiles, rememberProfile } from '@/api/accounts';
import { isPhoneApp } from '@/utils/platform';
import { ProfileChooser } from '@/components/ProfileChooser';
import { FirstProfile } from '@/components/FirstProfile';
import { InstallRunner } from '@/components/InstallRunner';
import { localSessionNeedsRenewal, renewLocalSession } from '@/api/sessionRenewal';
import '@/components/ai/ai-provider.css';
import './sidebar.css';

/**
 * Under the gates, on a computer: Moly always asks who is using it (profiles with a password; the first start makes the first profile).
 * The phone has one owner and opens its local profile by itself. There is no other login: a development backend started by hand uses the
 * same profile screens, and a connection that is missing is asked for again (it is lost when the browser's session area is emptied, for example).
 */
const SessionArea: React.FC<{ isAuthenticated: boolean }> = ({ isAuthenticated }) => {
  const setSession = useAuthStore((state) => state.setSession);
  const userId = useAuthStore((state) => state.session?.userId);
  // The connection to the app is published by the background worker; ask for it once if this page does not have it.
  const [asked, setAsked] = useState(getConnection() !== null);
  useEffect(() => {
    if (asked) return;
    let alive = true;
    ensureBackendConnection().then(() => {
      if (!alive) return;
      setAsked(true);
    });
    return () => {
      alive = false;
    };
  }, [asked]);
  const installed = asked; // after asking, a page without a connection talks to a development backend, which needs no token
  const phone = isPhoneApp();
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  // The names of the profiles that have a password; null while asking. The phone has one owner and no such profiles.
  const [profiles, setProfiles] = useState<string[] | null>(null);
  const [oldApp, setOldApp] = useState(false); // an app before 0.1.2 cannot make profiles

  // A local session close to its end is renewed quietly while the panel is open.
  useEffect(() => {
    if (isAuthenticated && localSessionNeedsRenewal()) void renewLocalSession();
  }, [isAuthenticated]);

  useEffect(() => {
    if (!installed || isAuthenticated) return;
    if (phone) {
      setProfiles([]);
      return;
    }
    let alive = true;
    setProfiles(null);
    Promise.all([appLacksProfiles(), listProfiles()]).then(([old, names]) => {
      if (!alive) return;
      setOldApp(old);
      setProfiles(names);
    });
    return () => {
      alive = false;
    };
  }, [installed, isAuthenticated, phone, attempt]);

  // On a computer every profile has a password and Moly always asks: the chooser when profiles exist, the first profile when none does.
  // Only the phone (one owner, a screen lock) opens its local profile by itself.
  const last = lastProfile();
  const choosing = installed && !phone && !isAuthenticated && profiles !== null && profiles.length > 0;
  const firstRun = installed && !phone && !isAuthenticated && profiles !== null && profiles.length === 0 && !oldApp;
  const autoLocal = installed && phone && !isAuthenticated;

  useEffect(() => {
    if (!autoLocal) return;
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
  }, [autoLocal, attempt, setSession]);

  // The AI provider belongs to the signed-in user, so it is asked for only once there is a session (the local profile or a login).
  // The key makes the whole panel start over when the profile changes, so nothing of the other profile stays in its state.
  if (isAuthenticated) {
    return (
      <ProviderGate key={userId}>
        <Sidebar key={userId} />
      </ProviderGate>
    );
  }
  if (choosing) {
    return (
      <ProfileChooser
        profiles={profiles ?? []}
        preferred={last}
        onSession={(name, session) => {
          rememberProfile(name);
          setSession(session);
        }}
      />
    );
  }
  if (firstRun) {
    return (
      <FirstProfile
        onSession={(name, session) => {
          rememberProfile(name);
          setSession(session);
        }}
      />
    );
  }
  if (installed && !phone && !isAuthenticated && oldApp) {
    return (
      <div className="ai-card" role="alert">
        <h2>Update the Moly app</h2>
        <p>The Moly app on this computer is older than this extension and has no sign-in. Install the newest Moly app; your conversations stay.</p>
        <div className="ai-actions">
          <InstallRunner label="Download and install the newest Moly app" />
          <button className="ai-btn secondary" onClick={() => setAttempt((n) => n + 1)}>Check again</button>
        </div>
      </div>
    );
  }
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
    // (a panel opened after Settings, or the other way round, takes the session the other page holds)
    void restoreSharedSession().then((saved) => {
      if (saved) setSession(saved);
      setAuthLoading(false);
    });
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

  return (
    <div className="sidebar-app">
      <ErrorBoundary>
        <BackendGate>
          <KeystoreGate>
            <SessionArea isAuthenticated={isAuthenticated} />
          </KeystoreGate>
        </BackendGate>
      </ErrorBoundary>
    </div>
  );
};
