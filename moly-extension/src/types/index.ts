export type ChatMode = 'socratic' | 'direct';
export type CommunicationContext = 'formal' | 'friendly' | 'dating';
export type MessageRole = 'user' | 'assistant' | 'system';
export type MessageType = 'user' | 'moly' | 'incoming' | 'suggestion';

// Context Binding Types - matches both frontend modal and backend schema
export interface AboutMeProfile {
  id?: number;
  communicationStyle?: string;
  coreValues?: string[];  // Maps to backend "values" field
  tonePreference?: string;  // Maps to backend "preferred_tone" field
  preferences?: {
    notes?: string;  // Maps to backend "notes" field
  };
  // Legacy fields (for backward compatibility)
  values?: string[];
  goals?: string[];
  patterns?: string[];
}

// Conversation Management Types
export type ConversationType = 'single' | 'group' | 'generic';

export interface ConversationMember {
  id?: number;
  name: string;
  relationship?: string;
  platform?: string;
  notes?: string;
}

// ConversationSettings - matches backend
export interface ConversationSettings {
  mode?: 'socratic' | 'direct';
  context?: 'formal' | 'friendly' | 'dating';
  llmProvider?: string;
}

// UNIFIED ConversationData - matches backend API response
export interface ConversationData {
  id: string;
  backendId?: string;  // Backend conversation ID (format: conv_<ts>_<ns>)
  userId?: string;
  name: string;
  type?: ConversationType;
  purpose?: string;
  description?: string;
  members?: ConversationMember[];
  settings?: ConversationSettings;
  notes?: string;
  createdAt?: number;
  created_at?: number;
  updatedAt?: number;
  updated_at?: number;
}

export interface ConversationContext {
  conversation: {
    id: string;
    name: string;
    type?: ConversationType;
    purpose?: string;
    notes?: string;
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

// Message type for conversation-based architecture
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

// Conversation type for managing per-contact conversations
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

// Legacy types (kept for backward compatibility)
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
  platform?: string;
  relationship?: string;
  notes?: string;
  interests?: string[];
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
