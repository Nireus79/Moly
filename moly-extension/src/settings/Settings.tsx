/**
 * Settings Component
 * Allows users to configure multiple LLM providers
 */

import React, { useState, useEffect } from 'react';
import { useSettingsStore } from '@/stores/settingsStore';
import { getProviderManager } from '@/api/providerManager';
import type { LLMProviderType } from '@/api/providers';
import './settings.css';

export const Settings: React.FC = () => {
  const { settings, loadSettings, updateProvider, setActiveProvider, error } = useSettingsStore();

  // Provider-specific state
  const [selectedProvider, setSelectedProvider] = useState<LLMProviderType>('claude');
  const [apiKey, setApiKey] = useState('');
  const [baseUrl, setBaseUrl] = useState('');
  const [model, setModel] = useState('');
  const [validating, setValidating] = useState(false);
  const [testMessage, setTestMessage] = useState('');

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

  const handleProviderChange = async (provider: LLMProviderType) => {
    setSelectedProvider(provider);
    if (settings) {
      const config = settings.providers[provider];
      setApiKey(config.apiKey?.slice(0, 20) + '...' + config.apiKey?.slice(-8) || '');
      setBaseUrl(config.baseUrl || '');
      setModel(config.model || '');

      // Discover models for this provider
      await discoverModels(provider, config.apiKey);
    }
    setTestMessage('');
  };

  const discoverModels = async (providerType: LLMProviderType, apiKey?: string) => {
    try {
      const provider = manager.getProvider(providerType);
      if (!provider) return;

      // For providers that need credentials, update them temporarily
      if (apiKey && 'apiKey' in provider) {
        (provider as any).apiKey = apiKey;
      }

      if ('discoverModels' in provider && typeof (provider as any).discoverModels === 'function') {
        setTestMessage('Discovering available models...');
        const discovered = await (provider as any).discoverModels();
        if (discovered.length > 0) {
          setTestMessage(`✅ Found ${discovered.length} models`);
          setTimeout(() => setTestMessage(''), 2000);
        }
      }
    } catch (error) {
      console.warn('Model discovery failed:', error);
    }
  };

  const handleSaveProvider = async () => {
    if (!selectedProvider) {
      setTestMessage('Please select a provider');
      return;
    }

    // Validate required fields based on provider
    if (selectedProvider !== 'ollama' && !apiKey.trim()) {
      setTestMessage('Please enter an API key');
      return;
    }

    if (apiKey.includes('...') && !apiKey.includes('..._')) {
      // If masked and not newly entered, just activate
      setTestMessage('Provider configuration already saved');
      return;
    }

    setValidating(true);
    setTestMessage('Configuring and discovering models...');

    try {
      // Discover models first with the new credentials
      const actualApiKey = !apiKey.includes('...') ? apiKey : undefined;
      if (actualApiKey) {
        await discoverModels(selectedProvider, actualApiKey);
      }

      await updateProvider(selectedProvider, {
        apiKey: actualApiKey,
        baseUrl: baseUrl || undefined,
        model: model || undefined,
        enabled: true,
      });

      setTestMessage('✅ Provider configured successfully!');
      setTimeout(() => setTestMessage(''), 3000);
    } catch (err) {
      setTestMessage(`❌ Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setValidating(false);
    }
  };

  const handleSetActiveProvider = async (provider: LLMProviderType) => {
    await setActiveProvider(provider);
    setTestMessage(`✅ ${provider} is now active`);
    setTimeout(() => setTestMessage(''), 2000);
  };

  const isConfigured = settings?.providers[selectedProvider]?.enabled;
  const availableModels = manager.getModels(selectedProvider);

  return (
    <div className="settings-container">
      <div className="settings-header">
        <h1>Moly Settings</h1>
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
                {provider === 'claude' && '🤖 Claude'}
                {provider === 'openai' && '🔷 OpenAI'}
                {provider === 'ollama' && '🐪 Ollama (Local)'}
              </button>
            ))}
          </div>
        </section>

        {/* Provider Configuration */}
        <section className="settings-section">
          <div className="section-header">
            <h2>Configure {selectedProvider.charAt(0).toUpperCase() + selectedProvider.slice(1)}</h2>
            {isConfigured && <span className="status-badge configured">✓ Configured</span>}
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
                    onChange={(e) => setApiKey(e.target.value)}
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
                <label className="form-label">Ollama Base URL</label>
                <input
                  type="text"
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  placeholder="http://localhost:11434"
                  className="key-input"
                  disabled={validating}
                />
                <p className="info-text">Must have Ollama running locally</p>
              </div>
            )}

            {/* Model Selection */}
            <div>
              <label className="form-label">Model</label>
              <select value={model} onChange={(e) => setModel(e.target.value)} className="key-input" disabled={validating}>
                <option value="">Select a model</option>
                {availableModels.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            </div>

            {/* Action Buttons */}
            <div className="form-actions">
              <button onClick={handleSaveProvider} disabled={validating || !apiKey.trim()} className="btn btn-primary">
                {validating ? 'Validating...' : 'Save & Validate'}
              </button>
              {isConfigured && (
                <button onClick={() => handleSetActiveProvider(selectedProvider)} className="btn btn-secondary">
                  Make Active
                </button>
              )}
            </div>

            {testMessage && (
              <div className={`form-message ${testMessage.includes('✅') ? 'success' : 'error'}`}>{testMessage}</div>
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
                <p className="success-text">✓ Provider is configured and ready</p>
              ) : (
                <p className="warning-text">⚠ Provider not configured yet</p>
              )}
            </div>
          </section>
        )}

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
