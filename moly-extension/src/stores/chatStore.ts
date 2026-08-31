import { create } from 'zustand';
import { Message, Conversation, ChatMode, CommunicationContext } from '@/types';

interface ChatStore {
  conversations: Conversation[];
  currentConversation: Conversation | null;

  startConversation: (contactName?: string, contactPlatform?: string) => void;
  loadConversation: (conversationId: string) => void;
  addMessage: (content: string, type: 'user' | 'moly' | 'incoming' | 'suggestion') => void;
  updateConversationSettings: (settings: Partial<Conversation['settings']>) => void;
  deleteConversation: (conversationId: string) => void;
  deleteMessage: (messageId: string) => void;
  exportConversation: (conversationId: string) => string;
  loadConversationsFromStorage: () => Promise<void>;
  saveConversationsToStorage: () => Promise<void>;
}

export const useChatStore = create<ChatStore>((set, get) => ({
  conversations: [],
  currentConversation: null,

  startConversation: (contactName = 'Unknown', contactPlatform = 'general') => {
    const newConversation: Conversation = {
      id: `conv-${Date.now()}`,
      contactName,
      contactPlatform,
      messages: [],
      settings: {
        mode: 'direct',
        context: 'friendly',
        llmProvider: 'claude',
      },
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };

    set((state) => ({
      conversations: [newConversation, ...state.conversations],
      currentConversation: newConversation,
    }));

    get().saveConversationsToStorage();
  },

  loadConversation: (conversationId: string) => {
    const { conversations } = get();
    const conversation = conversations.find((c) => c.id === conversationId);

    if (conversation) {
      set({ currentConversation: conversation });
    } else {
      console.warn(`Conversation ${conversationId} not found`);
    }
  },

  addMessage: (content: string, type: 'user' | 'moly' | 'incoming' | 'suggestion') => {
    const { currentConversation } = get();

    if (!currentConversation) {
      console.warn('No active conversation');
      return;
    }

    const newMessage: Message = {
      id: `msg-${Date.now()}`,
      type,
      content,
      timestamp: Date.now(),
      metadata: {
        mode: currentConversation.settings.mode,
        context: currentConversation.settings.context,
      },
    };

    const updatedConversation: Conversation = {
      ...currentConversation,
      messages: [...currentConversation.messages, newMessage],
      updatedAt: Date.now(),
    };

    set((state) => ({
      currentConversation: updatedConversation,
      conversations: state.conversations.map((c) =>
        c.id === updatedConversation.id ? updatedConversation : c
      ),
    }));

    get().saveConversationsToStorage();
  },

  updateConversationSettings: (settings: Partial<Conversation['settings']>) => {
    const { currentConversation } = get();

    if (!currentConversation) {
      console.warn('No active conversation');
      return;
    }

    const updatedConversation: Conversation = {
      ...currentConversation,
      settings: {
        ...currentConversation.settings,
        ...settings,
      },
      updatedAt: Date.now(),
    };

    set((state) => ({
      currentConversation: updatedConversation,
      conversations: state.conversations.map((c) =>
        c.id === updatedConversation.id ? updatedConversation : c
      ),
    }));

    get().saveConversationsToStorage();
  },

  deleteConversation: (conversationId: string) => {
    set((state) => ({
      conversations: state.conversations.filter((c) => c.id !== conversationId),
      currentConversation:
        state.currentConversation?.id === conversationId ? null : state.currentConversation,
    }));

    get().saveConversationsToStorage();
  },

  deleteMessage: (messageId: string) => {
    const { currentConversation } = get();

    if (!currentConversation) {
      console.warn('No active conversation');
      return;
    }

    const updatedConversation: Conversation = {
      ...currentConversation,
      messages: currentConversation.messages.filter((m) => m.id !== messageId),
      updatedAt: Date.now(),
    };

    set((state) => ({
      currentConversation: updatedConversation,
      conversations: state.conversations.map((c) =>
        c.id === updatedConversation.id ? updatedConversation : c
      ),
    }));

    get().saveConversationsToStorage();
  },

  exportConversation: (conversationId: string): string => {
    const { conversations } = get();
    const conversation = conversations.find((c) => c.id === conversationId);

    if (!conversation) {
      throw new Error(`Conversation ${conversationId} not found`);
    }

    return JSON.stringify(
      {
        id: conversation.id,
        contact: conversation.contactName,
        platform: conversation.contactPlatform,
        messages: conversation.messages,
        settings: conversation.settings,
        exportedAt: new Date().toISOString(),
      },
      null,
      2
    );
  },

  loadConversationsFromStorage: async () => {
    try {
      const result = await chrome.storage.local.get('conversations');
      if (result.conversations && Array.isArray(result.conversations)) {
        set({
          conversations: result.conversations,
          currentConversation: result.conversations.length > 0 ? result.conversations[0] : null,
        });
      }
    } catch (err) {
      console.error('Failed to load conversations from storage:', err);
    }
  },

  saveConversationsToStorage: async () => {
    try {
      const { conversations } = get();
      await chrome.storage.local.set({ conversations });
    } catch (err) {
      console.error('Failed to save conversations to storage:', err);
    }
  },
}));
