import React from 'react';

interface SettingsPanelProps {
  mode: 'socratic' | 'direct';
  context: 'formal' | 'friendly' | 'dating';
  llmProvider: string;
  onModeChange: (mode: 'socratic' | 'direct') => void;
  onContextChange: (context: 'formal' | 'friendly' | 'dating') => void;
  onSettingsOpen?: () => void;
}

export const SettingsPanel: React.FC<SettingsPanelProps> = ({
  mode,
  context,
  llmProvider,
  onModeChange,
  onContextChange,
  onSettingsOpen,
}) => {
  return (
    <div className="settings-panel" style={{ padding: '12px 16px', borderTop: '1px solid #e5e7eb', background: '#f9f9f9', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
      <div style={{ fontSize: '13px', color: '#666' }}>
        <span>Provider:</span>
        <strong style={{ marginLeft: '6px', color: '#1f2937' }}>{llmProvider}</strong>
      </div>
      {onSettingsOpen && (
        <button
          onClick={onSettingsOpen}
          style={{
            padding: '6px 12px',
            fontSize: '12px',
            background: '#fff',
            border: '1px solid #d1d5db',
            borderRadius: '4px',
            cursor: 'pointer',
            color: '#1f2937',
            fontWeight: '500',
          }}
          title="Configure LLM providers"
        >
          Settings
        </button>
      )}
    </div>
  );
};

export default SettingsPanel;
