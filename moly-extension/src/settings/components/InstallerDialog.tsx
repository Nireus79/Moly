import React, { useEffect, useState } from 'react';
import {
  detectPlatform,
  getInstallerStatus,
  completeSetupAfterInstall,
  openInstallerPage,
  type InstallerStatus,
} from '@/api/installerLauncher';
import { ModelSelectionDialog } from './ModelSelectionDialog';
import { NativeHostAutoDownloader } from './NativeHostAutoDownloader';

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
  const [showModelSelection, setShowModelSelection] = useState(false);
  const [showAutoDownloader, setShowAutoDownloader] = useState(false);

  useEffect(() => {
    const checkStatus = async () => {
      const installerStatus = await getInstallerStatus(platform);
      setStatus(installerStatus);
    };
    checkStatus();
  }, [platform]);

  const handleStartSetup = () => {
    // Show auto-downloader which handles everything
    setShowAutoDownloader(true);
  };

  const handleAutoDownloaderSuccess = async () => {
    // Native host is now installed, complete the setup
    setShowAutoDownloader(false);

    try {
      const result = await completeSetupAfterInstall(chrome.runtime.id);

      if (result.success) {
        // Setup complete, show model selection
        setShowModelSelection(true);
      }
    } catch (error) {
      console.error('Setup completion failed:', error);
    }
  };

  // Show auto-downloader if needed
  if (showAutoDownloader) {
    return (
      <NativeHostAutoDownloader
        onSuccess={handleAutoDownloaderSuccess}
        onClose={() => setShowAutoDownloader(false)}
      />
    );
  }

  // Show model selection after successful setup
  if (showModelSelection) {
    return (
      <ModelSelectionDialog
        onClose={() => {
          setShowModelSelection(false);
          onClose?.();
        }}
        onSuccess={() => {
          setShowModelSelection(false);
          onSuccess?.();
          onClose?.();
        }}
      />
    );
  }

  // Wait for status
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
          padding: '32px',
          maxWidth: '500px',
          boxShadow: '0 10px 40px rgba(0,0,0,0.2)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
          <div style={{ fontSize: '20px', fontWeight: '600', color: '#333' }}>
            Set Up Moly
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
            padding: '16px',
            background: '#e3f2fd',
            border: '1px solid #90caf9',
            borderRadius: '6px',
            marginBottom: '24px',
            fontSize: '13px',
            color: '#1565c0',
            lineHeight: '1.6',
          }}
        >
          Moly needs a native host installed to manage local models and services. This will be downloaded
          automatically and installed to your system.
        </div>

        <div style={{ display: 'flex', gap: '8px', marginBottom: '16px' }}>
          <button
            onClick={handleStartSetup}
            style={{
              flex: 1,
              padding: '12px 16px',
              fontSize: '14px',
              background: '#1976d2',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontWeight: '600',
            }}
          >
            Start Setup
          </button>

          <button
            onClick={() => onClose?.()}
            style={{
              flex: 1,
              padding: '12px 16px',
              fontSize: '14px',
              background: '#f5f5f5',
              color: '#666',
              border: '1px solid #ddd',
              borderRadius: '4px',
              cursor: 'pointer',
              fontWeight: '500',
            }}
          >
            Cancel
          </button>
        </div>

        <div
          style={{
            fontSize: '12px',
            color: '#999',
            lineHeight: '1.5',
            paddingTop: '16px',
            borderTop: '1px solid #e0e0e0',
          }}
        >
          The setup process includes:
          <div style={{ marginTop: '8px', marginLeft: '16px' }}>
            • Auto-download native host binary
            <br />
            • Extract and install to system
            <br />
            • Install optional CORS proxy (via npm)
            <br />
            • Detect local models
          </div>
        </div>
      </div>
    </div>
  );
};

export default InstallerDialog;
