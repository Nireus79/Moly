/**
 * V2 API Type Definitions
 * Types for communication with Moly v2 backend agents
 */

/**
 * Conversation Mode - how the agent should respond
 */
export type ConversationMode = 'socratic' | 'direct';

/**
 * Tone - communication tone preference
 */
export type CommunicationTone = 'formal' | 'friendly' | 'dating';

/**
 * V2 Conversation Request
 */
export interface V2ConversationRequest {
  conversationId: string;
  userId: string;
  userMessage: string;
  mode?: ConversationMode;
  tone?: CommunicationTone;
  metadata?: Record<string, unknown>;
}

/**
 * Generated suggestion from agent
 */
export interface V2Suggestion {
  index: number;
  text: string;
  tone: CommunicationTone;
  reasoning: string;
  confidence: number; // 0-1
}

/**
 * Risk warning from risk monitor agent
 */
export interface V2RiskWarning {
  riskLevel: 'immediate' | 'high' | 'medium' | 'low' | 'clear';
  pattern?: string;
  severity: number; // 0-10
  educationalQuestions: string[];
  principles: Array<{
    id: string;
    name: string;
    severity: string;
    description: string;
  }>;
  alternatives: string[];
  recommendation: 'proceed' | 'educate_first' | 'escalate';
  message: string;
}

/**
 * Crisis or illegal content alert
 */
export interface V2SafetyAlert {
  alert_type: 'crisis' | 'illegal' | 'none';
  severity: 'immediate' | 'high' | 'warning';
  title: string;
  message: string;
  indicators: string[];
  resources?: Array<{
    name: string;
    description: string;
    number: string;
    url: string;
    region?: string;
  }>;
  recommendations: string[];
}

/**
 * Reflection - extracted insights from conversation
 */
export interface V2Reflection {
  id: string;
  conversationId: string;
  contactId?: string;
  characteristics: string[];
  interests: string[];
  communicationPreferences: string;
  intentions: string[];
  userQuotes?: string[];
  status: 'pending_approval' | 'approved' | 'rejected';
  createdAt: number;
  approvedAt?: number;
}

/**
 * V2 Conversation Response
 */
export interface V2ConversationResponse {
  phase: string; // 'suggestions_ready', 'context_gathering', 'safety_alert', 'error'
  suggestions: V2Suggestion[];
  questions?: string[];
  reflection?: V2Reflection;
  riskWarning?: V2RiskWarning;
  safetyAlert?: V2SafetyAlert;
  constitutionConcerns?: unknown;
  processingTimeMs: number;
  metadata?: Record<string, unknown>;
  error?: string;
}

/**
 * Conversation feedback from user
 */
export interface V2ConversationFeedback {
  conversationId: string;
  userId: string;
  suggestionChosen: number;
  suggestionText: string;
  userModified: boolean;
  modificationRequest?: string;
  reflectionApproved: boolean;
  reflectionEdits?: Record<string, unknown>;
  timestamp: number;
}

/**
 * Context quality assessment
 */
export interface V2ContextQuality {
  overallScore: number; // 0-1
  hasAboutMe: boolean;
  hasContactProfile: boolean;
  hasHistory: boolean;
  hasBehaviorProfile: boolean;
  hasReflections: boolean;
  historyLength: number;
  reflectionCount: number;
  completenessLevel: 'complete' | 'partial' | 'minimal';
  recommendations?: string[];
}

/**
 * V2 Context Response
 */
export interface V2ContextResponse {
  conversationId: string;
  contextQuality: V2ContextQuality;
  missingContextGaps: string[];
  error?: string;
}

/**
 * Contact profile (user's observations)
 */
export interface V2Contact {
  id: string;
  userId: string;
  name: string;
  relationship: 'close_friend' | 'family' | 'work' | 'romantic' | 'new' | string;
  characteristics: string[];
  interests: string[];
  communicationPreferences: string;
  notes: string;
  createdAt: number;
  updatedAt: number;
}

/**
 * Contacts list response
 */
export interface V2ContactsListResponse {
  userId: string;
  contacts: V2Contact[];
  total: number;
  error?: string;
}

/**
 * AboutMe profile
 */
export interface V2AboutMeRequest {
  userId: string;
  communicationStyle?: string;
  values?: string[];
  preferredTone?: CommunicationTone;
  notes?: string;
}

export interface V2AboutMe {
  userId: string;
  communicationStyle?: string;
  values?: string[];
  preferredTone?: CommunicationTone;
  notes?: string;
  createdAt: number;
  updatedAt: number;
}

/**
 * V2 Health check response
 */
export interface V2HealthResponse {
  status: 'ok' | 'degraded' | 'error';
  timestamp: number;
  version: string;
}

/**
 * V2 API Error response
 */
export interface V2ErrorResponse {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
}

/**
 * Type guard to check if response is an error
 */
export function isV2ErrorResponse(response: unknown): response is V2ErrorResponse {
  return (
    typeof response === 'object' &&
    response !== null &&
    'error' in response &&
    typeof (response as Record<string, unknown>).error === 'string'
  );
}

/**
 * Type guard for safety alert
 */
export function hasV2SafetyAlert(response: V2ConversationResponse): response is V2ConversationResponse & { safetyAlert: V2SafetyAlert } {
  return response.safetyAlert !== undefined && response.safetyAlert !== null;
}

/**
 * Type guard for risk warning
 */
export function hasV2RiskWarning(response: V2ConversationResponse): response is V2ConversationResponse & { riskWarning: V2RiskWarning } {
  return response.riskWarning !== undefined && response.riskWarning !== null;
}
