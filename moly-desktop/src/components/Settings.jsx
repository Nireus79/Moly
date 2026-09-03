import React, { useState } from 'react';
import './Settings.css';

export default function Settings({ selectedModel, provider, onProviderChange, onModelChange }) {
  const [claudeKey, setClaudeKey] = useState('');
  const [openaiKey, setOpenaiKey] = useState('');
  const [proxyStatus, setProxyStatus] = useState('checking...');

  React.useEffect(() => {
    checkProxy();
  }, []);

  async function checkProxy() {
    try {
      const status = await window.moly.getProxyStatus();
      setProxyStatus(status ? '✓ Running' : '✗ Not running');
    } catch {
      setProxyStatus('✗ Error');
    }
  }

  function handleSaveSettings() {
    // Save to localStorage
    localStorage.setItem('moly-claude-key', claudeKey);
    localStorage.setItem('moly-openai-key', openaiKey);
    localStorage.setItem('moly-provider', provider);
    alert('Settings saved!');
  }

  return (
    <div className="settings">
      <h2>Settings</h2>

      <div className="settings-section">
        <h3>System Status</h3>
        <div className="status-item">
          <span>CORS Proxy:</span>
          <span className={`status ${proxyStatus.includes('Running') ? 'running' : 'stopped'}`}>
            {proxyStatus}
          </span>
        </div>
        <button onClick={checkProxy}>Refresh Status</button>
      </div>

      <div className="settings-section">
        <h3>AI Provider</h3>
        <div className="radio-group">
          <label>
            <input
              type="radio"
              value="ollama"
              checked={provider === 'ollama'}
              onChange={e => onProviderChange(e.target.value)}
            />
            Local Model (Ollama)
          </label>
          <label>
            <input
              type="radio"
              value="claude"
              checked={provider === 'claude'}
              onChange={e => onProviderChange(e.target.value)}
            />
            Claude API
          </label>
          <label>
            <input
              type="radio"
              value="openai"
              checked={provider === 'openai'}
              onChange={e => onProviderChange(e.target.value)}
            />
            OpenAI
          </label>
        </div>
      </div>

      {provider === 'ollama' && (
        <div className="settings-section">
          <h3>Local Model</h3>
          <label>
            Model:
            <select value={selectedModel} onChange={e => onModelChange(e.target.value)}>
              <option value="mistral">Mistral</option>
              <option value="neural-chat">Neural Chat</option>
              <option value="llama2">Llama 2</option>
            </select>
          </label>
          <p className="help-text">Requires Ollama installed and running</p>
        </div>
      )}

      {provider === 'claude' && (
        <div className="settings-section">
          <h3>Claude API</h3>
          <label>
            API Key:
            <input
              type="password"
              value={claudeKey}
              onChange={e => setClaudeKey(e.target.value)}
              placeholder="sk-ant-..."
            />
          </label>
          <p className="help-text">Get your API key from <a href="#" onClick={() => {}}>console.anthropic.com</a></p>
        </div>
      )}

      {provider === 'openai' && (
        <div className="settings-section">
          <h3>OpenAI API</h3>
          <label>
            API Key:
            <input
              type="password"
              value={openaiKey}
              onChange={e => setOpenaiKey(e.target.value)}
              placeholder="sk-..."
            />
          </label>
          <p className="help-text">Get your API key from <a href="#" onClick={() => {}}>platform.openai.com</a></p>
        </div>
      )}

      <div className="settings-actions">
        <button className="btn-primary" onClick={handleSaveSettings}>
          Save Settings
        </button>
      </div>

      <div className="settings-section about">
        <h3>About Moly</h3>
        <p>Version 1.0.0</p>
        <p>AI Coaching Chatbot for Better Messages</p>
      </div>
    </div>
  );
}
