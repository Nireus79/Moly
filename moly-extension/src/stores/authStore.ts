import { create } from 'zustand';
import { isPhoneApp } from '@/utils/platform';

export interface Session {
  sessionId: string;
  userId: string;
  expiresAt: number;
  /** The local profile of an installed Moly: no email, no password. */
  local?: boolean;
}

interface AuthState {
  session: Session | null;
  loading: boolean;
  error: string | null;

  // Actions
  setSession: (session: Session | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  clearAuth: () => void;

  // Initialize from the stored session
  initFromStorage: () => void;

  // Check if session is valid
  isAuthenticated: () => boolean;
}

const SESSION_KEY = 'moly_session';

/**
 * Where a session is kept. The local profile of the phone app has no password (a phone has one owner) and is shared by every page of the
 * extension, so it lives in localStorage. On a computer every profile has a password and a session lives in sessionStorage. An account session (a development backend with a login) lives in
 * sessionStorage only: the browser discards it when the panel closes, so reopening Moly asks for the login again.
 */
function readStored(): Session | null {
  try {
    const own = sessionStorage.getItem(SESSION_KEY);
    if (own) return JSON.parse(own) as Session;
    const shared = localStorage.getItem(SESSION_KEY);
    if (shared) {
      const session = JSON.parse(shared) as Session;
      if (session.local === true && isPhoneApp()) return session;
      localStorage.removeItem(SESSION_KEY); // an account session saved by an older version, or a passwordless local one on a computer, must not outlive the panel
    }
  } catch {
    /* storage unavailable or damaged: no session */
  }
  return null;
}

function writeStored(session: Session): void {
  const text = JSON.stringify(session);
  if (session.local === true) {
    localStorage.setItem(SESSION_KEY, text);
    sessionStorage.removeItem(SESSION_KEY);
  } else {
    sessionStorage.setItem(SESSION_KEY, text);
    localStorage.removeItem(SESSION_KEY);
  }
}

function removeStored(): void {
  try {
    sessionStorage.removeItem(SESSION_KEY);
    localStorage.removeItem(SESSION_KEY);
  } catch {
    /* storage unavailable */
  }
}

/** The stored session, if it exists and has not expired (expired ones are removed). */
export function getStoredSession(): Session | null {
  const session = readStored();
  if (!session) return null;
  if (!session.sessionId || !session.expiresAt || session.expiresAt <= Date.now()) {
    removeStored();
    return null;
  }
  return session;
}

export function getSessionToken(): string | null {
  return getStoredSession()?.sessionId ?? null;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  session: null,
  loading: false,
  error: null,

  setSession: (session) => {
    set({ session });
    if (session) {
      writeStored(session);
    } else {
      removeStored();
    // The cached About Me profile and the stored error reports belong to the user who is leaving: another login on this browser must not find them.
    localStorage.removeItem('moly_about_me_profile');
    localStorage.removeItem('moly_error_logs');
    }
  },

  setLoading: (loading) => set({ loading }),

  setError: (error) => set({ error }),

  clearAuth: async () => {
    set({ session: null, error: null, loading: false });
    removeStored();

    // Clear all user-specific data from chrome storage
    try {
      await chrome.storage.local.remove([
        'conversations',
        'contacts',
        'meProfile',
        'chatHistory',
        'aboutMe'
      ]);
      console.error('[AuthStore] ✓ Cleared all user data on logout');
    } catch (err) {
      console.error('[AuthStore] Failed to clear user data:', err);
    }
  },

  initFromStorage: () => {
    const session = getStoredSession();
    if (session) set({ session });
  },

  isAuthenticated: () => {
    const { session } = get();
    if (!session) return false;
    // Check if not expired
    return session.expiresAt > Date.now();
  },
}));

// The local profile's session is shared by every page of the extension (localStorage). When another page removes it (the profile
// was deleted, or the user signed out there), this page forgets it too instead of keeping a session the app no longer has.
if (typeof window !== 'undefined' && typeof window.addEventListener === 'function') {
  window.addEventListener('storage', (event) => {
    try {
      if (event.storageArea !== localStorage) return;
      if (event.key !== null && event.key !== SESSION_KEY) return;
      if (localStorage.getItem(SESSION_KEY)) return;
      if (useAuthStore.getState().session?.local === true) useAuthStore.setState({ session: null });
    } catch {
      /* storage unavailable */
    }
  });
}
