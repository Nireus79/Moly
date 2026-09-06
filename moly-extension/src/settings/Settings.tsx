/**
 * Settings Component - Simplified v2
 * Configure LLM providers, chat mode, and communication context
 */

import React, { useState, useEffect } from 'react';
import { useSettingsStore } from '@/stores/settingsStore';
import { getProviderManager } from '@/api/providerManager';
import type { LLMProviderType } from '@/api/providers';
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

  const manager = getProviderManager();

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  useEffect(() => {
    if (settings) {
      setSelectedProvider(settings.activeProvider);
      const config = settings.providers[settings.activeProvider];
      setApiKey(config.apiKey?.slice(0, 20) + '...' + config.apiKey?.slice(-8) || '');
      setBaseUrl(config.baseUrl || '');
      setModel(config.model || '');
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

  const discoverCloudModels = async (provider: 'claude' | 'openai', apiKey: string) => {
    setDiscoveringModels(true);
    try {
      const p = manager.getProvider(provider) as any;
      if (p && p.discoverModels) {
        const models = await p.discoverModels();
        setDiscoveredModels(models);
        if (models.length > 0 && !model) {
          setModel(models[0]);
        }
      }
    } catch (error) {
      console.error(`Failed to discover ${provider} models:`, error);
      setTestMessage(`Could not discover ${provider} models`);
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
      if (selectedProvider === 'claude' || selectedProvider === 'openai') {
        const keyToUse = !apiKey.includes('...') ? apiKey : undefined;
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
                      onClick={() => {
                        console.log('[Settings] Model selected:', m);
                        setModel(m);
                      }}
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
    </div>
  );
};

export default Settings;
