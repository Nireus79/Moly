import React, { useEffect, useState } from 'react';
import { useAuthStore, type Session } from '@/stores/authStore';
import { LoginScreen } from './components';
import Sidebar from './Sidebar';
import BackendStatus from './components/BackendStatus';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import type { BackendStatus as BackendStatusType } from '@/api/backendManager';
import './sidebar.css';

/**
 * V2.1 Sidebar App - Auth-gated chat interface
 * Shows LoginScreen if not authenticated, full Sidebar (with chat + settings) if authenticated
 * Shows BackendStatus if backend is unavailable
 */
export const SidebarApp: React.FC = () => {
  const session = useAuthStore((state) => state.session);
  const setSession = useAuthStore((state) => state.setSession);
  const initFromStorage = useAuthStore((state) => state.initFromStorage);
  const [backendStatus, setBackendStatus] = useState<BackendStatusType | null>(null);
  const [authLoading, setAuthLoading] = useState(true);

  // Initialize auth from storage on mount
  useEffect(() => {
    // Direct check of localStorage for valid session
    try {
      const saved = localStorage.getItem('moly_session');
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed.expiresAt > Date.now()) {
          setSession(parsed);
        } else {
          localStorage.removeItem('moly_session');
        }
      }
    } catch (e) {
      console.error('Failed to load session:', e);
    }
    setAuthLoading(false);
  }, [setSession]);

  useEffect(() => {
    // Check backend status periodically
    const checkBackend = async () => {
      const { getBackendManager } = await import('@/api/backendManager');
      const status = await getBackendManager().getStatus();
      setBackendStatus(status);
    };

    checkBackend();
    const interval = setInterval(checkBackend, 5000); // Check every 5 seconds
    return () => clearInterval(interval);
  }, []);

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

  // Show backend unavailable if not running (only after auth check)
  if (backendStatus && !backendStatus.running && isAuthenticated) {
    return (
      <div className="sidebar-app">
        <BackendStatus showDetails={true} />
      </div>
    );
  }

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
        {isAuthenticated ? (
          <Sidebar />
        ) : (
          <LoginScreen onLoginSuccess={handleLoginSuccess} />
        )}
      </ErrorBoundary>
    </div>
  );
};
