/**
 * Clarification Store Tests
 * Tests Zustand state management
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { useClarificationStore } from '../clarificationStore';

describe('useClarificationStore', () => {
  beforeEach(() => {
    // Reset store to initial state
    useClarificationStore.setState({
      questions: [],
      temporaryFacts: [],
      currentQuestionIndex: 0,
      answers: {},
      isOpen: false,
      isSubmitting: false,
      error: null
    });
  });

  describe('setQuestions', () => {
    it('should set questions and facts', () => {
      const questions = [
        {
          id: 'q1',
          type: 'user_context',
          question: 'In what situations?',
          options: ['A', 'B'],
          linkedFacts: ['f1'],
          priority: 1,
          status: 'pending' as const
        }
      ];

      const facts = [
        {
          factId: 'f1',
          type: 'trait',
          value: 'organized',
          attributedTo: 'user',
          linkedQuestionIds: ['q1'],
          answeredQuestionIds: [],
          status: 'pending' as const
        }
      ];

      const { setQuestions } = useClarificationStore.getState();
      setQuestions(questions, facts);

      const state = useClarificationStore.getState();
      expect(state.questions).toEqual(questions);
      expect(state.temporaryFacts).toEqual(facts);
      expect(state.currentQuestionIndex).toBe(0);
      expect(state.isOpen).toBe(true);
    });

    it('should open modal only if questions exist', () => {
      const { setQuestions } = useClarificationStore.getState();
      setQuestions([], []);

      const state = useClarificationStore.getState();
      expect(state.isOpen).toBe(false);
    });
  });

  describe('recordAnswer', () => {
    it('should record user answer to question', () => {
      const { recordAnswer } = useClarificationStore.getState();
      recordAnswer('q1', 'At work');

      const state = useClarificationStore.getState();
      expect(state.answers).toEqual({ q1: 'At work' });
    });

    it('should accumulate multiple answers', () => {
      const { recordAnswer } = useClarificationStore.getState();
      recordAnswer('q1', 'At work');
      recordAnswer('q2', 'Yes');
      recordAnswer('q3', 'I organize files');

      const state = useClarificationStore.getState();
      expect(Object.keys(state.answers)).toHaveLength(3);
      expect(state.answers.q2).toBe('Yes');
    });
  });

  describe('getCurrentQuestion', () => {
    it('should return current question', () => {
      const questions = [
        {
          id: 'q1',
          type: 'user_context',
          question: 'Q1?',
          linkedFacts: [],
          priority: 1,
          status: 'pending' as const
        },
        {
          id: 'q2',
          type: 'user_context',
          question: 'Q2?',
          linkedFacts: [],
          priority: 2,
          status: 'pending' as const
        }
      ];

      useClarificationStore.setState({ questions, currentQuestionIndex: 0 });

      const { getCurrentQuestion } = useClarificationStore.getState();
      const current = getCurrentQuestion();

      expect(current).toEqual(questions[0]);
    });

    it('should return null if no current question', () => {
      useClarificationStore.setState({ questions: [] });

      const { getCurrentQuestion } = useClarificationStore.getState();
      expect(getCurrentQuestion()).toBeNull();
    });
  });

  describe('getRemainingQuestions', () => {
    it('should return questions from current index to end', () => {
      const questions = [
        { id: 'q1', question: 'Q1?', type: 'user_context', linkedFacts: [], priority: 1, status: 'pending' as const },
        { id: 'q2', question: 'Q2?', type: 'user_context', linkedFacts: [], priority: 2, status: 'pending' as const },
        { id: 'q3', question: 'Q3?', type: 'user_context', linkedFacts: [], priority: 3, status: 'pending' as const }
      ];

      useClarificationStore.setState({ questions, currentQuestionIndex: 1 });

      const { getRemainingQuestions } = useClarificationStore.getState();
      const remaining = getRemainingQuestions();

      expect(remaining).toHaveLength(2);
      expect(remaining[0].id).toBe('q2');
      expect(remaining[1].id).toBe('q3');
    });
  });

  describe('getProgress', () => {
    it('should return current progress', () => {
      const questions = [
        { id: 'q1', question: 'Q1?', type: 'user_context', linkedFacts: [], priority: 1, status: 'pending' as const },
        { id: 'q2', question: 'Q2?', type: 'user_context', linkedFacts: [], priority: 2, status: 'pending' as const },
        { id: 'q3', question: 'Q3?', type: 'user_context', linkedFacts: [], priority: 3, status: 'pending' as const }
      ];

      useClarificationStore.setState({ questions, currentQuestionIndex: 1 });

      const { getProgress } = useClarificationStore.getState();
      const progress = getProgress();

      expect(progress.current).toBe(2); // 1 + 1 (0-indexed)
      expect(progress.total).toBe(3);
    });
  });

  describe('setOpen', () => {
    it('should set isOpen flag', () => {
      const { setOpen } = useClarificationStore.getState();
      setOpen(true);

      let state = useClarificationStore.getState();
      expect(state.isOpen).toBe(true);

      setOpen(false);
      state = useClarificationStore.getState();
      expect(state.isOpen).toBe(false);
    });

    it('should clear state when closing', () => {
      // Populate state
      const questions = [
        { id: 'q1', question: 'Q1?', type: 'user_context', linkedFacts: [], priority: 1, status: 'pending' as const }
      ];
      useClarificationStore.setState({
        questions,
        currentQuestionIndex: 1,
        answers: { q1: 'answer' }
      });

      const { setOpen } = useClarificationStore.getState();
      setOpen(false);

      const state = useClarificationStore.getState();
      expect(state.questions).toEqual([]);
      expect(state.answers).toEqual({});
      expect(state.currentQuestionIndex).toBe(0);
    });
  });

  describe('clear', () => {
    it('should reset all state to defaults', () => {
      const questions = [
        { id: 'q1', question: 'Q1?', type: 'user_context', linkedFacts: [], priority: 1, status: 'pending' as const }
      ];

      useClarificationStore.setState({
        questions,
        isOpen: true,
        isSubmitting: true,
        error: 'Some error',
        answers: { q1: 'answer' }
      });

      const { clear } = useClarificationStore.getState();
      clear();

      const state = useClarificationStore.getState();
      expect(state.questions).toEqual([]);
      expect(state.temporaryFacts).toEqual([]);
      expect(state.currentQuestionIndex).toBe(0);
      expect(state.answers).toEqual({});
      expect(state.isOpen).toBe(false);
      expect(state.isSubmitting).toBe(false);
      expect(state.error).toBeNull();
    });
  });
});
