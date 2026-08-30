/**
 * Contact Store
 * Manages user contacts and conversation history
 */

import { create } from 'zustand';
import type { Contact } from '@/types';

interface ContactStore {
  contacts: Contact[];
  selectedContact: Contact | null;
  isLoading: boolean;
  error: string | null;

  loadContacts: () => Promise<void>;
  addContact: (contact: Omit<Contact, 'id'>) => Promise<void>;
  updateContact: (id: string, updates: Partial<Contact>) => Promise<void>;
  deleteContact: (id: string) => Promise<void>;
  selectContact: (id: string | null) => void;
  searchContacts: (query: string) => Contact[];
  getContact: (id: string) => Contact | undefined;
  clearContacts: () => Promise<void>;
}

export const useContactStore = create<ContactStore>((set, get) => ({
  contacts: [],
  selectedContact: null,
  isLoading: false,
  error: null,

  loadContacts: async () => {
    set({ isLoading: true, error: null });
    try {
      const result = await chrome.storage.local.get('contacts');
      const contacts = (result.contacts || []) as Contact[];
      set({ contacts, isLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to load contacts';
      set({ error: message, isLoading: false });
    }
  },

  addContact: async (contact) => {
    try {
      const newContact: Contact = {
        ...contact,
        id: `contact_${Date.now()}`,
      };

      const contacts = [...get().contacts, newContact];
      await chrome.storage.local.set({ contacts });
      set({ contacts, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to add contact';
      set({ error: message });
    }
  },

  updateContact: async (id, updates) => {
    try {
      const contacts = get().contacts.map((c) =>
        c.id === id ? { ...c, ...updates } : c,
      );
      await chrome.storage.local.set({ contacts });

      const selectedContact = get().selectedContact;
      if (selectedContact?.id === id) {
        set({ selectedContact: { ...selectedContact, ...updates } });
      }

      set({ contacts, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update contact';
      set({ error: message });
    }
  },

  deleteContact: async (id) => {
    try {
      const contacts = get().contacts.filter((c) => c.id !== id);
      await chrome.storage.local.set({ contacts });

      let selectedContact = get().selectedContact;
      if (selectedContact?.id === id) {
        selectedContact = null;
      }

      set({ contacts, selectedContact, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to delete contact';
      set({ error: message });
    }
  },

  selectContact: (id) => {
    if (id === null) {
      set({ selectedContact: null });
      return;
    }

    const contact = get().contacts.find((c) => c.id === id);
    set({ selectedContact: contact || null });
  },

  searchContacts: (query) => {
    if (!query.trim()) {
      return get().contacts;
    }

    const lowerQuery = query.toLowerCase();
    return get().contacts.filter((c) =>
      c.name.toLowerCase().includes(lowerQuery) ||
      c.platform?.toLowerCase().includes(lowerQuery),
    );
  },

  getContact: (id) => {
    return get().contacts.find((c) => c.id === id);
  },

  clearContacts: async () => {
    try {
      await chrome.storage.local.remove('contacts');
      set({ contacts: [], selectedContact: null, error: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to clear contacts';
      set({ error: message });
    }
  },
}));

/**
 * Initialize contact store on app start
 */
export const initializeContacts = async () => {
  await useContactStore.getState().loadContacts();
};
