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
  private timeout: number = 2000; // 2 second timeout
  private requestCount: number = 0;
  private errorCount: number = 0;

  constructor(backendUrl: string) {
    this.backendUrl = backendUrl.replace(/\/$/, ''); // Remove trailing slash
  }

  /**
   * Generate conversation suggestions using V2 agents
   */
  async generateSuggestions(req: V2ConversationRequest): Promise<V2ConversationResponse> {
    const startTime = Date.now();
    this.requestCount++;

    try {
      const response = await this.fetchWithTimeout(
        `${this.backendUrl}/api/v2/conversation/generate`,
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
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json() as V2ConversationResponse;

      // Log metrics
      const elapsed = Date.now() - startTime;
      console.log(`[V2] generateSuggestions: ${elapsed}ms`, {
        suggestionsCount: data.suggestions?.length,
        hasError: !!data.error,
      });

      return data;
    } catch (error) {
      this.errorCount++;
      const elapsed = Date.now() - startTime;

      console.error(`[V2] generateSuggestions failed (${elapsed}ms):`, error);

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
        `${this.backendUrl}/api/v2/health`,
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
