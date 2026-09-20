import React, { useState, useCallback, useRef, useEffect } from 'react';
import { useAuth, type Session } from '@/hooks/useAuth';
import { useAboutMe } from '@/hooks/useAboutMe';
import { getBackendManager } from '@/api/backendManager';
import { LoginScreen } from './LoginScreen';
import { ReflectionsPanel } from './ReflectionsPanel';
import { MetricsPanel } from './MetricsPanel';
import { BehavioralInsightsPanel } from './BehavioralInsightsPanel';
import { ConversationHistoryPanel } from './ConversationHistoryPanel';
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
    // Socratic metadata
    socraticApproach?: string;
    expectedInsights?: string[];
    targetsPrinciple?: string;
    depthLevel?: number;
    violatedPrinciples?: string[];
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
  const [showMetrics, setShowMetrics] = useState(false);
  const [showInsights, setShowInsights] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [expandedEthicalNote, setExpandedEthicalNote] = useState<string | null>(null);
  const [browserSessionId, setBrowserSessionId] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [currentConversationId, setCurrentConversationId] = useState('');
  const messageCounterRef = useRef(0);

  // Initialize browser session ID on component mount
  useEffect(() => {
    const STORAGE_KEY = 'moly_browser_session_id';
    let sessionId = localStorage.getItem(STORAGE_KEY);

    if (!sessionId) {
      // Generate new session ID (UUID-like format)
      sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      localStorage.setItem(STORAGE_KEY, sessionId);
      console.log('[ChatInterface] Generated new browser session ID:', sessionId);
    } else {
      console.log('[ChatInterface] Using existing browser session ID:', sessionId);
    }

    setBrowserSessionId(sessionId);
  }, []);

  const generateUniqueId = () => {
    return `msg_${Date.now()}_${++messageCounterRef.current}`;
  };

  const handleSettingsClick = () => {
    chrome.runtime.openOptionsPage();
  };

  const handleSelectConversation = useCallback(async (conversationId: string) => {
    if (!session) return;

    try {
      console.log('[ChatInterface] Loading conversation:', conversationId);
      const apiBase = getBackendManager().getBackendUrl();
      const response = await fetch(`${apiBase}/api/v2/messages?conversationId=${conversationId}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${session.sessionId}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to load conversation: ${response.statusText}`);
      }

      const data = await response.json();
      const loadedMessages = (data.messages || []).map((msg: any, idx: number) => ({
        id: msg.id || `loaded_${idx}`,
        role: msg.role === 'assistant' ? 'assistant' : 'user',
        content: msg.content,
        timestamp: msg.created_at * 1000 || Date.now(),
      }));

      setMessages(loadedMessages);
      setCurrentConversationId(conversationId);
      setShowHistory(false);
      console.log('[ChatInterface] Loaded', loadedMessages.length, 'messages from conversation');
    } catch (err) {
      let message = 'Failed to load conversation';
      if (err instanceof Error) {
        message = err.message;
      }
      setError(String(message));
      console.error('[ChatInterface] Load error:', message);
    }
  }, [session]);

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
        conversationId: currentConversationId || '', // Use current conversation or leave empty to create new
        aboutMe: aboutMe ? {
          communicationStyle: aboutMe.communicationStyle,
          coreValues: aboutMe.coreValues,
          tonePreference: aboutMe.tonePreference,
          preferences: aboutMe.preferences,
          goals: aboutMe.goals,
          patterns: aboutMe.patterns,
        } : {},
        selectedContactIds: [],
        browserSessionId: browserSessionId, // Browser session ID for detecting new sessions
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
        console.log('[ChatInterface] ✓ Processing', data.action_required.clarificationQs.length, 'clarification questions');
        data.action_required.clarificationQs.forEach((q: any, idx: number) => {
          console.log(`[ChatInterface] Question ${idx + 1} metadata extraction:`, {
            id: q.id,
            question: q.question.substring(0, 50) + '...',
            type: q.type,
            // Socratic metadata
            socraticApproach: q.socraticApproach || 'MISSING',
            expectedInsights: q.expectedInsights?.length || 0,
            targetsPrinciple: q.targetsPrinciple || 'NONE',
            depthLevel: q.depthLevel || 'UNSET',
            linkedFacts: q.linkedFacts?.length || 0,
          });

          // Verify all Socratic fields are present
          if (!q.socraticApproach) {
            console.warn('[ChatInterface] ⚠️ Question missing socraticApproach');
          }
          if (!q.expectedInsights || q.expectedInsights.length === 0) {
            console.warn('[ChatInterface] ⚠️ Question missing expectedInsights');
          }
          if (!q.depthLevel) {
            console.warn('[ChatInterface] ⚠️ Question missing depthLevel');
          }

          setMessages(prev => [...prev, {
            id: generateUniqueId(),
            role: 'assistant',
            type: 'clarification_question',
            content: q.question,
            timestamp: Date.now(),
            metadata: {
              questionId: q.id,
              factId: q.linkedFacts?.[0] || '',
              // Socratic metadata - ALL fields should be extracted
              socraticApproach: q.socraticApproach,
              expectedInsights: q.expectedInsights,
              targetsPrinciple: q.targetsPrinciple,
              depthLevel: q.depthLevel,
            },
          }]);
          console.log(`[ChatInterface] ✓ Question ${idx + 1} added with all metadata`);
        });

        if (data.action_required.clarificationQs[0]) {
          setPendingClarificationId(data.action_required.clarificationQs[0].id);
          console.log('[ChatInterface] Set pending clarification ID:', data.action_required.clarificationQs[0].id);
        }
      }

      // Add Moly's actual response (always present per backend contract)
      if (data.response) {
        const responseText = typeof data.response === 'string' ? data.response : String(data.response);
        console.log('[ChatInterface] ✓ Adding Moly response:', {
          type: typeof data.response,
          isString: typeof data.response === 'string',
          length: responseText.length,
          preview: responseText.substring(0, 80),
        });

        // Extract ethical metadata with detailed logging
        const ethicalIntervention = data.metadata?.ethicalIntervention;
        const ethicalReason = data.metadata?.blockReason || data.metadata?.modificationReason || data.metadata?.warningReason;
        const ethicalNote = data.metadata?.ethicalNote || data.metadata?.ethicalWarning;
        const violatedPrinciples = data.metadata?.violatedPrinciples;

        console.log('[ChatInterface] Ethical metadata extraction:', {
          hasMetadata: !!data.metadata,
          ethicalIntervention: ethicalIntervention || 'NONE',
          hasReason: !!ethicalReason,
          hasNote: !!ethicalNote,
          violatedPrinciples: violatedPrinciples?.length || 0,
        });

        // Verify violated principles are present if expected
        if (violatedPrinciples && violatedPrinciples.length > 0) {
          console.log('[ChatInterface] ✓ Violated principles detected:', violatedPrinciples);
        } else if (ethicalIntervention) {
          console.log('[ChatInterface] ℹ️ Ethical intervention present but no violated principles:', ethicalIntervention);
        }

        const assistantMsg: ChatMessage = {
          id: generateUniqueId(),
          role: 'assistant',
          type: 'text',
          content: responseText,
          timestamp: Date.now(),
          metadata: data.metadata ? {
            ethicalIntervention: ethicalIntervention,
            ethicalReason: ethicalReason,
            ethicalNote: ethicalNote,
            violatedPrinciples: violatedPrinciples,
          } : undefined,
        };

        // Add ethical intervention badge if present
        let displayContent = responseText;
        if (ethicalIntervention) {
          const badge = `[Moly adjusted this response: ${ethicalIntervention}]`;
          displayContent = responseText + '\n\n_' + badge + '_';
          if (ethicalReason) {
            displayContent += '\n_Reason: ' + ethicalReason + '_';
          }
          console.log('[ChatInterface] ✓ Added ethical intervention badge');
        }

        // Update message content with badge if needed
        assistantMsg.content = displayContent;

        setMessages(prev => [...prev, assistantMsg]);
        console.log('[ChatInterface] ✓ Moly response added with metadata');
      } else {
        console.error('[ChatInterface] 🚨 CRITICAL: No response from backend', {
          hasResponseField: 'response' in data,
          dataKeys: Object.keys(data),
          metadata: data.metadata,
        });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to send message';
      setError(message);
      console.error('[ChatInterface] Send message error:', message, { session: session ? 'exists' : 'null' });
    } finally {
      setLoading(false);
    }
  }, [inputValue, session, logout, browserSessionId, aboutMe, currentConversationId]);

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
            onClick={() => { setShowHistory(!showHistory); setShowReflections(false); setShowMetrics(false); setShowInsights(false); }}
            title="Previous conversations"
            style={{ color: showHistory ? '#667eea' : undefined }}
          >📜</button>
          <button
            className="icon-btn"
            onClick={() => { setShowReflections(!showReflections); setShowMetrics(false); setShowHistory(false); setShowInsights(false); }}
            title="Pending insights"
            style={{ color: showReflections ? '#667eea' : undefined }}
          >💭</button>
          <button
            className="icon-btn"
            onClick={() => { setShowMetrics(!showMetrics); setShowReflections(false); setShowHistory(false); setShowInsights(false); }}
            title="Learning metrics"
            style={{ color: showMetrics ? '#667eea' : undefined }}
          >📊</button>
          <button
            className="icon-btn"
            onClick={() => { setShowInsights(!showInsights); setShowReflections(false); setShowMetrics(false); setShowHistory(false); }}
            title="How Moly sees you"
            style={{ color: showInsights ? '#667eea' : undefined }}
          >✨</button>
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

      {/* Tabs Content */}
      {showHistory ? (
        <ConversationHistoryPanel onSelectConversation={handleSelectConversation} />
      ) : showReflections ? (
        <ReflectionsPanel />
      ) : showMetrics ? (
        <MetricsPanel />
      ) : showInsights ? (
        <BehavioralInsightsPanel />
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
                      {/* Socratic approach badge */}
                      {msg.metadata?.socraticApproach && (
                        <div className="socratic-badge">
                          <span className="approach-label">
                            {msg.metadata.socraticApproach.split('_').map(word =>
                              word.charAt(0).toUpperCase() + word.slice(1)
                            ).join(' ')}
                          </span>
                        </div>
                      )}

                      <div className="question-text">{msg.content}</div>

                      {/* Expected insights collapsible */}
                      {msg.metadata?.expectedInsights && msg.metadata.expectedInsights.length > 0 && (
                        <details className="expected-insights">
                          <summary>Why we're asking this</summary>
                          <div className="insights-content">
                            {msg.metadata.expectedInsights.map((insight, idx) => (
                              <p key={idx}>{insight}</p>
                            ))}
                          </div>
                        </details>
                      )}

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

                  {/* Principle violations */}
                  {msg.metadata?.violatedPrinciples && msg.metadata.violatedPrinciples.length > 0 && (
                    <div className="principle-violations">
                      {msg.metadata.violatedPrinciples.map((principle, idx) => (
                        <div key={idx} className={`violation-badge ${principle.includes('critical') ? 'critical' : ''}`}>
                          {principle.split('_').map(word =>
                            word.charAt(0).toUpperCase() + word.slice(1)
                          ).join(' ')}
                        </div>
                      ))}
                    </div>
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
