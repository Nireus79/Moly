import React, { useState, useEffect } from 'react';
import type { ConversationData, ConversationMember, ConversationType } from '@/types';

interface Contact {
  id: string;
  name: string;
  platform: string;
  relationship: string;
  group?: string;
}

interface NewConversationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (conversation: ConversationData) => void;
}

const CONVERSATION_PURPOSES = [
  'relationship',
  'advice',
  'cover_letter',
  'networking',
  'customer_service',
  'sales',
  'other',
];

const CONVERSATION_PURPOSE_LABELS: Record<string, string> = {
  relationship: 'Romantic/Dating',
  advice: 'Seeking Advice',
  cover_letter: 'Cover Letter/Resume',
  networking: 'Networking',
  customer_service: 'Customer Service',
  sales: 'Sales/Business',
  other: 'Other',
};

export const NewConversationModal: React.FC<NewConversationModalProps> = ({
  isOpen,
  onClose,
  onSave,
}) => {
  const [name, setName] = useState('');
  const [type, setType] = useState<ConversationType>('single');
  const [purpose, setPurpose] = useState('relationship');
  const [notes, setNotes] = useState('');
  const [selectedContactIds, setSelectedContactIds] = useState<string[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      loadContacts();
    }
  }, [isOpen]);

  const loadContacts = async () => {
    try {
      setLoading(true);
      const result = await chrome.storage.local.get('contacts');
      console.log('[NewConversationModal] Loaded from storage:', result);
      if (result.contacts && Array.isArray(result.contacts)) {
        console.log('[NewConversationModal] Found', result.contacts.length, 'contacts');
        setContacts(result.contacts);
      } else {
        console.log('[NewConversationModal] No contacts in storage');
        setContacts([]);
      }
      setError(null);
    } catch (err) {
      console.error('[NewConversationModal] Failed to load contacts:', err);
      setError('Failed to load contacts');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleContact = (contactId: string) => {
    setSelectedContactIds(prev => {
      const updated = prev.includes(contactId)
        ? prev.filter(id => id !== contactId)
        : [...prev, contactId];
      console.log('[NewConversationModal] Selected contacts:', updated);
      return updated;
    });
  };

  const handleSave = async () => {
    console.log('[NewConversationModal] handleSave called');
    console.log('[NewConversationModal] name:', name);
    console.log('[NewConversationModal] selectedContactIds:', selectedContactIds);
    console.log('[NewConversationModal] selectedContactIds.length:', selectedContactIds.length);

    if (!name.trim()) {
      setError('Conversation name is required');
      console.log('[NewConversationModal] Validation failed: name required');
      return;
    }

    if (selectedContactIds.length === 0) {
      setError('Please select at least one contact');
      console.log('[NewConversationModal] Validation failed: no contacts selected');
      return;
    }

    const selectedMembers: ConversationMember[] = selectedContactIds
      .map(contactId => {
        const contact = contacts.find(c => c.id === contactId);
        if (!contact) return null;
        return {
          id: parseInt(contactId),
          name: contact.name,
          relationship: contact.relationship,
          platform: contact.platform,
          notes: '',
        };
      })
      .filter((m): m is ConversationMember => m !== null);

    const newConversation: ConversationData = {
      id: Date.now(),
      name: name.trim(),
      type,
      purpose,
      members: selectedMembers,
      notes: notes.trim(),
      created_at: Date.now(),
      updated_at: Date.now(),
    };

    try {
      const result = await chrome.storage.local.get('conversations');
      const conversations = result.conversations || [];
      conversations.push(newConversation);
      await chrome.storage.local.set({ conversations });

      onSave(newConversation);
      resetForm();
      onClose();
    } catch (err) {
      console.error('[NewConversationModal] Failed to save conversation:', err);
      setError('Failed to save conversation');
    }
  };

  const resetForm = () => {
    setName('');
    setType('single');
    setPurpose('relationship');
    setNotes('');
    setSelectedContactIds([]);
    setError(null);
  };

  const groupedContacts = (() => {
    const groups: Record<string, Contact[]> = {};
    contacts.forEach(contact => {
      const group = contact.group || 'general';
      if (!groups[group]) groups[group] = [];
      groups[group].push(contact);
    });
    return groups;
  })();

  if (!isOpen) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: 'white',
          borderRadius: '8px',
          padding: '24px',
          maxWidth: '500px',
          width: '90%',
          maxHeight: '90vh',
          overflow: 'auto',
          boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h2 style={{ marginTop: 0, marginBottom: '16px', fontSize: '18px' }}>
          New Conversation
        </h2>

        {error && (
          <div
            style={{
              padding: '12px',
              background: '#fee2e2',
              border: '1px solid #fca5a5',
              borderRadius: '4px',
              fontSize: '13px',
              color: '#991b1b',
              marginBottom: '16px',
            }}
          >
            {error}
          </div>
        )}

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
            Conversation Name
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., Sarah - Dating"
            style={{
              width: '100%',
              padding: '8px',
              border: '1px solid #ddd',
              borderRadius: '4px',
              fontSize: '14px',
              boxSizing: 'border-box',
            }}
          />
        </div>

        <div style={{ marginBottom: '16px', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
          <div>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
              Type
            </label>
            <select
              value={type}
              onChange={(e) => setType(e.target.value as ConversationType)}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                fontSize: '14px',
              }}
            >
              <option value="single">Single Contact</option>
              <option value="group">Group</option>
              <option value="generic">Generic/Purpose</option>
            </select>
          </div>

          <div>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
              Purpose
            </label>
            <select
              value={purpose}
              onChange={(e) => setPurpose(e.target.value)}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                fontSize: '14px',
              }}
            >
              {CONVERSATION_PURPOSES.map(p => (
                <option key={p} value={p}>
                  {CONVERSATION_PURPOSE_LABELS[p]}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '8px' }}>
            Select Contacts
          </label>
          {loading ? (
            <div style={{ fontSize: '12px', color: '#999', padding: '8px' }}>
              Loading contacts...
            </div>
          ) : contacts.length === 0 ? (
            <div style={{ fontSize: '12px', color: '#999', padding: '8px' }}>
              No contacts found. Please add contacts first.
            </div>
          ) : (
            <div style={{ border: '1px solid #ddd', borderRadius: '4px', maxHeight: '250px', overflow: 'auto' }}>
              {Object.entries(groupedContacts).map(([group, groupContacts]) => (
                <div key={group}>
                  <div style={{
                    padding: '8px 12px',
                    background: '#f3f4f6',
                    fontSize: '12px',
                    fontWeight: '600',
                    color: '#666',
                    borderBottom: '1px solid #ddd',
                    textTransform: 'capitalize',
                  }}>
                    {group}
                  </div>
                  {groupContacts.map(contact => (
                    <div
                      key={contact.id}
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        handleToggleContact(contact.id);
                      }}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        padding: '8px 12px',
                        borderBottom: '1px solid #eee',
                        fontSize: '13px',
                        cursor: 'pointer',
                        userSelect: 'none',
                        background: selectedContactIds.includes(contact.id) ? '#e0e7ff' : 'transparent',
                      }}
                    >
                      <input
                        type="checkbox"
                        checked={selectedContactIds.includes(contact.id)}
                        onChange={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          handleToggleContact(contact.id);
                        }}
                        onClick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                        }}
                        style={{ marginRight: '8px', cursor: 'pointer' }}
                      />
                      <span style={{ fontWeight: '500' }}>{contact.name}</span>
                      <span style={{ color: '#999', marginLeft: '4px', fontSize: '12px' }}>
                        ({contact.relationship} • {contact.platform})
                      </span>
                    </div>
                  ))}
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
            Notes (Optional)
          </label>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Any notes about this conversation..."
            style={{
              width: '100%',
              padding: '8px',
              border: '1px solid #ddd',
              borderRadius: '4px',
              fontSize: '13px',
              fontFamily: 'inherit',
              boxSizing: 'border-box',
              minHeight: '60px',
            }}
          />
        </div>

        <div style={{ display: 'flex', gap: '8px' }}>
          <button
            onClick={handleSave}
            style={{
              flex: 1,
              padding: '10px',
              background: '#10b981',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: '600',
            }}
          >
            Create
          </button>
          <button
            onClick={() => {
              resetForm();
              onClose();
            }}
            style={{
              flex: 1,
              padding: '10px',
              background: '#e5e7eb',
              color: '#333',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: '600',
            }}
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
};
