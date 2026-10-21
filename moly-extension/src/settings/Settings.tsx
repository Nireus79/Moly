/**
 * Settings Component - Simplified
 * Configure LLM providers, chat mode, and communication context
 */

import React, { useState, useEffect } from 'react';
import { useSettingsStore } from '@/stores/settingsStore';
import { useAuthStore } from '@/stores/authStore';
import { AiProviderPanel } from '@/components/ai/AiProviderPanel';
import { isPhoneApp } from '@/utils/platform';
import { ReportSection } from './ReportSection';
import { BACKEND_CONFIG } from '@/api/backendConfig';
import { fetchExport, importExport } from '@/api/dataTransfer';
import { Icon } from '@/components/Icon';
import { AboutPanel } from './AboutPanel';
import { ManualPanel } from './ManualPanel';
import { getThemeChoice, setThemeChoice, type ThemeChoice } from '@/styles/theme';
import '@/sidebar/components/about-me.css';
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
  const [theme, setTheme] = useState<ThemeChoice>(getThemeChoice());
  const [view, setView] = useState<'main' | 'about' | 'manual'>('main');
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
        setPrefsMessage('Preferences saved.');
        setTimeout(() => setPrefsMessage(''), 3000);
      } else {
        const error = await response.text();
        setPrefsMessage(`Could not save: ${error}`);
      }
    } catch (error) {
      setPrefsMessage(`Could not save: ${error instanceof Error ? error.message : 'unknown error'}`);
    } finally {
      setPrefsLoading(false);
    }
  };

  const handleClearAllSettings = async () => {
    if (confirm('Clear what this browser keeps for Moly (your sign-in here and your interface choices)? Your conversations stay in the Moly app.')) {
      try {
        // Everything the browser kept for this profile goes: the extension storage, and the page storage (the cached About Me profile, the session, error reports).
      try { localStorage.clear(); sessionStorage.clear(); } catch { /* storage may be unavailable */ }
      await chrome.storage.local.clear();
        await loadSettings();
        setTestMessage('Browser data cleared.');
        setTimeout(() => setTestMessage(''), 2000);
      } catch (error) {
        console.error('Error clearing settings:', error);
        setTestMessage('Failed to clear the browser data.');
      }
    }
  };

  const handleExport = async () => {
    setExporting(true);
    setExportMessage('');
    const result = await fetchExport();
    if (result.ok) {
      const url = URL.createObjectURL(new Blob([result.text], { type: 'application/json' }));
      const link = document.createElement('a');
      link.href = url;
      link.download = `moly-export-${new Date().toISOString().slice(0, 10)}.json`;
      link.click();
      URL.revokeObjectURL(url);
      setExportMessage('The file was saved to your downloads.');
    } else {
      setExportMessage(`Could not export: ${result.message}`);
    }
    setExporting(false);
  };

  const handleImportFile = async (file: File | undefined) => {
    if (!file) return;
    setExporting(true);
    setExportMessage('');
    const result = await importExport(await file.text());
    setExportMessage(result.ok ? result.message : `Could not import: ${result.message}`);
    setExporting(false);
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

      // Clear all local storage and session storage. The profile's own session is in the page storage (localStorage /
      // sessionStorage), which chrome.storage does not cover: left there, the deleted profile's session would still look valid.
      await chrome.storage.local.clear();
      await chrome.storage.session?.clear?.();
      await useAuthStore.getState().clearAuth();
      try {
        localStorage.clear();
        sessionStorage.clear();
      } catch {
        /* storage unavailable */
      }

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

  const withCurrent = (options: string[], current: string) => (current && !options.includes(current) ? [current, ...options] : options);
  const STYLES = withCurrent(['Casual', 'Direct', 'Formal', 'Authentic', 'Formal & Respectful'], communicationStyle);
  const TONES = withCurrent(['Friendly', 'Professional', 'Humorous', 'Supportive', 'Analytical'], tonePreference);
  const version = (() => {
    try {
      return chrome.runtime.getManifest().version;
    } catch {
      return '';
    }
  })();
  const isError = (m: string) => /^(Could not|Failed|Not authenticated)/.test(m);

  // The phone app has no browser to clear, and saving the export file needs a phone-specific way that is not built yet.
  const phone = isPhoneApp();

  return (
    <div className="settings-page">
      <header className="settings-topbar">
        <div className="settings-topbar-inner">
          {(view !== 'main' || onClose) && (
            <button className="btn btn-secondary btn-sm" onClick={() => (view !== 'main' ? setView('main') : onClose?.())}>
              <Icon name="back" /> {view !== 'main' ? 'Settings' : 'Back'}
            </button>
          )}
          <span className="brand-mark" aria-hidden="true" />
          <h1>{view === 'about' ? 'About Moly' : view === 'manual' ? 'User manual' : 'Settings'}</h1>
        </div>
      </header>

      <main className="settings-content">
        {view === 'about' && <AboutPanel />}
        {view === 'manual' && <ManualPanel />}
        {view === 'main' && (<>
        {/* AI provider and API key: saved by the Moly app in its encrypted database, never in the browser */}
        <AiProviderPanel />

        <section className="settings-section">
          <h2>Appearance</h2>
          <p className="section-description">Moly follows your system by default.</p>
          <div className="segmented" role="radiogroup" aria-label="Appearance">
            {(['system', 'light', 'dark'] as ThemeChoice[]).map((c) => (
              <button key={c} role="radio" aria-checked={theme === c} className={theme === c ? 'active' : ''} onClick={() => { setTheme(c); setThemeChoice(c); }}>
                {c === 'system' ? 'System' : c === 'light' ? 'Light' : 'Dark'}
              </button>
            ))}
          </div>
        </section>

        <section className="settings-section">
          <h2>Communication preferences</h2>
          <p className="section-description">Moly uses these to match your style. All of it is optional.</p>
          <div className="form-group">
            <label className="form-label" htmlFor="pref-style">Communication style</label>
            <select id="pref-style" className="select" value={communicationStyle} onChange={(e) => setCommunicationStyle(e.target.value)} disabled={prefsLoading}>
              <option value="">Not set</option>
              {STYLES.map((o) => <option key={o} value={o}>{o}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label className="form-label" htmlFor="pref-tone">Preferred tone</label>
            <select id="pref-tone" className="select" value={tonePreference} onChange={(e) => setTonePreference(e.target.value)} disabled={prefsLoading}>
              <option value="">Not set</option>
              {TONES.map((o) => <option key={o} value={o}>{o}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label className="form-label" htmlFor="pref-values">Core values</label>
            <input id="pref-values" className="input" type="text" value={coreValues} onChange={(e) => setCoreValues(e.target.value)} placeholder="For example: authenticity, loyalty, growth" disabled={prefsLoading} />
            <p className="form-hint">Separate them with commas.</p>
          </div>
          <div className="form-group">
            <label className="form-label" htmlFor="pref-goals">Communication goals</label>
            <input id="pref-goals" className="input" type="text" value={goals} onChange={(e) => setGoals(e.target.value)} placeholder="For example: be clearer, build confidence" disabled={prefsLoading} />
            <p className="form-hint">Separate them with commas.</p>
          </div>
          <div className="form-actions">
            <button onClick={handleSavePreferences} disabled={prefsLoading} className="btn btn-primary">
              {prefsLoading ? 'Saving...' : 'Save preferences'}
            </button>
          </div>
          {prefsMessage && (
            <div className={`notice ${isError(prefsMessage) ? 'danger' : 'ok'}`}>
              <Icon name={isError(prefsMessage) ? 'alert' : 'check'} /><span>{prefsMessage}</span>
            </div>
          )}
        </section>

        {!phone && (<>
        <section className="settings-section">
          <h2>Your data</h2>
          <p className="section-description">Download everything Moly has stored about you (conversations, contacts, what Moly learned) as one file. It stays on your computer; Moly sends it nowhere.</p>
          <div className="button-row">
            <button onClick={handleExport} disabled={exporting} className="btn btn-secondary">
              <Icon name="download" /> {exporting ? 'Working...' : 'Export my data'}
            </button>
            <label className="btn btn-secondary" style={{ cursor: exporting ? 'default' : 'pointer' }}>
              <Icon name="upload" /> Import data...
              <input type="file" accept="application/json,.json" style={{ display: 'none' }} disabled={exporting}
                onChange={(e) => { void handleImportFile(e.target.files?.[0]); e.target.value = ''; }} />
            </label>
          </div>
          <p className="section-info">Import restores a file made by "Export my data" into an empty profile, for example after reinstalling.</p>
          {exportMessage && (
            <div className={`notice ${exportMessage.startsWith('Could not') ? 'danger' : 'ok'}`}>
              <Icon name={exportMessage.startsWith('Could not') ? 'alert' : 'check'} /><span>{exportMessage}</span>
            </div>
          )}
        </section>

        <section className="settings-section">
          <h2>This browser</h2>
          <p className="section-description">Clears what this browser keeps for Moly: your sign-in here and your interface choices. Your conversations, contacts and AI key stay in the Moly app; use "Delete profile" below to remove them.</p>
          <button onClick={handleClearAllSettings} className="btn btn-secondary">Clear browser data</button>
          {testMessage && (
            <div className={`notice ${isError(testMessage) ? 'danger' : 'ok'}`}>
              <Icon name={isError(testMessage) ? 'alert' : 'check'} /><span>{testMessage}</span>
            </div>
          )}
        </section>
        </>)}

        <section className="settings-section danger-zone">
          <h2>Delete profile</h2>
          <p className="section-description">Permanently delete your profile and everything Moly stored about you. This cannot be undone{phone ? '.' : '; export your data first if you may want it.'}</p>
          <button onClick={() => setShowDeleteModal(true)} className="btn btn-danger">Delete profile permanently</button>
        </section>

        {phone && <ReportSection />}

        <section className="settings-section">
          <h2>About and help</h2>
          <div className="link-rows">
            <button className="link-row" onClick={() => setView('about')}>
              <span><strong>About Moly</strong><small>What it is, who makes it, how to reach us{version ? ` · version ${version}` : ''}</small></span>
              <Icon name="chevron" />
            </button>
            <button className="link-row" onClick={() => setView('manual')}>
              <span><strong>User manual</strong><small>How to use Moly, step by step, and what to do when something is wrong</small></span>
              <Icon name="chevron" />
            </button>
          </div>
        </section>
        </>)}
      </main>

      {showDeleteModal && (
        <div className="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="delete-title">
          <div className="modal-sheet">
            <h2 id="delete-title" className="danger">Delete profile?</h2>
            <p className="modal-lead">This permanently deletes your profile and everything stored with it:</p>
            <ul>
              <li>All conversations and messages</li>
              <li>All contacts and what Moly learned about them</li>
              <li>Your preferences and learned patterns</li>
            </ul>
            <div className="notice danger"><Icon name="alert" /><span>This cannot be undone.</span></div>

            <div className="form-group" style={{ marginTop: 16 }}>
              <label className="form-label" htmlFor="delete-confirm">
                {isLocalProfile ? 'Type DELETE to confirm' : 'Enter your password to confirm'}
              </label>
              <input
                id="delete-confirm"
                className="input"
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
              />
              {deleteError && <p className="form-hint" style={{ color: 'var(--danger)' }}>{deleteError}</p>}
            </div>

            <div className="modal-actions">
              <button
                className="btn btn-secondary"
                onClick={() => {
                  setShowDeleteModal(false);
                  setDeletePassword('');
                  setDeleteError('');
                }}
                disabled={deleteLoading}
              >
                Cancel
              </button>
              <button className="btn btn-danger-solid" onClick={handleDeleteProfile} disabled={deleteLoading || !deletePassword.trim()}>
                {deleteLoading ? 'Deleting...' : 'Delete forever'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Settings;
