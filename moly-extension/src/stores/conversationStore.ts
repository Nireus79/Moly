/**
 * Conversation Store
 * Manages conversation history per contact
 */

import { create } from 'zustand';
import type { ChatMessage } from '@/types';

interface ConversationStore {
  conversations: Record<string, ChatMessage[]>;
  currentContactId: string | null;
  isLoading: boolean;
  error: string | null;

  loadConversation: (contactId: string) => Promise<void>;
  saveMessage: (contactId: string, message: ChatMessage) => Promise<void>;
  saveMessages: (contactId: string, messages: ChatMessage[]) => Promise<void>;
  getConversation: (contactId: string) => ChatMessage[];
  clearConversation: (contactId: string) => Promise<void>;
  setCurrentContact: (contactId: string | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useConversationStore = create<ConversationStore>((set, get) => ({
  conversations: {},
  currentContactId: null,
  isLoading: false,
  error: null,

  loadConversation: async (contactId) => {
    set({ isLoading: true, error: null });
    try {
      const key = `conversation_${contactId}`;
      const result = await chrome.storage.local.get(key);
      const messages = (result[key] || []) as ChatMessage[];

      set((state) => ({
        conversations: {
          ...state.conversations,
          [contactId]: messages,
        },
        isLoading: false,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to load conversation';
      set({ error: message, isLoading: false });
    }
  },

  saveMessage: async (contactId, message) => {
    try {
      const key = `conversation_${contactId}`;
      const conversation = get().conversations[contactId] || [];
      const updatedMessages = [...conversation, message];

      await chrome.storage.local.set({ [key]: updatedMessages });

      set((state) => ({
        conversations: {
          ...state.conversations,
          [contactId]: updatedMessages,
        },
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to save message';
      set({ error: message });
    }
  },

  saveMessages: async (contactId, messages) => {
    try {
      const key = `conversation_${contactId}`;
      await chrome.storage.local.set({ [key]: messages });

      set((state) => ({
        conversations: {
          ...state.conversations,
          [contactId]: messages,
        },
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to save messages';
      set({ error: message });
    }
  },

  getConversation: (contactId) => {
    return get().conversations[contactId] || [];
  },

  clearConversation: async (contactId) => {
    try {
      const key = `conversation_${contactId}`;
      await chrome.storage.local.remove(key);

      set((state) => ({
        conversations: {
          ...state.conversations,
          [contactId]: [],
        },
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to clear conversation';
      set({ error: message });
    }
  },

  setCurrentContact: (contactId) => set({ currentContactId: contactId }),
  setLoading: (loading) => set({ isLoading: loading }),
  setError: (error) => set({ error }),
}));
