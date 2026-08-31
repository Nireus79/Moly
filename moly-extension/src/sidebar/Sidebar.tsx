import React, { useEffect } from 'react';
import { useChatStore } from '@/stores/chatStore';
import { useSettingsStore, initializeSettings } from '@/stores/settingsStore';
import { useConversationStore } from '@/stores/conversationStore';
import { useContactStore } from '@/stores/contactStore';
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
  const { saveMessage, currentContactId, setCurrentContact } = useConversationStore();
  useContactStore();

  useEffect(() => {
    initializeSettings();
    loadSettings();
    listenForDetectedMessages();
  }, [loadSettings]);

  useEffect(() => {
    const loadSelectedContact = async () => {
      const result = await chrome.storage.local.get('selectedContactId');
      if (result.selectedContactId) {
        setCurrentContact(result.selectedContactId);
      }
    };
    loadSelectedContact();
  }, [setCurrentContact]);

  const listenForDetectedMessages = () => {
    const { setDetectedMessage } = useChatStore.getState();

    chrome.runtime.onMessage.addListener((request) => {
      if (request.type === 'NEW_MESSAGE_DETECTED' || request.type === 'DETECTED_MESSAGE_BROADCAST') {
        console.log('Sidebar received detected message:', request.message?.sender);
        if (request.message) {
          setDetectedMessage(request.message);
        }
      }
    });

    chrome.storage.onChanged.addListener((changes, areaName) => {
      if (areaName === 'local' && changes.lastDetectedMessage) {
        console.log('Sidebar detected storage change for message');
        const message = changes.lastDetectedMessage.newValue;
        if (message) {
          setDetectedMessage(message);
        }
      }
    });

    chrome.storage.local.get('lastDetectedMessage', (result) => {
      if (result.lastDetectedMessage) {
        console.log('Sidebar found existing detected message on startup');
        setDetectedMessage(result.lastDetectedMessage);
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

    const userMsg = {
      id: Date.now().toString(),
      role: 'user' as const,
      content: userMessage,
      timestamp: Date.now(),
    };

    addMessage(userMsg);

    if (currentContactId) {
      await saveMessage(currentContactId, userMsg);
    }

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
        const assistantMsg = {
          id: (Date.now() + 1).toString(),
          role: 'assistant' as const,
          content: `Generated ${response.suggestions.length} message options`,
          timestamp: Date.now(),
        };
        addMessage(assistantMsg);

        if (currentContactId) {
          await saveMessage(currentContactId, assistantMsg);
        }
      } else if (!response.success) {
        setError(response.error || 'Failed to generate suggestions');
      }
    } catch (err) {
      setError(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenSettings = async () => {
    try {
      await chrome.runtime.openOptionsPage?.();
    } catch (error) {
      console.error('Failed to open settings:', error);
      setError('Could not open settings. Please try again.');
    }
  };

  const handleCopySuggestion = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  return (
    <div className="sidebar-container">
      {/* HEADER */}
      <div className="sidebar-header">
        <h2>Moly</h2>
        <div className="header-actions">
          <button className="icon-btn" onClick={handleOpenSettings} title="Settings">S</button>
          <button className="icon-btn" onClick={() => clearMessages()} title="Clear">C</button>
        </div>
      </div>

      {/* SECTION 1: DETECTED MESSAGE (TOP) */}
      <div className="detected-section">
        {detectedMessage ? (
          <div className="detected-message">
            <div className="message-label">Message from {detectedMessage.sender}</div>
            <div className="message-content">{detectedMessage.text}</div>
          </div>
        ) : (
          <div className="detected-placeholder">No message detected yet</div>
        )}
      </div>

      {/* MODE & CONTEXT SELECTORS */}
      <div className="controls-section">
        <div className="mode-selector">
          <span className="control-label">Mode:</span>
          <div className="button-group">
            <button
              className={`mode-btn ${chatMode === 'socratic' ? 'active' : ''}`}
              onClick={() => handleModeChange('socratic')}
              title="Socratic: Get guided questions to think deeper"
            >
              Socratic
            </button>
            <button
              className={`mode-btn ${chatMode === 'direct' ? 'active' : ''}`}
              onClick={() => handleModeChange('direct')}
              title="Direct: Get ready-to-use message suggestions"
            >
              Direct
            </button>
          </div>
        </div>

        <div className="context-selector">
          <span className="control-label">Context:</span>
          <div className="button-group">
            {(['formal', 'friendly', 'dating'] as const).map((ctx) => (
              <button
                key={ctx}
                className={`context-btn ${currentContext === ctx ? 'active' : ''}`}
                onClick={() => handleContextChange(ctx)}
                title={`${ctx.charAt(0).toUpperCase() + ctx.slice(1)} communication style`}
              >
                {ctx.charAt(0).toUpperCase() + ctx.slice(1)}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* SECTION 2: CHAT WITH MOLY (MIDDLE) */}
      <div className="chat-section">
        {error && (
          <div className="error-message">
            {error}
          </div>
        )}

        {messages.length === 0 && (
          <div className="empty-state">
            <p>Chat with Moly to get personalized message suggestions</p>
          </div>
        )}

        <div className="messages-container">
          {messages.map((msg) => (
            <div key={msg.id} className={`message ${msg.role}`}>
              <div className="message-content">{msg.content}</div>
              {msg.tone && <span className="message-tone">{msg.tone}</span>}
            </div>
          ))}

          {isLoading && (
            <div className="loading">
              <span className="spinner">...</span> Generating suggestions...
            </div>
          )}
        </div>
      </div>

      {/* SECTION 3: SUGGESTIONS (BOTTOM) */}
      {suggestions.length > 0 && (
        <div className="suggestions-section">
          <h4>Message Suggestions ({suggestions.length})</h4>
          <div className="suggestions-list">
            {suggestions.map((suggestion, idx) => (
              <div key={suggestion.id} className="suggestion-card">
                <div className="suggestion-header">
                  <span className="suggestion-number">Option {idx + 1}</span>
                  <span className="suggestion-confidence">
                    {Math.round(suggestion.confidence * 100)}% match
                  </span>
                </div>
                <div className="suggestion-text">{suggestion.text}</div>
                <div className="suggestion-footer">
                  <span className="suggestion-tone">{suggestion.tone}</span>
                  <button
                    className="copy-btn"
                    onClick={() => handleCopySuggestion(suggestion.text)}
                    title="Copy to clipboard"
                  >
                    Copy
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* CHAT INPUT (ALWAYS AT BOTTOM) */}
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
        placeholder="Ask Moly for suggestions..."
        value={input}
        onChange={(e) => setInput(e.target.value)}
        disabled={disabled}
        className="chat-input"
      />
      <button type="submit" disabled={disabled} className="send-btn" title="Send message to Moly">
        Send
      </button>
    </form>
  );
};
