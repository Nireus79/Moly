/**
 * Enhanced Popup Component
 * Quick access to recent contacts, conversations, and notifications
 */

import React, { useEffect, useState } from 'react';
import { useContactStore } from '@/stores/contactStore';
import { useChatStore } from '@/stores/chatStore';
import { useSettingsStore } from '@/stores/settingsStore';
import type { Contact } from '@/types';
import './popupEnhanced.css';

export const PopupEnhanced: React.FC = () => {
  const { contacts, loadContacts } = useContactStore();
  const { messages } = useChatStore();
  const { settings, loadSettings } = useSettingsStore();
  const [tab, setTab] = useState<'recent' | 'status'>('recent');
  const [recentContacts, setRecentContacts] = useState<Contact[]>([]);

  useEffect(() => {
    const init = async () => {
      await loadContacts();
      await loadSettings();
    };
    init();
  }, [loadContacts, loadSettings]);

  // Get 5 most recently contacted
  useEffect(() => {
    const sorted = [...contacts]
      .sort((a, b) => (b.lastMessaged || 0) - (a.lastMessaged || 0))
      .slice(0, 5);
    setRecentContacts(sorted);
  }, [contacts]);

  const handleOpenChat = async () => {
    try {
      const sidebarUrl = chrome.runtime.getURL('sidebar/sidebar.html');
      await chrome.tabs.create({ url: sidebarUrl });
      window.close();
    } catch (error) {
      console.error('Failed to open chat:', error);
      alert('Could not open Moly chat. Please try again.');
    }
  };

  const handleOpenSettings = async () => {
    try {
      await chrome.runtime.openOptionsPage?.();
      window.close();
    } catch (error) {
      console.error('Failed to open settings:', error);
      alert('Could not open settings. Please try again.');
    }
  };

  const handleContactClick = async (contact: Contact) => {
    try {
      await chrome.storage.local.set({ selectedContactId: contact.id });
      await chrome.runtime.sendMessage({ type: 'OPEN_SIDEPANEL' });
      window.close();
    } catch (error) {
      console.error('Failed to open chat:', error);
      alert('Could not open Moly chat. Please try again.');
    }
  };

  const isConfigured = settings?.providers[settings?.activeProvider]?.enabled;
  const messageCount = messages.length;
  const contactCount = contacts.length;

  return (
    <div className="popup-enhanced">
      {/* Header */}
      <div className="popup-header">
        <div className="header-title">
          <span className="title-icon">M</span>
          <div>
            <h1>Moly</h1>
            <p className="subtitle">Messaging Coach</p>
          </div>
        </div>
        <div className={`status-indicator ${isConfigured ? 'active' : 'inactive'}`} title={isConfigured ? 'Configured' : 'Not configured'} />
      </div>

      {/* Tab Navigation */}
      <div className="tab-navigation">
        <button
          className={`tab-btn ${tab === 'recent' ? 'active' : ''}`}
          onClick={() => setTab('recent')}
        >
          Recent
        </button>
        <button
          className={`tab-btn ${tab === 'status' ? 'active' : ''}`}
          onClick={() => setTab('status')}
        >
          Status
        </button>
      </div>

      {/* Tab Content */}
      <div className="tab-content">
        {tab === 'recent' && (
          <div className="recent-tab">
            {recentContacts.length > 0 ? (
              <div className="contacts-list">
                <h3 className="list-header">Recent Contacts</h3>
                {recentContacts.map((contact) => (
                  <div
                    key={contact.id}
                    className="contact-item"
                    onClick={() => handleContactClick(contact)}
                  >
                    <div className="contact-info">
                      <div className="contact-name">{contact.name}</div>
                      {contact.platform && (
                        <div className="contact-platform">{contact.platform}</div>
                      )}
                    </div>
                    <span className="contact-arrow">→</span>
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty-state">
                <p>No contacts yet</p>
                <p className="empty-hint">Start a conversation to see recent contacts</p>
              </div>
            )}
          </div>
        )}

        {tab === 'status' && (
          <div className="status-tab">
            <div className="status-section">
              <h3 className="status-title">Configuration</h3>
              <div className={`status-item ${isConfigured ? 'success' : 'warning'}`}>
                <span className="status-label">LLM Provider</span>
                <span className="status-value">
                  {isConfigured ? `${settings?.activeProvider || 'Unknown'}` : 'Not configured'}
                </span>
              </div>
            </div>

            <div className="status-section">
              <h3 className="status-title">Activity</h3>
              <div className="status-item">
                <span className="status-label">Contacts</span>
                <span className="status-value">{contactCount}</span>
              </div>
              <div className="status-item">
                <span className="status-label">Conversations</span>
                <span className="status-value">{messageCount}</span>
              </div>
            </div>

            <div className="status-section">
              <h3 className="status-title">Mode</h3>
              <div className="status-item">
                <span className="status-label">Chat Mode</span>
                <span className="status-value capitalize">
                  {settings?.chatMode === 'socratic' ? 'Socratic' : 'Direct'}
                </span>
              </div>
              <div className="status-item">
                <span className="status-label">Context</span>
                <span className="status-value capitalize">
                  {settings?.defaultContext === 'formal' && 'Formal'}
                  {settings?.defaultContext === 'friendly' && 'Friendly'}
                  {settings?.defaultContext === 'dating' && 'Dating'}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Action Buttons */}
      <div className="popup-footer">
        <button className="action-btn primary" onClick={handleOpenChat}>
          Open Chat
        </button>
        <button className="action-btn secondary" onClick={handleOpenSettings}>
          Settings
        </button>
      </div>
    </div>
  );
};

export default PopupEnhanced;
