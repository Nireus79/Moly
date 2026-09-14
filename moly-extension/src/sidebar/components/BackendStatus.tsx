import React, { useEffect, useState } from 'react';
import { getBackendManager } from '@/api/backendManager';

interface BackendStatusProps {
  onStatusChange?: (healthy: boolean) => void;
}

type OSType = 'linux' | 'windows' | 'macos' | 'unknown';
type CommandMode = 'backend' | 'ollama' | 'full-setup';

const detectOS = (): OSType => {
  if (typeof navigator === 'undefined') return 'unknown';
  const ua = navigator.userAgent.toLowerCase();
  if (ua.indexOf('win') !== -1) return 'windows';
  if (ua.indexOf('mac') !== -1) return 'macos';
  if (ua.indexOf('linux') !== -1) return 'linux';
  return 'unknown';
};

const getInstructions = (os: OSType, backendPort?: number, ollamaPort?: number) => {
  // Use detected ports if available, otherwise show generic command
  const backendCmd = backendPort
    ? `# Backend is on port ${backendPort}, if not running:`
    : '';
  const ollamaCmd = ollamaPort
    ? `# Ollama is on port ${ollamaPort}, if not running:`
    : '';

  const instructions: Record<OSType, { title: string; steps: string[]; commands: Record<CommandMode, string> }> = {
    linux: {
      title: 'Linux - Start Services',
      steps: [
        '1. Open Terminal',
        backendPort ? `2. Backend detected on port ${backendPort}` : '2. Start backend server',
        ollamaPort ? `3. Ollama detected on port ${ollamaPort}` : '3. Start Ollama',
        '4. Reload extension once services are running',
      ],
      commands: {
        backend: `cd ~/vs_projects/Moly/Moly/moly-go && go run main.go  # Starts on localhost:8080`,
        ollama: `ollama serve  # Starts on localhost:11434`,
        'full-setup': `# Terminal 1: Backend\ncd ~/vs_projects/Moly/Moly/moly-go && go run main.go\n\n# Terminal 2: Ollama\nollama serve`,
      },
    },
    macos: {
      title: 'macOS - Start Services',
      steps: [
        '1. Open Terminal.app',
        backendPort ? `2. Backend detected on port ${backendPort}` : '2. Start backend server',
        ollamaPort ? `3. Ollama detected on port ${ollamaPort}` : '3. Start Ollama',
        '4. Reload extension once services are running',
      ],
      commands: {
        backend: `cd ~/vs_projects/Moly/Moly/moly-go && go run main.go  # Starts on localhost:8080`,
        ollama: `ollama serve  # Starts on localhost:11434`,
        'full-setup': `# Terminal 1: Backend\ncd ~/vs_projects/Moly/Moly/moly-go && go run main.go\n\n# Terminal 2: Ollama\nollama serve`,
      },
    },
    windows: {
      title: 'Windows - Start Services',
      steps: [
        '1. Open PowerShell or Command Prompt',
        backendPort ? `2. Backend detected on port ${backendPort}` : '2. Start backend server',
        ollamaPort ? `3. Ollama detected on port ${ollamaPort}` : '3. Start Ollama',
        '4. Reload extension once services are running',
      ],
      commands: {
        backend: `cd ~/vs_projects/Moly/Moly/moly-go && go run main.go  # Starts on localhost:8080`,
        ollama: `ollama serve  # Starts on localhost:11434`,
        'full-setup': `# PowerShell 1: Backend\ncd ~/moly-go\ngo run main.go\n\n# PowerShell 2: Ollama\nollama serve`,
      },
    },
    unknown: {
      title: 'Start Services',
      steps: [
        '1. Open a terminal',
        backendPort ? `2. Backend detected on port ${backendPort}` : '2. Start backend: cd ~/vs_projects/Moly/Moly/moly-go && go run main.go',
        ollamaPort ? `3. Ollama detected on port ${ollamaPort}` : '3. Start Ollama: ollama serve',
        '4. Reload extension once services are running',
      ],
      commands: {
        backend: `cd ~/vs_projects/Moly/Moly/moly-go && go run main.go`,
        ollama: `ollama serve`,
        'full-setup': `# Terminal 1\ncd ~/vs_projects/Moly/Moly/moly-go && go run main.go\n\n# Terminal 2\nollama serve`,
      },
    },
  };

  return instructions[os] || instructions.unknown;
};

