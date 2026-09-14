/**
 * Clarification API Tests
 * Tests Phase 5 orchestrator integration
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ClarificationAPI, setUserId } from '../clarificationAPI';

describe('ClarificationAPI', () => {
  beforeEach(() => {
    setUserId('test-user-123');
    // Mock fetch
    global.fetch = vi.fn();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('processMessage', () => {
    it('should call Phase 5 endpoint with correct parameters', async () => {
      const mockResponse = {
        success: true,
        phase1: { facts: [], shifts: [] },
        phase2: { clarifications: [], resolved: {} },
        phase3: { unknown_contacts: [], created_contacts: [] },
        phase4: { saved_attributes: [], conflicts: [] },
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
        json: async () => mockResponse
      });

      const result = await ClarificationAPI.processMessage(
        'I am organized',
        'conv-123'
      );

      expect(global.fetch).toHaveBeenCalledWith(
        '/api/v2/phase5/process',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'X-User-ID': 'test-user-123',
            'Content-Type': 'application/json'
          })
        })
      );

      expect(result).toEqual(mockResponse);
    });

    it('should return clarification questions when needed', async () => {
      const mockResponse = {
        success: true,
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
            }
          ],
          temporaryFacts: [
            {
              factId: 'fact-1',
              type: 'personality_trait',
              value: 'organized',
              attributedTo: 'user',
              linkedQuestionIds: ['q1'],
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
        json: async () => mockResponse
      });

      const result = await ClarificationAPI.processMessage(
        'I am organized',
        'conv-123'
      );

      expect(result.action_required.needsClarification).toBe(true);
      expect(result.action_required.clarificationQs).toHaveLength(1);
      expect(result.action_required.temporaryFacts).toHaveLength(1);
    });

    it('should throw error on network failure', async () => {
      (global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        text: async () => 'Server error'
      });

      await expect(
        ClarificationAPI.processMessage('I am organized', 'conv-123')
      ).rejects.toThrow('Failed to process message');
    });
  });

  describe('submitAnswer', () => {
    it('should submit answer with option selection', async () => {
      const mockResponse = {
        questionAnswered: true,
        factId: 'fact-1',
        status: 'pending',
        remainingQuestions: ['q2', 'q3']
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      });

      const result = await ClarificationAPI.submitAnswer(
        'q1',
        'fact-1',
        'At work',
        'At work'
      );

      expect(global.fetch).toHaveBeenCalledWith(
        '/api/v2/clarification/respond',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'X-User-ID': 'test-user-123'
          }),
          body: expect.stringContaining('"selectedOption":"At work"')
        })
      );

      expect(result.status).toBe('pending');
      expect(result.remainingQuestions).toHaveLength(2);
    });

    it('should submit answer with text input', async () => {
      const mockResponse = {
        questionAnswered: true,
        factId: 'fact-1',
        status: 'saved',
        remainingQuestions: []
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      });

      const result = await ClarificationAPI.submitAnswer(
        'q3',
        'fact-1',
        'I organize my files systematically'
      );

      expect(global.fetch).toHaveBeenCalledWith(
        '/api/v2/clarification/respond',
        expect.objectContaining({
          body: expect.stringContaining('"userResponse"')
        })
      );

      expect(result.status).toBe('saved');
    });

    it('should throw error on failed submission', async () => {
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
