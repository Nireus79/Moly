import React, { useState, useEffect } from 'react';
import './SetupWizard.css';

export default function SetupWizard({ onComplete }) {
  const [step, setStep] = useState('welcome');
  const [installing, setInstalling] = useState(false);
  const [installStatus, setInstallStatus] = useState('');
  const [error, setError] = useState('');
  const [proxyRunning, setProxyRunning] = useState(false);

  async function handleInstall() {
    setInstalling(true);
    setError('');
    setInstallStatus('Starting installation...');

    try {
      setInstallStatus('Downloading native host...');
      const result = await window.moly.installNativeHost();

      if (result.success) {
        setInstallStatus('Verifying installation...');
        await new Promise(resolve => setTimeout(resolve, 2000));

        const proxyStatus = await window.moly.getProxyStatus();
        if (proxyStatus) {
          setProxyRunning(true);
          setInstallStatus('Installation complete!');
          setTimeout(() => {
            setStep('complete');
          }, 1500);
        } else {
          throw new Error('CORS proxy not responding');
        }
      } else {
        throw new Error(result.error || 'Installation failed');
      }
    } catch (err) {
      setError(err.message);
      setInstallStatus('Installation failed');
    } finally {
      setInstalling(false);
    }
  }

  async function handleComplete() {
    onComplete();
  }

  return (
    <div className="setup-wizard">
      {step === 'welcome' && (
        <div className="setup-step">
          <h1>Welcome to Moly</h1>
          <p>AI Coaching Chatbot</p>
          <div className="welcome-content">
            <p>Moly helps you craft better messages through intelligent dialogue.</p>
            <ul>
              <li>Works with local models (Ollama)</li>
              <li>Works with cloud APIs (Claude, OpenAI)</li>
              <li>Complete privacy - your data stays local</li>
            </ul>
          </div>
          <button className="btn-primary" onClick={() => setStep('setup')}>
            Get Started
          </button>
        </div>
      )}

      {step === 'setup' && (
        <div className="setup-step">
          <h2>System Setup</h2>
          <p>Installing required components...</p>

          <div className="setup-box">
            <h3>What's being installed:</h3>
            <ul>
              <li>Native messaging host</li>
              <li>CORS proxy for local models</li>
              <li>Automatic service startup</li>
            </ul>
          </div>

          {error && (
            <div className="error-box">
              <p><strong>Error:</strong> {error}</p>
            </div>
          )}

          {installStatus && (
            <div className="status-box">
              <p>{installStatus}</p>
              {installing && <div className="spinner"></div>}
            </div>
          )}

          <button
            className="btn-primary"
            onClick={handleInstall}
            disabled={installing}
          >
            {installing ? 'Installing...' : 'Install Now'}
          </button>

          {!installing && error && (
            <button className="btn-secondary" onClick={handleInstall}>
              Retry Installation
            </button>
          )}
        </div>
      )}

      {step === 'complete' && (
        <div className="setup-step">
          <div className="success-icon">✓</div>
          <h2>Setup Complete!</h2>
          <p>Moly is ready to use.</p>

          <div className="setup-box">
            <h3>What's next:</h3>
            <ul>
              <li>Configure your LLM provider (local or cloud)</li>
              <li>Start chatting with Moly</li>
              <li>Paste incoming messages for suggestions</li>
            </ul>
          </div>

          <button className="btn-primary" onClick={handleComplete}>
            Launch Moly
          </button>
        </div>
      )}
    </div>
  );
}
