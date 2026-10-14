/**
 * Keeps the local profile signed in. The local profile has no password, so when its session ends (or is about to) the
 * extension simply asks the app for a new one. Accounts with a password are never renewed here.
 */

import { setSessionRejectedHandler, setSessionRenewer } from './backendAuth';
import { startLocalSession } from './localProfile';
import { useAuthStore } from '@/stores/authStore';

const RENEW_WHEN_LEFT_MS = 3 * 24 * 60 * 60 * 1000; // renew when less than three days remain

let inFlight: Promise<string | null> | null = null;

/** Gets a new local session and stores it; one request at a time. Returns the new token, or null. */
export function renewLocalSession(): Promise<string | null> {
  if (inFlight) return inFlight;
  inFlight = (async () => {
    const current = useAuthStore.getState().session;
    if (!current?.local) return null;
    const next = await startLocalSession();
    if (!next) return null;
    useAuthStore.getState().setSession(next);
    return next.sessionId;
  })().finally(() => {
    inFlight = null;
  });
  return inFlight;
}

/** True when the local session is close to its end. */
export function localSessionNeedsRenewal(now: number = Date.now()): boolean {
  const s = useAuthStore.getState().session;
  return !!s?.local && s.expiresAt - now < RENEW_WHEN_LEFT_MS;
}

/** Registers the renewer with the fetch wrapper and renews at once if the session is close to its end. */
export function installSessionRenewal(): void {
  setSessionRenewer(renewLocalSession);
  setSessionRejectedHandler(() => useAuthStore.getState().setSession(null));
  if (localSessionNeedsRenewal()) void renewLocalSession();
}
