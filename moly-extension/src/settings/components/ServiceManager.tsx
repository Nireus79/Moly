import React, { useState, useEffect } from 'react';
import type { LocalModelStatus } from '@/api/detection';

interface ServiceManagerProps {
  status: LocalModelStatus;
  onStatusRefresh?: () => void;
}

export const ServiceManager: React.FC<ServiceManagerProps> = ({
  status,
  onStatusRefresh,
}) => {
  const [isOllamaRunning, setIsOllamaRunning] = useState(status.ollama.running);
  const [isControlling, setIsControlling] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    setIsOllamaRunning(status.ollama.running);
  }, [status.ollama.running]);

  const sendNativeMessage = (action: string): Promise<any> => {
    return new Promise((resolve) => {
      chrome.runtime.sendNativeMessage(
        'com.moly.native_host',
        { action },
        (response) => {
          if (chrome.runtime.lastError) {
            resolve({ error: chrome.runtime.lastError.message });
          } else {
            resolve(response || {});
          }
        }
      );
    });
  };

  const handleStartOllama = async () => {
    setIsControlling(true);
    setMessage('Starting Ollama...');

    try {
      const result = await sendNativeMessage('start-ollama');

      if (result.success) {
        setIsOllamaRunning(true);
        setMessage('Ollama started successfully!');
        setTimeout(() => {
          onStatusRefresh?.();
          setMessage('');
        }, 2000);
      } else {
        setMessage(`Error: ${result.error || 'Failed to start Ollama'}`);
      }
    } catch (error) {
      setMessage(
        `Error: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    } finally {
      setIsControlling(false);
    }
  };

  const handleStopOllama = async () => {
    setIsControlling(true);
    setMessage('Stopping Ollama...');

    try {
      const result = await sendNativeMessage('stop-ollama');

      if (result.success) {
        setIsOllamaRunning(false);
        setMessage('Ollama stopped successfully!');
        setTimeout(() => {
          onStatusRefresh?.();
          setMessage('');
        }, 2000);
      } else {
        setMessage(`Error: ${result.error || 'Failed to stop Ollama'}`);
      }
    } catch (error) {
      setMessage(
        `Error: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    } finally {
      setIsControlling(false);
    }
  };

  if (!status.ollama.installed && !status.lmStudio.installed) {
    return null;
  }

  return (
    <div
      style={{
        padding: '16px',
        background: '#f5f5f5',
        border: '1px solid #e0e0e0',
        borderRadius: '6px',
        marginBottom: '16px',
      }}
    >
      <div
        style={{
          fontSize: '14px',
          fontWeight: '600',
          color: '#333',
          marginBottom: '16px',
        }}
      >
        Service Control
      </div>

      {/* Ollama Service */}
      {status.ollama.installed && (
        <div
          style={{
            padding: '12px',
            background: 'white',
            border: '1px solid #e0e0e0',
            borderRadius: '4px',
            marginBottom: '12px',
          }}
        >
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: '12px',
            }}
          >
            <div>
              <div
                style={{
                  fontSize: '13px',
                  fontWeight: '600',
                  color: '#333',
                }}
              >
                Ollama Service
              </div>
              <div
                style={{
                  fontSize: '12px',
                  color: isOllamaRunning ? '#2e7d32' : '#d32f2f',
                  marginTop: '4px',
                }}
              >
                {isOllamaRunning ? '✓ Running' : '○ Not Running'}
              </div>
              {status.ollama.models.length > 0 && (
                <div
                  style={{
                    fontSize: '11px',
                    color: '#666',
                    marginTop: '4px',
                  }}
                >
                  Models: {status.ollama.models.length}
                </div>
              )}
            </div>

            <div style={{ display: 'flex', gap: '8px' }}>
              {!isOllamaRunning && (
                <button
                  onClick={handleStartOllama}
                  disabled={isControlling}
                  style={{
                    padding: '6px 12px',
                    fontSize: '12px',
                    background: '#2e7d32',
                    color: 'white',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: isControlling ? 'not-allowed' : 'pointer',
                    fontWeight: '600',
                    opacity: isControlling ? 0.7 : 1,
                  }}
                >
                  {isControlling ? 'Starting...' : 'Start'}
                </button>
              )}

              {isOllamaRunning && (
                <button
                  onClick={handleStopOllama}
                  disabled={isControlling}
                  style={{
                    padding: '6px 12px',
                    fontSize: '12px',
                    background: '#d32f2f',
                    color: 'white',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: isControlling ? 'not-allowed' : 'pointer',
                    fontWeight: '600',
                    opacity: isControlling ? 0.7 : 1,
                  }}
                >
                  {isControlling ? 'Stopping...' : 'Stop'}
                </button>
              )}
            </div>
          </div>

          {isOllamaRunning && status.ollama.models.length > 0 && (
            <div
              style={{
                fontSize: '11px',
                color: '#2e7d32',
                background: '#e8f5e9',
                padding: '8px',
                borderRadius: '3px',
                marginBottom: '8px',
              }}
            >
              Ready to use: {status.ollama.models.join(', ')}
            </div>
          )}

          {!isOllamaRunning && status.ollama.installed && (
            <div
              style={{
                fontSize: '11px',
                color: '#666',
                background: '#fafafa',
                padding: '8px',
                borderRadius: '3px',
              }}
            >
              Click "Start" to begin using Ollama, or enable auto-start from
              Settings.
            </div>
          )}
        </div>
      )}

      {/* Message */}
      {message && (
        <div
          style={{
            fontSize: '12px',
            color: message.includes('Error') ? '#d32f2f' : '#2e7d32',
            background:
              message.includes('Error') ? '#ffebee' : '#e8f5e9',
            padding: '8px 12px',
            borderRadius: '4px',
            marginTop: '12px',
          }}
        >
          {message}
        </div>
      )}

      {/* Info */}
      <div
        style={{
          marginTop: '12px',
          fontSize: '11px',
          color: '#2e7d32',
          lineHeight: '1.5',
        }}
      >
        ✓ Native messaging host is active. Service control is available.
      </div>
    </div>
  );
};

export default ServiceManager;
