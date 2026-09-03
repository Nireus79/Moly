import React, { useState, useEffect } from 'react';
import './SetupWizard.css';

export default function SetupWizard({ onComplete }) {
  const [step, setStep] = useState('welcome');
  const [installing, setInstalling] = useState(false);
  const [installStatus, setInstallStatus] = useState('');
  const [error, setError] = useState('');
  const [proxyRunning, setProxyRunning] = useState(false);
  const [useLocalModels, setUseLocalModels] = useState(null);

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

  async function handleInstallLocal() {
    setInstalling(true);
    setError('');
    setInstallStatus('Starting installation...');

    try {
      setInstallStatus('Downloading native host...');
      const result = await window.moly.installNativeHost();

      if (result.success) {
        setInstallStatus('Waiting for CORS proxy to start...');

        // Retry proxy check up to 5 times with 1 second delay
        let proxyReady = false;
        for (let i = 0; i < 5; i++) {
          await new Promise(resolve => setTimeout(resolve, 1000));
          try {
            const proxyStatus = await window.moly.getProxyStatus();
            if (proxyStatus) {
              proxyReady = true;
              break;
            }
          } catch (e) {
            console.log(`Proxy check ${i + 1}/5 failed, retrying...`);
          }
        }

        if (proxyReady) {
          setProxyRunning(true);
          setInstallStatus('Installation complete!');
          setTimeout(() => {
            setStep('complete');
          }, 1500);
        } else {
          throw new Error('CORS proxy failed to start. Make sure Ollama is running on your system.');
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
          <button className="btn-primary" onClick={() => setStep('choice')}>
            Get Started
          </button>
        </div>
      )}

      {step === 'choice' && (
        <div className="setup-step">
          <h2>How do you want to use Moly?</h2>
          <p>Choose your preferred AI provider</p>

          <div className="setup-box">
            <h3>Option 1: Local Models</h3>
            <p>Use Ollama (free, private, offline-capable)</p>
            <p style={{fontSize: '12px', color: '#666'}}>Requires Ollama to be installed and running on your machine</p>
            <button className="btn-primary" onClick={() => { setUseLocalModels(true); setStep('setup'); }}>
              Use Local Models
            </button>
          </div>

          <div className="setup-box">
            <h3>Option 2: Cloud APIs</h3>
            <p>Use Claude or OpenAI (requires API key)</p>
            <p style={{fontSize: '12px', color: '#666'}}>Pay as you go, works without local setup</p>
            <button className="btn-primary" onClick={() => { setUseLocalModels(false); setStep('complete'); }}>
              Use Cloud APIs
            </button>
          </div>
        </div>
      )}

      {step === 'setup' && useLocalModels && (
        <div className="setup-step">
          <h2>Local Model Setup</h2>
          <p>Installing CORS proxy for Ollama...</p>

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
            onClick={handleInstallLocal}
            disabled={installing}
          >
            {installing ? 'Installing...' : 'Install Now'}
          </button>

          {!installing && error && (
            <button className="btn-secondary" onClick={handleInstallLocal}>
              Retry Installation
            </button>
          )}
        </div>
      )}

      {step === 'complete' && (
        <div className="setup-step">
          <div className="success-icon">✓</div>
          <h2>Ready to Go!</h2>
          <p>Moly is configured and ready to use.</p>

          <div className="setup-box">
            <h3>What's next:</h3>
            {useLocalModels ? (
              <ul>
                <li>Make sure Ollama is running</li>
                <li>Go to Settings to verify Ollama connection</li>
                <li>Start chatting with local models</li>
              </ul>
            ) : (
              <ul>
                <li>Go to Settings and enter your Claude or OpenAI API key</li>
                <li>Select your preferred provider</li>
                <li>Start chatting</li>
              </ul>
            )}
          </div>

          <button className="btn-primary" onClick={handleComplete}>
            Launch Moly
          </button>
        </div>
      )}
    </div>
  );
}
