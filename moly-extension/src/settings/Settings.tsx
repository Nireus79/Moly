/**
 * Settings Component - Simplified v2
 * Configure LLM providers, chat mode, and communication context
 */

import React, { useState, useEffect } from 'react';
import { useSettingsStore } from '@/stores/settingsStore';
import { useAuthStore } from '@/stores/authStore';
import { getProviderManager } from '@/api/providerManager';
import { ClaudeProvider } from '@/api/providers/claude';
import { OpenAIProvider } from '@/api/providers/openai';
import type { LLMProviderType } from '@/api/providers';
import { BACKEND_CONFIG } from '@/api/backendConfig';
import './settings.css';

interface SettingsProps {
  onClose?: () => void;
}

export const Settings: React.FC<SettingsProps> = ({ onClose }) => {
  const { settings, loadSettings, updateProvider, setActiveProvider, error } = useSettingsStore();

  const [selectedProvider, setSelectedProvider] = useState<LLMProviderType>('claude');
  const [apiKey, setApiKey] = useState('');
  const [baseUrl, setBaseUrl] = useState('');
  const [model, setModel] = useState('');
  const [validating, setValidating] = useState(false);
  const [testMessage, setTestMessage] = useState('');
  const [discoveringModels, setDiscoveringModels] = useState(false);
  const [discoveredModels, setDiscoveredModels] = useState<string[]>([]);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deletePassword, setDeletePassword] = useState('');
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [deleteError, setDeleteError] = useState('');

  // Preferences state
  const [communicationStyle, setCommunicationStyle] = useState('');
  const [tonePreference, setTonePreference] = useState('');
  const [coreValues, setCoreValues] = useState('');
  const [goals, setGoals] = useState('');
  const [prefsLoading, setPrefsLoading] = useState(false);
  const [prefsMessage, setPrefsMessage] = useState('');

  const manager = getProviderManager();

  // Define discover functions BEFORE useEffects that call them (hoisting issue fix)
  const discoverCloudModels = async (provider: 'claude' | 'openai', apiKey: string) => {
    setDiscoveringModels(true);
    console.log(`[Settings] Starting ${provider} model discovery with key length: ${apiKey.length}`);
    try {
      if (!apiKey || !apiKey.trim()) {
        throw new Error(`No API key provided for ${provider}`);
      }

      // CRITICAL: Create fresh provider instance with the API key for discovery ONLY
      // Skip configureProvider() which validates with a hardcoded model that may not exist
      // We only need discoverModels(), which doesn't require full configuration or validation
      console.log(`[Settings] Creating fresh ${provider} provider instance for discovery...`);

      let p: any;
      if (provider === 'claude') {
        p = new ClaudeProvider(apiKey, '');
      } else if (provider === 'openai') {
        p = new OpenAIProvider(apiKey, '');
      } else {
        throw new Error(`Unknown provider: ${provider}`);
      }

      if (!p.discoverModels) {
        throw new Error(`Provider ${provider} has no discoverModels method`);
      }

      console.log(`[Settings] Calling ${provider}.discoverModels()...`);
      const models = await p.discoverModels();
      console.log(`[Settings] ${provider} discovery returned:`, models);

      if (!models || !Array.isArray(models)) {
        throw new Error(`${provider} returned invalid response: ${typeof models}`);
      }

      setDiscoveredModels(models);
      if (models.length > 0 && !model) {
        setModel(models[0]);
      }
      console.log(`[Settings] Discovered ${models.length} models for ${provider}`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      console.error(`[Settings] ${provider} discovery failed:`, errorMsg);
      setTestMessage(`${provider} model discovery failed: ${errorMsg}`);
      setDiscoveredModels([]);
    } finally {
      setDiscoveringModels(false);
    }
  };

  const discoverOllamaModels = async (url: string) => {
    setDiscoveringModels(true);
    try {
      const provider = manager.getProvider('ollama') as any;
      if (provider && provider.discoverModels) {
        const models = await provider.discoverModels();
        setDiscoveredModels(models);
        if (models.length > 0 && !model) {
          setModel(models[0]);
        }
      }
    } catch (error) {
      console.error('Failed to discover Ollama models:', error);
      setTestMessage('Could not connect to Ollama server');
    } finally {
      setDiscoveringModels(false);
    }
  };

  useEffect(() => {
    loadSettings();
    loadPreferences();
  }, [loadSettings]);

  const loadPreferences = async () => {
    try {
      const session = useAuthStore.getState().session;
      if (!session || !session.sessionId) return;

      const response = await fetch('http://localhost:8080/api/v2/about-me', {
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

      const response = await fetch('http://localhost:8080/api/v2/about-me', {
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

  useEffect(() => {
    if (settings) {
      setSelectedProvider(settings.activeProvider);
      const config = settings.providers[settings.activeProvider];
      setApiKey(config.apiKey?.slice(0, 20) + '...' + config.apiKey?.slice(-8) || '');
      setBaseUrl(config.baseUrl || '');
      setModel(config.model || '');

      // Auto-discover models when Settings component mounts
      if (settings.activeProvider === 'ollama' && config.baseUrl) {
        discoverOllamaModels(config.baseUrl);
      } else if ((settings.activeProvider === 'claude' || settings.activeProvider === 'openai') && config.apiKey) {
        discoverCloudModels(settings.activeProvider, config.apiKey);
      }
    }
  }, [settings]);

  const handleProviderChange = (provider: LLMProviderType) => {
    setSelectedProvider(provider);
    const config = settings?.providers[provider];
    setApiKey(config?.apiKey?.slice(0, 20) + '...' + config?.apiKey?.slice(-8) || '');
    setBaseUrl(config?.baseUrl || '');
    setModel(config?.model || '');
    setDiscoveredModels([]);

    // Auto-discover models when switching providers
    if (provider === 'ollama') {
      discoverOllamaModels(config?.baseUrl || 'http://127.0.0.1:11435');
    } else if ((provider === 'claude' || provider === 'openai') && config?.apiKey) {
      discoverCloudModels(provider, config.apiKey);
    }
  };

  const handleSaveProvider = async () => {
    if (!apiKey.trim() && selectedProvider !== 'ollama') return;
    if (!baseUrl.trim() && selectedProvider === 'ollama') return;

    setValidating(true);
    try {
      // Only pass new API key if user actually entered one (not masked)
      const newConfig: any = {
        baseUrl,
        model,
        enabled: true,
      };

      // If API key is not masked (doesn't contain '...'), it's a new key
      if (selectedProvider !== 'ollama' && !apiKey.includes('...')) {
        newConfig.apiKey = apiKey;
      }

      await updateProvider(selectedProvider, newConfig);

      // Auto-activate the provider after saving
      await setActiveProvider(selectedProvider);

      // Discover models for cloud providers
      // Get the actual API key: either new one from input (if not masked) or existing from storage
      if (selectedProvider === 'claude' || selectedProvider === 'openai') {
        const keyToUse = !apiKey.includes('...') ? apiKey : settings?.providers[selectedProvider]?.apiKey;
        if (keyToUse) {
          await discoverCloudModels(selectedProvider, keyToUse);
        }
      }

      setTestMessage('Provider configured and activated');
      setTimeout(() => setTestMessage(''), 2000);
    } catch (err) {
      setTestMessage(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setValidating(false);
    }
  };

  const handleSetActiveProvider = async (provider: LLMProviderType) => {
    try {
      await setActiveProvider(provider);
      setTestMessage('Active provider changed');
      setTimeout(() => setTestMessage(''), 2000);
    } catch (err) {
      setTestMessage(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleSelectModel = (modelName: string) => {
    console.log('[Settings] Model selected:', modelName);
    setModel(modelName);
  };

  const handleClearAllSettings = async () => {
    if (confirm('Are you sure you want to clear all settings? This cannot be undone.')) {
      try {
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

  const handleDeleteProfile = async () => {
    setDeleteError('');
    if (!deletePassword.trim()) {
      setDeleteError('Please enter your password');
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
        body: JSON.stringify({
          password: deletePassword,
        }),
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

  const isConfigured = settings?.providers[selectedProvider]?.enabled;
  // Use discoveredModels for all providers (Claude, OpenAI, Ollama)
  // Models are auto-discovered via provider.discoverModels()
  const availableModels = discoveredModels.length > 0 ? discoveredModels : manager.getModels(selectedProvider);

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
        {/* Provider Selection */}
        <section className="settings-section">
          <h2>Select LLM Provider</h2>
          <div className="provider-tabs">
            {(['claude', 'openai', 'ollama'] as const).map((provider) => (
              <button
                key={provider}
                className={`provider-tab ${selectedProvider === provider ? 'active' : ''}`}
                onClick={() => handleProviderChange(provider)}
              >
                {provider === 'claude' && 'Claude'}
                {provider === 'openai' && 'OpenAI'}
                {provider === 'ollama' && 'Ollama (Local)'}
              </button>
            ))}
          </div>
        </section>

        {/* Provider Configuration */}
        <section className="settings-section">
          <div className="section-header">
            <h2>Configure {selectedProvider.charAt(0).toUpperCase() + selectedProvider.slice(1)}</h2>
            {isConfigured && <span className="status-badge configured">Configured</span>}
            {!isConfigured && <span className="status-badge">Not configured</span>}
          </div>

          <div className="provider-config">
            {/* API Key Field */}
            {selectedProvider !== 'ollama' && (
              <div>
                <label className="form-label">
                  {selectedProvider === 'claude' ? 'Claude' : 'OpenAI'} API Key
                  <a href={selectedProvider === 'claude' ? 'https://console.anthropic.com/keys' : 'https://platform.openai.com/keys'} target="_blank" rel="noopener noreferrer" className="get-key-link">
                    Get key →
                  </a>
                </label>
                <div className="key-input-wrapper">
                  <input
                    type={apiKey.includes('...') ? 'password' : 'text'}
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value.trim())}
                    placeholder="sk-..."
                    className="key-input"
                    disabled={validating}
                  />
                </div>
              </div>
            )}

            {/* Ollama-specific fields */}
            {selectedProvider === 'ollama' && (
              <div>
                <label className="form-label">
                  Ollama Base URL
                  {discoveringModels && <span style={{ marginLeft: '8px', color: '#666' }}>discovering...</span>}
                </label>
                <input
                  type="text"
                  value={baseUrl}
                  onChange={(e) => {
                    setBaseUrl(e.target.value);
                    if (e.target.value.trim()) {
                      discoverOllamaModels(e.target.value);
                    }
                  }}
                  placeholder="http://127.0.0.1:11434"
                  className="key-input"
                  disabled={validating}
                />
                {discoveredModels.length > 0 && (
                  <p style={{ fontSize: '12px', color: '#22c55e', marginTop: '8px', marginBottom: '12px' }}>
                    ✓ Found {discoveredModels.length} model{discoveredModels.length !== 1 ? 's' : ''}
                  </p>
                )}
              </div>
            )}

            {/* Model Selection */}
            <div>
              <label className="form-label">Model</label>
              {availableModels.length > 0 ? (
                <div className="model-list">
                  {availableModels.map((m) => (
                    <button
                      key={m}
                      type="button"
                      onClick={() => handleSelectModel(m)}
                      className={`model-btn ${model === m ? 'active' : ''}`}
                    >
                      ✓ {m}
                    </button>
                  ))}
                </div>
              ) : (
                <p style={{ fontSize: '13px', color: '#9ca3af', fontStyle: 'italic' }}>
                  No models available for this provider
                </p>
              )}
            </div>

            {/* Action Buttons */}
            <div className="form-actions">
              <button
                onClick={handleSaveProvider}
                disabled={validating || (selectedProvider !== 'ollama' ? !apiKey.trim() : !baseUrl.trim())}
                className="btn btn-primary"
              >
                {validating ? 'Validating...' : 'Save & Validate'}
              </button>
              {isConfigured && settings?.activeProvider !== selectedProvider && (
                <button onClick={() => handleSetActiveProvider(selectedProvider)} className="btn btn-secondary">
                  Switch to This Provider
                </button>
              )}
            </div>

            {testMessage && (
              <div className={`form-message ${testMessage.includes('Error') ? 'error' : 'success'}`}>{testMessage}</div>
            )}
            {error && <div className="form-message error">{error}</div>}
          </div>
        </section>

        {/* Active Provider Info */}
        {settings && (
          <section className="settings-section">
            <h2>Active Provider</h2>
            <div className="info-box">
              <p>
                <strong>Current:</strong> {settings.activeProvider}
              </p>
              <p>
                <strong>Model:</strong> {settings.providers[settings.activeProvider]?.model}
              </p>
              {settings.providers[settings.activeProvider]?.enabled ? (
                <p className="success-text">Provider is configured and ready</p>
              ) : (
                <p className="warning-text">Provider not configured yet</p>
              )}
            </div>
          </section>
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
                Enter your password to confirm:
              </label>
              <input
                type="password"
                value={deletePassword}
                onChange={(e) => {
                  setDeletePassword(e.target.value);
                  setDeleteError('');
                }}
                placeholder="Password"
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
