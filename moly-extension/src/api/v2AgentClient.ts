/**
 * V2 Agent Client - Connection to Moly v2 backend agents
 * Handles API calls to /api/v2/ endpoints
 */

export interface V2ConversationRequest {
  conversationId: string;
  userId: string;
  userMessage: string;
  mode?: 'socratic' | 'direct';
  tone?: 'formal' | 'friendly' | 'dating';
  metadata?: Record<string, unknown>;
}

export interface V2Suggestion {
  index: number;
  text: string;
  tone: string;
  reasoning: string;
  confidence: number;
}

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
  }>;
  recommendations: string[];
}

export interface V2ConversationResponse {
  phase: string;
  suggestions: V2Suggestion[];
  questions?: string[];
  reflection?: unknown;
  riskWarning?: unknown;
  safetyAlert?: V2SafetyAlert;
  processingTimeMs: number;
  error?: string;
}

export interface V2ContextResponse {
  conversationId: string;
  contextQuality: {
    overallScore: number;
    completenessLevel: 'complete' | 'partial' | 'minimal';
    recommendations?: string[];
  };
  missingContextGaps: string[];
  error?: string;
}

export interface V2AboutMeRequest {
  userId: string;
  communicationStyle?: string;
  values?: string[];
  preferredTone?: string;
  notes?: string;
}

export class V2AgentClient {
  private backendUrl: string;
  private timeout: number = 190000; // 190 second timeout (Phase5 default is 180s + buffer)
  private requestCount: number = 0;
  private errorCount: number = 0;

  constructor(backendUrl: string) {
    this.backendUrl = backendUrl.replace(/\/$/, ''); // Remove trailing slash
  }

