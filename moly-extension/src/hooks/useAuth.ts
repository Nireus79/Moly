import { useCallback, useEffect } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { getBackendManager } from '@/api/backendManager';

export interface Session {
  sessionId: string;
  userId: string;
  expiresAt: number;
}

export function useAuth() {
  const session = useAuthStore((state) => state.session);
  const loading = useAuthStore((state) => state.loading);
  const error = useAuthStore((state) => state.error);
  const setSession = useAuthStore((state) => state.setSession);
  const setLoading = useAuthStore((state) => state.setLoading);
  const setError = useAuthStore((state) => state.setError);
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const initFromStorage = useAuthStore((state) => state.initFromStorage);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  // Load session from storage on mount
  useEffect(() => {
    initFromStorage();
  }, [initFromStorage]);

  // Periodic session validation (every 5 minutes)
  useEffect(() => {
    if (!session) return;

    const checkSessionValidity = async () => {
      if (!session) return;
      try {
        const apiBase = getBackendManager().getBackendUrl();
        const response = await fetch(`${apiBase}/api/auth/verify`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${session.sessionId}`,
          },
        });

        if (!response.ok) {
          console.warn('[useAuth] Session validation failed, clearing');
          await clearAuth();
        }
      } catch (err) {
        console.error('[useAuth] Session validation error:', err);
      }
    };

    const interval = setInterval(checkSessionValidity, 5 * 60 * 1000); // 5 minutes
    return () => clearInterval(interval);
  }, [session, clearAuth]);

  // Register with email/password
  const register = useCallback(
    async (name: string, email: string, password: string): Promise<boolean> => {
      setLoading(true);
      setError(null);
      try {
        const apiBase = getBackendManager().getBackendUrl();
        const response = await fetch(`${apiBase}/api/auth/register`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name, email, password }),
        });

        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.error || `Registration failed: ${response.statusText}`);
        }

        const data = await response.json();
        if (!data.token || !data.userId) {
          throw new Error('Invalid response: missing token or userId');
        }

        const newSession: Session = {
          sessionId: data.token,
          userId: data.userId,
          expiresAt: Date.now() + data.expiresIn * 1000, // Convert seconds to ms
        };

        console.log('[useAuth] Registration successful, userId:', data.userId);
        setSession(newSession);
        return true;
      } catch (err) {
        let message = err instanceof Error ? err.message : 'Unknown error';

        // Detect backend connection errors
        if (message.includes('Failed to fetch') || message.includes('ERR_CONNECTION_REFUSED')) {
          message = '❌ Backend server not responding\n\nPlease start the backend:\ncd /home/nireus79/vs_projects/Moly/Moly/moly-go && go run main.go';
        }

        setError(message);
        console.error('[useAuth] Registration error:', message);
        return false;
      } finally {
        setLoading(false);
      }
    },
    [setSession, setLoading, setError]
  );

  // Login with email/username/password (or register if name provided)
  const login = useCallback(
    async (emailOrUsername: string, password: string, name?: string): Promise<boolean> => {
      // If name provided, this is a registration request
      if (name) {
        return register(name, emailOrUsername, password);
      }

      setLoading(true);
      setError(null);
      try {
        const apiBase = getBackendManager().getBackendUrl();
        const response = await fetch(`${apiBase}/api/auth/login`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ emailOrUsername, password }),
        });

        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.error || `Login failed: ${response.statusText}`);
        }

        const data = await response.json();
        if (!data.token || !data.userId) {
          throw new Error('Invalid response: missing token or userId');
        }

        const newSession: Session = {
          sessionId: data.token,
          userId: data.userId,
          expiresAt: Date.now() + data.expiresIn * 1000, // Convert seconds to ms
        };

        console.log('[useAuth] Login successful, userId:', data.userId);
        setSession(newSession);
        return true;
      } catch (err) {
        let message = err instanceof Error ? err.message : 'Unknown error';

        // Detect backend connection errors
        if (message.includes('Failed to fetch') || message.includes('ERR_CONNECTION_REFUSED')) {
          message = '❌ Backend server not responding\n\nPlease start the backend:\ncd /home/nireus79/vs_projects/Moly/Moly/moly-go && go run main.go';
        }

        setError(message);
        console.error('[useAuth] Login error:', message);
        return false;
      } finally {
        setLoading(false);
      }
    },
    [setSession, setLoading, setError, register]
  );

  // Logout
  const logout = useCallback(async () => {
    if (!session) return;

    try {
      const apiBase = getBackendManager().getBackendUrl();
      await fetch(`${apiBase}/api/auth/logout`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${session.sessionId}`,
        },
      });
    } catch (err) {
      console.error('[useAuth] Logout error:', err);
    } finally {
      await clearAuth();
    }
  }, [session, clearAuth]);

  return {
    session,
    loading,
    error,
    register,
    login,
    logout,
    clearAuth,
    isAuthenticated: isAuthenticated(),
  };
}
