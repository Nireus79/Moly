import React from 'react';

export interface Message {
  id: string;
  type: 'user' | 'moly' | 'incoming' | 'suggestion';
  content: string;
  timestamp: number;
  metadata?: {
    mode?: 'socratic' | 'direct';
    context?: 'formal' | 'friendly' | 'dating';
  };
}

interface ChatHistoryProps {
  messages: Message[];
  onDeleteMessage?: (id: string) => void;
  onExport?: () => void;
}

export const ChatHistory: React.FC<ChatHistoryProps> = ({
  messages,
  onDeleteMessage,
  onExport,
}) => {
  const formatTime = (timestamp: number) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const getMessageClass = (type: string) => {
    switch (type) {
      case 'user':
        return 'message-user';
      case 'moly':
        return 'message-moly';
      case 'incoming':
        return 'message-incoming';
      case 'suggestion':
        return 'message-suggestion';
      default:
        return '';
    }
  };

  const getMessageLabel = (type: string) => {
    switch (type) {
      case 'user':
        return 'You';
      case 'moly':
        return 'Moly';
      case 'incoming':
        return 'Incoming';
      case 'suggestion':
        return 'Suggestion';
      default:
        return '';
    }
  };

  if (messages.length === 0) {
    return (
      <div className="chat-history empty">
        <div className="empty-state">
          <p>Start a conversation with Moly</p>
          <p className="hint">Tell Moly about the person you're responding to</p>
        </div>
      </div>
    );
  }

  return (
    <div className="chat-history">
      {messages.map((message) => (
        <div key={message.id} className={`message-item ${getMessageClass(message.type)}`}>
          <div className="message-header">
            <span className="message-label">{getMessageLabel(message.type)}</span>
            <span className="message-time">{formatTime(message.timestamp)}</span>
            {onDeleteMessage && (
              <button
                className="message-delete"
                onClick={() => onDeleteMessage(message.id)}
                title="Delete message"
              >
                ✕
              </button>
            )}
          </div>
          <div className="message-content">{message.content}</div>
          {message.metadata?.mode && (
            <div className="message-metadata">
              <span className="badge">{message.metadata.mode}</span>
              {message.metadata.context && (
                <span className="badge">{message.metadata.context}</span>
              )}
            </div>
          )}
        </div>
      ))}
      {onExport && (
        <div className="chat-actions">
          <button className="export-btn" onClick={onExport}>
            Export Conversation
          </button>
        </div>
      )}
    </div>
  );
};

export default ChatHistory;
