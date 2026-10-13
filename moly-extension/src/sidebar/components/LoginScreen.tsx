import React, { useState } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { useAuthStore } from '@/stores/authStore';
import { Icon } from '@/components/Icon';
import './login-screen.css';
import './about-me.css';

interface LoginScreenProps {
  /** Called with the new session once a login or registration succeeded. */
  onLoginSuccess?: (userId: string, token: string) => void;
}

export const LoginScreen: React.FC<LoginScreenProps> = ({ onLoginSuccess }) => {
  const { login, loading, error } = useAuth();
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [name, setName] = useState('');
  const [emailOrUsername, setEmailOrUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [fieldError, setFieldError] = useState<string | null>(null);

  const notifySuccess = () => {
    const session = useAuthStore.getState().session;
    if (session) onLoginSuccess?.(session.userId, session.sessionId);
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setFieldError(null);

    if (!emailOrUsername || !password) {
      setFieldError('Email or username and password are required');
      return;
    }

    if (await login(emailOrUsername, password)) notifySuccess();
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setFieldError(null);

    if (!name || !emailOrUsername || !password || !confirmPassword) {
      setFieldError('All fields are required');
      return;
    }

    if (password !== confirmPassword) {
      setFieldError('Passwords do not match');
      return;
    }

    if (password.length < 8) {
      setFieldError('Password must be at least 8 characters');
      return;
    }

    if (await login(emailOrUsername, password, name)) notifySuccess();
  };

  return (
    <div className="login-screen">
      <div className="login-card">
        <div className="login-brand">
          <span className="brand-mark" aria-hidden="true" />
          <h1 lang="el">Μώλυ</h1>
        </div>
        <p className="login-tagline">Think before you send.</p>

        <div className="segmented login-tabs" role="tablist">
          <button role="tab" aria-selected={mode === 'login'} className={mode === 'login' ? 'active' : ''} onClick={() => { setMode('login'); setFieldError(null); }}>Log in</button>
          <button role="tab" aria-selected={mode === 'register'} className={mode === 'register' ? 'active' : ''} onClick={() => { setMode('register'); setFieldError(null); }}>Register</button>
        </div>

        {(fieldError || error) && (
          <div className="notice danger" role="alert"><Icon name="alert" /><span>{fieldError || error}</span></div>
        )}

        {mode === 'login' && (
          <form onSubmit={handleLogin} className="login-form">
            <label className="form-label" htmlFor="login-id">Email or username</label>
            <input id="login-id" className="input" type="text" autoComplete="username" value={emailOrUsername} onChange={(e) => setEmailOrUsername(e.target.value)} disabled={loading} />
            <label className="form-label" htmlFor="login-password">Password</label>
            <input id="login-password" className="input" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={loading} />
            <button type="submit" className="btn btn-primary login-submit" disabled={loading}>{loading ? 'Logging in...' : 'Log in'}</button>
          </form>
        )}

        {mode === 'register' && (
          <form onSubmit={handleRegister} className="login-form">
            <label className="form-label" htmlFor="reg-name">Username</label>
            <input id="reg-name" className="input" type="text" autoComplete="username" value={name} onChange={(e) => setName(e.target.value)} disabled={loading} />
            <label className="form-label" htmlFor="reg-email">Email</label>
            <input id="reg-email" className="input" type="email" autoComplete="email" value={emailOrUsername} onChange={(e) => setEmailOrUsername(e.target.value)} disabled={loading} />
            <label className="form-label" htmlFor="reg-password">Password</label>
            <input id="reg-password" className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={loading} />
            <p className="form-hint">At least 8 characters.</p>
            <label className="form-label" htmlFor="reg-confirm">Confirm password</label>
            <input id="reg-confirm" className="input" type="password" autoComplete="new-password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} disabled={loading} />
            <button type="submit" className="btn btn-primary login-submit" disabled={loading}>{loading ? 'Creating account...' : 'Create account'}</button>
          </form>
        )}

        <p className="login-footnote">This sign-in is for a development backend. The installed Moly opens without an account.</p>
      </div>
    </div>
  );
};
