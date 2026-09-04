import React, { useEffect, useState } from 'react';
import { useChatStore } from '@/stores/chatStore';
import { useSettingsStore, initializeSettings } from '@/stores/settingsStore';
import { ChatHistory, MessageInput, Suggestions, SettingsPanel, ContactSelector } from './components';
import { Settings } from '@/settings/Settings';
import type { Message } from './components';
import type { CommunicationContext, ChatMode } from '@/types';
import './sidebar.css';

interface Contact {
  id: string;
  name: string;
  platform: string;
  relationship: string;
}

export const Sidebar: React.FC = () => {
  const [currentContact, setCurrentContact] = useState<Contact | null>(null);
  const [conversationMessages, setConversationMessages] = useState<Message[]>([]);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [chatMode, setChatMode] = useState<ChatMode>('direct');
  const [context, setContext] = useState<CommunicationContext>('friendly');
  const [showSettings, setShowSettings] = useState(false);
  const [activeProvider, setActiveProvider] = useState<string>('');
  const [ollamaDetected, setOllamaDetected] = useState<boolean>(false);

  const { settings, loadSettings } = useSettingsStore();

  useEffect(() => {
    initializeSettings();
    loadSettings();
    loadConversationHistory();
  }, [loadSettings]);

  const loadConversationHistory = async () => {
    try {
      const result = await chrome.storage.local.get('conversations');
      if (result.conversations && Array.isArray(result.conversations)) {
        // Load first conversation by default
        if (result.conversations.length > 0) {
          const conversation = result.conversations[0];
          setCurrentContact(conversation.contactName || 'Unknown');
          setConversationMessages(conversation.messages || []);
          setChatMode(conversation.settings?.mode || 'direct');
          setContext(conversation.settings?.context || 'friendly');
        }
      }
    } catch (err) {
      console.error('Failed to load conversation history:', err);
    }
  };

  const saveConversationHistory = async (messages: Message[]) => {
    try {
      const result = await chrome.storage.local.get('conversations');
      const conversations = result.conversations || [];

      if (conversations.length === 0) {
        conversations.push({
          id: Date.now().toString(),
          contactName: currentContact,
          messages,
          settings: { mode: chatMode, context, llmProvider: settings?.activeProvider || 'claude' },
          createdAt: Date.now(),
          updatedAt: Date.now(),
        });
      } else {
        conversations[0] = {
          ...conversations[0],
          messages,
          settings: { mode: chatMode, context, llmProvider: settings?.activeProvider || 'claude' },
          updatedAt: Date.now(),
        };
      }

      await chrome.storage.local.set({ conversations });
    } catch (err) {
      console.error('Failed to save conversation history:', err);
    }
  };

  const handleSendMessage = async (userMessage: string) => {
    if (!userMessage.trim()) return;

    // Validate LLM provider is configured
    const activeProvider = settings?.providers[settings?.activeProvider];
    if (!activeProvider?.enabled) {
      setError(`Please configure ${settings?.activeProvider || 'an LLM'} provider in Settings.`);
      return;
    }

    // Add user message to conversation
    const userMsg: Message = {
      id: Date.now().toString(),
      type: 'user',
      content: userMessage,
      timestamp: Date.now(),
      metadata: { mode: chatMode, context },
    };

    const updatedMessages = [...conversationMessages, userMsg];
    setConversationMessages(updatedMessages);
    saveConversationHistory(updatedMessages);
    setError(null);
    setIsLoading(true);

    try {
      // Call background script to generate suggestions
      const response = await chrome.runtime.sendMessage({
        type: 'GENERATE_SUGGESTIONS',
        data: {
          context: currentContact,
          communicationContext: context,
          userMessage,
          mode: chatMode,
          conversationHistory: conversationMessages,
        },
      });

      if (response.success && response.suggestions) {
        setSuggestions(response.suggestions);
        setActiveProvider(response.provider || 'Unknown');

        // Add Moly response with suggestions
        const molyMsg: Message = {
          id: (Date.now() + 1).toString(),
          type: 'moly',
          content: `I've generated ${response.suggestions.length} response suggestions for you.`,
          timestamp: Date.now(),
          metadata: { mode: chatMode, context },
        };

        const messagesWithResponse = [...updatedMessages, molyMsg];
        setConversationMessages(messagesWithResponse);
        saveConversationHistory(messagesWithResponse);
      } else if (!response.success) {
        setError(response.error || 'Failed to generate suggestions');
      }
    } catch (err) {
      setError(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopySuggestion = (text: string) => {
    // Add suggestion to conversation history
    const suggestionMsg: Message = {
      id: (Date.now() + 2).toString(),
      type: 'suggestion',
      content: text,
      timestamp: Date.now(),
    };

    const updatedMessages = [...conversationMessages, suggestionMsg];
    setConversationMessages(updatedMessages);
    saveConversationHistory(updatedMessages);
  };

  const handleDeleteMessage = (id: string) => {
    const filtered = conversationMessages.filter((msg) => msg.id !== id);
    setConversationMessages(filtered);
    saveConversationHistory(filtered);
  };

  const handleExportConversation = () => {
    const dataStr = JSON.stringify(
      {
        contact: currentContact,
        messages: conversationMessages,
        exportedAt: new Date().toISOString(),
      },
      null,
      2
    );

    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `moly-conversation-${currentContact}-${Date.now()}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleOpenSettings = () => {
    setShowSettings(!showSettings);
  };

  const handleModeChange = (newMode: ChatMode) => {
    setChatMode(newMode);
    saveConversationHistory(conversationMessages);
  };

  const handleContextChange = (newContext: CommunicationContext) => {
    setContext(newContext);
    saveConversationHistory(conversationMessages);
  };

  return (
    <div className="sidebar-container">
      {/* HEADER */}
      <div className="sidebar-header">
        <h2>Moly</h2>
        <div className="header-actions">
          <button
            className="icon-btn"
            onClick={handleOpenSettings}
            title="Open settings"
          >
            ⚙️
          </button>
        </div>
      </div>

      {/* MAIN CONTENT - SCROLLABLE */}
      <div className="sidebar-content">
        {showSettings ? (
          // FULL SETTINGS VIEW
          <div style={{ overflow: 'auto', height: '100%' }}>
            <div style={{ padding: '16px' }}>
              <button
                onClick={() => setShowSettings(false)}
                style={{
                  width: '100%',
                  padding: '8px',
                  marginBottom: '16px',
                  background: '#6366f1',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontWeight: '600',
                }}
              >
                Back to Chat
              </button>
            </div>
            <Settings />
          </div>
        ) : (
          <>
            {/* CONTACT SELECTOR */}
            <ContactSelector
              onSelectContact={setCurrentContact}
              currentContact={currentContact}
            />

            {/* CHAT HISTORY */}
            <ChatHistory
              messages={conversationMessages}
              onDeleteMessage={handleDeleteMessage}
              onExport={handleExportConversation}
            />

            {/* MESSAGE INPUT */}
            <MessageInput
              onSend={handleSendMessage}
              disabled={isLoading}
              placeholder="Type a message or paste from chat..."
            />

            {/* SUGGESTIONS */}
            {suggestions.length > 0 && (
              <Suggestions
                suggestions={suggestions}
                loading={isLoading}
                onCopy={handleCopySuggestion}
                error={error || undefined}
              />
            )}

            {/* PROVIDER STATUS */}
            {activeProvider && (
              <div className="provider-status">
                <p className="provider-label">Using: {activeProvider}</p>
              </div>
            )}

            {/* ERROR MESSAGE */}
            {error && (
              <div className="error-banner">
                <p>{error}</p>
                <button onClick={() => setError(null)} className="close-error">
                  ✕
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* SETTINGS PANEL - Bottom controls */}
      {!showSettings && (
        <SettingsPanel
          mode={chatMode}
          context={context}
          llmProvider={settings?.activeProvider || 'Not configured'}
          onModeChange={handleModeChange}
          onContextChange={handleContextChange}
          onSettingsOpen={handleOpenSettings}
        />
      )}
    </div>
  );
};

export default Sidebar;
