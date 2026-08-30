import React, { useEffect } from 'react';
import { useChatStore } from '@/stores/chatStore';
import { useSettingsStore, initializeSettings } from '@/stores/settingsStore';
import type { CommunicationContext, ChatMode } from '@/types';
import './sidebar.css';

export const Sidebar: React.FC = () => {
  const {
    chatMode,
    currentContext,
    messages,
    suggestions,
    isLoading,
    error,
    detectedMessage,
    setChatMode,
    setContext,
    addMessage,
    clearMessages,
    setSuggestions,
    setLoading,
    setError,
  } = useChatStore();

  const { settings, loadSettings } = useSettingsStore();

  useEffect(() => {
    initializeSettings();
    loadSettings();
    listenForDetectedMessages();
  }, [loadSettings]);

  const listenForDetectedMessages = () => {
    chrome.runtime.onMessage.addListener((request) => {
      if (request.type === 'NEW_MESSAGE_DETECTED') {
        console.log('New message detected:', request.message);
      }
    });
  };

  const handleContextChange = (context: CommunicationContext) => {
    setContext(context);
  };

  const handleModeChange = (mode: ChatMode) => {
    setChatMode(mode);
  };

  const handleSendMessage = async (userMessage: string) => {
    if (!userMessage.trim()) return;

    const activeProvider = settings?.providers[settings?.activeProvider];
    if (!activeProvider?.enabled) {
      setError(`Please configure ${settings?.activeProvider || 'an LLM'} provider in Settings.`);
      return;
    }

    addMessage({
      id: Date.now().toString(),
      role: 'user',
      content: userMessage,
      timestamp: Date.now(),
    });

    setLoading(true);
    setError(null);

    try {
      const response = await chrome.runtime.sendMessage({
        type: 'GENERATE_SUGGESTIONS',
        data: {
          context: detectedMessage?.sender || 'Unknown',
          communicationContext: currentContext,
          userMessage,
          mode: chatMode,
        },
      });

      if (response.success && response.suggestions) {
        setSuggestions(response.suggestions);
        addMessage({
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: `Generated ${response.suggestions.length} message options`,
          timestamp: Date.now(),
        });
      } else if (!response.success) {
        setError(response.error || 'Failed to generate suggestions');
      }
    } catch (err) {
      setError(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenSettings = () => {
    chrome.runtime.openOptionsPage?.();
  };

  return (
    <div className="sidebar-container">
      {/* Header */}
      <div className="sidebar-header">
        <h2>🧠 Moly</h2>
        <div className="header-actions">
          <button className="icon-btn" onClick={handleOpenSettings} title="Settings">⚙️</button>
          <button className="icon-btn" onClick={() => clearMessages()} title="Clear">🗑️</button>
        </div>
      </div>

      {/* Mode Selector */}
      <div className="mode-selector">
        <button
          className={`mode-btn ${chatMode === 'socratic' ? 'active' : ''}`}
          onClick={() => handleModeChange('socratic')}
        >
          📚 Socratic
        </button>
        <button
          className={`mode-btn ${chatMode === 'direct' ? 'active' : ''}`}
          onClick={() => handleModeChange('direct')}
        >
          ⚡ Direct
        </button>
      </div>

      {/* Context Selector */}
      <div className="context-selector">
        <label>Context:</label>
        <div className="context-buttons">
          {(['formal', 'friendly', 'dating'] as const).map((ctx) => (
            <button
              key={ctx}
              className={`context-btn ${currentContext === ctx ? 'active' : ''}`}
              onClick={() => handleContextChange(ctx)}
            >
              {ctx === 'formal' && '💼'}
              {ctx === 'friendly' && '👋'}
              {ctx === 'dating' && '💕'}
              {ctx.charAt(0).toUpperCase() + ctx.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Messages Display */}
      <div className="messages-container">
        {error && (
          <div className="error-message">
            ⚠️ {error}
          </div>
        )}

        {messages.length === 0 && !detectedMessage && (
          <div className="empty-state">
            <p>👋 Start a conversation or message someone to get suggestions</p>
          </div>
        )}

        {detectedMessage && (
          <div className="detected-message">
            <div className="message-label">📨 Message from {detectedMessage.sender}</div>
            <div className="message-content">{detectedMessage.text}</div>
          </div>
        )}

        {messages.map((msg) => (
          <div key={msg.id} className={`message ${msg.role}`}>
            <div className="message-content">{msg.content}</div>
            {msg.tone && <span className="message-tone">{msg.tone}</span>}
          </div>
        ))}

        {isLoading && (
          <div className="loading">
            <span className="spinner">⏳</span> Generating suggestions...
          </div>
        )}
      </div>

      {/* Suggestions Display */}
      {suggestions.length > 0 && (
        <div className="suggestions-container">
          <h4>💡 Suggestions ({suggestions.length})</h4>
          {suggestions.map((suggestion, idx) => (
            <div key={suggestion.id} className="suggestion-card">
              <div className="suggestion-number">Option {idx + 1}</div>
              <div className="suggestion-text">{suggestion.text}</div>
              <div className="suggestion-meta">
                <span className="suggestion-tone">{suggestion.tone}</span>
                <span className="suggestion-confidence">
                  {Math.round(suggestion.confidence * 100)}% match
                </span>
              </div>
              <div className="suggestion-action">
                <button className="copy-btn" onClick={() => {
                  navigator.clipboard.writeText(suggestion.text);
                }}>
                  📋 Copy
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Input Area */}
      <ChatInput
        onSendMessage={handleSendMessage}
        disabled={isLoading || !settings?.providers[settings?.activeProvider]?.enabled}
      />
    </div>
  );
};

interface ChatInputProps {
  onSendMessage: (message: string) => void;
  disabled: boolean;
}

const ChatInput: React.FC<ChatInputProps> = ({ onSendMessage, disabled }) => {
  const [input, setInput] = React.useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (input.trim()) {
      onSendMessage(input);
      setInput('');
    }
  };

  return (
    <form className="chat-input-form" onSubmit={handleSubmit}>
      <input
        type="text"
        placeholder="Ask for message suggestions..."
        value={input}
        onChange={(e) => setInput(e.target.value)}
        disabled={disabled}
        className="chat-input"
      />
      <button type="submit" disabled={disabled} className="send-btn">
        📤
      </button>
    </form>
  );
};
