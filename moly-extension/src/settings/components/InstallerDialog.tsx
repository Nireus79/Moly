import React, { useEffect, useState } from 'react';
import {
  detectPlatform,
  getInstallerStatus,
  downloadNativeHost,
  orchestrateSetup,
  completeSetupAfterInstall,
  testNativeHost,
  openInstallerPage,
  type InstallerStatus,
} from '@/api/installerLauncher';

interface InstallerDialogProps {
  onClose?: () => void;
  onSuccess?: () => void;
}

export const InstallerDialog: React.FC<InstallerDialogProps> = ({
  onClose,
  onSuccess,
}) => {
  const [platform] = useState(() => detectPlatform());
  const [status, setStatus] = useState<InstallerStatus | null>(null);
  const [isLaunching, setIsLaunching] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    const checkStatus = async () => {
      const installerStatus = await getInstallerStatus(platform);
      setStatus(installerStatus);
    };
    checkStatus();
  }, [platform]);

  const handleStartSetup = async () => {
    setIsLaunching(true);
    setMessage('Starting setup orchestration...');

    try {
      const result = await orchestrateSetup(platform, chrome.runtime.id);

      if (result.success && result.step === 'already-installed') {
        setMessage('Native host already installed! Ready to use.');
        setTimeout(() => {
          onSuccess?.();
          onClose?.();
        }, 2000);
      } else if (result.success && result.step === 'downloaded') {
        setMessage('Binary downloaded! Run the downloaded file to complete setup.');
        setTimeout(() => setMessage(''), 5000);
      } else {
        setMessage(result.error || 'Setup failed');
      }
    } catch (error) {
      setMessage(
        `Error: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    } finally {
      setIsLaunching(false);
    }
  };

  const handleVerifyAfterRun = async () => {
    setIsLaunching(true);
    setMessage('Verifying installation...');

    try {
      const result = await completeSetupAfterInstall(chrome.runtime.id);

      if (result.success) {
        setMessage('Setup complete! Services are configured.');
        setTimeout(() => {
          onSuccess?.();
          onClose?.();
        }, 2000);
      } else {
        setMessage(result.error || 'Verification failed. Try running the binary again.');
      }
    } catch (error) {
      setMessage(
        `Error: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    } finally {
      setIsLaunching(false);
    }
  };

  const handleDownloadOnly = async () => {
    setIsLaunching(true);
    setMessage('Starting download...');

    try {
      await downloadNativeHost(platform);
      setMessage('Download started! Check your Downloads folder.');
      setTimeout(() => setMessage(''), 3000);
    } catch (error) {
      setMessage(
        `Error: ${error instanceof Error ? error.message : 'Download failed'}`
      );
    } finally {
      setIsLaunching(false);
    }
  };

  if (!status) {
    return (
      <div
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(0,0,0,0.5)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 10000,
        }}
      >
        <div
          style={{
            background: 'white',
            borderRadius: '8px',
            padding: '24px',
            maxWidth: '500px',
            boxShadow: '0 10px 40px rgba(0,0,0,0.2)',
          }}
        >
          <div style={{ fontSize: '14px', color: '#666' }}>
            Detecting platform...
          </div>
        </div>
      </div>
    );
  }

  const platformName =
    status.platform === 'macos'
      ? 'macOS'
      : status.platform === 'linux'
        ? 'Linux'
        : status.platform === 'windows'
          ? 'Windows'
          : 'Your Platform';

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 10000,
      }}
      onClick={() => onClose?.()}
    >
      <div
        style={{
          background: 'white',
          borderRadius: '8px',
          padding: '24px',
          maxWidth: '500px',
          boxShadow: '0 10px 40px rgba(0,0,0,0.2)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <div style={{ fontSize: '18px', fontWeight: '600', color: '#333' }}>
            Install Moly
          </div>
          <button
            onClick={() => onClose?.()}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '24px',
              cursor: 'pointer',
              color: '#999',
            }}
          >
            ×
          </button>
        </div>

        <div
          style={{
            fontSize: '13px',
            color: '#666',
            marginBottom: '16px',
            lineHeight: '1.6',
          }}
        >
          <strong>Detected Platform: {platformName}</strong>
          <br />
          {status.platform !== 'unknown'
            ? 'Follow the steps below to install Ollama and set up Moly for local AI.'
            : 'Platform detection failed. Please visit the releases page.'}
        </div>

        {/* Instructions */}
        <div
          style={{
            background: '#f9f9f9',
            border: '1px solid #e0e0e0',
            borderRadius: '6px',
            padding: '16px',
            marginBottom: '16px',
          }}
        >
          <div
            style={{
              fontSize: '12px',
              fontWeight: '600',
              color: '#333',
              marginBottom: '8px',
            }}
          >
            Setup Steps:
          </div>
          <ol
            style={{
              fontSize: '12px',
              color: '#666',
              lineHeight: '1.6',
              margin: 0,
              paddingLeft: '20px',
            }}
          >
            {status.instructions?.map((instruction, idx) => (
              <li key={idx} style={{ marginBottom: '4px' }}>
                {instruction}
              </li>
            ))}
          </ol>
        </div>

        {message && (
          <div
            style={{
              fontSize: '12px',
              color: message.includes('Error') ? '#d32f2f' : '#2e7d32',
              background:
                message.includes('Error') ? '#ffebee' : '#e8f5e9',
              padding: '8px 12px',
              borderRadius: '4px',
              marginBottom: '16px',
            }}
          >
            {message}
          </div>
        )}

        {/* Action Buttons */}
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          <button
            onClick={handleStartSetup}
            disabled={isLaunching}
            style={{
              flex: 1,
              minWidth: '140px',
              padding: '10px 16px',
              fontSize: '13px',
              background: '#1976d2',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: isLaunching ? 'not-allowed' : 'pointer',
              fontWeight: '600',
              opacity: isLaunching ? 0.7 : 1,
            }}
          >
            {isLaunching ? 'Setting up...' : 'Download Setup'}
          </button>

          {message.includes('Run the downloaded file') && (
            <button
              onClick={handleVerifyAfterRun}
              disabled={isLaunching}
              style={{
                flex: 1,
                minWidth: '140px',
                padding: '10px 16px',
                fontSize: '13px',
                background: '#2e7d32',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: isLaunching ? 'not-allowed' : 'pointer',
                fontWeight: '600',
                opacity: isLaunching ? 0.7 : 1,
              }}
            >
              {isLaunching ? 'Verifying...' : 'Verify Setup'}
            </button>
          )}

          <button
            onClick={() => openInstallerPage()}
            style={{
              flex: 1,
              minWidth: '140px',
              padding: '10px 16px',
              fontSize: '13px',
              background: '#f5f5f5',
              color: '#666',
              border: '1px solid #ddd',
              borderRadius: '4px',
              cursor: 'pointer',
              fontWeight: '500',
            }}
          >
            Help
          </button>

          <button
            onClick={() => onClose?.()}
            style={{
              flex: 1,
              minWidth: '140px',
              padding: '10px 16px',
              fontSize: '13px',
              background: 'white',
              color: '#666',
              border: '1px solid #ddd',
              borderRadius: '4px',
              cursor: 'pointer',
              fontWeight: '500',
            }}
          >
            Close
          </button>
        </div>

        {/* Info Text */}
        <div
          style={{
            marginTop: '16px',
            fontSize: '11px',
            color: '#999',
            lineHeight: '1.5',
            borderTop: '1px solid #e0e0e0',
            paddingTop: '12px',
          }}
        >
          The installer will download Ollama and set up automatic startup on
          your system. Moly will detect the installation automatically.
        </div>
      </div>
    </div>
  );
};

export default InstallerDialog;
