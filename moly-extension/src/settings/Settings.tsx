/**
 * Settings Component
 * Allows users to configure multiple LLM providers
 */

import React, { useState, useEffect } from 'react';
import { useSettingsStore } from '@/stores/settingsStore';
import { getProviderManager } from '@/api/providerManager';
import { detectLocalModels } from '@/api/detection';
import { LocalModelStatusPanel } from './components/LocalModelStatus';
import { LocalModelSetup } from './components/LocalModelSetup';
import { ModelManagement } from './components/ModelManagement';
import { ServiceManager } from './components/ServiceManager';
import type { LLMProviderType } from '@/api/providers';
import type { LocalModelStatus } from '@/api/detection';
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
  const [chatMode, setChatMode] = useState<'socratic' | 'direct'>('socratic');
  const [communicationContext, setCommunicationContext] = useState<'formal' | 'friendly' | 'dating'>('friendly');
  const [discoveredModels, setDiscoveredModels] = useState<string[]>([]);
  const [localModelStatus, setLocalModelStatus] = useState<LocalModelStatus | null>(null);

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
      setChatMode(settings.chatMode || 'socratic');
      setCommunicationContext(settings.defaultContext || 'friendly');
    }
  }, [settings]);

  // Auto-discover models when API key or baseUrl changes (and it's a real new value, not masked display)
  useEffect(() => {
    const trimmedKey = apiKey.trim();
    const isMaskedDisplay = trimmedKey.includes('...');

    if (selectedProvider === 'ollama' && baseUrl && !isMaskedDisplay) {
      // Ollama with new base URL
      discoverModels(selectedProvider, undefined, baseUrl);
    } else if (selectedProvider !== 'ollama' && trimmedKey && !isMaskedDisplay && trimmedKey.length > 20) {
      // Claude/OpenAI with actual new key (not masked display, and looks like real key)
      discoverModels(selectedProvider, trimmedKey, undefined);
    }
  }, [apiKey, baseUrl, selectedProvider]);

  const handleProviderChange = async (provider: LLMProviderType) => {
    setSelectedProvider(provider);
    setDiscoveredModels([]); // Clear models when switching providers
    if (settings) {
      const config = settings.providers[provider];
      if (config.apiKey) {
        setApiKey(config.apiKey.slice(0, 20) + '...' + config.apiKey.slice(-8));
      } else {
        setApiKey('');
      }
      setBaseUrl(config.baseUrl || '');
      setModel(config.model || '');
    }
    setTestMessage('');
  };

  const discoverModels = async (providerType: LLMProviderType, apiKey?: string, baseUrl?: string) => {
    try {
      // Create temporary provider for discovery
      let tempProvider;

      if (providerType === 'claude') {
        const { ClaudeProvider } = await import('@/api/providers/claude');
        tempProvider = new ClaudeProvider(apiKey || '', model || 'claude-3-5-sonnet-20241022');
      } else if (providerType === 'openai') {
        const { OpenAIProvider } = await import('@/api/providers/openai');
        tempProvider = new OpenAIProvider(apiKey || '', model || 'gpt-4-turbo');
      } else if (providerType === 'ollama') {
        const { OllamaProvider } = await import('@/api/providers/ollama');
        tempProvider = new OllamaProvider(baseUrl || 'http://localhost:11434', model || 'mistral');
      } else {
        return;
      }

      // Discover models using temporary provider
      if ('discoverModels' in tempProvider && typeof (tempProvider as any).discoverModels === 'function') {
        const discovered = await (tempProvider as any).discoverModels();
        if (discovered && discovered.length > 0) {
          // Store in local state for immediate display
          setDiscoveredModels(discovered);

          // Also update manager's provider
          const managerProvider = manager.getProvider(providerType);
          if (managerProvider) {
            managerProvider.models = discovered;
          }

          setTestMessage(`Found ${discovered.length} models`);
          setTimeout(() => setTestMessage(''), 3000);
          return;
        }
      }

      setDiscoveredModels([]);
      setTestMessage('No models found');
      setTimeout(() => setTestMessage(''), 2000);
    } catch (error) {
      const msg = error instanceof Error ? error.message : 'Model discovery failed';
      setDiscoveredModels([]);
      setTestMessage(`Error: ${msg}`);
      console.error('Model discovery error:', error);
    }
  };

  const handleSaveProvider = async () => {
    if (!selectedProvider) {
      setTestMessage('Please select a provider');
      return;
    }

    // For non-Ollama providers, API key is required
    if (selectedProvider !== 'ollama' && !apiKey.trim()) {
      setTestMessage('Please enter an API key');
      return;
    }

    setValidating(true);
    setTestMessage('Validating configuration...');

    try {
      // Check if user entered a new key (not the masked display value)
      const trimmedKey = apiKey.trim();
      const isMaskedDisplay = trimmedKey.includes('...');

      // If it's not the masked display AND not empty, it's a new key
      let actualApiKey: string | undefined;
      if (!isMaskedDisplay && trimmedKey) {
        actualApiKey = trimmedKey;
      } else if (isMaskedDisplay && settings) {
        // User didn't change the key, use the one from settings
        actualApiKey = settings.providers[selectedProvider]?.apiKey;
      }

      // Validate and discover models before saving
      if (actualApiKey || selectedProvider === 'ollama') {
        await discoverModels(selectedProvider, actualApiKey, baseUrl);
      }

      // Save the provider configuration
      await updateProvider(selectedProvider, {
        apiKey: actualApiKey,
        baseUrl: baseUrl || undefined,
        model: model || undefined,
        enabled: true,
      });

      // Set as active provider
      await setActiveProvider(selectedProvider);

      if (!apiKey.includes('...')) {
        setTestMessage('Configuration saved and validated successfully');
      } else {
        setTestMessage('Configuration saved (using existing credentials)');
      }
      setTimeout(() => setTestMessage(''), 3000);
    } catch (err) {
      setTestMessage(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setValidating(false);
    }
  };

  const handleSetActiveProvider = async (provider: LLMProviderType) => {
    await setActiveProvider(provider);
    setTestMessage(`${provider} is now active`);
    setTimeout(() => setTestMessage(''), 2000);
  };

  const handleSaveChatMode = async (mode: 'socratic' | 'direct') => {
    try {
      setChatMode(mode);
      await chrome.storage.local.set({ chatMode: mode });
      setTestMessage(`Chat mode set to ${mode}`);
      setTimeout(() => setTestMessage(''), 2000);
    } catch (error) {
      console.error('Error saving chat mode:', error);
      setTestMessage('Failed to save chat mode');
    }
  };

  const handleSaveCommunicationContext = async (context: 'formal' | 'friendly' | 'dating') => {
    try {
      setCommunicationContext(context);
      await chrome.storage.local.set({ defaultContext: context });
      setTestMessage(`Communication context set to ${context}`);
      setTimeout(() => setTestMessage(''), 2000);
    } catch (error) {
      console.error('Error saving context:', error);
      setTestMessage('Failed to save context');
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
  // Use discovered models, fallback to manager's models for already-configured providers
  const availableModels = discoveredModels.length > 0 ? discoveredModels : manager.getModels(selectedProvider);

  const handleStatusRefresh = async () => {
    const newStatus = await detectLocalModels();
    setLocalModelStatus(newStatus);
  };

  return (
    <div className="settings-container">
      <div className="settings-header">
        <h1>Moly Settings</h1>
        <p className="settings-subtitle">Configure your LLM providers and preferences</p>
      </div>

      <div className="settings-content">
        {/* Local Model Status */}
        <section className="settings-section">
          <h2>Local Models Status</h2>
          <LocalModelStatusPanel onStatusChange={setLocalModelStatus} />
          {localModelStatus && <LocalModelSetup status={localModelStatus} />}
          {localModelStatus && (
            <ServiceManager
              status={localModelStatus}
              onStatusRefresh={handleStatusRefresh}
            />
          )}
          {localModelStatus &&
            (localModelStatus.ollama.models.length > 0 ||
              localModelStatus.lmStudio.models.length > 0) && (
              <ModelManagement
                status={localModelStatus}
                currentModel={selectedProvider === 'ollama' ? model : undefined}
                onModelSelect={(selectedModel) => {
                  setModel(selectedModel);
                  setSelectedProvider('ollama');
                }}
              />
            )}
        </section>

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
                <label className="form-label">Ollama Base URL</label>
                <input
                  type="text"
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  placeholder="http://localhost:11435"
                  className="key-input"
                  disabled={validating}
                />
                <p className="info-text">
                  Moly detects Ollama at localhost:11434. <br />
                  Optional CORS proxy at 11435 for better performance. <br />
                  Moly uses whichever connection is available.
                </p>
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
              <button
                onClick={handleSaveProvider}
                disabled={validating || (selectedProvider !== 'ollama' ? !apiKey.trim() : !baseUrl.trim())}
                className="btn btn-primary"
              >
                {validating ? 'Validating...' : 'Save & Validate'}
              </button>
              {isConfigured && (
                <button onClick={() => handleSetActiveProvider(selectedProvider)} className="btn btn-secondary">
                  Make Active
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
          <h2>Preferences</h2>

          <div className="preferences-group">
            <div className="preference-item">
              <label className="form-label">Chat Mode</label>
              <p className="preference-description">Choose how Moly assists you with messages</p>
              <div className="option-buttons">
                <button
                  className={`option-btn ${chatMode === 'socratic' ? 'active' : ''}`}
                  onClick={() => handleSaveChatMode('socratic')}
                >
                  Socratic
                  <span className="option-hint">Guiding questions to refine your message</span>
                </button>
                <button
                  className={`option-btn ${chatMode === 'direct' ? 'active' : ''}`}
                  onClick={() => handleSaveChatMode('direct')}
                >
                  Direct
                  <span className="option-hint">Ready-to-use message suggestions</span>
                </button>
              </div>
            </div>

            <div className="preference-item">
              <label className="form-label">Communication Context</label>
              <p className="preference-description">Set the default tone for message suggestions</p>
              <div className="option-buttons">
                <button
                  className={`option-btn ${communicationContext === 'formal' ? 'active' : ''}`}
                  onClick={() => handleSaveCommunicationContext('formal')}
                >
                  Formal
                  <span className="option-hint">Professional and respectful</span>
                </button>
                <button
                  className={`option-btn ${communicationContext === 'friendly' ? 'active' : ''}`}
                  onClick={() => handleSaveCommunicationContext('friendly')}
                >
                  Friendly
                  <span className="option-hint">Warm and approachable</span>
                </button>
                <button
                  className={`option-btn ${communicationContext === 'dating' ? 'active' : ''}`}
                  onClick={() => handleSaveCommunicationContext('dating')}
                >
                  Dating
                  <span className="option-hint">Flirty and romantic</span>
                </button>
              </div>
            </div>
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
