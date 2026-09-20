import React, { useState, useEffect } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { getBackendManager } from '@/api/backendManager';
import './conversation-history.css';

interface ConversationPreview {
  id: string;
  name: string;
  description?: string;
  createdAt: number;
  updatedAt: number;
  recentMessages?: Array<{
    role: string;
    content: string;
    timestamp: number;
  }>;
}

interface ConversationHistoryPanelProps {
  onSelectConversation: (conversationId: string) => void;
}

export const ConversationHistoryPanel: React.FC<ConversationHistoryPanelProps> = ({
  onSelectConversation,
}) => {
  const { session } = useAuth();
  const [conversations, setConversations] = useState<ConversationPreview[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchConversations = async () => {
      if (!session) return;

      try {
        setLoading(true);
        const apiBase = getBackendManager().getBackendUrl();
        const response = await fetch(`${apiBase}/api/v2/conversations`, {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${session.sessionId}`,
          },
        });

        if (response.status === 404) {
          console.log('[ConversationHistory] No conversations (404 is normal when empty)');
          setConversations([]);
          setLoading(false);
          return;
        }

        if (!response.ok) {
          if (response.status === 401) {
            setError('Session expired');
            setLoading(false);
            return;
          }
          throw new Error(`Failed to fetch conversations: ${response.statusText}`);
        }

        const data = await response.json();
        const conversationList = data.conversations || [];
        setConversations(conversationList);
        console.log('[ConversationHistory] Loaded', conversationList.length, 'conversations');
      } catch (err) {
        let message = 'Failed to load conversations';
        if (err instanceof Error) {
          message = err.message;
        } else if (typeof err === 'string') {
          message = err;
        }
        setError(String(message));
        console.error('[ConversationHistory] Fetch error:', message);
      } finally {
        setLoading(false);
      }
    };

    fetchConversations();
  }, [session]);

  const formatDate = (timestamp: number) => {
    const date = new Date(timestamp * 1000);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } else if (diffDays === 1) {
      return 'Yesterday';
    } else if (diffDays < 7) {
      return `${diffDays} days ago`;
    } else {
      return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
    }
  };

  const handleResumeConversation = (conversationId: string) => {
    console.log('[ConversationHistory] Resuming conversation:', conversationId);
    onSelectConversation(conversationId);
  };

  if (loading) {
    return (
      <div className="conversation-history-panel">
        <div className="history-loading">
          <div className="spinner">⏳</div>
          <p>Loading your conversations...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="conversation-history-panel">
      <div className="history-header">
        <h3>Previous Conversations</h3>
        <p className="history-subtitle">Click to resume a conversation</p>
      </div>

      {error && (
        <div className="history-error">
          <span>⚠️</span>
          <p>{error}</p>
        </div>
      )}

      {conversations.length === 0 ? (
        <div className="history-empty">
          <div className="empty-icon">📝</div>
          <p>No previous conversations yet</p>
          <p className="empty-subtitle">Start a new conversation to get started</p>
        </div>
      ) : (
        <div className="history-list">
          {conversations.map(conversation => (
            <button
              key={conversation.id}
              className="conversation-card"
              onClick={() => handleResumeConversation(conversation.id)}
              title={`Resume: ${conversation.name || conversation.description || 'Untitled'}`}
            >
              <div className="conversation-header">
                <h4 className="conversation-name">
                  {conversation.name || conversation.description || 'Untitled'}
                </h4>
                <span className="conversation-date">
                  {formatDate(conversation.updatedAt)}
                </span>
              </div>

              {conversation.recentMessages && conversation.recentMessages.length > 0 && (
                <div className="conversation-preview">
                  {conversation.recentMessages.map((msg, idx) => (
                    <div key={idx} className={`preview-message preview-${msg.role}`}>
                      <span className="preview-role">
                        {msg.role === 'user' ? 'You' : 'Moly'}:
                      </span>
                      <span className="preview-text">{msg.content}</span>
                    </div>
                  ))}
                </div>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};
