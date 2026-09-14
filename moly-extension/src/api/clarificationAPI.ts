/**
 * Clarification API - Handle user responses to clarification questions
 * Integrates with backend /api/v2/clarification/respond endpoint
 */

import { getBackendManager } from './backendManager';
import { logger } from '@/utils/logger';

// userId is now stored in authStore.session.userId, not here
// Use getAuthToken() to get the auth header which validates the session

export type ClarificationQuestionType =
  | "user_context"      // Questions about the user
  | "contact_context"   // Questions about a contact
  | "subject"           // Clarifying an ambiguous subject
  | "contact_confirmation"; // Confirming a contact match

export interface ClarificationQuestion {
  id: string;
  type: ClarificationQuestionType;
  question: string;
  options?: string[];
  linkedFacts: string[];
  priority: number;
  status: "pending" | "answered" | "skipped";
  context?: string;
}

export interface TemporaryFact {
  id: string;  // Backend sends 'id', not 'factId'
  type: string;
  value: string;
  subject: string;  // Who the fact is about
  pendingQuestions?: string[];
  timestamp?: number;
  needsUserContext?: boolean;
  needsSubjectMatch?: boolean;
}

export interface ClarificationResponse {
  questionAnswered: boolean;
  factId: string;
  status: "pending" | "saved";
  remainingQuestions: ClarificationQuestion[];
  savedAttribute?: {
    id: number;
    factType: string;
    factValue: string;
    attributedTo: string;
    context: string;
    confidence: number;
    evidence: string;
  };
  createdContact?: {
    id: string;
    name: string;
    relationship: string;
  };
  error?: string;
}

export interface Phase5ProcessResponse {
  success: boolean;
  phase1: {
    facts: Array<{
      id: string;
      type: string;
      value: string;
      confidence: number;
      evidence: string;
    }>;
    shifts: any[];
  };
  phase2: {
    clarifications: ClarificationQuestion[];
    resolved: Record<string, string>;
  };
  phase3: {
    unknown_contacts: any[];
    created_contacts: any[];
  };
  phase4: {
    saved_attributes: any[];
    conflicts: any[];
  };
  action_required: {
    needsClarification: boolean;
    clarificationQs: ClarificationQuestion[];
    temporaryFacts: TemporaryFact[];
    hasConflicts: boolean;
    conflicts: any[];
  };
}

let userId: string | null = null;

const getAuthToken = (): string => {
  try {
    // Read from new auth store (moly_session)
    const sessionStr = localStorage.getItem('moly_session');
    if (sessionStr) {
      const session = JSON.parse(sessionStr);
      if (session.sessionId && session.expiresAt) {
        if (session.expiresAt > Date.now()) {
          return session.sessionId;
        }
      }
    }

    // Fallback to old auth method for backward compatibility
    const token = localStorage.getItem('authToken');
    if (token) {
      return token;
    }
  } catch (err) {
    console.warn('Could not read authToken from localStorage:', err);
  }
  throw new Error('Auth token not available');
};

export class ClarificationAPI {
  /**
   * Process a message through Phase 5 orchestrator
   * Returns questions if clarification needed, or suggestions if not
   */
  static async processMessage(
    message: string,
    conversationId: string,
    aboutMe?: any,
    selectedContactIds?: string[]
  ): Promise<Phase5ProcessResponse> {
    logger.info('ClarificationAPI', 'Processing message for Phase 5', {
      messageLength: message.length,
      conversationId: conversationId || 'unknown',
      hasAboutMe: !!aboutMe,
      contactCount: selectedContactIds?.length || 0,
      messagePreview: message.substring(0, 50)
    });

    const backendUrl = getBackendManager().getBackendUrl();
    const response = await fetch(`${backendUrl}/api/v2/message-processor`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getAuthToken()}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        message,
        conversationId,
        aboutMe,
        selectedContactIds: selectedContactIds || []
      })
    });

    if (!response.ok) {
      const error = await response.text();
      logger.error('ClarificationAPI', 'Process message failed', null, {
        status: response.status,
        statusText: response.statusText,
        error
      });
      throw new Error(`Failed to process message: ${response.statusText}`);
    }

    const result = await response.json();
    logger.info('ClarificationAPI', 'Phase 5 response received', {
      needsClarification: result.action_required?.needsClarification,
      clarificationQsCount: result.action_required?.clarificationQs?.length || 0,
      temporaryFactsCount: result.action_required?.temporaryFacts?.length || 0,
      savedAttributesCount: result.phase4?.saved_attributes?.length || 0,
      conflictsCount: result.action_required?.conflicts?.length || 0
    });

    return result;
  }

  /**
   * Submit answer to a clarification question
   * Returns status: "pending" (more questions) or "saved" (fact saved)
   */
  static async submitAnswer(
    questionId: string,
    factId: string,
    answer: string,
    selectedOption?: string
  ): Promise<ClarificationResponse> {
    logger.info('ClarificationAPI', 'Submitting answer to clarification question', {
      questionId: questionId.substring(0, 8) + '...',
      factId: factId.substring(0, 8) + '...',
      answerType: selectedOption ? 'selected_option' : 'text_input',
      answerPreview: (selectedOption || answer).substring(0, 30)
    });

    const backendUrl = getBackendManager().getBackendUrl();
    const response = await fetch(`${backendUrl}/api/v2/clarification/respond`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getAuthToken()}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        questionId,
        factId,
        selectedOption: selectedOption || undefined,
        userResponse: selectedOption ? undefined : answer
      })
    });

    if (!response.ok) {
      const error = await response.text();
      logger.error('ClarificationAPI', 'Submit answer failed', null, {
        status: response.status,
        statusText: response.statusText,
        error
      });
      throw new Error(`Failed to submit answer: ${response.statusText}`);
    }

    const result = await response.json();
    logger.info('ClarificationAPI', 'Answer response received', {
      status: result.status,
      questionAnswered: result.questionAnswered,
      remainingQuestionsCount: result.remainingQuestions?.length || 0,
      factSaved: !!result.savedAttribute,
      contactCreated: !!result.createdContact
    });

    return result;
  }
}
