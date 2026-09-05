export type ChatMode = 'socratic' | 'direct';
export type CommunicationContext = 'formal' | 'friendly' | 'dating';
export type MessageRole = 'user' | 'assistant' | 'system';
export type MessageType = 'user' | 'moly' | 'incoming' | 'suggestion';

// NEW: Message type for conversation-based architecture
export interface Message {
  id: string;
  type: MessageType;
  content: string;
  timestamp: number;
  metadata?: {
    mode?: ChatMode;
    context?: CommunicationContext;
  };
}

// NEW: Conversation type for managing per-contact conversations
export interface Conversation {
  id: string;
  contactName?: string;
  contactPlatform?: string;
  messages: Message[];
  settings: {
    mode: ChatMode;
    context: CommunicationContext;
    llmProvider: string;
  };
  createdAt: number;
  updatedAt: number;
}

// OLD: Legacy types (kept for backward compatibility)
export interface ChatMessage {
  id: string;
  role: MessageRole;
  content: string;
  timestamp: number;
  tone?: string;
  confidence?: number;
  savedAsVersion?: string;
}

export interface Contact {
  id: string;
  name: string;
  platform: string;
  interests?: string[];
  notes?: string;
  createdAt?: number;
  updatedAt?: number;
  lastMessageAt?: number;
  lastMessaged?: number;
  conversationHistory?: ChatMessage[];
}

export interface DetectedMessage {
  sender: string;
  text: string;
  timestamp: number;
  platform: string;
  url: string;
  profileId?: string;
  context?: CommunicationContext;
}

export interface MessageSuggestion {
  id: string;
  text: string;
  tone: string;
  confidence: number;
  reasoning: string;
  bestFor: string;
}

export interface LLMConfig {
  apiKey: string;
  provider: 'claude' | 'openai' | 'groq';
  model: string;
  temperature: number;
  maxTokens: number;
  isConfigured: boolean;
}

export interface AppState {
  chatMode: ChatMode;
  currentContext: CommunicationContext;
  isLoading: boolean;
  error: string | null;
  messages: ChatMessage[];
  selectedContact: Contact | null;
  suggestions: MessageSuggestion[];
}

// NEW: Conversation Management Types (Phase 1)
export type ConversationType = 'single' | 'group' | 'generic';

export interface ConversationMember {
  id: number;
  name: string;
  relationship: string;
  platform: string;
  notes: string;
}

export interface ConversationData {
  id: number;
  name: string;
  type: ConversationType;
  purpose: string;
  members: ConversationMember[];
  notes: string;
  created_at: number;
  updated_at: number;
}

export interface ConversationContext {
  conversation: {
    id: number;
    name: string;
    type: ConversationType;
    purpose: string;
    notes: string;
  };
  members: ConversationMember[];
  recent_interactions: Array<{
    date: string;
    topic: string;
    sentiment: string;
    summary: string;
  }>;
  context_summary: string;
}
