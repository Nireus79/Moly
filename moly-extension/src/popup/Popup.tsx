import React, { useEffect, useState } from 'react';
import type { Contact } from '@/types';
import './popup.css';

export const Popup: React.FC = () => {
  const [recentContacts, setRecentContacts] = useState<Contact[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadRecentContacts();
  }, []);

  const loadRecentContacts = async () => {
    try {
      const result = await chrome.storage.local.get('contacts');
      const contacts: Contact[] = result.contacts || [];
      setRecentContacts(contacts.slice(-5).reverse());
    } catch (error) {
      console.error('Error loading contacts:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const openSidebar = async () => {
    const tab = await chrome.tabs.query({ active: true, currentWindow: true });
    if (tab[0]?.id) {
      // Open sidebar using sidePanel API (Manifest V3)
      (chrome.sidePanel as any).open?.({ tabId: tab[0].id });
    }
  };

  const openSettings = () => {
    chrome.runtime.openOptionsPage();
  };

  return (
    <div className="popup-container">
      <div className="popup-header">
        <h1>🧠 Moly</h1>
        <p className="subtitle">Messaging Coach</p>
      </div>

      <div className="popup-content">
        <button className="btn btn-primary" onClick={openSidebar}>
          💬 Open Chat
        </button>

        {!isLoading && (
          <>
            <div className="recent-section">
              <h3>Recent Contacts</h3>
              {recentContacts.length > 0 ? (
                <ul className="contact-list">
                  {recentContacts.map((contact) => (
                    <li key={contact.id} className="contact-item">
                      <span className="contact-name">{contact.name}</span>
                      <span className="contact-platform">{contact.platform}</span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="empty-state">No contacts yet</p>
              )}
            </div>
          </>
        )}

        <div className="popup-footer">
          <button className="btn btn-secondary" onClick={openSettings}>
            ⚙️ Settings
          </button>
        </div>
      </div>
    </div>
  );
};
