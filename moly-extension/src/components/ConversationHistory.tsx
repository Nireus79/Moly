/**
 * Conversation History Component
 * Display and manage conversation history for a contact
 */

import React, { useEffect } from 'react';
import { useConversationStore } from '@/stores/conversationStore';
import { useContactStore } from '@/stores/contactStore';
import type { ChatMessage } from '@/types';
import './conversationHistory.css';

interface ConversationHistoryProps {
  contactId?: string;
  messages?: ChatMessage[];
}

export const ConversationHistory: React.FC<ConversationHistoryProps> = ({ contactId, messages }) => {
  const { getConversation, clearConversation, loadConversation } = useConversationStore();
  const { selectedContact } = useContactStore();
  const currentId = contactId || selectedContact?.id;

  useEffect(() => {
    if (currentId) {
      loadConversation(currentId);
    }
  }, [currentId, loadConversation]);

  const displayMessages = messages || (currentId ? getConversation(currentId) : []);

  const handleClearHistory = async () => {
    if (!currentId) return;
    if (confirm('Clear conversation history with this contact?')) {
      await clearConversation(currentId);
    }
  };

  if (displayMessages.length === 0) {
    return (
      <div className="conversation-history empty">
        <p>No conversation history yet</p>
      </div>
    );
  }

  return (
    <div className="conversation-history">
      <div className="history-header">
        <h4>📜 Conversation History</h4>
        <button
          className="clear-history-btn"
          onClick={handleClearHistory}
          title="Clear history"
        >
          🗑️
        </button>
      </div>

      <div className="history-messages">
        {displayMessages.map((msg) => (
          <div key={msg.id} className={`history-message ${msg.role}`}>
            <div className="message-time">
              {new Date(msg.timestamp).toLocaleTimeString([], {
                hour: '2-digit',
                minute: '2-digit',
              })}
            </div>
            <div className="message-role">{msg.role === 'user' ? '👤' : '🤖'}</div>
            <div className="message-content">{msg.content}</div>
            {msg.tone && <span className="message-tone">{msg.tone}</span>}
          </div>
        ))}
      </div>

      <div className="history-stats">
        <span>{displayMessages.length} message{displayMessages.length !== 1 ? 's' : ''}</span>
        {displayMessages.length > 0 && (
          <span className="last-updated">
            Last: {new Date(displayMessages[displayMessages.length - 1].timestamp).toLocaleDateString()}
          </span>
        )}
      </div>
    </div>
  );
};

export default ConversationHistory;
