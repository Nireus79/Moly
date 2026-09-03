import React, { useState } from 'react';
import Sidebar from './Sidebar';
import Settings from './Settings';
import './MainApp.css';

export default function MainApp() {
  const [activeTab, setActiveTab] = useState('chat');
  const [selectedModel, setSelectedModel] = useState('mistral');
  const [provider, setProvider] = useState('ollama');

  return (
    <div className="main-app">
      <div className="app-header">
        <h1>Moly</h1>
        <div className="header-status">
          <span className="status-badge active">● Running</span>
        </div>
      </div>

      <div className="app-content">
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
        <button
          className={`tab-btn ${activeTab === 'chat' ? 'active' : ''}`}
          onClick={() => setActiveTab('chat')}
        >
          💬 Chat
        </button>
        <button
          className={`tab-btn ${activeTab === 'settings' ? 'active' : ''}`}
          onClick={() => setActiveTab('settings')}
        >
          ⚙️ Settings
        </button>
      </div>
    </div>
  );
}
