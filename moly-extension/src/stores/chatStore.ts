import { create } from 'zustand';
import type { ChatMode, CommunicationContext, ChatMessage, DetectedMessage, MessageSuggestion } from '@/types';

interface ChatStore {
  chatMode: ChatMode;
  currentContext: CommunicationContext;
  messages: ChatMessage[];
  suggestions: MessageSuggestion[];
  isLoading: boolean;
  error: string | null;
  detectedMessage: DetectedMessage | null;

  setChatMode: (mode: ChatMode) => void;
  setContext: (context: CommunicationContext) => void;
  addMessage: (message: ChatMessage) => void;
  clearMessages: () => void;
  setSuggestions: (suggestions: MessageSuggestion[]) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setDetectedMessage: (message: DetectedMessage | null) => void;
}

export const useChatStore = create<ChatStore>((set) => ({
  chatMode: 'direct',
  currentContext: 'dating',
  messages: [],
  suggestions: [],
  isLoading: false,
  error: null,
  detectedMessage: null,

  setChatMode: (mode) => set({ chatMode: mode }),
  setContext: (context) => set({ currentContext: context }),
  addMessage: (message) => set((state) => ({
    messages: [...state.messages, message],
  })),
  clearMessages: () => set({ messages: [] }),
  setSuggestions: (suggestions) => set({ suggestions }),
  setLoading: (loading) => set({ isLoading: loading }),
  setError: (error) => set({ error }),
  setDetectedMessage: (message) => set({ detectedMessage: message }),
}));
