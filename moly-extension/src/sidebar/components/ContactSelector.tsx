import React, { useState, useEffect } from 'react';

interface Contact {
  id: string;
  name: string;
  platform: string;
  relationship: string;
}

interface ContactSelectorProps {
  onSelectContact: (contact: Contact) => void;
  currentContact: Contact | null;
}

export const ContactSelector: React.FC<ContactSelectorProps> = ({
  onSelectContact,
  currentContact,
}) => {
  console.log('[ContactSelector] RENDERING - currentContact:', currentContact);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [showNewForm, setShowNewForm] = useState(false);
  const [newContactName, setNewContactName] = useState('');
  const [newContactPlatform, setNewContactPlatform] = useState('email');
  const [newContactRelationship, setNewContactRelationship] = useState('friend');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    try {
      loadContacts();
    } catch (err) {
      console.error('[ContactSelector] Mount error:', err);
      setError('Failed to load contacts');
    }
  }, []);

  const loadContacts = async () => {
    try {
      const result = await chrome.storage.local.get('contacts');
      if (result.contacts && Array.isArray(result.contacts)) {
        setContacts(result.contacts);
      }
    } catch (err) {
      console.error('Failed to load contacts:', err);
    }
  };

  const handleAddContact = async () => {
    if (!newContactName.trim()) return;

    const contact: Contact = {
      id: Date.now().toString(),
      name: newContactName,
      platform: newContactPlatform,
      relationship: newContactRelationship,
    };

    const updatedContacts = [...contacts, contact];
    try {
      await chrome.storage.local.set({ contacts: updatedContacts });
      setContacts(updatedContacts);
      onSelectContact(contact);
      setNewContactName('');
      setShowNewForm(false);
    } catch (err) {
      console.error('Failed to add contact:', err);
    }
  };

  const handleDeleteContact = async (id: string) => {
    const updated = contacts.filter(c => c.id !== id);
    try {
      await chrome.storage.local.set({ contacts: updated });
      setContacts(updated);
      if (currentContact?.id === id) {
        if (updated.length > 0) {
          onSelectContact(updated[0]);
        }
      }
    } catch (err) {
      console.error('Failed to delete contact:', err);
    }
  };

  try {
    return (
      <div style={{ marginBottom: '16px', padding: '12px', background: '#f9f9f9', borderRadius: '4px' }}>
        <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', color: '#666', marginBottom: '8px' }}>
          Select Contact
        </label>

      <div style={{ display: 'flex', gap: '8px', marginBottom: '8px', flexWrap: 'wrap' }}>
        <select
          value={currentContact?.id || ''}
          onChange={(e) => {
            const contact = contacts.find(c => c.id === e.target.value);
            if (contact) onSelectContact(contact);
          }}
          style={{
            flex: 1,
            padding: '8px',
            border: '1px solid #ddd',
            borderRadius: '4px',
            fontSize: '14px',
          }}
        >
          <option value="">-- Select a contact --</option>
          {contacts.map(contact => (
            <option key={contact.id} value={contact.id}>
              {contact.name} ({contact.relationship})
            </option>
          ))}
        </select>
        <button
          onClick={() => setShowNewForm(!showNewForm)}
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

      {showNewForm && (
        <div style={{ background: 'white', padding: '12px', borderRadius: '4px', border: '1px solid #ddd' }}>
          <div style={{ marginBottom: '8px' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '4px' }}>
              Name
            </label>
            <input
              type="text"
              value={newContactName}
              onChange={(e) => setNewContactName(e.target.value)}
              placeholder="Contact name..."
              style={{
                width: '100%',
                padding: '6px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                fontSize: '12px',
              }}
            />
          </div>

          <div style={{ marginBottom: '8px', display: 'flex', gap: '8px' }}>
            <div style={{ flex: 1 }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '4px' }}>
                Platform
              </label>
              <select
                value={newContactPlatform}
                onChange={(e) => setNewContactPlatform(e.target.value)}
                style={{ width: '100%', padding: '6px', border: '1px solid #ddd', borderRadius: '4px', fontSize: '12px' }}
              >
                <option value="email">Email</option>
                <option value="whatsapp">WhatsApp</option>
                <option value="slack">Slack</option>
                <option value="instagram">Instagram</option>
                <option value="facebook">Facebook</option>
                <option value="other">Other</option>
              </select>
            </div>

            <div style={{ flex: 1 }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '4px' }}>
                Relationship
              </label>
              <select
                value={newContactRelationship}
                onChange={(e) => setNewContactRelationship(e.target.value)}
                style={{ width: '100%', padding: '6px', border: '1px solid #ddd', borderRadius: '4px', fontSize: '12px' }}
              >
                <option value="friend">Friend</option>
                <option value="colleague">Colleague</option>
                <option value="romantic">Romantic Interest</option>
                <option value="family">Family</option>
                <option value="acquaintance">Acquaintance</option>
              </select>
            </div>
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
                fontSize: '12px',
                fontWeight: '600',
              }}
            >
              Add
            </button>
            <button
              onClick={() => setShowNewForm(false)}
              style={{
                flex: 1,
                padding: '8px',
                background: '#e5e7eb',
                color: '#333',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontSize: '12px',
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {currentContact && (
        <div style={{ marginTop: '8px', padding: '8px', background: 'white', borderRadius: '4px', fontSize: '12px', color: '#666', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <strong>{currentContact.name}</strong>
            <div>{currentContact.platform} • {currentContact.relationship}</div>
          </div>
          <button
            onClick={() => handleDeleteContact(currentContact.id)}
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
            title="Delete this contact"
          >
            Delete
          </button>
        </div>
      )}
      </div>
    );
  } catch (renderError) {
    console.error('[ContactSelector] Render error:', renderError);
    return (
      <div style={{ marginBottom: '16px', padding: '12px', background: '#fff3cd', borderRadius: '4px', border: '1px solid #ffc107' }}>
        <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', color: '#856404', marginBottom: '8px' }}>
          Contact Manager Error
        </label>
        <p style={{ fontSize: '11px', color: '#856404', margin: 0 }}>
          Failed to load contact selector. Please reload the extension.
        </p>
      </div>
    );
  }
};
