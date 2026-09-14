/**
 * Clarification Integration Tests
 * Tests the complete flow from message to clarification to answer
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ClarificationAPI, setUserId } from '../api/clarificationAPI';
import { useClarificationStore } from '../stores/clarificationStore';

describe('Clarification Integration Flow', () => {
  beforeEach(() => {
    setUserId('test-user-123');
    global.fetch = vi.fn();

    // Reset store
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

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Full Clarification Flow', () => {
    it('should handle complete user fact clarification flow', async () => {
      // Step 1: User sends message about themselves
      const userMessage = 'I am very organized';
      const conversationId = 'conv-123';

      // Mock Phase 5 response with clarification questions
      const phase5Response = {
        success: true,
        phase1: { facts: [], shifts: [] },
        phase2: { clarifications: [], resolved: {} },
        phase3: { unknown_contacts: [], created_contacts: [] },
        phase4: { saved_attributes: [], conflicts: [] },
        action_required: {
          needsClarification: true,
          clarificationQs: [
            {
              id: 'q1',
              type: 'user_context',
              question: 'In what situations are you organized?',
              options: ['At work', 'Personal', 'All', 'Depends'],
              linkedFacts: ['fact-1'],
              priority: 1,
              status: 'pending'
            },
            {
              id: 'q2',
              type: 'user_context',
              question: 'Is this consistent?',
              options: ['Yes', 'No', 'Depends'],
              linkedFacts: ['fact-1'],
              priority: 2,
              status: 'pending'
            },
            {
              id: 'q3',
              type: 'user_context',
              question: 'Tell me about a time when this was important.',
              linkedFacts: ['fact-1'],
              priority: 3,
              status: 'pending'
            }
          ],
          temporaryFacts: [
            {
              factId: 'fact-1',
              type: 'personality_trait',
              value: 'organized',
              attributedTo: 'user',
              linkedQuestionIds: ['q1', 'q2', 'q3'],
              answeredQuestionIds: [],
              status: 'pending'
            }
          ],
          hasConflicts: false,
          conflicts: []
        }
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => phase5Response
      });

      // Step 1: Process message through Phase 5
      const response = await ClarificationAPI.processMessage(userMessage, conversationId);

      expect(response.action_required.needsClarification).toBe(true);
      expect(response.action_required.clarificationQs).toHaveLength(3);

      // Step 2: Populate clarification store (simulating Sidebar.tsx behavior)
      const { setQuestions, recordAnswer } = useClarificationStore.getState();
      setQuestions(
        response.action_required.clarificationQs,
        response.action_required.temporaryFacts
      );

      let state = useClarificationStore.getState();
      expect(state.isOpen).toBe(true);
      expect(state.questions).toHaveLength(3);
      expect(state.temporaryFacts).toHaveLength(1);

      // Step 3: User answers first question (option selection)
      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          questionAnswered: true,
          factId: 'fact-1',
          status: 'pending',
          remainingQuestions: ['q2', 'q3']
        })
      });

      const answer1Response = await ClarificationAPI.submitAnswer(
        'q1',
        'fact-1',
        'At work',
        'At work'
      );

      expect(answer1Response.status).toBe('pending');
      expect(answer1Response.remainingQuestions).toHaveLength(2);

      recordAnswer('q1', 'At work');
      state = useClarificationStore.getState();
      expect(state.answers.q1).toBe('At work');

      // Advance to next question
      useClarificationStore.setState({ currentQuestionIndex: 1 });

      // Step 4: User answers second question
      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          questionAnswered: true,
          factId: 'fact-1',
          status: 'pending',
          remainingQuestions: ['q3']
        })
      });

      const answer2Response = await ClarificationAPI.submitAnswer(
        'q2',
        'fact-1',
        'Yes',
        'Yes'
      );

      expect(answer2Response.status).toBe('pending');
      recordAnswer('q2', 'Yes');

      useClarificationStore.setState({ currentQuestionIndex: 2 });

      // Step 5: User answers third question (text input)
      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          questionAnswered: true,
          factId: 'fact-1',
          status: 'saved',
          remainingQuestions: [],
          savedAttribute: {
            id: 1,
            factType: 'personality_trait',
            factValue: 'organized',
            attributedTo: 'user',
            context: 'at_work',
            confidence: 0.95
          }
        })
      });

      const answer3Response = await ClarificationAPI.submitAnswer(
        'q3',
        'fact-1',
        'I organize my work files and calendar systematically'
      );

      expect(answer3Response.status).toBe('saved');
      expect(answer3Response.savedAttribute).toBeDefined();

      recordAnswer('q3', 'I organize my work files and calendar systematically');

      state = useClarificationStore.getState();
      expect(Object.keys(state.answers)).toHaveLength(3);

      // Step 6: Close modal (simulating successful completion)
      const { setOpen, clear } = useClarificationStore.getState();
      setOpen(false);

      state = useClarificationStore.getState();
      expect(state.isOpen).toBe(false);
      expect(state.questions).toEqual([]);
    });
  });

  describe('Contact Clarification Flow', () => {
    it('should handle unknown contact clarification', async () => {
      const userMessage = 'My friend Alex is very creative';
      const conversationId = 'conv-123';

      // Mock Phase 5 response for unknown contact
      const phase5Response = {
        success: true,
        action_required: {
          needsClarification: true,
          clarificationQs: [
            {
              id: 'q1',
              type: 'contact_context',
              question: 'How long have you known Alex?',
              options: ['Less than a year', '1-3 years', '3+ years'],
              linkedFacts: ['fact-1'],
              priority: 1,
              status: 'pending'
            },
            {
              id: 'q2',
              type: 'contact_context',
              question: 'How often do you interact?',
              options: ['Occasionally', 'Regularly', 'Daily'],
              linkedFacts: ['fact-1'],
              priority: 2,
              status: 'pending'
            },
            {
              id: 'q3',
              type: 'contact_context',
              question: 'What brings them joy?',
              linkedFacts: ['fact-1'],
              priority: 3,
              status: 'pending'
            }
          ],
          temporaryFacts: [
            {
              factId: 'fact-1',
              type: 'contact_trait',
              value: 'creative',
              attributedTo: 'contact_alex',
              linkedQuestionIds: ['q1', 'q2', 'q3'],
              answeredQuestionIds: [],
              status: 'pending'
            }
          ],
          hasConflicts: false,
          conflicts: []
        }
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => phase5Response
      });

      const response = await ClarificationAPI.processMessage(userMessage, conversationId);

      expect(response.action_required.clarificationQs[0].type).toBe('contact_context');
      expect(response.action_required.temporaryFacts[0].attributedTo).toBe('contact_alex');

      // Populate store
      const { setQuestions } = useClarificationStore.getState();
      setQuestions(
        response.action_required.clarificationQs,
        response.action_required.temporaryFacts
      );

      const state = useClarificationStore.getState();
      expect(state.isOpen).toBe(true);
      expect(state.temporaryFacts[0].attributedTo).toBe('contact_alex');
    });
  });

  describe('No Clarification Needed Flow', () => {
    it('should skip clarification when not needed', async () => {
      const userMessage = 'Hello';
      const conversationId = 'conv-123';

      // Mock Phase 5 response without clarification
      const phase5Response = {
        success: true,
        action_required: {
          needsClarification: false,
          clarificationQs: [],
          temporaryFacts: [],
          hasConflicts: false,
          conflicts: []
        }
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => phase5Response
      });

      const response = await ClarificationAPI.processMessage(userMessage, conversationId);

      expect(response.action_required.needsClarification).toBe(false);
      expect(response.action_required.clarificationQs).toHaveLength(0);

      // Store should not open
      const { setQuestions } = useClarificationStore.getState();
      setQuestions(
        response.action_required.clarificationQs,
        response.action_required.temporaryFacts
      );

      const state = useClarificationStore.getState();
      expect(state.isOpen).toBe(false);
    });
  });

  describe('Error Handling', () => {
    it('should handle network errors gracefully', async () => {
      (global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        text: async () => 'Server error'
      });

      await expect(
        ClarificationAPI.processMessage('test message', 'conv-123')
      ).rejects.toThrow('Failed to process message');
    });

    it('should handle answer submission errors', async () => {
      (global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        text: async () => 'Invalid answer'
      });

      await expect(
        ClarificationAPI.submitAnswer('q1', 'fact-1', 'answer')
      ).rejects.toThrow('Failed to submit answer');
    });
  });
});
