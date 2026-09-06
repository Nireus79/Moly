import React, { useEffect, useState } from 'react';
import { getBackendManager } from '@/api/backendManager';

interface BackendStatusProps {
  onStatusChange?: (healthy: boolean) => void;
}

export const BackendStatus: React.FC<BackendStatusProps> = ({ onStatusChange }) => {
  const [status, setStatus] = useState<'checking' | 'healthy' | 'unavailable'>('checking');
  const [message, setMessage] = useState('Starting backend...');
  const [copied, setCopied] = useState(false);

  const command = 'MOLY_PROXY_PATH=~/vs_projects/Moly/Moly/moly-proxy/bin/moly-proxy.js ~/.local/bin/moly &';

  useEffect(() => {
    const checkBackend = async () => {
      const manager = getBackendManager();
      const result = await manager.initialize();

      if (result.running) {
        setStatus('healthy');
        setMessage('Backend connected');
        onStatusChange?.(true);
      } else {
        setStatus('unavailable');
        setMessage(
          'You are in development mode. To start backend, open a terminal and run:'
        );
        onStatusChange?.(false);
      }
    };

    checkBackend();
  }, [onStatusChange]);

  const copyToClipboard = () => {
    try {
      // Create temporary textarea to copy text
      const textarea = document.createElement('textarea');
      textarea.value = command;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);

      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

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
      <div
        style={{
          display: 'flex',
          gap: '8px',
          alignItems: 'center',
          background: '#fca5a5',
          padding: '8px',
          borderRadius: '4px',
          fontFamily: 'monospace',
          fontSize: '11px',
          wordBreak: 'break-all',
        }}
      >
        <span style={{ flex: 1 }}>{command}</span>
        <button
          onClick={copyToClipboard}
          style={{
            padding: '4px 8px',
            background: '#7f1d1d',
            color: '#fee2e2',
            border: 'none',
            borderRadius: '3px',
            cursor: 'pointer',
            fontSize: '11px',
            fontWeight: '600',
            whiteSpace: 'nowrap',
            flexShrink: 0,
          }}
          title="Copy command"
        >
          {copied ? '✓' : 'Copy'}
        </button>
      </div>
    </div>
  );
};

export default BackendStatus;
