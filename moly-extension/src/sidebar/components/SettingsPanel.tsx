import React from 'react';
import { Settings } from '@/settings/Settings';

interface SettingsPanelProps {
  onClose?: () => void;
}

export const SettingsPanel: React.FC<SettingsPanelProps> = ({ onClose }) => {
  return (
    <div className="settings-panel">
      <Settings onClose={onClose} />
    </div>
  );
};
