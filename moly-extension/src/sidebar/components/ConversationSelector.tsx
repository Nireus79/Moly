import React, { useState, useEffect } from 'react';
import type { ConversationData, ConversationMember } from '@/types';

interface ConversationSelectorProps {
  onSelectConversation: (conversation: ConversationData) => void;
  onNewConversation: () => void;
  currentConversation: ConversationData | null;
}

export const ConversationSelector: React.FC<ConversationSelectorProps> = ({
  onSelectConversation,
  onNewConversation,
  currentConversation,
}) => {
  const [conversations, setConversations] = useState<ConversationData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadConversations();
  }, []);

  const loadConversations = async () => {
    try {
      setLoading(true);
      const result = await chrome.storage.local.get('conversations');
      if (result.conversations && Array.isArray(result.conversations)) {
        setConversations(result.conversations);
      }
      setError(null);
    } catch (err) {
      console.error('[ConversationSelector] Failed to load conversations:', err);
      setError('Failed to load conversations');
    } finally {
      setLoading(false);
    }
  };

  const handleSelectConversation = (conversation: ConversationData) => {
    onSelectConversation(conversation);
  };

  const handleDeleteConversation = async (id: number) => {
    const updated = conversations.filter(c => c.id !== id);
    try {
      await chrome.storage.local.set({ conversations: updated });
      setConversations(updated);
      if (currentConversation?.id === id && updated.length > 0) {
        onSelectConversation(updated[0]);
      }
    } catch (err) {
      console.error('[ConversationSelector] Failed to delete conversation:', err);
      setError('Failed to delete conversation');
    }
  };

  const getConversationLabel = (conv: ConversationData): string => {
    if (conv.type === 'single' && conv.members.length > 0) {
      return `${conv.name}`;
    }
    if (conv.type === 'group') {
      return `${conv.name} (${conv.members.length} members)`;
    }
    return `${conv.name} (${conv.type})`;
  };

  const getMembersDisplay = (members: ConversationMember[]): string => {
    if (members.length === 0) return 'No members';
    if (members.length === 1) return members[0].name;
    return `${members[0].name} + ${members.length - 1} more`;
  };

  if (loading) {
    return (
      <div style={{ marginBottom: '16px', padding: '12px', background: '#f9f9f9', borderRadius: '4px' }}>
        <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', color: '#666', marginBottom: '8px' }}>
          Conversations
        </label>
        <div style={{ fontSize: '12px', color: '#999', textAlign: 'center', padding: '8px' }}>
          Loading conversations...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ marginBottom: '16px', padding: '12px', background: '#fff3cd', borderRadius: '4px', border: '1px solid #ffc107' }}>
        <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', color: '#856404', marginBottom: '8px' }}>
          Conversation Manager Error
        </label>
        <p style={{ fontSize: '11px', color: '#856404', margin: 0 }}>{error}</p>
        <button
          onClick={loadConversations}
          style={{
            marginTop: '8px',
            padding: '4px 8px',
            background: '#856404',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
            fontSize: '11px',
            fontWeight: '600',
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div style={{ marginBottom: '16px', padding: '12px', background: '#f9f9f9', borderRadius: '4px' }}>
      <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', color: '#666', marginBottom: '8px' }}>
        Conversations
      </label>

      <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
        <select
          value={currentConversation?.id || ''}
          onChange={(e) => {
            const conv = conversations.find(c => c.id === Number(e.target.value));
            if (conv) handleSelectConversation(conv);
          }}
          style={{
            flex: 1,
            padding: '8px',
            border: '1px solid #ddd',
            borderRadius: '4px',
            fontSize: '14px',
          }}
        >
          <option value="">-- Select a conversation --</option>
          {conversations.map(conv => (
            <option key={conv.id} value={conv.id}>
              {getConversationLabel(conv)}
            </option>
          ))}
        </select>
        <button
          onClick={onNewConversation}
          style={{
            padding: '8px 12px',
            background: '#6366f1',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
            fontSize: '12px',
            fontWeight: '600',
          }}
        >
          + New
        </button>
      </div>

      {currentConversation && (
        <div style={{ background: 'white', padding: '12px', borderRadius: '4px', border: '1px solid #ddd' }}>
          <div style={{ marginBottom: '8px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', marginBottom: '8px' }}>
              <div>
                <strong style={{ display: 'block', fontSize: '14px', marginBottom: '4px' }}>
                  {currentConversation.name}
                </strong>
                <div style={{ fontSize: '12px', color: '#666' }}>
                  Type: <span style={{ fontWeight: '600' }}>{currentConversation.type}</span>
                </div>
                {currentConversation.purpose && (
                  <div style={{ fontSize: '12px', color: '#666' }}>
                    Purpose: <span style={{ fontWeight: '600' }}>{currentConversation.purpose}</span>
                  </div>
                )}
              </div>
              <button
                onClick={() => handleDeleteConversation(currentConversation.id)}
                style={{
                  padding: '4px 8px',
                  background: '#ef4444',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontSize: '12px',
                  fontWeight: '600',
                }}
                title="Delete this conversation"
              >
                Delete
              </button>
            </div>

            {currentConversation.members.length > 0 && (
              <div style={{ marginTop: '8px', paddingTop: '8px', borderTop: '1px solid #eee' }}>
                <div style={{ fontSize: '12px', fontWeight: '600', marginBottom: '6px', color: '#666' }}>
                  Members ({currentConversation.members.length})
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  {currentConversation.members.map(member => (
                    <div
                      key={member.id}
                      style={{
                        padding: '6px',
                        background: '#f3f4f6',
                        borderRadius: '4px',
                        fontSize: '12px',
                      }}
                    >
                      <div style={{ fontWeight: '600' }}>{member.name}</div>
                      <div style={{ color: '#666', fontSize: '11px' }}>
                        {member.relationship} • {member.platform}
                      </div>
                      {member.notes && (
                        <div style={{ color: '#999', fontSize: '11px', marginTop: '2px' }}>
                          {member.notes}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {currentConversation.notes && (
              <div style={{ marginTop: '8px', paddingTop: '8px', borderTop: '1px solid #eee', fontSize: '12px', color: '#666' }}>
                <strong>Notes:</strong> {currentConversation.notes}
              </div>
            )}
          </div>
        </div>
      )}

      {conversations.length === 0 && (
        <div style={{ textAlign: 'center', padding: '16px', color: '#999', fontSize: '12px' }}>
          No conversations yet. Create your first one!
        </div>
      )}
    </div>
  );
};
