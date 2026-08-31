/**
 * Contact List Component
 * Display and manage contacts
 */

import React, { useEffect, useState } from 'react';
import { useContactStore } from '@/stores/contactStore';
import type { Contact } from '@/types';
import './contactList.css';

interface ContactListProps {
  onSelectContact?: (contact: Contact) => void;
  onAddContact?: () => void;
}

export const ContactList: React.FC<ContactListProps> = ({ onSelectContact, onAddContact }) => {
  const { contacts, selectedContact, loadContacts, deleteContact, searchContacts } = useContactStore();
  const [searchQuery, setSearchQuery] = useState('');
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const initContacts = async () => {
      await loadContacts();
      setIsLoading(false);
    };
    initContacts();
  }, [loadContacts]);

  const filteredContacts = searchQuery.trim() ? searchContacts(searchQuery) : contacts;

  const handleSelectContact = (contact: Contact) => {
    onSelectContact?.(contact);
  };

  const handleDeleteContact = (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    if (confirm('Are you sure you want to delete this contact?')) {
      deleteContact(id);
    }
  };

  if (isLoading) {
    return <div className="contact-list loading">Loading contacts...</div>;
  }

  return (
    <div className="contact-list">
      <div className="contact-list-header">
        <h3>Contacts</h3>
        <button className="add-contact-btn" onClick={onAddContact} title="Add new contact">
          +
        </button>
      </div>

      <div className="contact-search">
        <input
          type="text"
          placeholder="Search contacts..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="search-input"
        />
      </div>

      <div className="contact-items">
        {filteredContacts.length === 0 ? (
          <div className="empty-state">
            {contacts.length === 0 ? 'No contacts yet' : 'No matching contacts'}
          </div>
        ) : (
          filteredContacts.map((contact) => (
            <div
              key={contact.id}
              className={`contact-item ${selectedContact?.id === contact.id ? 'selected' : ''}`}
              onClick={() => handleSelectContact(contact)}
            >
              <div className="contact-info">
                <div className="contact-name">{contact.name}</div>
                <div className="contact-platform">
                  {contact.platform && `${contact.platform}`}
                  {contact.lastMessaged && (
                    <span className="contact-time">
                      {' • ' + new Date(contact.lastMessaged).toLocaleDateString()}
                    </span>
                  )}
                </div>
              </div>
              <button
                className="delete-btn"
                onClick={(e) => handleDeleteContact(e, contact.id)}
                title="Delete contact"
              >
                Delete
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default ContactList;
