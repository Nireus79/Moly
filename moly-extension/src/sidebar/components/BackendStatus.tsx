import React, { useEffect, useState } from 'react';
import { getBackendManager } from '@/api/backendManager';

interface BackendStatusProps {
  onStatusChange?: (healthy: boolean) => void;
}

export const BackendStatus: React.FC<BackendStatusProps> = ({ onStatusChange }) => {
  const [status, setStatus] = useState<'checking' | 'healthy' | 'unavailable'>('checking');
  const [message, setMessage] = useState('Starting backend...');

  useEffect(() => {
    const checkBackend = async () => {
      const manager = getBackendManager();
      const result = await manager.ensureRunning();

      if (result) {
        setStatus('healthy');
        setMessage('Backend connected');
        onStatusChange?.(true);
      } else {
        setStatus('unavailable');
        setMessage(
          'Backend not running. Start with: cd moly-go && ./moly'
        );
        onStatusChange?.(false);
      }
    };

    checkBackend();
  }, [onStatusChange]);

  if (status === 'healthy') {
    return (
      <div
        style={{
          padding: '8px 12px',
          marginBottom: '12px',
          background: '#dcfce7',
          border: '1px solid #86efac',
          borderRadius: '6px',
          fontSize: '12px',
          color: '#166534',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
        }}
      >
        <span style={{ fontSize: '14px' }}>✓</span>
        {message}
      </div>
    );
  }

  if (status === 'checking') {
    return (
      <div
        style={{
          padding: '8px 12px',
          marginBottom: '12px',
          background: '#fef3c7',
          border: '1px solid #fcd34d',
          borderRadius: '6px',
          fontSize: '12px',
          color: '#92400e',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
        }}
      >
        <span style={{ fontSize: '14px' }}>⏳</span>
        {message}
      </div>
    );
  }

  return (
    <div
      style={{
        padding: '12px',
        marginBottom: '12px',
        background: '#fee2e2',
        border: '2px solid #fca5a5',
        borderRadius: '6px',
        fontSize: '12px',
        color: '#7f1d1d',
      }}
    >
      <div style={{ fontWeight: '600', marginBottom: '8px' }}>
        ⚠️ {message}
      </div>
      <div style={{ fontSize: '11px', marginBottom: '8px', opacity: 0.9 }}>
        Go backend enables safety checks, ethics evaluation, and contextual questions.
        Without it, Moly will use LLM providers only.
      </div>
      <div style={{ fontSize: '11px', fontFamily: 'monospace', background: '#fca5a5', padding: '6px', borderRadius: '3px' }}>
        cd moly-go && ./moly
      </div>
    </div>
  );
};

export default BackendStatus;
