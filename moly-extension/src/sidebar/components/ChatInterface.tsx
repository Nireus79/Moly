import React, { useState, useCallback, useRef, useEffect } from 'react';
import { useAuth, type Session } from '@/hooks/useAuth';
import { useAboutMe } from '@/hooks/useAboutMe';
import { getBackendManager } from '@/api/backendManager';
import { LoginScreen } from './LoginScreen';
import { ReflectionsPanel } from './ReflectionsPanel';
import { ConversationHistoryPanel } from './ConversationHistoryPanel';
import { OrchestratorInsightsPanel } from './OrchestratorInsightsPanel';
import { OrchestratorInsights } from '@/types/apiTypes';
import { SponsorCard } from '@/components/SponsorCard';
import { Icon } from '@/components/Icon';
import { markChatStarted } from '../chatSession';
import { isPhoneApp, phoneBridge } from '@/utils/platform';
import {
  PENDING_POLL_MS,
  clearPending,
  lastConversation,
  loadPending,
  pickNewConversation,
  rememberConversation,
  replyArrived,
  savePending,
} from '../pendingTurn';
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
    // The server says the user may skip Moly's question and take the result with what is known
    canSkip?: boolean;
  };
}

// The system-insights panel shows Moly's internal scores. It is for developers: switch it on in the console with
// localStorage.setItem('moly_debug_insights', '1').
function showDeveloperInsights(): boolean {
  try {
    return localStorage.getItem('moly_debug_insights') === '1';
  } catch {
    return false;
  }
}

// What the skip button sends as the user's message. The server reads skipQuestions, not this text.
const SKIP_MESSAGE = 'Go ahead with what you have.';

interface ChatResponse {
  messageId: string;
  response: string;
  suggestions?: Array<{ text: string }>;
  contextLearned?: Record<string, unknown>;
  timestamp: number;
}

interface ChatInterfaceProps {
  onSettingsClick?: () => void;
}

