import React, { useState, useEffect } from 'react';

interface Contact {
  id: string;
  name: string;
  platform: string;
  relationship: string;
  group?: string;
  notes?: string;
}

interface ContactManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ContactManager: React.FC<ContactManagerProps> = ({ isOpen, onClose }) => {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [newName, setNewName] = useState('');
  const [newPlatform, setNewPlatform] = useState('text');
  const [newRelationship, setNewRelationship] = useState('friend');
  const [newGroup, setNewGroup] = useState('general');
  const [newNotes, setNewNotes] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editNotes, setEditNotes] = useState('');
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
      if (result.contacts && Array.isArray(result.contacts)) {
        setContacts(result.contacts);
      }
      setError(null);
    } catch (err) {
      console.error('[ContactManager] Failed to load contacts:', err);
      setError('Failed to load contacts');
    } finally {
      setLoading(false);
    }
  };

  const handleAddContact = async () => {
    if (!newName.trim()) {
      setError('Name is required');
      return;
    }

    const contact: Contact = {
      id: Date.now().toString(),
      name: newName.trim(),
      platform: newPlatform,
      relationship: newRelationship,
      group: newGroup || 'general',
      notes: newNotes.trim(),
    };

    try {
      const updated = [...contacts, contact];
      await chrome.storage.local.set({ contacts: updated });
      setContacts(updated);
      setNewName('');
      setNewPlatform('text');
      setNewRelationship('friend');
      setNewGroup('general');
      setNewNotes('');
      setShowForm(false);
      setError(null);
    } catch (err) {
      console.error('[ContactManager] Failed to add contact:', err);
      setError('Failed to add contact');
    }
  };

  const handleDeleteContact = async (id: string) => {
    try {
      const updated = contacts.filter(c => c.id !== id);
      await chrome.storage.local.set({ contacts: updated });
      setContacts(updated);
      setError(null);
    } catch (err) {
      console.error('[ContactManager] Failed to delete contact:', err);
      setError('Failed to delete contact');
    }
  };

  const handleUpdateNotes = async (id: string) => {
    try {
      const updated = contacts.map(c =>
        c.id === id ? { ...c, notes: editNotes.trim() } : c
      );
      await chrome.storage.local.set({ contacts: updated });
      setContacts(updated);
      setEditingId(null);
      setEditNotes('');
      setError(null);
    } catch (err) {
      console.error('[ContactManager] Failed to update notes:', err);
      setError('Failed to update notes');
    }
  };

  if (!isOpen) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.7)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 10001,
        pointerEvents: 'auto',
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
          Contacts
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
            <button
              onClick={() => setError(null)}
              style={{
                marginLeft: '8px',
                background: 'none',
                border: 'none',
                color: '#991b1b',
                cursor: 'pointer',
                fontWeight: 'bold',
              }}
            >
              ✕
            </button>
          </div>
        )}

        {loading ? (
          <div style={{ fontSize: '14px', color: '#999', textAlign: 'center', padding: '20px' }}>
            Loading contacts...
          </div>
        ) : contacts.length === 0 ? (
          <div style={{ fontSize: '14px', color: '#999', textAlign: 'center', padding: '20px' }}>
            No contacts yet
          </div>
        ) : (
          <div style={{ marginBottom: '16px', maxHeight: '300px', overflow: 'auto', border: '1px solid #ddd', borderRadius: '4px' }}>
            {contacts.map(contact => (
              <div
                key={contact.id}
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  padding: '12px',
                  borderBottom: '1px solid #eee',
                  fontSize: '13px',
                }}
              >
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: '600' }}>{contact.name}</div>
                  <div style={{ color: '#666', fontSize: '12px' }}>
                    {contact.relationship} • {contact.platform}
                  </div>
                  {contact.group && (
                    <div style={{ color: '#999', fontSize: '11px', marginTop: '2px' }}>
                      {contact.group}
                    </div>
                  )}
                  {contact.notes && (
                    <div style={{ color: '#999', fontSize: '11px', marginTop: '4px', fontStyle: 'italic' }}>
                      "{contact.notes}"
                    </div>
                  )}
                </div>
                <div style={{ display: 'flex', gap: '4px' }}>
                  <button
                    onClick={() => {
                      setEditingId(contact.id);
                      setEditNotes(contact.notes || '');
                    }}
                    style={{
                      padding: '4px 8px',
                      background: '#6366f1',
                      color: 'white',
                      border: 'none',
                      borderRadius: '4px',
                      cursor: 'pointer',
                      fontSize: '12px',
                      fontWeight: '600',
                    }}
                  >
                    Notes
                  </button>
                  <button
                    onClick={() => handleDeleteContact(contact.id)}
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
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {editingId && (
          <div style={{ marginBottom: '16px', background: '#f0f9ff', padding: '12px', borderRadius: '4px', border: '1px solid #bfdbfe' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '8px' }}>
              Edit Notes
            </label>
            <textarea
              value={editNotes}
              onChange={(e) => setEditNotes(e.target.value)}
              placeholder="Add notes about this contact..."
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                fontSize: '13px',
                fontFamily: 'inherit',
                boxSizing: 'border-box',
                minHeight: '60px',
                marginBottom: '8px',
                resize: 'vertical',
              }}
            />
            <div style={{ display: 'flex', gap: '8px' }}>
              <button
                onClick={() => handleUpdateNotes(editingId)}
                style={{
                  flex: 1,
                  padding: '6px',
                  background: '#10b981',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontSize: '12px',
                  fontWeight: '600',
                }}
              >
                Save Notes
              </button>
              <button
                onClick={() => {
                  setEditingId(null);
                  setEditNotes('');
                }}
                style={{
                  flex: 1,
                  padding: '6px',
                  background: '#e5e7eb',
                  color: '#333',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontSize: '12px',
                  fontWeight: '600',
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {!showForm ? (
          <button
            onClick={() => setShowForm(true)}
            style={{
              width: '100%',
              padding: '10px',
              background: '#6366f1',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: '600',
              marginBottom: '8px',
            }}
          >
            + Add Contact
          </button>
        ) : (
          <div style={{ background: '#f9f9f9', padding: '16px', borderRadius: '4px', marginBottom: '8px', border: '1px solid #ddd' }}>
            <div style={{ marginBottom: '12px' }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '4px' }}>
                Name
              </label>
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="Contact name..."
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '13px',
                  boxSizing: 'border-box',
                }}
              />
            </div>

            <div style={{ marginBottom: '12px', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
              <div>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '4px' }}>
                  Platform
                </label>
                <select
                  value={newPlatform}
                  onChange={(e) => setNewPlatform(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '8px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '13px',
                  }}
                >
                  <option value="text">Text</option>
                  <option value="email">Email</option>
                  <option value="whatsapp">WhatsApp</option>
                  <option value="slack">Slack</option>
                  <option value="instagram">Instagram</option>
                  <option value="facebook">Facebook</option>
                  <option value="dating_app">Dating App</option>
                  <option value="other">Other</option>
                </select>
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '4px' }}>
                  Relationship
                </label>
                <select
                  value={newRelationship}
                  onChange={(e) => setNewRelationship(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '8px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '13px',
                  }}
                >
                  <option value="friend">Friend</option>
                  <option value="colleague">Colleague</option>
                  <option value="romantic">Romantic</option>
                  <option value="family">Family</option>
                  <option value="acquaintance">Acquaintance</option>
                </select>
              </div>
            </div>

            <div style={{ marginBottom: '12px' }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '4px' }}>
                Group (optional)
              </label>
              <select
                value={newGroup}
                onChange={(e) => setNewGroup(e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '13px',
                }}
              >
                <option value="general">General</option>
                <option value="friends">Friends</option>
                <option value="work">Work</option>
                <option value="dating">Dating</option>
                <option value="family">Family</option>
                <option value="other">Other</option>
              </select>
            </div>

            <div style={{ marginBottom: '12px' }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '4px' }}>
                Notes (optional)
              </label>
              <textarea
                value={newNotes}
                onChange={(e) => setNewNotes(e.target.value)}
                placeholder="e.g., Likes hiking, prefers calls over texts..."
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '13px',
                  fontFamily: 'inherit',
                  boxSizing: 'border-box',
                  minHeight: '60px',
                  resize: 'vertical',
                }}
              />
            </div>

            <div style={{ display: 'flex', gap: '8px' }}>
              <button
                onClick={handleAddContact}
                style={{
                  flex: 1,
                  padding: '8px',
                  background: '#10b981',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontSize: '13px',
                  fontWeight: '600',
                }}
              >
                Save
              </button>
              <button
                onClick={() => {
                  setShowForm(false);
                  setNewName('');
                  setNewPlatform('text');
                  setNewRelationship('friend');
                }}
                style={{
                  flex: 1,
                  padding: '8px',
                  background: '#e5e7eb',
                  color: '#333',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontSize: '13px',
                  fontWeight: '600',
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        <button
          onClick={onClose}
          style={{
            width: '100%',
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
          Close
        </button>
      </div>
    </div>
  );
};
