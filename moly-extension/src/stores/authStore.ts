import { create } from 'zustand';

export interface Session {
  sessionId: string;
  userId: string;
  expiresAt: number;
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

  // Initialize from localStorage
  initFromStorage: () => void;

  // Check if session is valid
  isAuthenticated: () => boolean;
}

const SESSION_KEY = 'moly_session';

export const useAuthStore = create<AuthState>((set, get) => ({
  session: null,
  loading: false,
  error: null,

  setSession: (session) => {
    set({ session });
    if (session) {
      localStorage.setItem(SESSION_KEY, JSON.stringify(session));
    } else {
      localStorage.removeItem(SESSION_KEY);
    }
  },

  setLoading: (loading) => set({ loading }),

  setError: (error) => set({ error }),

  clearAuth: async () => {
    set({ session: null, error: null, loading: false });
    localStorage.removeItem(SESSION_KEY);

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
    try {
      const saved = localStorage.getItem(SESSION_KEY);
      if (saved) {
        const session = JSON.parse(saved) as Session;
        // Check if not expired
        if (session.expiresAt > Date.now()) {
          set({ session });
        } else {
          // Expired, clear it
          localStorage.removeItem(SESSION_KEY);
        }
      }
    } catch (e) {
      console.warn('[AuthStore] Failed to load session from storage:', e);
    }
  },

  isAuthenticated: () => {
    const { session } = get();
    if (!session) return false;
    // Check if not expired
    return session.expiresAt > Date.now();
  },
}));
