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
    <div className="settings-panel">
      <div className="settings-section">
        <label className="settings-label">Mode</label>
        <div className="button-group">
          <button
            className={`mode-btn ${mode === 'socratic' ? 'active' : ''}`}
            onClick={() => onModeChange('socratic')}
            title="Guiding questions to help you think through responses"
          >
            Socratic
          </button>
          <button
            className={`mode-btn ${mode === 'direct' ? 'active' : ''}`}
            onClick={() => onModeChange('direct')}
            title="Ready-to-use response suggestions"
          >
            Direct
          </button>
        </div>
      </div>

      <div className="settings-section">
        <label className="settings-label">Context</label>
        <div className="button-group">
          <button
            className={`context-btn ${context === 'formal' ? 'active' : ''}`}
            onClick={() => onContextChange('formal')}
            title="Professional and respectful tone"
          >
            Formal
          </button>
          <button
            className={`context-btn ${context === 'friendly' ? 'active' : ''}`}
            onClick={() => onContextChange('friendly')}
            title="Warm and casual tone"
          >
            Friendly
          </button>
          <button
            className={`context-btn ${context === 'dating' ? 'active' : ''}`}
            onClick={() => onContextChange('dating')}
            title="Playful and engaging tone"
          >
            Dating
          </button>
        </div>
      </div>

      <div className="settings-section">
        <label className="settings-label">LLM</label>
        <div className="provider-info">
          <span className="provider-name">{llmProvider}</span>
          {onSettingsOpen && (
            <button
              onClick={onSettingsOpen}
              className="btn-settings"
              title="Open full settings"
            >
              ⚙️
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default SettingsPanel;