  /**
   * Generate conversation suggestions using V2 Phase5 orchestrator
   * Routes through /api/v2/phase5/process endpoint
   */
  async generateSuggestions(req: V2ConversationRequest): Promise<V2ConversationResponse> {
    const startTime = Date.now();
    this.requestCount++;

    console.log('[V2AgentClient] generateSuggestions: Starting Phase5 request', {
      conversationId: req.conversationId,
      userId: req.userId?.substring(0, 8) + '...',
      messageLength: req.userMessage?.length,
      mode: req.mode,
      tone: req.tone,
      requestNumber: this.requestCount,
    });

    try {
      // Get auth token from localStorage
      let authToken = '';
      try {
        authToken = localStorage.getItem('authToken') || '';
      } catch {
        console.warn('[V2AgentClient] Could not read authToken from localStorage');
      }

      if (!authToken) {
        throw new Error('No auth token available - user not logged in');
      }

      // Use message-processor endpoint (Phase5 orchestrator)
      const endpoint = `${this.backendUrl}/api/v2/message-processor`;
      console.log('[V2AgentClient] generateSuggestions: Calling Phase5 at', endpoint);

      // Transform request to Phase5 format
      const phase5Request = {
        message: req.userMessage,
        conversationId: req.conversationId,
      };

      const response = await this.fetchWithTimeout(
        endpoint,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${authToken}`,
          },
          body: JSON.stringify(phase5Request),
        },
        this.timeout
      );

      console.log('[V2AgentClient] generateSuggestions: Response received', {
        status: response.status,
        statusText: response.statusText,
        ok: response.ok,
        contentType: response.headers.get('content-type'),
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('[V2AgentClient] generateSuggestions: HTTP error response', {
          status: response.status,
          statusText: response.statusText,
          body: errorText,
        });
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const responseText = await response.text();
      console.warn('[V2AgentClient] generateSuggestions: FULL RAW RESPONSE', responseText);
      console.log('[V2AgentClient] generateSuggestions: Raw response text', {
        length: responseText.length,
        preview: responseText.substring(0, 200),
      });

      let phase5Response: any;
      try {
        phase5Response = JSON.parse(responseText);
      } catch (parseErr) {
        console.error('[V2AgentClient] generateSuggestions: JSON parse error', {
          error: parseErr,
          responseText: responseText.substring(0, 500),
        });
        throw parseErr;
      }

      // Transform Phase5 response to V2ConversationResponse format
      const data: V2ConversationResponse = {
        phase: phase5Response.action_required?.needsClarification ? 'clarification_needed' : 'suggestions_ready',
        suggestions: [], // Will populate below
        questions: phase5Response.action_required?.clarificationQs?.map((q: any) => q.question) || [],
        processingTimeMs: Date.now() - startTime,
        error: phase5Response.error,
      };

      // If clarifications needed, don't generate suggestions yet
      if (phase5Response.action_required?.needsClarification && phase5Response.action_required?.clarificationQs?.length > 0) {
        console.log('[V2AgentClient] Phase5 returned clarification questions, skipping suggestions', {
          questionsCount: data.questions.length,
        });
        // Return questions for the UI to handle
        data.phase = 'questions_only';
        data.suggestions = [];
      } else {
        // No clarification needed - generate generic suggestions as fallback
        // (Phase5 doesn't return pre-generated suggestions, just facts)
        data.suggestions = this.generateGenericSuggestions(req.userMessage);
        data.phase = 'suggestions_ready';
      }

      // Validate response structure
      console.log('[V2AgentClient] generateSuggestions: Response transformed', {
        phase: data.phase,
        hasSuggestions: !!data.suggestions,
        suggestionsLength: data.suggestions?.length || 0,
        questionsLength: data.questions?.length || 0,
        hasError: !!data.error,
        error: data.error,
      });

      if (!Array.isArray(data.suggestions)) {
        console.error('[V2AgentClient] generateSuggestions: suggestions is not an array!', {
          type: typeof data.suggestions,
          value: data.suggestions,
        });
        data.suggestions = [];
      }

      // Log metrics
      const elapsed = Date.now() - startTime;
      console.log(`[V2AgentClient] generateSuggestions: SUCCESS (${elapsed}ms)`, {
        suggestionsCount: data.suggestions.length,
        phase: data.phase,
        processingTimeMs: data.processingTimeMs,
        totalTimeMs: elapsed,
      });

      return data;
    } catch (error) {
      this.errorCount++;
      const elapsed = Date.now() - startTime;

      console.error(`[V2AgentClient] generateSuggestions FAILED (${elapsed}ms):`, {
        error,
        errorMessage: error instanceof Error ? error.message : String(error),
        errorStack: error instanceof Error ? error.stack : 'no stack',
        requestCount: this.requestCount,
        errorCount: this.errorCount,
      });

      return {
        phase: 'error',
        suggestions: [],
        processingTimeMs: elapsed,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  /**
   * Get conversation context
   */
  async getContext(conversationId: string, userId: string): Promise<V2ContextResponse> {
    const startTime = Date.now();

    try {
      const url = new URL(`${this.backendUrl}/api/v2/context`);
      url.searchParams.append('conversationId', conversationId);
      url.searchParams.append('userId', userId);

      const response = await this.fetchWithTimeout(url.toString(), {}, this.timeout);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json() as V2ContextResponse;
      const elapsed = Date.now() - startTime;

      console.log(`[V2] getContext: ${elapsed}ms`);
      return data;
    } catch (error) {
      const elapsed = Date.now() - startTime;
      console.error(`[V2] getContext failed:`, error);

      return {
        conversationId,
        contextQuality: {
          overallScore: 0,
          completenessLevel: 'minimal',
        },
        missingContextGaps: [],
        error: error instanceof Error ? error.message : 'Failed to get context',
      };
    }
  }

  /**
   * Save/update About Me profile
   */
  async setAboutMe(req: V2AboutMeRequest): Promise<{ success: boolean; error?: string }> {
    try {
      const response = await this.fetchWithTimeout(
        `${this.backendUrl}/api/v2/about-me`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(req),
        },
        this.timeout
      );

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return { success: true };
    } catch (error) {
      console.error(`[V2] setAboutMe failed:`, error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to save profile',
      };
    }
  }

  /**
   * Health check for V2 backend
   */
  async healthCheck(): Promise<boolean> {
    try {
      const response = await this.fetchWithTimeout(
        `${this.backendUrl}/api/status`,
        {},
        this.timeout
      );

      return response.ok;
    } catch {
      return false;
    }
  }

  /**
   * Get metrics for monitoring
   */
  getMetrics() {
    return {
      totalRequests: this.requestCount,
      errorCount: this.errorCount,
      errorRate: this.requestCount > 0 ? this.errorCount / this.requestCount : 0,
    };
  }

  /**
   * Generate generic conversation starters when no specific context is available
   */
  private generateGenericSuggestions(userMessage: string): V2Suggestion[] {
    const messageLength = userMessage.trim().length;
    const suggestions: string[] = [];

    // Analyze message length to provide appropriate response
    if (messageLength < 20) {
      // Short messages - provide engagement starters
      suggestions.push(
        "That sounds important. Tell me more.",
        "I'm listening. What's on your mind?",
        "How are you feeling about this?"
      );
    } else if (messageLength < 100) {
      // Medium messages - acknowledge and explore
      suggestions.push(
        "Can you elaborate on that?",
        "What happened next?",
        "How did that make you feel?"
      );
    } else {
      // Long messages - reflect and summarize
      suggestions.push(
        "So what I'm hearing is... is that right?",
        "That's a lot to consider. What matters most to you?",
        "How would you like to move forward?"
      );
    }

    // Convert to V2Suggestion format
    return suggestions.map((text, index) => ({
      index,
      text,
      tone: 'conversational',
      reasoning: 'Generic conversation starter',
      confidence: 0.7,
    }));
  }

  /**
   * Fetch with timeout
   */
  private async fetchWithTimeout(
    url: string,
    options: RequestInit,
    timeoutMs: number
  ): Promise<Response> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
      });

      return response;
    } finally {
      clearTimeout(timeoutId);
    }
  }
}

/**
 * Create V2 agent client instance
 */
export function createV2AgentClient(backendUrl: string): V2AgentClient {
  return new V2AgentClient(backendUrl);
}
