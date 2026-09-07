/**
 * Enhanced Moly Agent Hook with V2 Support
 * Wraps existing useMolyAgent with V2 agent integration and fallback
 */

import { useCallback, useEffect, useState } from 'react';
import { useMolyAgent } from './useMolyAgent';
import { useV2Agent } from './useV2Agent';
import { getBackendManager } from '../api/backendManager';

export interface EnhancedMolyAgentState {
  // Existing state
  analyzing: boolean;
  error: string | null;
  suggestions: string[];
  safety: any;
  constitution: any;
  questions: string[];

  // V2-specific state
  usingV2: boolean;
  v2Fallback: boolean;
  processingTimeMs: number;
}

export interface MolyAgentOptions {
  userId?: string;
  conversationId: string;
  mode?: 'socratic' | 'direct';
  tone?: 'formal' | 'friendly' | 'dating';
}

/**
 * Enhanced hook combining V1 (existing) and V2 (new agents) paths
 */
export function useMolyAgentV2(options: MolyAgentOptions) {
  const [usingV2, setUsingV2] = useState(false);
  const [v2Fallback, setV2Fallback] = useState(false);
  const [processingTimeMs, setProcessingTimeMs] = useState(0);

  // V1 (existing) agent
  const {
    analyze: v1Analyze,
    safety: v1Safety,
    constitution: v1Constitution,
    questions: v1Questions,
    loading: v1Loading,
    clear: v1Clear,
  } = useMolyAgent();

  // V2 (new agents) hook
  const { generateSuggestions: v2Generate, ...v2State } = useV2Agent({
    ...options,
  });

  // Initialize V2 agents on mount
  useEffect(() => {
    const initV2 = async () => {
      try {
        const backendManager = getBackendManager();
        await backendManager.initializeV2Agents(options.userId);
      } catch (error) {
        console.warn('[useMolyAgentV2] Failed to initialize V2:', error);
      }
    };

    initV2();
  }, [options.userId]);

  /**
   * Enhanced analyze function that tries V2 first, falls back to V1
   */
  const analyze = useCallback(
    async (userMessage: string, contactName: string, contextString: string) => {
      const startTime = Date.now();
      const backendManager = getBackendManager();

      // Try V2 path first (if enabled)
      if (backendManager.isV2AgentsEnabled()) {
        try {
          console.info('[useMolyAgentV2] Trying V2 agent path');

          // TODO: Call V2 safety checker and analysis
          // For now, skip V2 analysis and just use suggestions

          setUsingV2(true);
          setV2Fallback(false);
          return;
        } catch (error) {
          console.warn('[useMolyAgentV2] V2 analysis failed, falling back to V1:', error);
          setV2Fallback(true);
          backendManager.recordV2Request(false);
        }
      }

      // Fallback to V1 (existing logic)
      console.info('[useMolyAgentV2] Using V1 fallback path');
      setUsingV2(false);

      try {
        const result = await v1Analyze(userMessage, contactName, contextString);
        setProcessingTimeMs(Date.now() - startTime);
        return result;
      } catch (error) {
        setProcessingTimeMs(Date.now() - startTime);
        throw error;
      }
    },
    [v1Analyze, options.userId]
  );

  /**
   * Enhanced suggestion generation with V2 + fallback
   */
  const generateSuggestions = useCallback(
    async (userMessage: string): Promise<string[]> => {
      const startTime = Date.now();
      const backendManager = getBackendManager();

      // Try V2 path first (if enabled)
      if (backendManager.isV2AgentsEnabled()) {
        try {
          console.info('[useMolyAgentV2] Generating suggestions with V2 agents');

          const suggestions = await v2Generate(userMessage);

          setUsingV2(true);
          setV2Fallback(false);
          setProcessingTimeMs(Date.now() - startTime);

          return suggestions;
        } catch (error) {
          console.warn('[useMolyAgentV2] V2 generation failed, falling back:', error);
          setV2Fallback(true);
          backendManager.recordV2Request(false);
        }
      }

      // Fallback to V1 (existing suggestion generation)
      console.info('[useMolyAgentV2] Using V1 fallback for suggestions');
      setUsingV2(false);

      try {
        const result = await v1Analyze(userMessage, '', '');
        setProcessingTimeMs(Date.now() - startTime);
        return result.suggestions || [];
      } catch (error) {
        console.error('[useMolyAgentV2] V1 fallback also failed:', error);
        setProcessingTimeMs(Date.now() - startTime);
        return [];
      }
    },
    [v2Generate, v1Analyze, options.userId]
  );

  /**
   * Get current state
   */
  const getState = useCallback((): EnhancedMolyAgentState => {
    return {
      analyzing: v1Loading || v2State.loading,
      error: v1Loading ? null : v2State.error,
      suggestions: v2State.suggestions,
      safety: v1Safety,
      constitution: v1Constitution,
      questions: v1Questions || v2State.questions || [],
      usingV2,
      v2Fallback,
      processingTimeMs,
    };
  }, [v1Loading, v2State, v1Safety, v1Constitution, v1Questions, usingV2, v2Fallback, processingTimeMs]);

  /**
   * Clear all analysis
   */
  const clear = useCallback(() => {
    v1Clear();
    setUsingV2(false);
    setV2Fallback(false);
    setProcessingTimeMs(0);
  }, [v1Clear]);

  return {
    analyze,
    generateSuggestions,
    getState,
    clear,
    ...getState(),
  };
}

/**
 * Get V2 diagnostics for debugging
 */
export function useV2Diagnostics() {
  const [diagnostics, setDiagnostics] = useState<any>(null);

  const refresh = useCallback(() => {
    try {
      const backendManager = getBackendManager();
      setDiagnostics(backendManager.getDiagnostics());
    } catch (error) {
      console.error('[useV2Diagnostics] Failed to get diagnostics:', error);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { diagnostics, refresh };
}
