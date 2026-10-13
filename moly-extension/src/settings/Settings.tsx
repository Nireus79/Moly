/**
 * Settings Component - Simplified
 * Configure LLM providers, chat mode, and communication context
 */

import React, { useState, useEffect } from 'react';
import { useSettingsStore } from '@/stores/settingsStore';
import { useAuthStore } from '@/stores/authStore';
import { AiProviderPanel } from '@/components/ai/AiProviderPanel';
import { BACKEND_CONFIG } from '@/api/backendConfig';
import './settings.css';

interface SettingsProps {
  onClose?: () => void;
}

export const Settings: React.FC<SettingsProps> = ({ onClose }) => {
  const { loadSettings } = useSettingsStore();
  const isLocalProfile = useAuthStore((state) => state.session?.local === true);

  const [testMessage, setTestMessage] = useState('');
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deletePassword, setDeletePassword] = useState('');
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [deleteError, setDeleteError] = useState('');
  const [exporting, setExporting] = useState(false);
  const [exportMessage, setExportMessage] = useState('');

  // Preferences state
  const [communicationStyle, setCommunicationStyle] = useState('');
  const [tonePreference, setTonePreference] = useState('');
  const [coreValues, setCoreValues] = useState('');
  const [goals, setGoals] = useState('');
  const [prefsLoading, setPrefsLoading] = useState(false);
  const [prefsMessage, setPrefsMessage] = useState('');

  useEffect(() => {
    loadSettings();
    loadPreferences();
  }, [loadSettings]);

  const loadPreferences = async () => {
    try {
      const session = useAuthStore.getState().session;
      if (!session || !session.sessionId) return;

      const aboutMeUrl = `${BACKEND_CONFIG.baseUrl}${BACKEND_CONFIG.endpoints.context.aboutMe}`;
      const response = await fetch(aboutMeUrl, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${session.sessionId}`,
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        const data = await response.json();
        if (data.profile) {
          setCommunicationStyle(data.profile.communicationStyle || '');
          setTonePreference(data.profile.tonePreference || '');
          setCoreValues((data.profile.coreValues || []).join(', '));
          setGoals((data.profile.goals || []).join(', '));
        }
      }
    } catch (error) {
      console.error('Failed to load preferences:', error);
    }
  };

  const handleSavePreferences = async () => {
    setPrefsLoading(true);
    setPrefsMessage('');
    try {
      const session = useAuthStore.getState().session;
      if (!session || !session.sessionId) {
        setPrefsMessage('Not authenticated. Please login first.');
        setPrefsLoading(false);
        return;
      }
      const token = session.sessionId;

      const aboutMeUrl = `${BACKEND_CONFIG.baseUrl}${BACKEND_CONFIG.endpoints.context.aboutMe}`;
      const response = await fetch(aboutMeUrl, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          communicationStyle: communicationStyle.trim(),
          tonePreference: tonePreference.trim(),
          coreValues: coreValues.split(',').map(v => v.trim()).filter(v => v),
          goals: goals.split(',').map(v => v.trim()).filter(v => v),
        }),
      });

      if (response.ok) {
        setPrefsMessage('✓ Preferences saved successfully');
        setTimeout(() => setPrefsMessage(''), 3000);
      } else {
        const error = await response.text();
        setPrefsMessage(`Error: ${error}`);
      }
    } catch (error) {
      setPrefsMessage(`Error: ${error instanceof Error ? error.message : 'Failed to save'}`);
    } finally {
      setPrefsLoading(false);
    }
  };

  const handleClearAllSettings = async () => {
    if (confirm('Are you sure you want to clear all settings? This cannot be undone.')) {
      try {
        // Everything the browser kept for this profile goes: the extension storage, and the page storage (the cached About Me profile, the session, error reports).
      try { localStorage.clear(); sessionStorage.clear(); } catch { /* storage may be unavailable */ }
      await chrome.storage.local.clear();
        await loadSettings();
        setTestMessage('All settings cleared');
        setTimeout(() => setTestMessage(''), 2000);
      } catch (error) {
        console.error('Error clearing settings:', error);
        setTestMessage('Failed to clear settings');
      }
    }
  };

  const handleExport = async () => {
    setExporting(true);
    setExportMessage('');
    try {
      const session = useAuthStore.getState().session;
      if (!session?.sessionId) throw new Error('not signed in');
      const response = await fetch(`${BACKEND_CONFIG.baseUrl}/api/user/export`, { headers: { Authorization: `Bearer ${session.sessionId}` } });
      if (!response.ok) throw new Error(`status ${response.status}`);
      const url = URL.createObjectURL(await response.blob());
      const link = document.createElement('a');
      link.href = url;
      link.download = `moly-export-${new Date().toISOString().slice(0, 10)}.json`;
      link.click();
      URL.revokeObjectURL(url);
      setExportMessage('The file was saved to your downloads.');
    } catch (err) {
      setExportMessage(`Could not export: ${err instanceof Error ? err.message : 'unknown error'}`);
    } finally {
      setExporting(false);
    }
  };

  const handleDeleteProfile = async () => {
    setDeleteError('');
    const isLocal = useAuthStore.getState().session?.local === true;
    if (!deletePassword.trim()) {
      setDeleteError(isLocal ? 'Type DELETE to confirm' : 'Please enter your password');
      return;
    }

    setDeleteLoading(true);
    try {
      // Get auth token from authStore
      const session = useAuthStore.getState().session;
      if (!session || !session.sessionId) {
        throw new Error('Not authenticated. Please login first.');
      }
      const token = session.sessionId;

      // Call backend API to delete user and all their data
      const deleteUrl = `${BACKEND_CONFIG.baseUrl}${BACKEND_CONFIG.endpoints.account.deleteProfile}`;
      const response = await fetch(deleteUrl, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        // The local profile has no password: it is confirmed by typing DELETE.
        body: JSON.stringify(isLocal ? { confirm: deletePassword.trim() } : { password: deletePassword }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Delete failed with status ${response.status}`);
      }

      // Clear all local storage and session storage
      await chrome.storage.local.clear();
      await chrome.storage.session?.clear?.();

      // Redirect to login or close the extension
      alert('Your profile and all associated data have been permanently deleted.');
      window.close();
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Unknown error occurred';
      console.error('Error deleting profile:', error);
      setDeleteError(`Failed to delete profile: ${errorMsg}`);
    } finally {
      setDeleteLoading(false);
    }
  };

  return (
    <div className="settings-container">
      <div className="settings-header">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <h1 style={{ margin: 0 }}>Moly Settings</h1>
          {onClose && (
            <button
              onClick={onClose}
              style={{
                background: 'rgba(255, 255, 255, 0.2)',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                padding: '6px 12px',
                cursor: 'pointer',
                fontSize: '14px',
                fontWeight: '600',
                transition: 'background 0.2s',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255, 255, 255, 0.3)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(255, 255, 255, 0.2)'}
            >
              ← Back
            </button>
          )}
        </div>
        <p className="settings-subtitle">Configure your LLM providers and preferences</p>
      </div>

      <div className="settings-content">
        {/* AI provider and API key: saved by the Moly app in its encrypted database, never in the browser */}
        <section className="settings-section" style={{ padding: 0, background: 'transparent' }}>
          <AiProviderPanel />
        </section>
        {testMessage && (
          <div className={`form-message ${testMessage.includes('Failed') ? 'error' : 'success'}`}>{testMessage}</div>
        )}

        {/* Preferences Section */}
        <section className="settings-section">
          <h2>Communication Preferences</h2>
          <div className="prefs-form">
            {/* Current Preferences Display */}
            <div style={{
              background: 'rgba(255, 255, 255, 0.05)',
              border: '1px solid rgba(255, 255, 255, 0.2)',
              borderRadius: '8px',
              padding: '1rem',
              marginBottom: '1.5rem',
              fontSize: '14px',
            }}>
              <p style={{ margin: '0 0 0.5rem 0', color: '#d1d5db' }}>
                <strong>Current Settings:</strong>
              </p>
              <p style={{ margin: '0.25rem 0', color: '#9ca3af' }}>
                • Style: <span style={{ color: '#e5e7eb' }}>{communicationStyle || '(not set)'}</span>
              </p>
              <p style={{ margin: '0.25rem 0', color: '#9ca3af' }}>
                • Tone: <span style={{ color: '#e5e7eb' }}>{tonePreference || '(not set)'}</span>
              </p>
              <p style={{ margin: '0.25rem 0', color: '#9ca3af' }}>
                • Values: <span style={{ color: '#e5e7eb' }}>{coreValues || '(not set)'}</span>
              </p>
              <p style={{ margin: '0.25rem 0', color: '#9ca3af' }}>
                • Goals: <span style={{ color: '#e5e7eb' }}>{goals || '(not set)'}</span>
              </p>
            </div>

            <div className="form-group">
              <label className="form-label">Communication Style</label>
              <select
                value={communicationStyle}
                onChange={(e) => setCommunicationStyle(e.target.value)}
                style={{
                  width: '100%',
                  padding: '12px',
                  background: 'rgba(255, 255, 255, 0.1)',
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  borderRadius: '8px',
                  color: 'white',
                  fontSize: '14px',
                  cursor: 'pointer',
                }}
                disabled={prefsLoading}
              >
                <option value="">-- Select a style --</option>
                <option value="Casual">Casual</option>
                <option value="Direct">Direct</option>
                <option value="Formal">Formal</option>
                <option value="Authentic">Authentic</option>
                <option value="Formal & Respectful">Formal & Respectful</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">Preferred Tone</label>
              <select
                value={tonePreference}
                onChange={(e) => setTonePreference(e.target.value)}
                style={{
                  width: '100%',
                  padding: '12px',
                  background: 'rgba(255, 255, 255, 0.1)',
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  borderRadius: '8px',
                  color: 'white',
                  fontSize: '14px',
                  cursor: 'pointer',
                }}
                disabled={prefsLoading}
              >
                <option value="">-- Select a tone --</option>
                <option value="Friendly">Friendly</option>
                <option value="Professional">Professional</option>
                <option value="Humorous">Humorous</option>
                <option value="Supportive">Supportive</option>
                <option value="Analytical">Analytical</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">Core Values</label>
              <input
                type="text"
                value={coreValues}
                onChange={(e) => setCoreValues(e.target.value)}
                placeholder="e.g., authenticity, loyalty, growth"
                style={{
                  width: '100%',
                  padding: '12px',
                  background: 'rgba(255, 255, 255, 0.1)',
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  borderRadius: '8px',
                  color: 'white',
                  fontSize: '14px',
                  boxSizing: 'border-box',
                }}
                disabled={prefsLoading}
              />
              <p style={{ fontSize: '12px', color: '#9ca3af', marginTop: '4px' }}>
                Your core values (comma-separated)
              </p>
            </div>

            <div className="form-group">
              <label className="form-label">Communication Goals</label>
              <input
                type="text"
                value={goals}
                onChange={(e) => setGoals(e.target.value)}
                placeholder="e.g., improve communication, build confidence, understand patterns"
                style={{
                  width: '100%',
                  padding: '12px',
                  background: 'rgba(255, 255, 255, 0.1)',
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  borderRadius: '8px',
                  color: 'white',
                  fontSize: '14px',
                  boxSizing: 'border-box',
                }}
                disabled={prefsLoading}
              />
              <p style={{ fontSize: '12px', color: '#9ca3af', marginTop: '4px' }}>
                Your goals for communication coaching (comma-separated)
              </p>
            </div>

            <div className="form-actions">
              <button
                onClick={handleSavePreferences}
                disabled={prefsLoading}
                className="btn btn-primary"
              >
                {prefsLoading ? 'Saving...' : '💾 Save Preferences'}
              </button>
            </div>

            {prefsMessage && (
              <div className={`form-message ${prefsMessage.includes('Error') ? 'error' : 'success'}`}>
                {prefsMessage}
              </div>
            )}
          </div>
        </section>

        {/* Advanced Section */}
        <section className="settings-section">
          <h2>Advanced</h2>
          <div className="advanced-options">
            <p className="section-description">Reset or manage your extension data</p>
            <button onClick={handleClearAllSettings} className="btn btn-danger">
              Clear All Settings
            </button>
            <p className="section-info">This will reset all provider configurations and preferences to defaults. You will need to re-enter API keys.</p>
          </div>
        </section>

        {/* Your data: a copy of everything Moly stores about you */}
        <section className="settings-section">
          <h2>Your data</h2>
          <p className="section-description">Download everything Moly has stored about you (conversations, contacts, what Moly learned) as one file. It stays on your computer; Moly sends it nowhere.</p>
          <button onClick={handleExport} disabled={exporting} className="btn btn-secondary">
            {exporting ? 'Preparing...' : 'Export my data'}
          </button>
          {exportMessage && <div className={`form-message ${exportMessage.startsWith('Could not') ? 'error' : 'success'}`}>{exportMessage}</div>}
        </section>

        {/* Danger Zone - Delete Profile */}
        <section className="settings-section" style={{ borderTop: '2px solid #ef4444', paddingTop: '2rem', marginTop: '2rem' }}>
          <h2 style={{ color: '#ef4444' }}>Danger Zone</h2>
          <div className="advanced-options">
            <p className="section-description" style={{ color: '#7f1d1d' }}>⚠️ Permanently delete your profile and all associated data</p>
            <button
              onClick={() => setShowDeleteModal(true)}
              className="btn btn-danger"
              style={{ backgroundColor: '#dc2626', borderColor: '#991b1b' }}
            >
              Delete Profile Permanently
            </button>
            <p className="section-info" style={{ color: '#7f1d1d' }}>
              This action cannot be undone. Your profile, all conversations, contacts, and settings will be permanently deleted from the system.
            </p>
          </div>
        </section>

        {/* About Section */}
        <section className="settings-section">
          <h2>About Moly</h2>
          <div className="about-content">
            <p>Moly is an AI-powered messaging assistant for dating and social apps.</p>
            <p>
              <strong>Supported Providers:</strong>
            </p>
            <ul>
              <li>
                <strong>Claude</strong> - Anthropic's advanced AI model (recommended)
              </li>
              <li>
                <strong>OpenAI</strong> - GPT-4 and GPT-3.5 Turbo models
              </li>
              <li>
                <strong>Ollama</strong> - Run open-source models locally (no internet required)
              </li>
            </ul>
            <p className="version">Version 1.0.0 - Phase 1 (Multi-Provider)</p>
          </div>
        </section>
      </div>

      {/* Delete Profile Modal */}
      {showDeleteModal && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0, 0, 0, 0.5)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 1000,
        }}>
          <div style={{
            backgroundColor: '#1f2937',
            borderRadius: '8px',
            padding: '2rem',
            maxWidth: '500px',
            width: '90%',
            boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.3)',
          }}>
            <h2 style={{ margin: '0 0 1rem 0', color: '#ef4444' }}>Delete Profile?</h2>
            <p style={{ color: '#d1d5db', marginBottom: '1rem' }}>
              This will permanently delete your profile and all associated data including:
            </p>
            <ul style={{ color: '#d1d5db', marginBottom: '1.5rem', paddingLeft: '1.5rem' }}>
              <li>All conversations and messages</li>
              <li>All contacts and contact information</li>
              <li>Profile settings and preferences</li>
              <li>Learning data and behavioral patterns</li>
            </ul>
            <p style={{ color: '#fca5a5', fontWeight: 'bold', marginBottom: '1rem' }}>
              ⚠️ This action cannot be undone!
            </p>

            <div style={{ marginBottom: '1.5rem' }}>
              <label style={{ display: 'block', color: '#d1d5db', marginBottom: '0.5rem' }}>
                {isLocalProfile ? 'Type DELETE to confirm:' : 'Enter your password to confirm:'}
              </label>
              <input
                type={isLocalProfile ? 'text' : 'password'}
                value={deletePassword}
                onChange={(e) => {
                  setDeletePassword(e.target.value);
                  setDeleteError('');
                }}
                placeholder={isLocalProfile ? 'DELETE' : 'Password'}
                disabled={deleteLoading}
                onKeyPress={(e) => {
                  if (e.key === 'Enter' && !deleteLoading) {
                    handleDeleteProfile();
                  }
                }}
                style={{
                  width: '100%',
                  padding: '0.75rem',
                  backgroundColor: '#374151',
                  color: 'white',
                  border: deleteError ? '2px solid #ef4444' : '1px solid #4b5563',
                  borderRadius: '4px',
                  fontSize: '14px',
                  boxSizing: 'border-box',
                }}
              />
              {deleteError && (
                <p style={{ color: '#fca5a5', fontSize: '12px', marginTop: '0.5rem' }}>
                  {deleteError}
                </p>
              )}
            </div>

            <div style={{ display: 'flex', gap: '1rem' }}>
              <button
                onClick={() => {
                  setShowDeleteModal(false);
                  setDeletePassword('');
                  setDeleteError('');
                }}
                disabled={deleteLoading}
                style={{
                  flex: 1,
                  padding: '0.75rem',
                  backgroundColor: '#4b5563',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: deleteLoading ? 'not-allowed' : 'pointer',
                  fontSize: '14px',
                  fontWeight: '600',
                  opacity: deleteLoading ? 0.5 : 1,
                }}
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteProfile}
                disabled={deleteLoading || !deletePassword.trim()}
                style={{
                  flex: 1,
                  padding: '0.75rem',
                  backgroundColor: '#dc2626',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: (deleteLoading || !deletePassword.trim()) ? 'not-allowed' : 'pointer',
                  fontSize: '14px',
                  fontWeight: '600',
                  opacity: (deleteLoading || !deletePassword.trim()) ? 0.5 : 1,
                }}
              >
                {deleteLoading ? 'Deleting...' : 'Delete Forever'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Settings;
