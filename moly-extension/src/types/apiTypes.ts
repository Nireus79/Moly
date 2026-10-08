/**
 * API Type Definitions
 * Types for communication with Moly backend agents
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
 * Orchestrator Insights - results from 11-layer orchestrator
 */
export interface OrchestratorInsights {
  // Layer 1: Context Extraction
  extractionConfidence?: number;

  // Layer 2: Principle Checking
  principleViolations?: string[];

  // Layer 3: Maturity Assessment
  maturityScore?: number;
  contextQuality?: string;

  // Layer 4: Gap Detection
  detectedGaps?: Array<{
    type: string;
    severity: string;
    confidence?: number;
  }>;

  // Layer 5: Conflict Detection
  detectedConflicts?: Array<{
    type: string;
    severity: string;
    confidence?: number;
  }>;

  // Layer 6: Ambiguous Request
  isAmbiguous?: boolean;
  ambiguousElements?: string[];

  // Layer 8: Socratic Deepening
  socraticQuestions?: string[];

  // Layer 9: Topic Shift
  topicShifts?: Array<{
    type: string;
    severity: string;
    confidence?: number;
  }>;
}

/**
 * Conversation Request
 */
export interface ConversationRequest {
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
export interface Suggestion {
  index: number;
  text: string;
  tone: CommunicationTone;
  reasoning: string;
  confidence: number; // 0-1
}

/**
 * Risk warning from risk monitor agent
 */
export interface RiskWarning {
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
export interface SafetyAlert {
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
export interface Reflection {
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
 * Conversation Response
 */
export interface ConversationResponse {
  phase: string; // 'suggestions_ready', 'context_gathering', 'safety_alert', 'error'
  suggestions: Suggestion[];
  questions?: string[];
  reflection?: Reflection;
  riskWarning?: RiskWarning;
  safetyAlert?: SafetyAlert;
  constitutionConcerns?: unknown;
  processingTimeMs: number;
  metadata?: Record<string, unknown>;
  error?: string;
}

/**
 * Conversation feedback from user
 */
export interface ConversationFeedback {
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
export interface ContextQuality {
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
 * Context Response
 */
export interface ContextResponse {
  conversationId: string;
  contextQuality: ContextQuality;
  missingContextGaps: string[];
  error?: string;
}

/**
 * Contact profile (user's observations)
 */
export interface Contact {
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
export interface ContactsListResponse {
  userId: string;
  contacts: Contact[];
  total: number;
  error?: string;
}

/**
 * AboutMe profile
 */
export interface AboutMeRequest {
  userId: string;
  communicationStyle?: string;
  values?: string[];
  preferredTone?: CommunicationTone;
  notes?: string;
}

export interface AboutMe {
  userId: string;
  communicationStyle?: string;
  values?: string[];
  preferredTone?: CommunicationTone;
  notes?: string;
  createdAt: number;
  updatedAt: number;
}

/**
 * Health check response
 */
export interface HealthResponse {
  status: 'ok' | 'degraded' | 'error';
  timestamp: number;
  version: string;
}

/**
 * API Error response
 */
export interface ErrorResponse {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
}

/**
 * Type guard to check if response is an error
 */
export function isErrorResponse(response: unknown): response is ErrorResponse {
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
export function hasSafetyAlert(response: ConversationResponse): response is ConversationResponse & { safetyAlert: SafetyAlert } {
  return response.safetyAlert !== undefined && response.safetyAlert !== null;
}

/**
 * Type guard for risk warning
 */
export function hasRiskWarning(response: ConversationResponse): response is ConversationResponse & { riskWarning: RiskWarning } {
  return response.riskWarning !== undefined && response.riskWarning !== null;
}
