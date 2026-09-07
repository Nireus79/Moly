/**
 * Hook for V2 agent integration
 * Handles dual-path routing: V2 agents (new) vs inline logic (fallback)
 */

import { useCallback, useState } from 'react';
import { getBackendManager } from '../api/backendManager';
import { V2ConversationRequest, V2ConversationResponse } from '../types/v2ApiTypes';

export interface UseV2AgentOptions {
  userId?: string;
  conversationId: string;
  mode?: 'socratic' | 'direct';
  tone?: 'formal' | 'friendly' | 'dating';
}

export interface UseV2AgentState {
  loading: boolean;
  error: string | null;
  suggestions: string[];
  phase: string;
  processingTimeMs: number;
  usingV2: boolean;
}

/**
 * Hook to generate suggestions using V2 agents or fallback
 */
export function useV2Agent(options: UseV2AgentOptions) {
  const [state, setState] = useState<UseV2AgentState>({
    loading: false,
    error: null,
    suggestions: [],
    phase: 'idle',
    processingTimeMs: 0,
    usingV2: false,
  });

  /**
   * Generate suggestions with fallback
   */
  const generateSuggestions = useCallback(
    async (userMessage: string): Promise<string[]> => {
      setState(prev => ({ ...prev, loading: true, error: null }));

      try {
        const backendManager = getBackendManager();

        // Initialize V2 if not already done
        if (!backendManager.getV2Client()) {
          await backendManager.initializeV2Agents(options.userId);
        }

        // Try V2 first (if enabled)
        if (backendManager.isV2AgentsEnabled()) {
          try {
            const v2Client = backendManager.getV2Client();
            if (v2Client) {
              const startTime = Date.now();

              const request: V2ConversationRequest = {
                conversationId: options.conversationId,
                userId: options.userId || 'anonymous',
                userMessage,
                mode: options.mode,
                tone: options.tone,
              };

              const response = await v2Client.generateSuggestions(request);
              const processingTime = Date.now() - startTime;

              // Record metrics
              backendManager.recordV2Request(!response.error);

              // Check for errors in response
              if (response.error) {
                console.warn('[useV2Agent] V2 returned error:', response.error);
                throw new Error(response.error);
              }

              // Extract suggestions text
              const suggestions = response.suggestions.map(s => s.text);

              setState(prev => ({
                ...prev,
                loading: false,
                suggestions,
                phase: response.phase,
                processingTimeMs: processingTime,
                usingV2: true,
              }));

              console.info('[useV2Agent] Used V2 agents', {
                suggestionsCount: suggestions.length,
                processingTimeMs: processingTime,
              });

              return suggestions;
            }
          } catch (error) {
            // V2 failed, fall through to inline logic
            console.warn('[useV2Agent] V2 request failed, falling back:', error);
            backendManager.recordV2Request(false);
          }
        }

        // Fallback: Use inline logic (V1)
        console.info('[useV2Agent] Using fallback (inline logic)');

        const startTime = Date.now();
        const suggestions = await generateSuggestionsInline(userMessage, options);
        const processingTime = Date.now() - startTime;

        setState(prev => ({
          ...prev,
          loading: false,
          suggestions,
          phase: 'suggestions_ready',
          processingTimeMs: processingTime,
          usingV2: false,
        }));

        return suggestions;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error';

        setState(prev => ({
          ...prev,
          loading: false,
          error: errorMessage,
          suggestions: [],
          usingV2: false,
        }));

        console.error('[useV2Agent] Failed to generate suggestions:', error);
        throw error;
      }
    },
    [options]
  );

  return {
    ...state,
    generateSuggestions,
  };
}

/**
 * Fallback: Inline suggestion generation (context-aware without LLM)
 */
async function generateSuggestionsInline(
  userMessage: string,
  options: UseV2AgentOptions
): Promise<string[]> {
  console.info('[useV2Agent] Using inline suggestion generation (no LLM)');

  const message = userMessage.toLowerCase();
  const tone = options.tone || 'friendly';

  // Generate suggestions based on detected intention
  if (message.includes('congratul') || message.includes('promot') || message.includes('great')) {
    return [
      `That's amazing news! I'm so happy for you! 🎉`,
      `Congratulations! You deserve this. Tell me everything!`,
      `This is huge! I'd love to hear all about it.`,
    ];
  }

  if (message.includes('sorry') || message.includes('apologize') || message.includes('wrong')) {
    return [
      `I understand. What happened?`,
      `Tell me your side of the story.`,
      `How can I help make this right?`,
    ];
  }

  if (message.includes('help') || message.includes('advice') || message.includes('what should')) {
    return [
      `I'm here to help. Tell me more about the situation.`,
      `Let's think through this together.`,
      `What do you think would work best?`,
    ];
  }

  // Default suggestions
  return [
    `That sounds important. Tell me more.`,
    `I'm listening. What's on your mind?`,
    `How are you feeling about this?`,
  ];
}

/**
 * Hook to get V2 agent diagnostics
 */
export function useV2AgentDiagnostics() {
  const [diagnostics, setDiagnostics] = useState<ReturnType<typeof getBackendManager>['getDiagnostics'] | null>(null);

  const refresh = useCallback(() => {
    const backendManager = getBackendManager();
    setDiagnostics(backendManager.getDiagnostics());
  }, []);

  return { diagnostics, refresh };
}
