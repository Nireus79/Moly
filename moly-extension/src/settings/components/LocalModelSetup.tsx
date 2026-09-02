import React, { useState } from 'react';
import { InstallerDialog } from './InstallerDialog';
import type { LocalModelStatus } from '@/api/detection';

interface LocalModelSetupProps {
  status: LocalModelStatus;
  onSetupStart?: () => void;
}

export const LocalModelSetup: React.FC<LocalModelSetupProps> = ({
  status,
  onSetupStart,
}) => {
  const [showInstructions, setShowInstructions] = useState<string | null>(null);
  const [showInstallerDialog, setShowInstallerDialog] = useState(false);

  const hasLocalRunning =
    (status.ollama.running || status.lmStudio.running) &&
    (status.ollama.models.length > 0 || status.lmStudio.models.length > 0);

  const hasAnyCloud =
    status.cloudProviders.claude || status.cloudProviders.openai;

  if (hasLocalRunning && hasAnyCloud) {
    // Best case - local running and cloud configured
    return (
      <>
        <div
          style={{
            padding: '16px',
            background: '#e8f5e9',
            border: '1px solid #c8e6c9',
            borderRadius: '6px',
            marginBottom: '16px',
          }}
        >
          <div style={{ fontSize: '14px', fontWeight: '600', color: '#2e7d32' }}>
            All Set!
          </div>
          <div style={{ fontSize: '13px', color: '#558b2f', marginTop: '8px' }}>
            Local model running with cloud fallback configured. Everything works!
          </div>
          <div style={{ marginTop: '12px', display: 'flex', gap: '8px' }}>
            <button
              onClick={() => setShowInstallerDialog(true)}
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                background: '#2e7d32',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontWeight: '600',
              }}
            >
              Reconfigure Local Setup
            </button>
          </div>
        </div>
        {showInstallerDialog && (
          <InstallerDialog
            onClose={() => setShowInstallerDialog(false)}
            onSuccess={() => setShowInstallerDialog(false)}
          />
        )}
      </>
    );
  }

  if (hasLocalRunning) {
    // Local working but no cloud backup
    return (
      <>
        <div
          style={{
            padding: '16px',
            background: '#fff3e0',
            border: '1px solid #ffe0b2',
            borderRadius: '6px',
            marginBottom: '16px',
          }}
        >
          <div style={{ fontSize: '14px', fontWeight: '600', color: '#e65100' }}>
            Local Only
          </div>
          <div style={{ fontSize: '13px', color: '#bf360c', marginTop: '8px' }}>
            Local model is working. Consider adding a cloud API key for backup
            when offline.
          </div>
          <div style={{ marginTop: '12px', display: 'flex', gap: '8px' }}>
            <a
              href="https://console.anthropic.com/keys"
              target="_blank"
              rel="noopener noreferrer"
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                background: '#e65100',
                color: 'white',
                borderRadius: '4px',
                textDecoration: 'none',
                border: 'none',
                cursor: 'pointer',
              }}
            >
              Get Claude Key
            </a>
            <button
              onClick={() => setShowInstallerDialog(true)}
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                background: '#f57f17',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontWeight: '600',
              }}
            >
              Reconfigure Setup
            </button>
          </div>
        </div>
        {showInstallerDialog && (
          <InstallerDialog
            onClose={() => setShowInstallerDialog(false)}
            onSuccess={() => setShowInstallerDialog(false)}
          />
        )}
      </>
    );
  }

  // No local model running - show setup options
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
      <div style={{ fontSize: '14px', fontWeight: '600', color: '#333' }}>
        Set Up Local Model
      </div>

      <div style={{ marginTop: '16px' }}>
        <div style={{ marginBottom: '12px' }}>
          <div
            style={{
              fontSize: '13px',
              fontWeight: '600',
              color: '#333',
              marginBottom: '8px',
            }}
          >
            Option 1: One-Click Setup (Recommended)
          </div>
          <div
            style={{
              fontSize: '12px',
              color: '#666',
              marginBottom: '8px',
              lineHeight: '1.5',
            }}
          >
            Automatically downloads and installs Ollama + Mistral 7B model.
            Works offline after setup.
          </div>
          <button
            onClick={() => {
              onSetupStart?.();
              setShowInstallerDialog(true);
            }}
            style={{
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
            Start Setup
          </button>
        </div>

        <div style={{ marginBottom: '12px', borderTop: '1px solid #e0e0e0', paddingTop: '12px' }}>
          <div
            style={{
              fontSize: '13px',
              fontWeight: '600',
              color: '#333',
              marginBottom: '8px',
            }}
          >
            Option 2: Manual Setup
          </div>
          <div
            style={{
              fontSize: '12px',
              color: '#666',
              marginBottom: '8px',
              lineHeight: '1.5',
            }}
          >
            Install Ollama or LM Studio manually and we'll detect it
            automatically.
          </div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            <a
              href="https://ollama.ai"
              target="_blank"
              rel="noopener noreferrer"
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                background: '#f5f5f5',
                border: '1px solid #999',
                color: '#333',
                borderRadius: '4px',
                textDecoration: 'none',
                cursor: 'pointer',
              }}
            >
              Download Ollama
            </a>
            <a
              href="https://lmstudio.ai"
              target="_blank"
              rel="noopener noreferrer"
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                background: '#f5f5f5',
                border: '1px solid #999',
                color: '#333',
                borderRadius: '4px',
                textDecoration: 'none',
                cursor: 'pointer',
              }}
            >
              Download LM Studio
            </a>
          </div>
          <button
            onClick={() => setShowInstructions('manual')}
            style={{
              marginTop: '8px',
              padding: '6px 12px',
              fontSize: '12px',
              background: 'white',
              border: '1px solid #ccc',
              borderRadius: '4px',
              cursor: 'pointer',
              color: '#666',
            }}
          >
            Show Instructions
          </button>
          {showInstructions === 'manual' && (
            <div
              style={{
                marginTop: '8px',
                padding: '12px',
                background: '#fafafa',
                border: '1px solid #ddd',
                borderRadius: '4px',
                fontSize: '12px',
                lineHeight: '1.6',
              }}
            >
              <strong>Ollama Setup:</strong>
              <div style={{ marginTop: '4px', fontFamily: 'monospace' }}>
                1. Download from ollama.ai
                <br />
                2. Install (follow installer)
                <br />
                3. In terminal: <code>ollama pull mistral</code>
                <br />
                4. Moly will detect it automatically
              </div>
              <br />
              <strong>LM Studio Setup:</strong>
              <div style={{ marginTop: '4px', fontFamily: 'monospace' }}>
                1. Download from lmstudio.ai
                <br />
                2. Install and launch app
                <br />
                3. Search and download "Mistral 7B"
                <br />
                4. Load the model
                <br />
                5. Moly will detect it automatically
              </div>
            </div>
          )}
        </div>

        <div style={{ borderTop: '1px solid #e0e0e0', paddingTop: '12px' }}>
          <div
            style={{
              fontSize: '13px',
              fontWeight: '600',
              color: '#333',
              marginBottom: '8px',
            }}
          >
            Option 3: Cloud Only (No Local Setup)
          </div>
          <div
            style={{
              fontSize: '12px',
              color: '#666',
              marginBottom: '8px',
              lineHeight: '1.5',
            }}
          >
            Use Claude or OpenAI instead. Requires API key but no local
            installation.
          </div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            <button
              onClick={() =>
                window.open('https://console.anthropic.com/keys', '_blank')
              }
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                background: '#f5f5f5',
                border: '1px solid #999',
                color: '#333',
                borderRadius: '4px',
                cursor: 'pointer',
              }}
            >
              Claude API Key
            </button>
            <button
              onClick={() =>
                window.open('https://platform.openai.com/api-keys', '_blank')
              }
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                background: '#f5f5f5',
                border: '1px solid #999',
                color: '#333',
                borderRadius: '4px',
                cursor: 'pointer',
              }}
            >
              OpenAI API Key
            </button>
          </div>
        </div>
      </div>

      {showInstallerDialog && (
        <InstallerDialog
          onClose={() => setShowInstallerDialog(false)}
          onSuccess={() => {
            setShowInstallerDialog(false);
            // Optional: reload detection after successful setup
          }}
        />
      )}
    </div>
  );
};

export default LocalModelSetup;
