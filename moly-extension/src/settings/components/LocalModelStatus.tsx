import React, { useEffect, useState } from 'react';
import {
  detectLocalModels,
  formatStatus,
  type LocalModelStatus,
} from '@/api/detection';

interface LocalModelStatusProps {
  onStatusChange?: (status: LocalModelStatus) => void;
}

export const LocalModelStatusPanel: React.FC<LocalModelStatusProps> = ({
  onStatusChange,
}) => {
  const [status, setStatus] = useState<LocalModelStatus | null>(null);
  const [loading, setLoading] = useState(true);

  const checkStatus = async () => {
    setLoading(true);
    try {
      const detectedStatus = await detectLocalModels();
      setStatus(detectedStatus);
      onStatusChange?.(detectedStatus);
    } catch (error) {
      console.error('[LocalModelStatus] Detection failed:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    checkStatus();
    // Re-check every 30 seconds
    const interval = setInterval(checkStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div style={{ padding: '12px', color: '#666', fontSize: '13px' }}>
        Checking for local models...
      </div>
    );
  }

  if (!status) {
    return (
      <div style={{ padding: '12px', color: '#d32f2f', fontSize: '13px' }}>
        Unable to detect local models
      </div>
    );
  }

  const statusLines = formatStatus(status);
  const hasLocalModel =
    (status.ollama.running || status.lmStudio.running) &&
    (status.ollama.models.length > 0 || status.lmStudio.models.length > 0);

  return (
    <div
      style={{
        padding: '16px',
        background: hasLocalModel ? '#f0f7f4' : '#fafafa',
        border: `1px solid ${hasLocalModel ? '#c8e6c9' : '#e0e0e0'}`,
        borderRadius: '6px',
        marginBottom: '16px',
      }}
    >
      <div
        style={{
          fontSize: '12px',
          lineHeight: '1.6',
          color: '#333',
        }}
      >
        {statusLines.map((line, idx) => (
          <div key={idx}>{line}</div>
        ))}
      </div>

      <div
        style={{
          marginTop: '12px',
          display: 'flex',
          gap: '8px',
          flexWrap: 'wrap',
        }}
      >
        {status.ollama.running && (
          <button
            onClick={() => checkStatus()}
            style={{
              padding: '6px 12px',
              fontSize: '12px',
              background: '#e8f5e9',
              border: '1px solid #c8e6c9',
              borderRadius: '4px',
              cursor: 'pointer',
              color: '#2e7d32',
              fontWeight: '500',
            }}
          >
            Refresh Models
          </button>
        )}

        {status.ollama.installed && !status.ollama.running && (
          <div
            style={{
              padding: '6px 12px',
              fontSize: '12px',
              color: '#d32f2f',
              background: '#ffebee',
              borderRadius: '4px',
              border: '1px solid #ef5350',
            }}
          >
            Start with: ollama serve
          </div>
        )}

      </div>

      {status.ollama.models.length > 0 && (
        <div
          style={{
            marginTop: '12px',
            fontSize: '11px',
            color: '#666',
          }}
        >
          Available models: {status.ollama.models.join(', ')}
        </div>
      )}
    </div>
  );
};

export default LocalModelStatusPanel;
