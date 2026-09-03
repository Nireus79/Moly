import React, { useState } from 'react';
import Sidebar from './Sidebar';
import Settings from './Settings';
import './MainApp.css';

export default function MainApp() {
  const [activeTab, setActiveTab] = useState('chat');
  const [selectedModel, setSelectedModel] = useState(() => localStorage.getItem('moly-model') || 'mistral');
  const [provider, setProvider] = useState(() => localStorage.getItem('moly-provider') || 'ollama');
  const [showSetupHint, setShowSetupHint] = React.useState(true);
  const isSidebar = new URLSearchParams(window.location.search).get('mode') === 'sidebar';

  React.useEffect(() => {
    localStorage.setItem('moly-provider', provider);
    localStorage.setItem('moly-model', selectedModel);
  }, [provider, selectedModel]);

  async function handleOpenSidebar() {
    if (window.moly) {
      await window.moly.openSidebar();
    }
  }

  return (
    <div className="main-app">
      <div className="app-header">
        <h1>Moly</h1>
        <div className="header-status">
          <span className="status-badge active">● Running</span>
        </div>
      </div>

      <div className="app-content">
        {showSetupHint && activeTab === 'chat' && (
          <div style={{padding: '16px', background: '#e3f2fd', borderBottom: '1px solid #90caf9', color: '#1976d2', fontSize: '14px'}}>
            First time? Go to Settings to configure Claude/OpenAI API key or use local Ollama.
            <button onClick={() => setShowSetupHint(false)} style={{marginLeft: '12px', background: 'none', border: 'none', color: '#1976d2', cursor: 'pointer', textDecoration: 'underline'}}>Dismiss</button>
          </div>
        )}
        {activeTab === 'chat' && (
          <Sidebar
            selectedModel={selectedModel}
            provider={provider}
            onModelChange={setSelectedModel}
          />
        )}

        {activeTab === 'settings' && (
          <Settings
            selectedModel={selectedModel}
            provider={provider}
            onProviderChange={setProvider}
            onModelChange={setSelectedModel}
          />
        )}
      </div>

      <div className="app-footer">
        {!isSidebar && (
          <>
            <button
              className={`tab-btn ${activeTab === 'chat' ? 'active' : ''}`}
              onClick={() => setActiveTab('chat')}
            >
              Chat
            </button>
            <button
              className={`tab-btn ${activeTab === 'settings' ? 'active' : ''}`}
              onClick={() => setActiveTab('settings')}
            >
              Settings
            </button>
            <button
              className="tab-btn"
              onClick={handleOpenSidebar}
              style={{marginLeft: 'auto'}}
            >
              Open Sidebar
            </button>
          </>
        )}
      </div>
    </div>
  );
}
