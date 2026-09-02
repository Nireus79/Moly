import React, { useEffect, useState } from 'react';
import { detectPlatform, getNativeHostDownloadUrl, testNativeHost } from '@/api/installerLauncher';
import type { Platform } from '@/api/installerLauncher';

interface NativeHostAutoDownloaderProps {
  onSuccess?: () => void;
  onClose?: () => void;
}

export const NativeHostAutoDownloader: React.FC<NativeHostAutoDownloaderProps> = ({
  onSuccess,
  onClose,
}) => {
  const [platform] = useState<Platform>(() => detectPlatform());
  const [status, setStatus] = useState<
    'detecting' | 'ready-to-download' | 'downloading' | 'waiting-install' | 'checking' | 'success' | 'error'
  >('detecting');
  const [downloadUrl, setDownloadUrl] = useState('');
  const [message, setMessage] = useState('Checking for native host...');
  const [error, setError] = useState('');
  const [checkAttempts, setCheckAttempts] = useState(0);

  const platformName =
    platform === 'macos'
      ? 'macOS'
      : platform === 'linux'
        ? 'Linux'
        : platform === 'windows'
          ? 'Windows'
          : 'Your Platform';

  const downloadFileName =
    platform === 'macos'
      ? 'moly-install-macos.sh'
      : platform === 'linux'
        ? 'moly-install-linux.sh'
        : platform === 'windows'
          ? 'moly-install-windows.bat'
          : 'moly-install';

  // Initial check for native host
  useEffect(() => {
    const checkNativeHost = async () => {
      try {
        const isAvailable = await testNativeHost();
        if (isAvailable) {
          setStatus('success');
          setMessage('Native host already installed and ready!');
        } else {
          setStatus('ready-to-download');
          const url = getNativeHostDownloadUrl(platform);
          setDownloadUrl(url);
          setMessage(`Click below to download native host for ${platformName}`);
        }
      } catch (err) {
        setStatus('error');
        setError('Failed to check native host status');
      }
    };

    checkNativeHost();
  }, [platform]);

  const handleDownload = () => {
    if (!downloadUrl) {
      setError('Download URL not available');
      return;
    }

    setStatus('downloading');
    setMessage(`Downloading ${downloadFileName}...`);

    // Use Chrome downloads API (more reliable for extensions)
    chrome.downloads.download({
      url: downloadUrl,
      filename: downloadFileName,
      saveAs: false,
    }, (downloadId) => {
      if (chrome.runtime.lastError) {
        setStatus('error');
        setError(`Download failed: ${chrome.runtime.lastError.message}`);
        return;
      }

      // After a short delay, move to waiting state
      setTimeout(() => {
        setStatus('waiting-install');
        setMessage(
          platform === 'macos'
            ? 'Installer Downloaded!\n\n' +
              '1. Open Terminal\n' +
              '2. Run: chmod +x ~/Downloads/moly-install-macos.sh\n' +
              '3. Run: sudo ~/Downloads/moly-install-macos.sh\n' +
              '4. Enter your password when prompted\n' +
              '5. Wait for "Installation Complete!"\n\n' +
              'Then click "Verify Installation"'
            : platform === 'linux'
              ? 'Installer Downloaded!\n\n' +
                '1. Open Terminal (Ctrl+Alt+T)\n' +
                '2. Run: chmod +x ~/Downloads/moly-install-linux.sh\n' +
                '3. Run: sudo ~/Downloads/moly-install-linux.sh\n' +
                '4. Enter your password when prompted\n' +
                '5. Wait for "Installation Complete!"\n\n' +
                'Then click "Verify Installation"'
              : 'Installer Downloaded!\n\n' +
                '1. Open File Explorer (Windows key + E)\n' +
                '2. Go to Downloads\n' +
                '3. Right-click moly-install-windows.bat\n' +
                '4. Select "Run as administrator"\n' +
                '5. Click "Yes" if prompted\n' +
                '6. Wait for "Installation Complete!"\n\n' +
                'Then click "Verify Installation"'
        );
      }, 1000);
    });
  };


  const handleVerifyInstall = async () => {
    setStatus('checking');
    setMessage('Checking for installed native host...');
    setCheckAttempts((prev) => prev + 1);

    try {
      const isAvailable = await testNativeHost();
      if (isAvailable) {
        setStatus('success');
        setMessage('Native host installed successfully! Click "Continue" to proceed.');
      } else {
        if (checkAttempts < 2) {
          setStatus('waiting-install');
          setMessage(
            'Native host not detected yet. Make sure you:\n\n' +
              '1. Extracted the downloaded file\n' +
              '2. Ran the moly-native-host executable\n' +
              '3. Waited a few seconds\n\n' +
              'Then click "Verify Again"'
          );
        } else {
          setStatus('error');
          setError(
            'Native host not found after multiple verification attempts.\n\n' +
              'Troubleshooting:\n' +
              '1. Check installer output - should show "Installation Complete!"\n' +
              '2. Verify binary exists: /usr/local/bin/moly-native-host\n' +
              '3. Check native messaging config exists\n' +
              '4. Restart Chrome completely\n' +
              '5. Run installer again if needed\n\n' +
              'Contact support with installer output if issue persists'
          );
        }
      }
    } catch (err) {
      setStatus('error');
      setError('Error checking native host installation');
    }
  };

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: 'rgba(0, 0, 0, 0.5)',
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
          padding: '32px',
          maxWidth: '500px',
          width: '90%',
          maxHeight: '90vh',
          overflowY: 'auto',
          boxShadow: '0 10px 40px rgba(0, 0, 0, 0.2)',
        }}
      >
        {/* Title */}
        <div
          style={{
            fontSize: '20px',
            fontWeight: '600',
            color: '#333',
            marginBottom: '16px',
          }}
        >
          {status === 'success'
            ? 'All Set!'
            : status === 'error'
              ? 'Setup Failed'
              : 'Install Native Host'}
        </div>

        {/* Status Icon & Message */}
        <div
          style={{
            padding: '24px',
            background:
              status === 'success'
                ? '#e8f5e9'
                : status === 'error'
                  ? '#ffebee'
                  : '#f5f5f5',
            border:
              status === 'success'
                ? '1px solid #c8e6c9'
                : status === 'error'
                  ? '1px solid #ffcdd2'
                  : '1px solid #e0e0e0',
            borderRadius: '8px',
            marginBottom: '24px',
            minHeight: '120px',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            alignItems: 'center',
          }}
        >
          {status === 'detecting' || status === 'downloading' || status === 'checking' ? (
            <div
              style={{
                width: '100%',
                textAlign: 'center',
              }}
            >
              <div
                style={{
                  fontSize: '14px',
                  color: '#666',
                  marginBottom: '12px',
                }}
              >
                {message}
              </div>
              <div
                style={{
                  width: '100%',
                  height: '4px',
                  background: '#e0e0e0',
                  borderRadius: '2px',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    width: '33%',
                    height: '100%',
                    background: '#1976d2',
                    animation: 'pulse 1.5s infinite',
                  }}
                />
              </div>
            </div>
          ) : status === 'success' ? (
            <div style={{ textAlign: 'center' }}>
              <div
                style={{
                  fontSize: '32px',
                  marginBottom: '8px',
                }}
              >
                ✓
              </div>
              <div
                style={{
                  fontSize: '14px',
                  color: '#2e7d32',
                  fontWeight: '500',
                }}
              >
                {message}
              </div>
            </div>
          ) : status === 'error' ? (
            <div style={{ textAlign: 'center' }}>
              <div
                style={{
                  fontSize: '32px',
                  marginBottom: '8px',
                  color: '#d32f2f',
                }}
              >
                ✕
              </div>
              <div
                style={{
                  fontSize: '13px',
                  color: '#c62828',
                  lineHeight: '1.6',
                  whiteSpace: 'pre-line',
                }}
              >
                {error}
              </div>
            </div>
          ) : status === 'ready-to-download' ? (
            <div style={{ textAlign: 'center', width: '100%' }}>
              <div
                style={{
                  fontSize: '14px',
                  color: '#666',
                  marginBottom: '16px',
                }}
              >
                {message}
              </div>
              <div
                style={{
                  fontSize: '12px',
                  color: '#999',
                  background: '#fafafa',
                  padding: '8px 12px',
                  borderRadius: '4px',
                  fontFamily: 'monospace',
                  overflowX: 'auto',
                }}
              >
                {downloadFileName}
              </div>
            </div>
          ) : (
            <div
              style={{
                fontSize: '13px',
                color: '#666',
                lineHeight: '1.6',
                whiteSpace: 'pre-line',
              }}
            >
              {message}
            </div>
          )}
        </div>

        {/* Action Buttons */}
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          {status === 'ready-to-download' && (
            <>
              <button
                onClick={handleDownload}
                style={{
                  flex: 1,
                  minWidth: '140px',
                  padding: '10px 16px',
                  fontSize: '13px',
                  background: '#1976d2',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontWeight: '600',
                }}
              >
                Download Now
              </button>
              <button
                onClick={onClose}
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
                Cancel
              </button>
            </>
          )}

          {status === 'waiting-install' && (
            <>
              <button
                onClick={handleVerifyInstall}
                style={{
                  flex: 1,
                  minWidth: '140px',
                  padding: '10px 16px',
                  fontSize: '13px',
                  background: '#2e7d32',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontWeight: '600',
                }}
              >
                Verify Installation
              </button>
              <button
                onClick={onClose}
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
                Cancel
              </button>
            </>
          )}

          {status === 'error' && (
            <>
              <button
                onClick={handleVerifyInstall}
                style={{
                  flex: 1,
                  minWidth: '140px',
                  padding: '10px 16px',
                  fontSize: '13px',
                  background: '#1976d2',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontWeight: '600',
                }}
              >
                Try Again
              </button>
              <button
                onClick={onClose}
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
                Cancel
              </button>
            </>
          )}

          {status === 'detecting' || status === 'downloading' || status === 'checking' ? (
            <button
              onClick={onClose}
              disabled
              style={{
                flex: 1,
                minWidth: '140px',
                padding: '10px 16px',
                fontSize: '13px',
                background: '#f5f5f5',
                color: '#999',
                border: '1px solid #ddd',
                borderRadius: '4px',
                cursor: 'not-allowed',
                fontWeight: '500',
                opacity: 0.6,
              }}
            >
              Please wait...
            </button>
          ) : null}

          {status === 'success' && (
            <button
              onClick={onClose}
              style={{
                flex: 1,
                minWidth: '140px',
                padding: '10px 16px',
                fontSize: '13px',
                background: '#2e7d32',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontWeight: '600',
              }}
            >
              Continue
            </button>
          )}
        </div>

        {/* Info */}
        {(status === 'ready-to-download' || status === 'waiting-install') && (
          <div
            style={{
              marginTop: '16px',
              paddingTop: '16px',
              borderTop: '1px solid #e0e0e0',
              fontSize: '11px',
              color: '#999',
              lineHeight: '1.5',
            }}
          >
            The native host is a small system tool that allows Moly to manage local models and services.
            It runs in the background and handles all system-level operations.
          </div>
        )}
      </div>
    </div>
  );
};

export default NativeHostAutoDownloader;
