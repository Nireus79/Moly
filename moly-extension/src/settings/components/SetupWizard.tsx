import React, { useState, useEffect } from 'react';
import type { LocalModelStatus } from '@/api/detection';
import type { LLMProviderType } from '@/api/providers';
import { detectPlatform, downloadNativeHost } from '@/api/installerLauncher';

interface SetupWizardProps {
  status: LocalModelStatus;
  onSetupComplete?: () => void;
  onCancel?: () => void;
}

type WizardStep = 'welcome' | 'components' | 'provider' | 'model' | 'verify' | 'complete';

export const SetupWizard: React.FC<SetupWizardProps> = ({
  status,
  onSetupComplete,
  onCancel,
}) => {
  const [step, setStep] = useState<WizardStep>('welcome');
  const [selectedProvider, setSelectedProvider] = useState<LLMProviderType>('ollama');
  const [apiKey, setApiKey] = useState('');
  const [selectedModel, setSelectedModel] = useState('mistral');
  const [isInstalling, setIsInstalling] = useState(false);
  const [installMessage, setInstallMessage] = useState('');
  const [discoveredModels, setDiscoveredModels] = useState<string[]>([]);

  // Step 1: Welcome
  const StepWelcome = () => (
    <div style={{ textAlign: 'center', padding: '40px 20px' }}>
      <h2 style={{ fontSize: '28px', marginBottom: '12px', color: '#333' }}>
        Welcome to Moly
      </h2>
      <p style={{ fontSize: '14px', color: '#666', marginBottom: '24px', lineHeight: '1.6' }}>
        Let's set up your AI coaching assistant in just a few steps.
      </p>

      <div style={{
        background: '#f5f5f5',
        padding: '20px',
        borderRadius: '8px',
        marginBottom: '24px',
        textAlign: 'left',
      }}>
        <div style={{ fontSize: '13px', color: '#333', marginBottom: '12px' }}>
          <strong>Setup includes:</strong>
        </div>
        <div style={{ fontSize: '12px', color: '#666', lineHeight: '1.8' }}>
          ✓ Install Moly components<br/>
          ✓ Set up local AI (Ollama) or cloud API<br/>
          ✓ Configure your communication style<br/>
          ✓ Test everything is working
        </div>
      </div>

      <button
        onClick={() => setStep('components')}
        style={{
          padding: '12px 24px',
          background: '#6366f1',
          color: 'white',
          border: 'none',
          borderRadius: '6px',
          fontWeight: '600',
          cursor: 'pointer',
          marginBottom: '12px',
          width: '100%',
        }}
      >
        Get Started
      </button>

      {onCancel && (
        <button
          onClick={onCancel}
          style={{
            padding: '12px 24px',
            background: 'transparent',
            color: '#6366f1',
            border: '1px solid #6366f1',
            borderRadius: '6px',
            fontWeight: '600',
            cursor: 'pointer',
            width: '100%',
          }}
        >
          Skip for Now
        </button>
      )}
    </div>
  );

  // Step 2: Install Components
  const StepComponents = () => (
    <div style={{ padding: '20px' }}>
      <h3 style={{ marginBottom: '16px', color: '#333' }}>Step 1: Install Components</h3>

      <div style={{
        background: '#f5f5f5',
        padding: '16px',
        borderRadius: '6px',
        marginBottom: '16px',
        fontSize: '13px',
        color: '#666',
      }}>
        Missing components will be installed on your system:
        <ul style={{ marginTop: '8px', marginBottom: '0', paddingLeft: '20px' }}>
          {!status.nativeHost.installed && <li>Native messaging host</li>}
          {!status.ollama.installed && <li>Ollama (local AI runtime)</li>}
          {!status.corsProxy.running && <li>CORS proxy (browser communication)</li>}
        </ul>
      </div>

      <button
        onClick={async () => {
          setIsInstalling(true);
          setInstallMessage('Preparing installation...');

          try {
            // Check if native host is already installed
            const testResult = await new Promise<boolean>((resolve) => {
              const timeout = setTimeout(() => resolve(false), 2000);
              chrome.runtime.sendNativeMessage(
                'com.moly.native_host',
                { action: 'ping' },
                (response) => {
                  clearTimeout(timeout);
                  if (chrome.runtime.lastError) {
                    resolve(false);
                  } else {
                    resolve(response?.pong === true);
                  }
                }
              );
            });

            // If native host doesn't exist, download the installer
            if (!testResult) {
              setInstallMessage('Downloading installer...');
              const platform = detectPlatform();

              if (platform === 'unknown') {
                throw new Error('Unable to detect your operating system');
              }

              await downloadNativeHost(platform);

              let runCommand = '';
              if (platform === 'linux') {
                runCommand = 'bash ~/Downloads/moly-install-linux.sh';
              } else if (platform === 'macos') {
                runCommand = 'bash ~/Downloads/moly-install-macos.sh';
              } else if (platform === 'windows') {
                runCommand = 'Open PowerShell and run: C:\\Users\\YourName\\Downloads\\moly-install-windows.bat';
              }

              setInstallMessage(
                `✓ Installer downloaded to your Downloads folder.\n\n` +
                `Open a terminal and run:\n${runCommand}\n\n` +
                `After installation completes, click "Verify Installation" to continue.`
              );
              return;
            }

            // Native host exists, proceed with setup
            setInstallMessage('Setting up components via native host...');
            const result = await new Promise<any>((resolve) => {
              chrome.runtime.sendNativeMessage(
                'com.moly.native_host',
                { action: 'setup-all' },
                (response) => {
                  if (chrome.runtime.lastError) {
                    resolve({ success: false, error: chrome.runtime.lastError.message });
                  } else {
                    resolve(response || { success: false });
                  }
                }
              );
            });

            if (result.success) {
              setInstallMessage('✓ Components installed successfully');
              setTimeout(() => setStep('provider'), 2000);
            } else {
              setInstallMessage(`✗ Error: ${result.error || 'Installation failed'}`);
            }
          } catch (error) {
            setInstallMessage(
              `✗ Error: ${error instanceof Error ? error.message : 'Unknown error'}`
            );
          } finally {
            setIsInstalling(false);
          }
        }}
        disabled={isInstalling}
        style={{
          padding: '12px',
          background: '#2e7d32',
          color: 'white',
          border: 'none',
          borderRadius: '6px',
          fontWeight: '600',
          cursor: isInstalling ? 'not-allowed' : 'pointer',
          width: '100%',
          opacity: isInstalling ? 0.7 : 1,
          marginBottom: '12px',
        }}
      >
        {isInstalling ? 'Preparing...' : 'Install Components'}
      </button>

      {installMessage && (
        <div style={{
          fontSize: '12px',
          color: installMessage.includes('✓') && !installMessage.includes('run:') ? '#2e7d32' : '#d32f2f',
          background: installMessage.includes('✓') && !installMessage.includes('run:') ? '#e8f5e9' : '#fff3e0',
          padding: '12px',
          borderRadius: '4px',
          marginTop: '12px',
          whiteSpace: 'pre-wrap',
        }}>
          {installMessage}

          {installMessage.includes('Downloaded to your Downloads folder') && (
            <button
              onClick={async () => {
                setInstallMessage('Verifying installation...');
                const testResult = await new Promise<boolean>((resolve) => {
                  const timeout = setTimeout(() => resolve(false), 2000);
                  chrome.runtime.sendNativeMessage(
                    'com.moly.native_host',
                    { action: 'ping' },
                    (response) => {
                      clearTimeout(timeout);
                      if (chrome.runtime.lastError) {
                        resolve(false);
                      } else {
                        resolve(response?.pong === true);
                      }
                    }
                  );
                });

                if (testResult) {
                  setInstallMessage('✓ Installation verified successfully!');
                  setTimeout(() => setStep('provider'), 2000);
                } else {
                  setInstallMessage(
                    `✗ Native host still not found.\n\n` +
                    `Make sure you ran the installer script and it completed successfully.\n\n` +
                    `If you're still having issues, check the terminal output for errors.`
                  );
                }
              }}
              style={{
                marginTop: '12px',
                padding: '8px 16px',
                background: '#ff9800',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                fontSize: '12px',
                fontWeight: '600',
                cursor: 'pointer',
              }}
            >
              Verify Installation
            </button>
          )}
        </div>
      )}
    </div>
  );

  // Step 3: Choose Provider
  const StepProvider = () => (
    <div style={{ padding: '20px' }}>
      <h3 style={{ marginBottom: '16px', color: '#333' }}>Step 2: Choose AI Provider</h3>

      <div style={{
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: '12px',
        marginBottom: '16px',
      }}>
        {[
          { type: 'ollama' as LLMProviderType, name: 'Local Ollama', desc: 'Free, no API key' },
          { type: 'claude' as LLMProviderType, name: 'Claude API', desc: 'Cloud-based' },
          { type: 'openai' as LLMProviderType, name: 'OpenAI', desc: 'Cloud-based' },
        ].map((option) => (
          <button
            key={option.type}
            onClick={() => setSelectedProvider(option.type)}
            style={{
              padding: '16px',
              background: selectedProvider === option.type ? '#e3f2fd' : '#f5f5f5',
              border: selectedProvider === option.type ? '2px solid #1976d2' : '1px solid #ddd',
              borderRadius: '6px',
              cursor: 'pointer',
              textAlign: 'left',
            }}
          >
            <div style={{ fontWeight: '600', fontSize: '13px', color: '#333' }}>
              {option.name}
            </div>
            <div style={{ fontSize: '11px', color: '#666', marginTop: '4px' }}>
              {option.desc}
            </div>
          </button>
        ))}
      </div>

      {selectedProvider !== 'ollama' && (
        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px', color: '#333' }}>
            API Key
          </label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="Enter your API key"
            style={{
              width: '100%',
              padding: '10px',
              border: '1px solid #ddd',
              borderRadius: '4px',
              fontSize: '12px',
              boxSizing: 'border-box',
            }}
          />
          <div style={{ fontSize: '11px', color: '#666', marginTop: '6px' }}>
            {selectedProvider === 'claude'
              ? 'Get your API key from https://console.anthropic.com'
              : 'Get your API key from https://platform.openai.com'}
          </div>
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
        <button
          onClick={() => setStep('components')}
          style={{
            padding: '12px',
            background: 'transparent',
            color: '#6366f1',
            border: '1px solid #6366f1',
            borderRadius: '6px',
            fontWeight: '600',
            cursor: 'pointer',
          }}
        >
          Back
        </button>
        <button
          onClick={() => setStep('model')}
          disabled={selectedProvider !== 'ollama' && !apiKey}
          style={{
            padding: '12px',
            background: '#6366f1',
            color: 'white',
            border: 'none',
            borderRadius: '6px',
            fontWeight: '600',
            cursor: selectedProvider !== 'ollama' && !apiKey ? 'not-allowed' : 'pointer',
            opacity: selectedProvider !== 'ollama' && !apiKey ? 0.7 : 1,
          }}
        >
          Next
        </button>
      </div>
    </div>
  );

  // Step 4: Select Model
  const StepModel = () => (
    <div style={{ padding: '20px' }}>
      <h3 style={{ marginBottom: '16px', color: '#333' }}>Step 3: Select Model</h3>

      {selectedProvider === 'ollama' && (
        <div style={{
          background: '#e8f5e9',
          padding: '12px',
          borderRadius: '6px',
          marginBottom: '16px',
          fontSize: '12px',
          color: '#2e7d32',
        }}>
          ✓ Ollama is ready with {status.ollama.models.length} model(s) installed
        </div>
      )}

      <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px', color: '#333' }}>
        {selectedProvider === 'ollama' ? 'Available Models' : 'Model'}
      </label>

      {selectedProvider === 'ollama' ? (
        <select
          value={selectedModel}
          onChange={(e) => setSelectedModel(e.target.value)}
          style={{
            width: '100%',
            padding: '10px',
            border: '1px solid #ddd',
            borderRadius: '4px',
            fontSize: '12px',
            marginBottom: '16px',
            boxSizing: 'border-box',
          }}
        >
          {status.ollama.models.map((model) => (
            <option key={model} value={model}>
              {model}
            </option>
          ))}
        </select>
      ) : (
        <div style={{ fontSize: '12px', color: '#666', marginBottom: '16px' }}>
          Using {selectedProvider === 'claude' ? 'claude-3-5-sonnet' : 'gpt-4-turbo'}
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
        <button
          onClick={() => setStep('provider')}
          style={{
            padding: '12px',
            background: 'transparent',
            color: '#6366f1',
            border: '1px solid #6366f1',
            borderRadius: '6px',
            fontWeight: '600',
            cursor: 'pointer',
          }}
        >
          Back
        </button>
        <button
          onClick={() => setStep('verify')}
          style={{
            padding: '12px',
            background: '#6366f1',
            color: 'white',
            border: 'none',
            borderRadius: '6px',
            fontWeight: '600',
            cursor: 'pointer',
          }}
        >
          Next
        </button>
      </div>
    </div>
  );

  // Step 5: Verify & Test
  const StepVerify = () => (
    <div style={{ padding: '20px' }}>
      <h3 style={{ marginBottom: '16px', color: '#333' }}>Step 4: Verify Setup</h3>

      <div style={{
        background: '#f5f5f5',
        padding: '16px',
        borderRadius: '6px',
        marginBottom: '16px',
        fontSize: '12px',
        color: '#666',
        lineHeight: '1.6',
      }}>
        <strong>Setup Summary:</strong><br/>
        Provider: {selectedProvider === 'ollama' ? 'Ollama (Local)' : selectedProvider.toUpperCase()}<br/>
        Model: {selectedModel}<br/>
        Status: Ready to test
      </div>

      <button
        onClick={() => setStep('complete')}
        style={{
          padding: '12px',
          background: '#2e7d32',
          color: 'white',
          border: 'none',
          borderRadius: '6px',
          fontWeight: '600',
          cursor: 'pointer',
          width: '100%',
          marginBottom: '12px',
        }}
      >
        Complete Setup
      </button>

      <button
        onClick={() => setStep('model')}
        style={{
          padding: '12px',
          background: 'transparent',
          color: '#6366f1',
          border: '1px solid #6366f1',
          borderRadius: '6px',
          fontWeight: '600',
          cursor: 'pointer',
          width: '100%',
        }}
      >
        Back
      </button>
    </div>
  );

  // Step 6: Complete
  const StepComplete = () => (
    <div style={{ textAlign: 'center', padding: '40px 20px' }}>
      <div style={{ fontSize: '48px', marginBottom: '16px' }}>✓</div>
      <h2 style={{ fontSize: '24px', color: '#2e7d32', marginBottom: '12px' }}>
        Setup Complete!
      </h2>
      <p style={{ fontSize: '13px', color: '#666', marginBottom: '24px' }}>
        Moly is ready to help you craft better messages.
      </p>

      <button
        onClick={() => {
          onSetupComplete?.();
          window.location.reload();
        }}
        style={{
          padding: '12px 24px',
          background: '#6366f1',
          color: 'white',
          border: 'none',
          borderRadius: '6px',
          fontWeight: '600',
          cursor: 'pointer',
          width: '100%',
        }}
      >
        Start Using Moly
      </button>
    </div>
  );

  // Render current step
  const renderStep = () => {
    switch (step) {
      case 'welcome':
        return <StepWelcome />;
      case 'components':
        return <StepComponents />;
      case 'provider':
        return <StepProvider />;
      case 'model':
        return <StepModel />;
      case 'verify':
        return <StepVerify />;
      case 'complete':
        return <StepComplete />;
    }
  };

  return (
    <div style={{
      maxWidth: '600px',
      margin: '0 auto',
      background: 'white',
      borderRadius: '8px',
      boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
      overflow: 'hidden',
    }}>
      {/* Progress bar */}
      {step !== 'welcome' && step !== 'complete' && (
        <div style={{ height: '3px', background: '#ddd' }}>
          <div
            style={{
              height: '100%',
              background: '#6366f1',
              width: `${
                step === 'components' ? '25%' :
                step === 'provider' ? '50%' :
                step === 'model' ? '75%' :
                '100%'
              }`,
              transition: 'width 0.3s ease',
            }}
          />
        </div>
      )}

      {renderStep()}
    </div>
  );
};

export default SetupWizard;
