import React, { useState } from 'react';
import { pullModel } from '@/api/installerLauncher';

interface ModelSelectionDialogProps {
  onClose: () => void;
  onSuccess?: () => void;
}

const RECOMMENDED_MODELS = [
  {
    name: 'mistral',
    display: 'Mistral 7B',
    size: '~4GB',
    speed: 'Fast',
    quality: 'Excellent',
    recommended: true,
  },
  {
    name: 'llama2',
    display: 'Llama 2 7B',
    size: '~4GB',
    speed: 'Fast',
    quality: 'Good',
    recommended: false,
  },
  {
    name: 'neural-chat',
    display: 'Neural Chat 7B',
    size: '~4GB',
    speed: 'Fast',
    quality: 'Very Good',
    recommended: false,
  },
];

export const ModelSelectionDialog: React.FC<ModelSelectionDialogProps> = ({
  onClose,
  onSuccess,
}) => {
  const [selectedModel, setSelectedModel] = useState('mistral');
  const [isInstalling, setIsInstalling] = useState(false);
  const [progress, setProgress] = useState('');
  const [error, setError] = useState('');

  const handleInstall = async () => {
    setIsInstalling(true);
    setProgress('Starting download...');
    setError('');

    try {
      const result = await pullModel(selectedModel);

      if (result.success) {
        setProgress('Download complete! Model is ready.');
        setTimeout(() => {
          onSuccess?.();
          onClose();
        }, 1000);
      } else {
        setError(result.error || 'Failed to install model');
        setIsInstalling(false);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Installation failed');
      setIsInstalling(false);
    }
  };

  const model = RECOMMENDED_MODELS.find((m) => m.name === selectedModel);

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
        zIndex: 1000,
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: 'white',
          borderRadius: '8px',
          padding: '24px',
          maxWidth: '500px',
          width: '90%',
          maxHeight: '80vh',
          overflowY: 'auto',
          boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {isInstalling ? (
          <>
            <div style={{ fontSize: '16px', fontWeight: '600', marginBottom: '16px' }}>
              Installing {model?.display}...
            </div>
            <div
              style={{
                padding: '20px',
                background: '#f5f5f5',
                borderRadius: '6px',
                textAlign: 'center',
                marginBottom: '16px',
              }}
            >
              <div style={{ fontSize: '14px', color: '#666', marginBottom: '12px' }}>
                {progress}
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
                    animation: 'pulse 1s infinite',
                  }}
                />
              </div>
            </div>
            <div style={{ fontSize: '12px', color: '#999', textAlign: 'center' }}>
              This may take a few minutes depending on your internet speed...
            </div>
          </>
        ) : error ? (
          <>
            <div style={{ fontSize: '16px', fontWeight: '600', color: '#d32f2f', marginBottom: '16px' }}>
              Installation Failed
            </div>
            <div
              style={{
                padding: '12px',
                background: '#ffebee',
                border: '1px solid #ffcdd2',
                borderRadius: '4px',
                marginBottom: '16px',
                fontSize: '13px',
                color: '#c62828',
                lineHeight: '1.5',
              }}
            >
              {error}
            </div>
            <div style={{ display: 'flex', gap: '8px' }}>
              <button
                onClick={() => {
                  setError('');
                  setProgress('');
                }}
                style={{
                  flex: 1,
                  padding: '8px 16px',
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
                  padding: '8px 16px',
                  fontSize: '13px',
                  background: '#f5f5f5',
                  color: '#333',
                  border: '1px solid #ccc',
                  borderRadius: '4px',
                  cursor: 'pointer',
                }}
              >
                Cancel
              </button>
            </div>
          </>
        ) : (
          <>
            <div style={{ fontSize: '16px', fontWeight: '600', marginBottom: '16px' }}>
              Choose a Local Model
            </div>

            <div style={{ marginBottom: '16px' }}>
              {RECOMMENDED_MODELS.map((m) => (
                <div
                  key={m.name}
                  onClick={() => setSelectedModel(m.name)}
                  style={{
                    padding: '12px',
                    marginBottom: '8px',
                    border: selectedModel === m.name ? '2px solid #1976d2' : '1px solid #ddd',
                    borderRadius: '6px',
                    cursor: 'pointer',
                    background: selectedModel === m.name ? '#e3f2fd' : 'white',
                    transition: 'all 0.2s',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      marginBottom: '4px',
                    }}
                  >
                    <div style={{ fontWeight: '600', fontSize: '13px' }}>
                      {m.display}
                      {m.recommended && (
                        <span
                          style={{
                            marginLeft: '8px',
                            fontSize: '11px',
                            background: '#4caf50',
                            color: 'white',
                            padding: '2px 6px',
                            borderRadius: '3px',
                          }}
                        >
                          Recommended
                        </span>
                      )}
                    </div>
                    <div
                      style={{
                        fontSize: '11px',
                        color: '#999',
                      }}
                    >
                      {m.size}
                    </div>
                  </div>
                  <div style={{ fontSize: '12px', color: '#666' }}>
                    Speed: {m.speed} | Quality: {m.quality}
                  </div>
                </div>
              ))}
            </div>

            {model && (
              <div
                style={{
                  padding: '12px',
                  background: '#e3f2fd',
                  border: '1px solid #90caf9',
                  borderRadius: '4px',
                  fontSize: '12px',
                  color: '#1565c0',
                  marginBottom: '16px',
                  lineHeight: '1.5',
                }}
              >
                {model.display} is a great balance of speed and quality. Perfect for offline
                use.
              </div>
            )}

            <div style={{ display: 'flex', gap: '8px' }}>
              <button
                onClick={handleInstall}
                style={{
                  flex: 1,
                  padding: '8px 16px',
                  fontSize: '13px',
                  background: '#1976d2',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontWeight: '600',
                }}
              >
                Install {model?.display}
              </button>
              <button
                onClick={onClose}
                style={{
                  flex: 1,
                  padding: '8px 16px',
                  fontSize: '13px',
                  background: '#f5f5f5',
                  color: '#333',
                  border: '1px solid #ccc',
                  borderRadius: '4px',
                  cursor: 'pointer',
                }}
              >
                Skip for Now
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
};
