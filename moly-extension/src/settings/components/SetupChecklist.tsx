import React, { useState } from 'react';
import type { LocalModelStatus } from '@/api/detection';

interface SetupChecklistProps {
  status: LocalModelStatus;
  onSetupComplete?: () => void;
}

export const SetupChecklist: React.FC<SetupChecklistProps> = ({
  status,
  onSetupComplete,
}) => {
  const [isInstalling, setIsInstalling] = useState(false);
  const [installMessage, setInstallMessage] = useState('');

  const handleInstallAll = async () => {
    setIsInstalling(true);
    setInstallMessage('Starting setup...');

    try {
      // Call native host to set up everything
      const result = await new Promise<any>((resolve) => {
        chrome.runtime.sendNativeMessage(
          'com.moly.native_host',
          { action: 'setup-all' },
          (response) => {
            if (chrome.runtime.lastError) {
              resolve({ success: false, error: chrome.runtime.lastError.message });
            } else {
              resolve(response || { success: false, error: 'No response' });
            }
          }
        );
      });

      if (result.success) {
        setInstallMessage('Setup complete! Refresh the page.');
        setTimeout(() => {
          window.location.reload();
        }, 2000);
      } else {
        setInstallMessage(`Error: ${result.error || 'Setup failed'}`);
      }
    } catch (error) {
      setInstallMessage(
        `Error: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    } finally {
      setIsInstalling(false);
    }
  };

  if (status.components.allConfigured) {
    return (
      <div style={{
        padding: '16px',
        background: '#e8f5e9',
        border: '1px solid #2e7d32',
        borderRadius: '6px',
        marginBottom: '16px',
      }}>
        <div style={{
          color: '#2e7d32',
          fontWeight: '600',
          marginBottom: '8px',
        }}>
          ✓ All Set Up!
        </div>
        <div style={{ fontSize: '12px', color: '#1b5e20' }}>
          Moly is ready to use. Configure your preferred AI provider in the settings above.
        </div>
      </div>
    );
  }

  return (
    <div style={{
      padding: '16px',
      background: '#fff3cd',
      border: '1px solid #856404',
      borderRadius: '6px',
      marginBottom: '16px',
    }}>
      <div style={{
        color: '#856404',
        fontWeight: '600',
        marginBottom: '12px',
      }}>
        Setup Required
      </div>

      <div style={{ marginBottom: '12px' }}>
        {status.components.needsSetup.length > 0 && (
          <div style={{ fontSize: '13px', color: '#333', marginBottom: '8px' }}>
            Missing components:
          </div>
        )}
        {!status.nativeHost.installed && (
          <div style={{ fontSize: '12px', color: '#555', marginLeft: '16px', marginBottom: '4px' }}>
            • Native messaging host
          </div>
        )}
        {!status.ollama.installed && (
          <div style={{ fontSize: '12px', color: '#555', marginLeft: '16px', marginBottom: '4px' }}>
            • Ollama (local AI)
          </div>
        )}
        {status.ollama.installed && !status.corsProxy.running && (
          <div style={{ fontSize: '12px', color: '#555', marginLeft: '16px', marginBottom: '4px' }}>
            • CORS proxy (browser communication)
          </div>
        )}
        {!status.cloudProviders.claude &&
          !status.cloudProviders.openai &&
          !status.ollama.installed && (
            <div style={{ fontSize: '12px', color: '#555', marginLeft: '16px', marginBottom: '4px' }}>
              • AI provider (Claude API or OpenAI)
            </div>
          )}
      </div>

      <button
        onClick={handleInstallAll}
        disabled={isInstalling}
        style={{
          width: '100%',
          padding: '10px',
          background: '#ffc107',
          color: '#333',
          border: 'none',
          borderRadius: '4px',
          fontWeight: '600',
          cursor: isInstalling ? 'not-allowed' : 'pointer',
          opacity: isInstalling ? 0.7 : 1,
          marginBottom: '8px',
        }}
      >
        {isInstalling ? 'Installing...' : 'Install Everything'}
      </button>

      {installMessage && (
        <div style={{
          fontSize: '12px',
          color: installMessage.includes('Error') ? '#d32f2f' : '#2e7d32',
          background: installMessage.includes('Error') ? '#ffebee' : '#e8f5e9',
          padding: '8px',
          borderRadius: '3px',
          marginTop: '8px',
        }}>
          {installMessage}
        </div>
      )}

      <div style={{
        fontSize: '11px',
        color: '#666',
        marginTop: '12px',
        lineHeight: '1.4',
      }}>
        Moly will automatically detect and install missing components for your operating system.
        This requires admin/sudo access.
      </div>
    </div>
  );
};

export default SetupChecklist;
