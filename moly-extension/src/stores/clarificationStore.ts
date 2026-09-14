/**
 * Clarification Store - State management for clarification questions flow
 * Tracks questions, answers, and temporary facts
 */

import { create } from 'zustand';
import type { ClarificationQuestion, TemporaryFact } from '@/api/clarificationAPI';

export interface ClarificationState {
  // Questions and facts
  questions: ClarificationQuestion[];
  temporaryFacts: TemporaryFact[];
  currentQuestionIndex: number;

  // Answers tracking
  answers: Record<string, string>; // questionId -> answer

  // UI state
  isOpen: boolean;
  isSubmitting: boolean;
  error: string | null;

  // Actions
  setQuestions: (questions: ClarificationQuestion[], facts: TemporaryFact[]) => void;
  recordAnswer: (questionId: string, answer: string) => void;
  getCurrentQuestion: () => ClarificationQuestion | null;
  getRemainingQuestions: () => ClarificationQuestion[];
  getProgress: () => { current: number; total: number };
  setSubmitting: (submitting: boolean) => void;
  setError: (error: string | null) => void;
  setOpen: (open: boolean) => void;
  clear: () => void;
}

// Hydrate from localStorage on store creation
const getInitialState = () => {
  try {
    const savedState = localStorage.getItem('clarification_state');
    const savedAnswers = localStorage.getItem('clarification_answers');

    if (savedState) {
      const parsed = JSON.parse(savedState);
      return {
        questions: parsed.questions || [],
        temporaryFacts: parsed.temporaryFacts || [],
        answers: savedAnswers ? JSON.parse(savedAnswers) : {},
      };
    }
  } catch (err) {
    console.warn('[ClarificationStore] Failed to hydrate from localStorage:', err);
  }

  return {
    questions: [],
    temporaryFacts: [],
    answers: {},
  };
};

export const useClarificationStore = create<ClarificationState>((set, get) => {
  const initial = getInitialState();

  return {
    questions: initial.questions,
    temporaryFacts: initial.temporaryFacts,
    currentQuestionIndex: 0,
    answers: initial.answers,
    isOpen: initial.questions.length > 0,
    isSubmitting: false,
    error: null,

    setQuestions: (questions, facts) => {
    // Filter out facts without id (malformed from backend)
    // Backend sends 'id', not 'factId'
    const validFacts = facts.filter((f: any) => f && (f.id || f.factId));

    console.info('[ClarificationStore] Setting questions:', {
      questionsCount: questions.length,
      factsCount: facts.length,
      validFactsCount: validFacts.length,
      willOpen: questions.length > 0
    });

    if (facts.length !== validFacts.length) {
      console.warn('[ClarificationStore] Filtered out malformed facts', {
        original: facts.length,
        valid: validFacts.length,
        malformed: facts.filter((f: any) => !f || !f.factId)
      });
    }

    const newState = {
      questions,
      temporaryFacts: validFacts,
      currentQuestionIndex: 0,
      answers: {},
      error: null,
      isOpen: questions.length > 0
    };

    // Persist to localStorage
    try {
      localStorage.setItem('clarification_state', JSON.stringify({
        questions,
        temporaryFacts: validFacts,
        timestamp: Date.now(),
      }));
      localStorage.setItem('clarification_answers', JSON.stringify({}));
    } catch (err) {
      console.warn('[ClarificationStore] Failed to persist to localStorage:', err);
    }

    set(newState);
    },

    recordAnswer: (questionId, answer) => {
    console.info('[ClarificationStore] Recording answer:', {
      questionId: questionId.substring(0, 8) + '...',
      answerLength: answer.length,
      answerPreview: answer.substring(0, 30)
    });
    set(state => {
      const newAnswers = {
        ...state.answers,
        [questionId]: answer
      };

      // Persist answers to localStorage
      try {
        localStorage.setItem('clarification_answers', JSON.stringify(newAnswers));
      } catch (err) {
        console.warn('[ClarificationStore] Failed to persist answers:', err);
      }

      return { answers: newAnswers };
    });
    },

    getCurrentQuestion: () => {
      const state = get();
      return state.questions[state.currentQuestionIndex] || null;
    },

    getRemainingQuestions: () => {
      const state = get();
      return state.questions.slice(state.currentQuestionIndex);
    },

    getProgress: () => {
      const state = get();
      return {
        current: state.currentQuestionIndex + 1,
        total: state.questions.length
      };
    },

    setSubmitting: (submitting) => {
      set({ isSubmitting: submitting });
    },

    setError: (error) => {
      set({ error });
    },

    setOpen: (open) => {
      console.info('[ClarificationStore] Setting isOpen:', { open });
      set({ isOpen: open });
      if (!open) {
        console.info('[ClarificationStore] Clearing state on close');
        set({
          questions: [],
          temporaryFacts: [],
          answers: {},
          currentQuestionIndex: 0
        });
      }
    },

    clear: () => {
      console.info('[ClarificationStore] Clearing all state');

      // Clear localStorage
      try {
        localStorage.removeItem('clarification_state');
        localStorage.removeItem('clarification_answers');
      } catch (err) {
        console.warn('[ClarificationStore] Failed to clear localStorage:', err);
      }

      set({
        questions: [],
        temporaryFacts: [],
        currentQuestionIndex: 0,
        answers: {},
        isSubmitting: false,
        error: null,
        isOpen: false
      });
    }
  };
});
