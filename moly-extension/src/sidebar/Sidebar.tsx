import React, { useEffect, useState } from 'react';
import { useChatStore } from '@/stores/chatStore';
import { useSettingsStore, initializeSettings } from '@/stores/settingsStore';
import { useMolyAgent } from '@/hooks/useMolyAgent';
import { ChatHistory, MessageInput, Suggestions, SettingsPanel, ConversationSelector, NewConversationModal, ContactManager, ReflectionModal, BackendStatus, SafetyAlert, MeProfileModal } from './components';
import { Settings } from '@/settings/Settings';
import { ConversationAPI, type ConversationContextResponse } from '@/api/conversationAPI';
import { extractContactContextFromConversation, type ExtractedContactContext } from '@/utils/contextExtractor';
import type { Message } from './components';
import type { CommunicationContext, ChatMode, ConversationData } from '@/types';
import './sidebar.css';

interface Contact {
  id: string;
  name: string;
  platform: string;
  relationship: string;
}

export const Sidebar: React.FC = () => {
  const [currentConversation, setCurrentConversation] = useState<ConversationData | null>(null);
  const [conversationContext, setConversationContext] = useState<ConversationContextResponse | null>(null);
  const [showNewConversationModal, setShowNewConversationModal] = useState(false);
  const [showContactManager, setShowContactManager] = useState(false);
  const [showReflection, setShowReflection] = useState(false);
  const [conversationMessages, setConversationMessages] = useState<Message[]>([]);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [chatMode, setChatMode] = useState<ChatMode>('direct');
  const [context, setContext] = useState<CommunicationContext>('friendly');
  const [showSettings, setShowSettings] = useState(false);
  const [activeProvider, setActiveProvider] = useState<string>('');
  const [extractedContactContext, setExtractedContactContext] = useState<ExtractedContactContext | undefined>(undefined);
  const [isExtractingContext, setIsExtractingContext] = useState(false);
  const [showMeProfile, setShowMeProfile] = useState(false);
  const [meProfile, setMeProfile] = useState<any>(null);

  const { settings, loadSettings } = useSettingsStore();
  const { analyze, safety, constitution, questions, loading: analyzing, clear: clearAnalysis } = useMolyAgent();

  useEffect(() => {
    initializeSettings();
    loadSettings();
    loadConversationHistory();
    loadMeProfile();

  }, [loadSettings]);

  const loadMeProfile = async () => {
    try {
      const result = await chrome.storage.local.get('meProfile');
      if (result.meProfile) {
        setMeProfile(result.meProfile);
      } else {
        // Create default Me profile
        const defaultProfile = {
          id: 'me',
          name: 'Me',
          description: '',
          communicationStyle: '',
          values: '',
          goals: '',
          patterns: '',
          notes: '',
        };
        await chrome.storage.local.set({ meProfile: defaultProfile });
        setMeProfile(defaultProfile);
      }
    } catch (err) {
      console.error('[Sidebar] Failed to load Me profile:', err);
    }
  };

  // Fetch conversation context when conversation is selected
  useEffect(() => {
    if (!currentConversation) {
      setConversationContext(null);
      return;
    }

    const loadContext = async () => {
      try {
        // Try to fetch from backend first
        const backendAvailable = await ConversationAPI.isBackendAvailable();
        if (backendAvailable) {
          const ctx = await ConversationAPI.getConversationContext(
            currentConversation.id,
            false // don't include full history for now
          );
          if (ctx.success) {
            setConversationContext(ctx);
            console.log('[Sidebar] Loaded conversation context from backend');
          }
        }
      } catch (err) {
        console.warn('[Sidebar] Could not load context from backend:', err);
        // Fall back to local data in currentConversation
      }
    };

    loadContext();
  }, [currentConversation]);

  const loadConversationHistory = async () => {
    try {
      const result = await chrome.storage.local.get('conversations');
      if (result.conversations && Array.isArray(result.conversations)) {
        if (result.conversations.length > 0) {
          const conversation = result.conversations[0];
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

    const activeProviderConfig = settings?.providers[settings?.activeProvider];
    if (!activeProviderConfig?.enabled) {
      setError(`Please configure ${settings?.activeProvider || 'an LLM'} provider in Settings.`);
      return;
    }

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

    // Phase 1: Analyze for safety and ethics
    let analysisResults: any = null;
    try {
      // Build context string with full conversation info
      let contextString = 'No conversation selected';

      // Add user context first
      if (meProfile) {
        let meContext = 'About me: ';
        const parts = [];
        if (meProfile.communicationStyle) parts.push(`Communication style: ${meProfile.communicationStyle}`);
        if (meProfile.values) parts.push(`Values: ${meProfile.values}`);
        if (meProfile.goals) parts.push(`Goals: ${meProfile.goals}`);
        if (meProfile.patterns) parts.push(`Patterns: ${meProfile.patterns}`);
        if (parts.length > 0) {
          contextString = meContext + parts.join('. ') + '.';
        }
      }

      if (currentConversation) {
        // Build member list with names and notes for personalization
        const membersList = currentConversation.members
          .map(m => m.notes ? `${m.name} (${m.notes})` : m.name)
          .join('; ');
        const purpose = currentConversation.purpose ? ` Purpose: ${currentConversation.purpose}.` : '';
        const conversationContext = `Conversation: "${currentConversation.name}" (${currentConversation.type}). Members: ${membersList}.${purpose}`;

        contextString = contextString === 'No conversation selected'
          ? conversationContext
          : contextString + ' ' + conversationContext;

        // If we have full context from backend, include member details
        if (conversationContext?.members && conversationContext.members.length > 0) {
          const memberDetails = conversationContext.members
            .map(m => {
              let detail = `${m.name} (${m.relationship})`;
              if (m.notes) detail += `: ${m.notes}`;
              return detail;
            })
            .join('; ');
          contextString += ` Details: ${memberDetails}.`;
        }
      }

      analysisResults = await analyze(
        userMessage,
        currentConversation?.name || 'Unknown',
        contextString
      );
    } catch (err) {
      console.warn('[Moly] Backend analysis not available:', err);
    }

    // Phase 2: Check if we should gate suggestions based on analysis results
    const hasCrisis = analysisResults?.safety?.alert_type === 'crisis' || analysisResults?.safety?.alert_type === 'illegal';
    const hasEthicsViolations = analysisResults?.constitution?.violations && analysisResults.constitution.violations.length > 0;

    // If crisis detected, don't generate suggestions - show resources instead
    if (hasCrisis) {
      const molyMsg: Message = {
        id: (Date.now() + 1).toString(),
        type: 'moly',
        content: 'I detected a safety concern. Your wellbeing comes first. Please reach out to one of the resources shown above.',
        timestamp: Date.now(),
        metadata: { mode: chatMode, context },
      };
      const messagesWithResponse = [...updatedMessages, molyMsg];
      setConversationMessages(messagesWithResponse);
      saveConversationHistory(messagesWithResponse);
      setIsLoading(false);
      return;
    }

    // Phase 3: If ethics violations, ask clarifying questions instead of suggesting
    if (hasEthicsViolations) {
      const clarifyingQuestions = [
        'Can you tell me more about why you want to do this?',
        'How do you think this might affect the other person?',
        'Are there other options you\'ve considered?',
        'What outcome are you hoping for?'
      ];

      const molyMsg: Message = {
        id: (Date.now() + 1).toString(),
        type: 'moly',
        content: `I want to understand this better. ${clarifyingQuestions[Math.floor(Math.random() * clarifyingQuestions.length)]}`,
        timestamp: Date.now(),
        metadata: { mode: chatMode, context },
      };

      const messagesWithResponse = [...updatedMessages, molyMsg];
      setConversationMessages(messagesWithResponse);
      saveConversationHistory(messagesWithResponse);
      setIsLoading(false);
      return;
    }

    // Phase 4: Only generate suggestions if safe and ethical
    try {
      const response = await chrome.runtime.sendMessage({
        type: 'GENERATE_SUGGESTIONS',
        data: {
          context: 'Conversation history provided for context',
          communicationContext: context,
          userMessage,
          mode: chatMode,
          conversationHistory: updatedMessages,
        },
      });

      if (response.success && response.suggestions) {
        setSuggestions(response.suggestions);
        setActiveProvider(response.provider || 'Unknown');

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

        // Extract context and show reflection modal after a brief delay
        setTimeout(() => {
          if (currentConversation) {
            setIsExtractingContext(true);
            // Extract contact context from conversation
            const extracted = extractContactContextFromConversation(
              messagesWithResponse,
              currentConversation.name
            );
            setExtractedContactContext(extracted);
            setIsExtractingContext(false);
            setShowReflection(true);
          }
        }, 1500);
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
    link.download = `moly-conversation-${Date.now()}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleOpenSettings = () => {
    setShowSettings(!showSettings);
  };

  const handleSaveContactContext = async (context: ExtractedContactContext) => {
    if (!currentConversation || currentConversation.members.length === 0) return;

    // Get the first contact from the conversation members
    const contactId = currentConversation.members[0].id.toString();

    try {
      // Load contacts from storage
      const result = await chrome.storage.local.get('contacts');
      const contacts = result.contacts || [];

      // Find and update the contact
      const updatedContacts = contacts.map(c => {
        if (c.id === contactId) {
          // Build new notes from extracted context
          const contextNotes: string[] = [];
          if (context.characteristics?.length) {
            contextNotes.push(`Characteristics: ${context.characteristics.join('; ')}`);
          }
          if (context.intentions?.length) {
            contextNotes.push(`Intentions: ${context.intentions.join('; ')}`);
          }
          if (context.behaviors?.length) {
            contextNotes.push(`Behaviors: ${context.behaviors.join('; ')}`);
          }

          const newContextText = contextNotes.join('\n');
          const updatedNotes = c.notes
            ? `${c.notes}\n\n${newContextText}`
            : newContextText;

          return {
            ...c,
            notes: updatedNotes,
          };
        }
        return c;
      });

      // Save updated contacts
      await chrome.storage.local.set({ contacts: updatedContacts });
      console.log('[Sidebar] Saved contact context to', currentConversation.members[0].name);
    } catch (err) {
      console.error('[Sidebar] Failed to save contact context:', err);
    }
  };

  const handleSaveReflection = async (notes: string) => {
    if (!currentConversation) return;

    // Update conversation with notes
    const updated: ConversationData = {
      ...currentConversation,
      notes: (currentConversation.notes ? currentConversation.notes + '\n\n' : '') + notes,
      updated_at: Date.now(),
    };

    try {
      // Update local storage
      const result = await chrome.storage.local.get('conversations');
      const conversations = (result.conversations || []).map(c =>
        c.id === currentConversation.id ? updated : c
      );
      await chrome.storage.local.set({ conversations });

      // Sync to backend
      ConversationAPI.syncConversationToBackend(updated).catch(err =>
        console.warn('[Sidebar] Could not sync reflection to backend:', err)
      );

      setCurrentConversation(updated);
      setShowReflection(false);
      setExtractedContactContext(undefined);
    } catch (err) {
      console.error('[Sidebar] Failed to save reflection:', err);
    }
  };

  return (
    <div className="sidebar-container">
      <div className="sidebar-header">
        <h2>Moly</h2>
        <div className="header-actions">
          <button
            className="icon-btn"
            onClick={() => setShowMeProfile(true)}
            title="About me"
          >
            ℹ️
          </button>
          <button
            className="icon-btn"
            onClick={() => setShowContactManager(true)}
            title="Manage contacts"
          >
            👥
          </button>
          <button
            className="icon-btn"
            onClick={handleOpenSettings}
            title="Open settings"
          >
            ⚙️
          </button>
          <button
            className="icon-btn"
            onClick={() => {
              console.log('[Sidebar] Sending CLOSE_SIDEPANEL message...');
              chrome.runtime.sendMessage({ type: 'CLOSE_SIDEPANEL' });
            }}
            title="Close sidebar"
          >
            ✕
          </button>
        </div>
      </div>

      <div className="sidebar-content">
        {showSettings ? (
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
            {!showContactManager && (
              <>
                <BackendStatus />

                {safety && safety.alert_type !== 'none' && (
                  <SafetyAlert alert={safety} onDismiss={clearAnalysis} />
                )}
              </>
            )}

            {constitution && constitution.violations && constitution.violations.length > 0 && (
              <div style={{
                padding: '12px',
                marginBottom: '12px',
                background: '#fef3c7',
                border: '1px solid #fcd34d',
                borderRadius: '6px',
                fontSize: '13px',
                color: '#78350f'
              }}>
                <strong>Ethics Check:</strong> {constitution.violations.length} concern(s)
                {constitution.violations.slice(0, 2).map((v) => (
                  <div key={v.principle_id} style={{ marginTop: '6px', fontSize: '12px' }}>
                    • {v.principle}: {v.description}
                  </div>
                ))}
                {constitution.violations.length > 2 && (
                  <div style={{ marginTop: '6px', fontSize: '12px', opacity: 0.8 }}>
                    +{constitution.violations.length - 2} more...
                  </div>
                )}
              </div>
            )}

            {analyzing && (
              <div style={{
                padding: '8px 12px',
                marginBottom: '12px',
                fontSize: '12px',
                color: '#6366f1',
                textAlign: 'center'
              }}>
                ⏳ Analyzing message for safety & ethics...
              </div>
            )}

            {!showContactManager && (
              <>
                <ConversationSelector
                  onSelectConversation={setCurrentConversation}
                  onNewConversation={() => setShowNewConversationModal(true)}
                  currentConversation={currentConversation}
                />
              </>
            )}

            <NewConversationModal
              isOpen={showNewConversationModal}
              onClose={() => setShowNewConversationModal(false)}
              onSave={(conversation) => {
                setCurrentConversation(conversation);
                setShowNewConversationModal(false);

                // Sync to backend (optional - local storage is primary)
                ConversationAPI.syncConversationToBackend(conversation).catch(err =>
                  console.warn('[Sidebar] Could not sync conversation to backend:', err)
                );
              }}
            />

            <ContactManager
              isOpen={showContactManager}
              onClose={() => setShowContactManager(false)}
            />

            <MeProfileModal
              isOpen={showMeProfile}
              onClose={() => setShowMeProfile(false)}
              onSave={(profile) => {
                setMeProfile(profile);
              }}
            />

            <ReflectionModal
              isOpen={showReflection}
              conversationName={currentConversation?.name || 'this person'}
              onClose={() => {
                setShowReflection(false);
                setExtractedContactContext(undefined);
              }}
              onSave={handleSaveReflection}
              onSaveContactContext={handleSaveContactContext}
              extractedContext={extractedContactContext}
              isExtractingContext={isExtractingContext}
            />

            <ChatHistory
              messages={conversationMessages}
              onDeleteMessage={handleDeleteMessage}
              onExport={handleExportConversation}
            />

            <MessageInput
              onSend={handleSendMessage}
              disabled={isLoading}
              placeholder="Type a message or paste from chat..."
            />

            {analyzing && suggestions.length === 0 && (
              <div style={{
                padding: '16px',
                marginBottom: '12px',
                background: '#ede9fe',
                border: '1px solid #c4b5fd',
                borderRadius: '6px',
                textAlign: 'center',
                color: '#6d28d9'
              }}>
                <div style={{ marginBottom: '8px' }}>⏳ Generating suggestions...</div>
                <div style={{
                  display: 'inline-block',
                  width: '20px',
                  height: '20px',
                  border: '2px solid #c4b5fd',
                  borderTop: '2px solid #6d28d9',
                  borderRadius: '50%',
                  animation: 'spin 0.8s linear infinite'
                }} />
              </div>
            )}

            {suggestions.length > 0 && (
              <Suggestions
                suggestions={suggestions}
                loading={isLoading}
                onCopy={handleCopySuggestion}
                error={error || undefined}
              />
            )}

            {activeProvider && (
              <div className="provider-status">
                <p className="provider-label">Using: {activeProvider}</p>
              </div>
            )}

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

      {!showSettings && (
        <SettingsPanel
          llmProvider={settings?.activeProvider || 'Not configured'}
          onSettingsOpen={handleOpenSettings}
        />
      )}
    </div>
  );
};

export default Sidebar;
