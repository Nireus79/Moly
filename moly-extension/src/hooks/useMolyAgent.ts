/**
 * React Hook: useMolyAgent
 * Provides access to Moly backend analysis systems with loading/error states
 */

import { useState, useCallback } from 'react';
import { getMolyAgent, type SafetyCheckResult, type ConstitutionalAnalysis, type QuestionGeneratorResult } from '@/api/molyAgent';

export interface UseMolyAgentState {
  loading: boolean;
  error: string | null;
  safety: SafetyCheckResult | null;
  constitution: ConstitutionalAnalysis | null;
  questions: QuestionGeneratorResult | null;
}

export function useMolyAgent() {
  const [state, setState] = useState<UseMolyAgentState>({
    loading: false,
    error: null,
    safety: null,
    constitution: null,
    questions: null,
  });

  const agent = getMolyAgent();

  /**
   * Analyze a message with all available checks
   */
  const analyze = useCallback(
    async (message: string, contactName?: string, context?: string) => {
      setState({ loading: true, error: null, safety: null, constitution: null, questions: null });

      try {
        const results = await agent.analyzeMessage(message, contactName, context);

        setState({
          loading: false,
          error: null,
          safety: results.safety,
          constitution: results.constitution || null,
          questions: results.questions || null,
        });

        return results;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Analysis failed';
        setState({ loading: false, error: errorMessage, safety: null, constitution: null, questions: null });
        throw error;
      }
    },
    [agent]
  );

  /**
   * Check just safety (crisis/illegal language)
   */
  const checkSafety = useCallback(
    async (message: string) => {
      setState((prev) => ({ ...prev, loading: true, error: null }));

      try {
        const result = await agent.checkSafety(message);
        setState((prev) => ({ ...prev, loading: false, safety: result }));
        return result;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Safety check failed';
        setState((prev) => ({ ...prev, loading: false, error: errorMessage }));
        throw error;
      }
    },
    [agent]
  );

  /**
   * Evaluate ethics (constitution)
   */
  const evaluateConstitution = useCallback(
    async (message: string) => {
      setState((prev) => ({ ...prev, loading: true, error: null }));

      try {
        const result = await agent.evaluateConstitution(message);
        setState((prev) => ({ ...prev, loading: false, constitution: result }));
        return result;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Constitution evaluation failed';
        setState((prev) => ({ ...prev, loading: false, error: errorMessage }));
        throw error;
      }
    },
    [agent]
  );

  /**
   * Analyze relationship mode shift
   */
  const analyzeModeShift = useCallback(
    async (currentMode: string, proposedMode: string, context: string) => {
      setState((prev) => ({ ...prev, loading: true, error: null }));

      try {
        const result = await agent.analyzeModeShift(currentMode, proposedMode, context);
        return result;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Mode shift analysis failed';
        setState((prev) => ({ ...prev, loading: false, error: errorMessage }));
        throw error;
      }
    },
    [agent]
  );

  /**
   * Generate contextual questions
   */
  const generateQuestions = useCallback(
    async (contactName: string, context: string) => {
      setState((prev) => ({ ...prev, loading: true, error: null }));

      try {
        const result = await agent.generateQuestions(contactName, context);
        setState((prev) => ({ ...prev, loading: false, questions: result }));
        return result;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Question generation failed';
        setState((prev) => ({ ...prev, loading: false, error: errorMessage }));
        throw error;
      }
    },
    [agent]
  );

  /**
   * Clear all analysis results
   */
  const clear = useCallback(() => {
    setState({ loading: false, error: null, safety: null, constitution: null, questions: null });
  }, []);

  return {
    ...state,
    analyze,
    checkSafety,
    evaluateConstitution,
    analyzeModeShift,
    generateQuestions,
    clear,
  };
}