export const BackendStatus: React.FC<BackendStatusProps> = ({ onStatusChange }) => {
  const [status, setStatus] = useState<'checking' | 'healthy' | 'unavailable'>('checking');
  const [message, setMessage] = useState('Starting backend...');
  const [copied, setCopied] = useState(false);
  const [os, setOS] = useState<OSType>('unknown');
  const [isDevMode, setIsDevMode] = useState(false);
  const [detectedBackendPort, setDetectedBackendPort] = useState<number | undefined>();
  const [detectedOllamaPort, setDetectedOllamaPort] = useState<number | undefined>();

  useEffect(() => {
    setOS(detectOS());
    setIsDevMode(true); // Assume development mode if backend not running
  }, []);

  console.log('[BackendStatus] Rendering with status:', status, 'OS:', os);

  const instructions = getInstructions(os, detectedBackendPort, detectedOllamaPort);
  const backendCommand = instructions.commands.backend;

  const checkBackend = async () => {
    const manager = getBackendManager();
    const result = await manager.initialize();

    // Extract port from detected URL if available
    if (result.url && result.url !== 'unknown') {
      try {
        const url = new URL(result.url);
        const port = parseInt(url.port, 10);
        if (!isNaN(port)) {
          setDetectedBackendPort(port);
        }
      } catch {
        // URL parsing failed, no port to extract
      }
    }

    // Try to detect Ollama
    const commonOllamaPorts = [11434, 11435, 8000, 5000];
    for (const port of commonOllamaPorts) {
      try {
        const response = await fetch(`http://127.0.0.1:${port}/api/tags`, {
          method: 'GET',
          signal: AbortSignal.timeout(1000),
        });
        if (response.ok) {
          setDetectedOllamaPort(port);
          break;
        }
      } catch {
        // Try next port
      }
    }

    if (result.running) {
      setStatus('healthy');
      setMessage('Backend connected');
      setIsDevMode(false);
      onStatusChange?.(true);
    } else {
      setStatus('unavailable');
      setMessage(
        'Backend not running. Development mode: Follow instructions below to start it manually.'
      );
      // In development mode, show instructions to start backend
      // In production (Web Store), backend auto-starts, so this message won't appear
      setIsDevMode(true);
      onStatusChange?.(false);
    }
  };

  useEffect(() => {
    checkBackend();

    // Auto-check every 3 seconds to detect when backend comes online
    const interval = setInterval(checkBackend, 3000);
    return () => clearInterval(interval);
  }, [onStatusChange]);

  const copyToClipboard = (text: string) => {
    try {
      const textarea = document.createElement('textarea');
      textarea.value = text;
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

      {isDevMode && (
        <>
          <div style={{ background: '#fca5a5', padding: '8px', borderRadius: '4px', marginBottom: '8px', fontSize: '11px' }}>
            <div style={{ fontWeight: '600', marginBottom: '6px' }}>
              📋 {instructions.title}
            </div>
            <div style={{ marginBottom: '8px', lineHeight: '1.4' }}>
              {instructions.steps.map((step, i) => (
                <div key={i}>{step}</div>
              ))}
            </div>
          </div>

          <div style={{ marginBottom: '8px' }}>
            <div style={{ fontWeight: '600', marginBottom: '4px' }}>Backend Command:</div>
            <div
              style={{
                display: 'flex',
                gap: '8px',
                alignItems: 'center',
                background: '#7f1d1d',
                padding: '8px',
                borderRadius: '4px',
                fontFamily: 'monospace',
                fontSize: '10px',
                wordBreak: 'break-all',
                color: '#fee2e2',
              }}
            >
              <span style={{ flex: 1 }}>{backendCommand}</span>
              <button
                onClick={() => copyToClipboard(backendCommand)}
                style={{
                  padding: '4px 8px',
                  background: '#fee2e2',
                  color: '#7f1d1d',
                  border: 'none',
                  borderRadius: '3px',
                  cursor: 'pointer',
                  fontSize: '10px',
                  fontWeight: '600',
                  whiteSpace: 'nowrap',
                  flexShrink: 0,
                }}
                title="Copy backend command"
              >
                {copied ? '✓' : 'Copy'}
              </button>
            </div>
          </div>

          <div style={{ marginBottom: '8px' }}>
            <div style={{ fontWeight: '600', marginBottom: '4px' }}>Ollama Command (separate terminal):</div>
            <div
              style={{
                display: 'flex',
                gap: '8px',
                alignItems: 'center',
                background: '#7f1d1d',
                padding: '8px',
                borderRadius: '4px',
                fontFamily: 'monospace',
                fontSize: '10px',
                wordBreak: 'break-all',
                color: '#fee2e2',
              }}
            >
              <span style={{ flex: 1 }}>ollama serve</span>
              <button
                onClick={() => copyToClipboard('ollama serve')}
                style={{
                  padding: '4px 8px',
                  background: '#fee2e2',
                  color: '#7f1d1d',
                  border: 'none',
                  borderRadius: '3px',
                  cursor: 'pointer',
                  fontSize: '10px',
                  fontWeight: '600',
                  whiteSpace: 'nowrap',
                  flexShrink: 0,
                }}
                title="Copy Ollama command"
              >
                {copied ? '✓' : 'Copy'}
              </button>
            </div>
          </div>
        </>
      )}

      <button
        onClick={() => {
          console.log('[BackendStatus] Refresh clicked');
          checkBackend();
        }}
        style={{
          marginTop: '8px',
          padding: '6px 12px',
          background: '#7f1d1d',
          color: '#fee2e2',
          border: 'none',
          borderRadius: '4px',
          cursor: 'pointer',
          fontSize: '11px',
          fontWeight: '600',
          width: '100%',
        }}
      >
        🔄 Refresh Status
      </button>
    </div>
  );
};

export default BackendStatus;
