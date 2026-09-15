import React, { useState, useCallback, useRef, useEffect } from 'react';
import { useAuth, type Session } from '@/hooks/useAuth';
import { useAboutMe } from '@/hooks/useAboutMe';
import { getBackendManager } from '@/api/backendManager';
import { LoginScreen } from './LoginScreen';
import { ReflectionsPanel } from './ReflectionsPanel';
import './chat-interface.css';

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  type?: 'text' | 'clarification_question' | 'incoming_message' | 'suggestions' | 'conflict_resolution';
  content: string;
  timestamp: number;
  metadata?: {
    linkedFacts?: any[];
    conflictOptions?: string[];
    suggestions?: string[];
    sender?: string;
    questionId?: string;
    factId?: string;
    ethicalIntervention?: 'blocked' | 'modified' | 'warned';
    ethicalReason?: string;
    ethicalNote?: string;
  };
}

interface ChatResponse {
  messageId: string;
  response: string;
  suggestions?: Array<{ text: string }>;
  contextLearned?: Record<string, unknown>;
  timestamp: number;
}

export const ChatInterface: React.FC = () => {
  const { session, logout } = useAuth();
  const { profile: aboutMe } = useAboutMe();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [incomingMessageInput, setIncomingMessageInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pendingClarificationId, setPendingClarificationId] = useState<string | null>(null);
  const [showIncomingInput, setShowIncomingInput] = useState(false);
  const [showReflections, setShowReflections] = useState(false);
  const [expandedEthicalNote, setExpandedEthicalNote] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [currentConversationId, setCurrentConversationId] = useState('');
  const messageCounterRef = useRef(0);

  const generateUniqueId = () => {
    return `msg_${Date.now()}_${++messageCounterRef.current}`;
  };

  const handleSettingsClick = () => {
    chrome.runtime.openOptionsPage();
  };

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const sendMessage = useCallback(async () => {
    if (!inputValue.trim() || !session) return;

    const userMessage = inputValue.trim();
    setInputValue('');
    setError(null);
    console.log('[ChatInterface] Sending message:', userMessage.substring(0, 50) + '...');

    // Add user message to display
    const userMsg: ChatMessage = {
      id: generateUniqueId(),
      role: 'user',
      content: userMessage,
      timestamp: Date.now(),
    };
    setMessages(prev => [...prev, userMsg]);
    setLoading(true);

    try {
      const apiBase = getBackendManager().getBackendUrl();
      const fullUrl = `${apiBase}/api/v2/message-processor`;
      const requestBody = {
        message: userMessage,
        conversationId: '', // Leave empty - backend creates conversation if needed
        aboutMe: aboutMe ? {
          communicationStyle: aboutMe.communicationStyle,
          coreValues: aboutMe.coreValues,
          tonePreference: aboutMe.tonePreference,
          preferences: aboutMe.preferences,
          goals: aboutMe.goals,
          patterns: aboutMe.patterns,
        } : {},
        selectedContactIds: [],
      };

      console.log('[ChatInterface] === DETAILED DEBUG ===');
      console.log('[ChatInterface] Backend URL:', apiBase);
      console.log('[ChatInterface] Full URL:', fullUrl);
      console.log('[ChatInterface] Session ID:', session.sessionId.substring(0, 20) + '...');
      console.log('[ChatInterface] Message:', userMessage);
      console.log('[ChatInterface] Request body:', JSON.stringify(requestBody));
      console.log('[ChatInterface] About Me available:', !!aboutMe);

      console.log('[ChatInterface] Starting fetch...');
      const response = await fetch(fullUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${session.sessionId}`,
        },
        body: JSON.stringify(requestBody),
      });

      console.log('[ChatInterface] === RESPONSE RECEIVED ===');
      console.log('[ChatInterface] Status:', response.status, response.statusText);
      console.log('[ChatInterface] Headers:', response.headers);
      console.log('[ChatInterface] URL that was called:', response.url);

      if (!response.ok) {
        const responseText = await response.text();
        console.error('[ChatInterface] Response failed:', response.status, response.statusText);
        console.error('[ChatInterface] Response body:', responseText);
        if (response.status === 401) {
          console.error('[ChatInterface] Session expired');
          setError('Session expired. Please log in again.');
          await logout();
          return;
        }
        throw new Error(`Chat failed: ${response.statusText}`);
      }

      const data = await response.json();
      console.log('[ChatInterface] Response received:', {
        facts: data.phase1?.facts?.length || 0,
        clarifications: data.action_required?.clarificationQs?.length || 0,
        response: data.response ? 'present' : 'MISSING',
      });

      // Handle clarification questions
      if (data.action_required?.clarificationQs && data.action_required.clarificationQs.length > 0) {
        console.log('[ChatInterface] Adding', data.action_required.clarificationQs.length, 'clarification questions');
        data.action_required.clarificationQs.forEach((q: any) => {
          console.log('[ChatInterface] Clarification question details:', { id: q.id, question: q.question, type: q.type });
          setMessages(prev => [...prev, {
            id: generateUniqueId(),
            role: 'assistant',
            type: 'clarification_question',
            content: q.question,
            timestamp: Date.now(),
            metadata: {
              questionId: q.id,
              factId: q.linkedFacts?.[0] || '',
            },
          }]);
        });

        if (data.action_required.clarificationQs[0]) {
          setPendingClarificationId(data.action_required.clarificationQs[0].id);
        }
      }

      // Add Moly's actual response (always present per backend contract)
      if (data.response) {
        const responseText = typeof data.response === 'string' ? data.response : String(data.response);
        console.log('[ChatInterface] Adding response from Moly', {
          type: typeof data.response,
          isString: typeof data.response === 'string',
          preview: responseText.substring(0, 100),
        });
        const assistantMsg: ChatMessage = {
          id: generateUniqueId(),
          role: 'assistant',
          type: 'text',
          content: responseText,
          timestamp: Date.now(),
          metadata: data.metadata ? {
            ethicalIntervention: data.metadata.ethicalIntervention,
            ethicalReason: data.metadata.blockReason || data.metadata.modificationReason || data.metadata.warningReason,
            ethicalNote: data.metadata.ethicalNote || data.metadata.ethicalWarning,
          } : undefined,
        };
        setMessages(prev => [...prev, assistantMsg]);
      } else {
        console.error('[ChatInterface] CRITICAL: No response from backend', {
          hasResponseField: 'response' in data,
          dataKeys: Object.keys(data),
        });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to send message';
      setError(message);
      console.error('[ChatInterface] Send message error:', message, { session: session ? 'exists' : 'null' });
    } finally {
      setLoading(false);
    }
  }, [inputValue, session, logout]);

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  // Handle clarification answer
  const handleClarificationAnswer = useCallback(async (answer: string, questionId: string, factId: string) => {
    if (!session) return;

    console.log('[ChatInterface] Processing clarification answer for question:', questionId);

    // Add user's answer to chat
    setMessages(prev => [...prev, {
      id: generateUniqueId(),
      role: 'user',
      content: answer,
      timestamp: Date.now(),
    }]);

    setLoading(true);
    setPendingClarificationId(null);

    try {
      const apiBase = getBackendManager().getBackendUrl();
      console.log('[ChatInterface] Sending answer to backend');
      const response = await fetch(`${apiBase}/api/v2/clarification/respond`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${session.sessionId}`,
        },
        body: JSON.stringify({
          questionId,
          userResponse: answer,
          factId,
        }),
      });

      if (!response.ok) {
        console.error('[ChatInterface] Answer processing failed:', response.status, response.statusText);
        throw new Error(`Failed to process answer: ${response.statusText}`);
      }

      const result = await response.json();
      console.log('[ChatInterface] Answer processed successfully');

      // Add Moly's acknowledgment
      if (result.molyReply) {
        console.log('[ChatInterface] Adding Moly reply');
        setMessages(prev => [...prev, {
          id: generateUniqueId(),
          role: 'assistant',
          type: 'text',
          content: result.molyReply,
          timestamp: Date.now(),
        }]);
      }

      // Check if more clarification needed
      if (result.needsMore && result.needsMore > 0) {
        console.log('[ChatInterface] More clarifications needed:', result.needsMore);
        // Generate next question (this would be handled by backend in real flow)
        setPendingClarificationId(null);
      } else {
        console.log('[ChatInterface] Clarification complete');
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to process answer';
      setError(message);
      console.error('[ChatInterface] Answer error:', message);
    } finally {
      setLoading(false);
    }
  }, [session]);

  // Handle incoming message analysis
  const handleAnalyzeIncomingMessage = useCallback(async () => {
    if (!incomingMessageInput.trim() || !session) return;

    const incomingMsg = incomingMessageInput.trim();
    console.log('[ChatInterface] Analyzing incoming message:', incomingMsg.substring(0, 50) + '...');
    setIncomingMessageInput('');
    setError(null);

    // Add incoming message to chat as system message
    setMessages(prev => [...prev, {
      id: generateUniqueId(),
      role: 'system',
      type: 'incoming_message',
      content: `📨 Incoming message: "${incomingMsg}"`,
      timestamp: Date.now(),
      metadata: { sender: '' },
    }]);

    setLoading(true);

    try {
      const apiBase = getBackendManager().getBackendUrl();
      console.log('[ChatInterface] Sending to incoming-message/analyze endpoint');
      const response = await fetch(`${apiBase}/api/v2/incoming-message/analyze`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${session.sessionId}`,
        },
        body: JSON.stringify({
          incomingMessage: incomingMsg,
          conversationId: '', // Leave empty - backend handles optional conversationId
        }),
      });

      if (!response.ok) {
        console.error('[ChatInterface] Analysis failed:', response.status, response.statusText);
        throw new Error(`Failed to analyze message: ${response.statusText}`);
      }

      const result = await response.json();
      console.log('[ChatInterface] Analysis complete. Sender:', result.sender, 'Suggestions:', result.suggestions?.length || 0);

      // Add Moly's response with suggestions
      setMessages(prev => [...prev, {
        id: generateUniqueId(),
        role: 'assistant',
        type: 'suggestions',
        content: result.sender
          ? `Here are some ways to respond to ${result.sender}:`
          : 'Here are some ways to respond:',
        timestamp: Date.now(),
        metadata: {
          sender: result.sender || 'Unknown',
          suggestions: result.suggestions || [],
        },
      }]);

      setShowIncomingInput(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to analyze message';
      setError(message);
      console.error('[ChatInterface] Incoming message error:', message);
    } finally {
      setLoading(false);
    }
  }, [incomingMessageInput, session]);

  if (!session) {
    return <LoginScreen />;
  }

  return (
    <div className="chat-interface">
      {/* Header */}
      <div className="chat-header">
        <div className="chat-title-section">
          <h2>Moly</h2>
        </div>
        <div className="header-actions" style={{ display: 'flex', gap: '4px' }}>
          <button
            className="icon-btn"
            onClick={() => setShowReflections(!showReflections)}
            title="Pending insights"
            style={{ color: showReflections ? '#667eea' : undefined }}
          >💭</button>
          <button
            className="icon-btn"
            onClick={handleSettingsClick}
            title="Settings"
          >⚙️</button>
          <button
            className="icon-btn"
            onClick={logout}
            title="Sign out"
            style={{ color: '#dc2626' }}
          >🚪</button>
        </div>
      </div>

      {/* Reflections Panel or Messages */}
      {showReflections ? (
        <ReflectionsPanel />
      ) : (
        <div className="chat-messages">
          {messages.length === 0 ? (
            <div className="chat-empty">
            <div className="empty-icon">💬</div>
            <h3>Start a conversation</h3>
            <p>Share what's on your mind and I'll help you think it through.</p>
          </div>
        ) : (
          messages.map(msg => {
            const avatar = msg.role === 'user' ? '👤' : msg.role === 'system' ? '📨' : '🤖';

            return (
              <div key={msg.id} className={`chat-message message-${msg.role} ${msg.type ? `message-type-${msg.type}` : ''}`}>
                <div className="message-avatar">{avatar}</div>
                <div className="message-content">
                  {/* Clarification question */}
                  {msg.type === 'clarification_question' && (
                    <div className="clarification-question">
                      <div className="question-text">{msg.content}</div>
                      <input
                        type="text"
                        className="clarification-input"
                        placeholder="Your answer..."
                        onKeyPress={(e) => {
                          if (e.key === 'Enter' && e.currentTarget.value.trim()) {
                            handleClarificationAnswer(
                              e.currentTarget.value,
                              msg.metadata?.questionId || '',
                              msg.metadata?.factId || ''
                            );
                            e.currentTarget.value = '';
                          }
                        }}
                      />
                    </div>
                  )}

                  {/* Suggestions */}
                  {msg.type === 'suggestions' && (
                    <div className="suggestions-container">
                      <div className="message-text">{msg.content}</div>
                      <div className="suggestions-list">
                        {msg.metadata?.suggestions?.map((suggestion, idx) => (
                          <div key={idx} className="suggestion-item">
                            <span className="suggestion-text">{suggestion}</span>
                            <button
                              className="copy-btn"
                              onClick={() => {
                                navigator.clipboard.writeText(suggestion);
                              }}
                              title="Copy to clipboard"
                            >
                              📋
                            </button>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Default text message */}
                  {(!msg.type || msg.type === 'text') && (
                    <div className="message-text">{msg.content}</div>
                  )}

                  {/* Ethical intervention disclosure */}
                  {msg.metadata?.ethicalIntervention && (
                    <div className="ethical-disclosure">
                      <button
                        className="ethical-disclosure-toggle"
                        onClick={() => setExpandedEthicalNote(
                          expandedEthicalNote === msg.id ? null : msg.id
                        )}
                        title={msg.metadata.ethicalIntervention === 'blocked' ?
                          'Moly chose not to help with this' :
                          'Moly adjusted this response'}
                      >
                        {msg.metadata.ethicalIntervention === 'blocked' ? '🚫' : '⚠️'}
                        {' '}
                        {msg.metadata.ethicalIntervention === 'blocked' ?
                          'Response not sent' :
                          msg.metadata.ethicalIntervention === 'modified' ?
                          'Response adjusted' :
                          'Response note'}
                      </button>
                      {expandedEthicalNote === msg.id && (
                        <div className="ethical-disclosure-content">
                          {msg.metadata.ethicalReason && (
                            <p className="ethical-reason">{msg.metadata.ethicalReason}</p>
                          )}
                          {msg.metadata.ethicalNote && (
                            <p className="ethical-note">{msg.metadata.ethicalNote}</p>
                          )}
                        </div>
                      )}
                    </div>
                  )}

                  <div className="message-time">
                    {new Date(msg.timestamp).toLocaleTimeString([], {
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </div>
                </div>
              </div>
            );
          })
        )}
        {loading && (
          <div className="chat-message message-assistant">
            <div className="message-avatar">🤖</div>
            <div className="message-content">
              <div className="typing-indicator">
                <span></span>
                <span></span>
                <span></span>
              </div>
            </div>
          </div>
        )}
          <div ref={messagesEndRef} />
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="chat-error">
          <span className="error-icon">⚠️</span>
          <p>{error}</p>
          <button onClick={() => setError(null)}>Dismiss</button>
        </div>
      )}

      {/* Incoming Message Input */}
      {showIncomingInput && (
        <div className="incoming-message-area">
          <div className="incoming-message-wrapper">
            <textarea
              className="incoming-message-input"
              placeholder="Paste incoming message here (e.g., 'From: Sarah\nHey, how are you?')..."
              value={incomingMessageInput}
              onChange={e => setIncomingMessageInput(e.target.value)}
              disabled={loading}
              rows={3}
            />
            <button
              className="analyze-button"
              onClick={handleAnalyzeIncomingMessage}
              disabled={!incomingMessageInput.trim() || loading}
              title="Analyze incoming message"
            >
              {loading ? '⏳' : '📨'}
            </button>
            <button
              className="close-incoming-button"
              onClick={() => setShowIncomingInput(false)}
              title="Close"
            >
              ✕
            </button>
          </div>
        </div>
      )}

      {/* Input */}
      <div className="chat-input-area">
        <div className="chat-input-wrapper">
          <textarea
            className="chat-input"
            placeholder="Share what's on your mind..."
            value={inputValue}
            onChange={e => setInputValue(e.target.value)}
            onKeyPress={handleKeyPress}
            disabled={loading}
            rows={3}
          />
          <button
            className="send-button"
            onClick={sendMessage}
            disabled={!inputValue.trim() || loading}
            title="Send message (Enter)"
          >
            {loading ? '⏳' : '→'}
          </button>
          <button
            className="incoming-message-toggle"
            onClick={() => setShowIncomingInput(!showIncomingInput)}
            title="Paste incoming message"
          >
            📨
          </button>
        </div>
      </div>
    </div>
  );
};
