export type ChatMode = 'socratic' | 'direct';
export type CommunicationContext = 'formal' | 'friendly' | 'dating';
export type MessageRole = 'user' | 'assistant' | 'system';

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