export const ChatInterface: React.FC<ChatInterfaceProps> = ({ onSettingsClick }) => {
  const { session, logout } = useAuth();
  // The local profile of an installed Moly has nothing to sign out of.
  const isLocalProfile = (session as { local?: boolean } | null)?.local === true;
  const { profile: aboutMe } = useAboutMe();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pendingClarificationId, setPendingClarificationId] = useState<string | null>(null);
  const [showReflections, setShowReflections] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [expandedEthicalNote, setExpandedEthicalNote] = useState<string | null>(null);
  const [browserSessionId, setBrowserSessionId] = useState('');
  const [orchestratorInsights, setOrchestratorInsights] = useState<OrchestratorInsights | null>(null);
  const messagesListRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const [currentConversationId, setCurrentConversationId] = useState('');
  const messageCounterRef = useRef(0);
  // Phone app: a sent message whose reply is being waited for after the app was closed or lost its connection
  const phone = isPhoneApp();
  const [waiting, setWaiting] = useState(false);
  const restoredRef = useRef(false);
  const startedNewChatRef = useRef(false);

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

  // The skip button is always shown and faded when it cannot be used. The server decides (canSkip on Moly's last reply:
  // at least one question answered, and the question is one the user may skip) and checks again when it is pressed.
  const lastMessage = messages[messages.length - 1];
  const canSkip = !loading && lastMessage?.role === 'assistant' && lastMessage.metadata?.canSkip === true;

  const generateUniqueId = () => {
    return `msg_${Date.now()}_${++messageCounterRef.current}`;
  };

  const handleSettingsClick = () => {
    onSettingsClick?.();
  };

  const fetchMessages = useCallback(async (conversationId: string): Promise<ChatMessage[]> => {
    if (!session) return [];
    const apiBase = getBackendManager().getBackendUrl();
    const response = await fetch(`${apiBase}/api/messages?conversationId=${conversationId}`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${session.sessionId}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to load conversation: ${response.statusText}`);
    }

    const data = await response.json();
    return (data.messages || []).map((msg: any, idx: number) => ({
      id: msg.id || `loaded_${idx}`,
      role: msg.role === 'assistant' ? 'assistant' : 'user',
      content: msg.content,
      // the API sends createdAt in whole seconds
      timestamp: (msg.createdAt ?? msg.created_at) * 1000 || Date.now(),
    }));
  }, [session]);

  const handleSelectConversation = useCallback(async (conversationId: string) => {
    if (!session) return;

    try {
      console.log('[ChatInterface] Loading conversation:', conversationId);
      const loadedMessages = await fetchMessages(conversationId);

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
  }, [session, fetchMessages]);

  // Phone app: Android may close the app while Moly is writing. The engine goes on (it is a foreground service) and saves
  // the reply when it is finished, so the next start shows the last conversation again and, if a message was still unanswered,
  // waits for its reply (see pendingTurn.ts).
  // (An empty id is remembered only when the person started a new chat: the first, empty render must not erase the saved id.)
  useEffect(() => {
    if (phone && (currentConversationId || startedNewChatRef.current)) rememberConversation(currentConversationId);
  }, [phone, currentConversationId]);

  useEffect(() => {
    if (!phone || !session || restoredRef.current) return;
    restoredRef.current = true;
    (async () => {
      const pending = loadPending();
      const id = pending ? pending.conversationId : lastConversation();
      let loaded: ChatMessage[] = [];
      if (id) {
        try {
          loaded = await fetchMessages(id);
          setCurrentConversationId(id);
        } catch (err) {
          console.warn('[ChatInterface] Could not restore the last conversation:', err);
        }
      }
      if (pending && !(id && replyArrived(loaded, pending))) {
        const shown = loaded.some((m) => m.role === 'user' && m.content === pending.text);
        setMessages(shown ? loaded : [...loaded, { id: generateUniqueId(), role: 'user', content: pending.text, timestamp: pending.startedAt }]);
        setLoading(true);
        setWaiting(true);
        return;
      }
      if (pending) clearPending();
      if (loaded.length > 0) setMessages(loaded);
    })();
  }, [phone, session, fetchMessages]);

  useEffect(() => {
    if (!waiting || !session) return;
    const tick = async () => {
      const pending = loadPending();
      if (!pending) {
        // given up (or cleared elsewhere): the engine did not finish the reply
        setWaiting(false);
        setLoading(false);
        setError('Moly did not finish the reply. Send the message again.');
        return;
      }
      try {
        let id = pending.conversationId;
        if (!id) {
          const apiBase = getBackendManager().getBackendUrl();
          const r = await fetch(`${apiBase}/api/conversations`, { headers: { 'Authorization': `Bearer ${session.sessionId}` } });
          if (r.ok) {
            const body = await r.json();
            const list = Array.isArray(body) ? body : body.conversations || [];
            id = pickNewConversation(list, pending) || '';
            if (id) savePending({ ...pending, conversationId: id });
          }
        }
        if (!id) return;
        const loaded = await fetchMessages(id);
        if (replyArrived(loaded, pending)) {
          setMessages(loaded);
          setCurrentConversationId(id);
          clearPending();
          setWaiting(false);
          setLoading(false);
        }
      } catch (err) {
        console.debug('[ChatInterface] Reply not there yet:', err);
      }
    };
    void tick();
    const timer = setInterval(tick, PENDING_POLL_MS);
    return () => clearInterval(timer);
  }, [waiting, session, fetchMessages]);

  // Phone app: text shared to Moly from another app, and the Paste button, fill the writing box. Nothing is read unless the
  // person shares it or taps Paste.
  const addToDraft = useCallback((text: string) => {
    const t = text.trim();
    if (!t) return;
    setInputValue((v) => (v.trim() ? `${v}\n\n${t}` : t));
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    if (!phone) return;
    const take = async () => {
      try {
        addToDraft((await phoneBridge()?.takeSharedText()) || '');
      } catch {
        /* nothing shared */
      }
    };
    void take();
    window.addEventListener('molyShared', take);
    return () => window.removeEventListener('molyShared', take);
  }, [phone, addToDraft]);

  const pasteFromClipboard = async () => {
    try {
      const text = ((await phoneBridge()?.readClipboard()) || '').trim();
      if (text) {
        setError(null);
        addToDraft(text);
      } else setError('The clipboard is empty. Copy the message first.');
    } catch {
      setError('Could not read the clipboard.');
    }
  };

  // A new chat from the middle of another one: no conversation id (the server opens a new conversation for the next message),
  // an empty list and writing box, nothing left over from the old one. The old conversation stays in the history.
  const handleNewChat = useCallback(() => {
    if (loading) return;
    startedNewChatRef.current = true;
    setMessages([]);
    setInputValue('');
    setError(null);
    setPendingClarificationId(null);
    setOrchestratorInsights(null);
    setExpandedEthicalNote(null);
    setCurrentConversationId('');
    setShowHistory(false);
    setShowReflections(false);
    inputRef.current?.focus();
  }, [loading]);

  // The writing box grows with the text (up to about six lines), then scrolls.
  useEffect(() => {
    const box = inputRef.current;
    if (!box) return;
    box.style.height = 'auto';
    box.style.height = `${Math.min(box.scrollHeight, 140)}px`;
  }, [inputValue]);

  // Auto-scroll to bottom when new messages arrive
  // (The list scrolls itself: scrollIntoView would also scroll the page around it.)
  useEffect(() => {
    const list = messagesListRef.current;
    if (list) list.scrollTop = list.scrollHeight;
  }, [messages, loading]);

  const sendMessage = useCallback(async (skip: boolean = false) => {
    if (!session || (!skip && !inputValue.trim())) return;

    const userMessage = skip ? SKIP_MESSAGE : inputValue.trim();
    if (!skip) setInputValue(''); // a draft typed before pressing skip is kept
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
    markChatStarted();
    if (phone) savePending({ conversationId: currentConversationId, text: userMessage, startedAt: Date.now() });
    let handedOver = false; // the reply is being waited for by the poll (phone), not by this request
    // The message is on screen before the request starts, however long the model then takes.
    await new Promise<void>(resolve => requestAnimationFrame(() => resolve()));

    try {
      const apiBase = getBackendManager().getBackendUrl();
      const fullUrl = `${apiBase}/api/message-processor`;
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
        skipQuestions: skip, // the skip button: go ahead with what is known (the server checks it is allowed)
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
        throw new Error(
          response.status === 503 || response.status === 423
            ? 'Moly is still starting, or the AI model is not answering. Try again in a moment.'
            : `Moly could not answer (error ${response.status}). Try again.`,
        );
      }

      const data = await response.json();
      console.log('[ChatInterface] Response received:', {
        facts: data.phase1?.facts?.length || 0,
        clarifications: data.action_required?.clarificationQs?.length || 0,
        response: data.response ? 'present' : 'MISSING',
        conversationId: data.conversationId || 'MISSING',
      });

      // Extract and store conversationId from response (critical for multi-message conversations)
      if (data.conversationId && data.conversationId !== currentConversationId) {
        console.log('[ChatInterface] ✓ Storing conversation ID:', data.conversationId);
        setCurrentConversationId(data.conversationId);
      }

      // NOTE: Clarification questions are now handled entirely through Moly's response messages
      // No separate modals or dialog elements - everything stays in natural conversation flow
      // Backend integrates clarification requests into the response text itself

      // Add Moly's actual response (always present per backend contract)
      if (data.response) {
        const responseText = typeof data.response === 'string' ? data.response : String(data.response);
        console.log('[ChatInterface] ✓ Adding Moly response:', {
          type: typeof data.response,
          isString: typeof data.response === 'string',
          length: responseText.length,
          preview: responseText.substring(0, 80),
        });

        // Extract orchestrator insights from response metadata
        const insights = data.metadata?.orchestratorInsights as OrchestratorInsights || null;
        if (insights) {
          console.log('[ChatInterface] ✓ Orchestrator insights extracted:', {
            hasConfidence: !!insights.extractionConfidence,
            hasMaturity: !!insights.maturityScore,
            gapCount: insights.detectedGaps?.length || 0,
            conflictCount: insights.detectedConflicts?.length || 0,
            isAmbiguous: insights.isAmbiguous,
          });
          setOrchestratorInsights(insights);
        }

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
            canSkip: data.metadata?.canSkip === true,
          } : undefined,
        };

        // The note about an adjusted or refused reply is the disclosure button under the message (its reason is inside);
        // nothing is appended to the text, which would show the internal word of the intervention ("warned").
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
      if (phone && err instanceof TypeError) {
        // The connection to the engine broke (the app was paused or the web view was reset); the engine may still be writing.
        handedOver = true;
        setWaiting(true);
        return;
      }
      const message = err instanceof Error ? err.message : 'Failed to send message';
      setError(message);
      console.error('[ChatInterface] Send message error:', message, { session: session ? 'exists' : 'null' });
    } finally {
      if (!handedOver) {
        if (phone) clearPending();
        setLoading(false);
      }
    }
  }, [inputValue, session, logout, browserSessionId, aboutMe, currentConversationId, phone]);

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage(false);
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
      const response = await fetch(`${apiBase}/api/clarification/respond`, {
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
  if (!session) {
    return <LoginScreen />;
  }

  return (
    <div className="chat-interface">
      {/* Header */}
      <header className="chat-header">
        <div className="chat-brand">
          <span className="brand-mark" aria-hidden="true" />
          <h1 lang="el">Μώλυ</h1>
        </div>
        <div className="header-actions">
          <button
            className="icon-btn"
            onClick={handleNewChat}
            disabled={loading}
            title="New chat"
            aria-label="New chat"
          ><Icon name="plus" /></button>
          <button
            className="icon-btn"
            aria-pressed={showHistory}
            onClick={() => { setShowHistory(!showHistory); setShowReflections(false); }}
            title="Previous conversations"
            aria-label="Previous conversations"
          ><Icon name="history" /></button>
          <button
            className="icon-btn"
            aria-pressed={showReflections}
            onClick={() => { setShowReflections(!showReflections); setShowHistory(false); }}
            title="Pending insights"
            aria-label="Pending insights"
          ><Icon name="insight" /></button>
          <button
            className="icon-btn"
            onClick={handleSettingsClick}
            title="Settings"
            aria-label="Settings"
          ><Icon name="settings" /></button>
          {!isLocalProfile && (
            <button className="icon-btn" onClick={logout} title="Sign out" aria-label="Sign out">
              <Icon name="back" />
            </button>
          )}
        </div>
      </header>

      {/* Tabs Content */}
      {showHistory ? (
        <ConversationHistoryPanel onSelectConversation={handleSelectConversation} />
      ) : showReflections ? (
        <ReflectionsPanel />
      ) : (
        <div className="chat-messages" ref={messagesListRef}>
          {messages.length === 0 ? (
            <div className="chat-empty">
              <span className="brand-mark chat-empty-mark" aria-hidden="true" />
              <h2>What would you like to say?</h2>
              <p>Tell me who it is for and what you want. I will ask what is missing, then help you write it.</p>
            </div>
        ) : (
          messages.map(msg => {
            return (
              <div key={msg.id} className={`chat-message message-${msg.role} ${msg.type ? `message-type-${msg.type}` : ''}`}>
                <div className="message-content">
                  {/* Clarification question */}
                  {/* Clarification questions are now integrated into Moly's response messages */}
                  {/* No separate modals or dialog elements - all in natural conversation */}

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
                              aria-label="Copy to clipboard"
                            >
                              <Icon name="copy" />
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
                        <Icon name={msg.metadata.ethicalIntervention === 'blocked' ? 'block' : 'alert'} />
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
            <div className="message-content">
              <div className="typing-indicator">
                <span></span>
                <span></span>
                <span></span>
              </div>
              {waiting && <p className="chat-waiting">Moly is still writing. You can leave the app, the reply will be here.</p>}
            </div>
          </div>
        )}
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="chat-error" role="alert">
          <Icon name="alert" />
          <p>{error}</p>
          <button onClick={() => setError(null)} aria-label="Dismiss" title="Dismiss"><Icon name="close" /></button>
        </div>
      )}

      {/* Orchestrator Insights */}
      {orchestratorInsights && showDeveloperInsights() && (
        <OrchestratorInsightsPanel insights={orchestratorInsights} loading={loading} />
      )}

      {/* Input */}
      <div className="chat-input-area">
        <div className="chat-composer">
          <textarea
            ref={inputRef}
            className="chat-input"
            placeholder="Write what is on your mind"
            aria-label="Your message"
            value={inputValue}
            onChange={e => setInputValue(e.target.value)}
            onKeyPress={handleKeyPress}
            disabled={loading}
            rows={2}
          />
          <div className="chat-composer-row">
            <div className="composer-left">
            <button
              className="skip-button"
              onClick={() => sendMessage(true)}
              disabled={!canSkip}
              title={canSkip
                ? 'Moly goes ahead with what it knows. You can correct the result afterwards, and the skipped questions stay open.'
                : 'Available once you have answered at least one of Moly\'s questions, while Moly is asking a question you may skip.'}
            >
              Skip questions
            </button>
            {phone && (
              <button className="skip-button paste-button" onClick={pasteFromClipboard} disabled={loading} title="Paste the copied text into the box">
                Paste
              </button>
            )}
            </div>
            <button
              className="send-button"
              onClick={() => sendMessage(false)}
              disabled={!inputValue.trim() || loading}
              title="Send (Enter)"
              aria-label="Send"
            >
              <Icon name={loading ? 'spinner' : 'send'} className={loading ? 'spin' : undefined} />
            </button>
          </div>
        </div>
      </div>

      {/* The Moly / sponsor message has its own area, below the writing box, never between messages */}
      <SponsorCard />

    </div>
  );
};
