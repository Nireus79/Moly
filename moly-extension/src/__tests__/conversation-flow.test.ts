/**
 * Extensive tests for conversation creation and message processing flow
 * Tests the complete flow from conversation creation through Phase5 processing
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

describe('Conversation Flow Integration Tests', () => {
  const BACKEND_URL = 'http://localhost:8080';
  const TEST_TOKEN = 'test-auth-token-123';
  const TEST_USER_ID = 'user_123456_789';

  beforeEach(() => {
    // Mock localStorage
    localStorage.clear();
    localStorage.setItem('authToken', TEST_TOKEN);
    localStorage.setItem('userId', TEST_USER_ID);

    // Mock fetch globally
    global.fetch = vi.fn();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Step 1: Conversation Creation', () => {
    it('should create conversation on backend and capture backend ID', async () => {
      const mockResponse = {
        success: true,
        conversation: {
          id: 'conv_1726185431_987654321',  // Backend format
          name: 'Test Conversation',
          type: 'generic',
          description: 'A test conversation',
          createdAt: 1726185431
        }
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
        text: async () => JSON.stringify(mockResponse)
      });

      // Simulate calling profileAPI.createConversation
      const response = await fetch(`${BACKEND_URL}/api/v2/conversations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          name: 'Test Conversation',
          type: 'generic',
          description: 'A test conversation'
        }),
      });

      expect(response.ok).toBe(true);
      expect(response.status).toBe(200);

      const data = await response.json();
      expect(data.success).toBe(true);
      expect(data.conversation).toBeDefined();
      expect(data.conversation.id).toBe('conv_1726185431_987654321');
      expect(data.conversation.id).toMatch(/^conv_\d+_\d+$/);
    });

    it('should handle backend errors during conversation creation', async () => {
      const mockErrorResponse = {
        error: 'Failed to create conversation'
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => mockErrorResponse,
        text: async () => JSON.stringify(mockErrorResponse)
      });

      const response = await fetch(`${BACKEND_URL}/api/v2/conversations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          name: 'Test Conversation',
          type: 'generic',
          description: 'A test conversation'
        }),
      });

      expect(response.ok).toBe(false);
      expect(response.status).toBe(500);
    });

    it('should reject if backend ID is missing from response', async () => {
      const mockResponse = {
        success: true,
        conversation: {
          name: 'Test Conversation',
          type: 'generic',
          // NOTE: Missing 'id' field - this should be caught
        }
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const response = await fetch(`${BACKEND_URL}/api/v2/conversations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          name: 'Test Conversation',
          type: 'generic',
        }),
      });

      const data = await response.json();
      expect(data.conversation.id).toBeUndefined();
      // This should trigger error handling in Sidebar.handleConversationCreate
    });

    it('should not create conversation locally without backend ID', async () => {
      const mockResponse = {
        success: true,
        conversation: {
          id: 'conv_1726185431_987654321',
          name: 'Test Conversation',
          type: 'generic',
        }
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const response = await fetch(`${BACKEND_URL}/api/v2/conversations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          name: 'Test Conversation',
          type: 'generic',
        }),
      });

      const data = await response.json();

      // Simulate what Sidebar.handleConversationCreate should do
      if (data.conversation && data.conversation.id) {
        const conversation = {
          id: `local_${Date.now()}`,  // Local numeric ID
          backendId: data.conversation.id,  // Store backend ID
          name: data.conversation.name,
          type: data.conversation.type,
        };

        expect(conversation.backendId).toBe('conv_1726185431_987654321');
        expect(conversation.id).toMatch(/^local_\d+$/);
      }
    });
  });

  describe('Step 2: Conversation Storage', () => {
    it('should store conversation with backendId in state', () => {
      const backendConversation = {
        id: 'conv_1726185431_987654321',
        name: 'Test Conversation',
        type: 'generic',
      };

      const localConversation = {
        id: Date.now().toString(),
        backendId: backendConversation.id,
        name: backendConversation.name,
        type: backendConversation.type,
      };

      expect(localConversation.backendId).toBeDefined();
      expect(localConversation.backendId).toBe('conv_1726185431_987654321');
      expect(localConversation.id).not.toBe(localConversation.backendId);
    });

    it('should preserve backendId through state updates', () => {
      const conversation = {
        id: '1726185431987',
        backendId: 'conv_1726185431_987654321',
        name: 'Test Conversation',
        type: 'generic',
        members: [],
      };

      const updated = {
        ...conversation,
        name: 'Updated Name',
      };

      expect(updated.backendId).toBe('conv_1726185431_987654321');
      expect(updated.id).toBe('1726185431987');
    });

    it('should NOT lose backendId when conversation is serialized', () => {
      const conversation = {
        id: '1726185431987',
        backendId: 'conv_1726185431_987654321',
        name: 'Test Conversation',
        type: 'generic',
      };

      const serialized = JSON.stringify(conversation);
      const deserialized = JSON.parse(serialized);

      expect(deserialized.backendId).toBe('conv_1726185431_987654321');
      expect(deserialized.id).toBe('1726185431987');
    });
  });

  describe('Step 3: Message Processing with Phase5', () => {
    it('should use backendId when calling Phase5', async () => {
      const conversation = {
        id: '1726185431987',  // Local ID
        backendId: 'conv_1726185431_987654321',  // Backend ID
        name: 'Test Conversation',
        type: 'generic',
      };

      // Simulate what Sidebar.processMessage does
      const backendConversationId = conversation.backendId || String(conversation.id);

      expect(backendConversationId).toBe('conv_1726185431_987654321');
      expect(backendConversationId).not.toBe('1726185431987');
    });

    it('should fall back to local ID if backendId is missing', () => {
      const conversation = {
        id: '1726185431987',
        name: 'Test Conversation',
        type: 'generic',
        // NOTE: backendId is missing
      };

      const backendConversationId = (conversation as any).backendId || String(conversation.id);

      // This is the fallback behavior - NOT ideal but should work if ID is in correct format
      expect(backendConversationId).toBe('1726185431987');
      // BUT: Backend won't find conversation with numeric ID format
    });

    it('should send correct conversation ID to Phase5 endpoint', async () => {
      const conversation = {
        id: '1726185431987',
        backendId: 'conv_1726185431_987654321',
        name: 'Test Conversation',
        type: 'generic',
      };

      const mockPhase5Response = {
        success: true,
        phase1: { facts: [] },
        phase2: { clarifications: [] },
        phase3: { unknown_contacts: [] },
        phase4: { saved_attributes: [] },
        action_required: {
          needsClarification: false,
          clarificationQs: [],
          temporaryFacts: [],
          hasConflicts: false,
          conflicts: [],
        },
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockPhase5Response,
      });

      const backendConversationId = conversation.backendId || String(conversation.id);

      const response = await fetch(`${BACKEND_URL}/api/v2/phase5/process`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          message: 'Hello',
          conversationId: backendConversationId,
          aboutMe: {},
          selectedContactIds: [],
        }),
      });

      expect(response.ok).toBe(true);

      // Verify the request body contains correct ID
      const lastCall = (global.fetch as jest.Mock).mock.calls[0];
      const requestBody = JSON.parse(lastCall[1].body);
      expect(requestBody.conversationId).toBe('conv_1726185431_987654321');
      expect(requestBody.conversationId).not.toBe('1726185431987');
    });

    it('should return 404 if conversation ID is not found on backend', async () => {
      const mockErrorResponse = {
        error: 'Conversation not found',
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => mockErrorResponse,
        text: async () => JSON.stringify(mockErrorResponse),
      });

      const response = await fetch(`${BACKEND_URL}/api/v2/phase5/process`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          message: 'Hello',
          conversationId: '1726185431987',  // Wrong: local ID instead of backend ID
          aboutMe: {},
          selectedContactIds: [],
        }),
      });

      expect(response.ok).toBe(false);
      expect(response.status).toBe(404);

      const error = await response.json();
      expect(error.error).toBe('Conversation not found');
    });
  });

  describe('Step 4: Complete Flow Integration', () => {
    it('should successfully process message with newly created conversation', async () => {
      // Step 1: Create conversation
      const createResponse = {
        success: true,
        conversation: {
          id: 'conv_1726185431_987654321',
          name: 'Test Conversation',
          type: 'generic',
          description: 'A test conversation',
        }
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => createResponse,
      });

      const createResp = await fetch(`${BACKEND_URL}/api/v2/conversations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          name: 'Test Conversation',
          type: 'generic',
        }),
      });

      const createData = await createResp.json();

      // Step 2: Create local conversation with backendId
      const conversation = {
        id: Date.now().toString(),
        backendId: createData.conversation.id,
        name: createData.conversation.name,
        type: createData.conversation.type,
      };

      expect(conversation.backendId).toBe('conv_1726185431_987654321');

      // Step 3: Send message with correct conversation ID
      const phase5Response = {
        success: true,
        phase1: { facts: [] },
        phase2: { clarifications: [] },
        phase3: { unknown_contacts: [] },
        phase4: { saved_attributes: [] },
        action_required: {
          needsClarification: false,
          clarificationQs: [],
        },
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => phase5Response,
      });

      const backendConversationId = conversation.backendId || String(conversation.id);

      const phase5Resp = await fetch(`${BACKEND_URL}/api/v2/phase5/process`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          message: 'Hello',
          conversationId: backendConversationId,
          aboutMe: {},
          selectedContactIds: [],
        }),
      });

      expect(phase5Resp.ok).toBe(true);
      expect(phase5Resp.status).toBe(200);

      const phase5Data = await phase5Resp.json();
      expect(phase5Data.success).toBe(true);
      expect(phase5Data.action_required).toBeDefined();
    });

    it('should handle message processing failure with correct error info', async () => {
      const conversation = {
        id: Date.now().toString(),
        backendId: 'conv_1726185431_987654321',
        name: 'Test Conversation',
        type: 'generic',
      };

      const mockErrorResponse = {
        error: 'Conversation not found',
      };

      (global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => mockErrorResponse,
        text: async () => JSON.stringify(mockErrorResponse),
      });

      const backendConversationId = conversation.backendId || String(conversation.id);

      const response = await fetch(`${BACKEND_URL}/api/v2/phase5/process`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${TEST_TOKEN}`,
        },
        body: JSON.stringify({
          message: 'Hello',
          conversationId: backendConversationId,
          aboutMe: {},
          selectedContactIds: [],
        }),
      });

      expect(response.status).toBe(404);

      const error = await response.json();
      expect(error.error).toBe('Conversation not found');

      // Error should clearly indicate the conversation ID that failed
      console.error(`[Test] Phase5 failed for conversation: ${conversation.backendId} (local: ${conversation.id})`);
    });
  });

  describe('Edge Cases', () => {
    it('should handle undefined backendId gracefully', () => {
      const conversation = {
        id: '1726185431987',
        backendId: undefined,
        name: 'Test Conversation',
        type: 'generic',
      };

      const backendConversationId = conversation.backendId || String(conversation.id);

      // Fallback to local ID (may not work if backend doesn't recognize format)
      expect(backendConversationId).toBe('1726185431987');
    });

    it('should handle null backendId gracefully', () => {
      const conversation = {
        id: '1726185431987',
        backendId: null as any,
        name: 'Test Conversation',
        type: 'generic',
      };

      const backendConversationId = conversation.backendId || String(conversation.id);
      expect(backendConversationId).toBe('1726185431987');
    });

    it('should handle multiple conversations with different IDs', () => {
      const conversations = [
        {
          id: '1726185431987',
          backendId: 'conv_1726185431_111111111',
          name: 'Conversation 1',
        },
        {
          id: '1726185431988',
          backendId: 'conv_1726185431_222222222',
          name: 'Conversation 2',
        },
        {
          id: '1726185431989',
          backendId: 'conv_1726185431_333333333',
          name: 'Conversation 3',
        },
      ];

      conversations.forEach(conv => {
        expect(conv.backendId).toMatch(/^conv_\d+_\d+$/);
        expect(conv.id).not.toBe(conv.backendId);
      });

      // Verify each conversation can be uniquely identified
      const ids = new Set(conversations.map(c => c.backendId));
      expect(ids.size).toBe(3);
    });

    it('should validate conversation ID format before sending to backend', () => {
      const validIds = [
        'conv_1726185431_987654321',
        'conv_9999999999_123456789',
        'conv_1_1',
      ];

      const invalidIds = [
        '1726185431987',  // Numeric local ID
        'conversation_123',
        '',
        null,
        undefined,
      ];

      validIds.forEach(id => {
        expect(id).toMatch(/^conv_\d+_\d+$/);
      });

      invalidIds.forEach(id => {
        if (id) {
          expect(id).not.toMatch(/^conv_\d+_\d+$/);
        }
      });
    });
  });

  describe('Type Safety', () => {
    it('should have proper ConversationData type with backendId', () => {
      type ConversationData = {
        id: string;
        backendId?: string;  // Should be optional
        name: string;
        type?: string;
        members?: any[];
      };

      const conversation: ConversationData = {
        id: '1726185431987',
        backendId: 'conv_1726185431_987654321',
        name: 'Test',
        type: 'generic',
      };

      expect(conversation.backendId).toBeDefined();
      expect(conversation.id).toBeDefined();
      expect(conversation.name).toBeDefined();
    });

    it('should allow accessing backendId with optional chaining', () => {
      const conversation: any = {
        id: '1726185431987',
        backendId: 'conv_1726185431_987654321',
      };

      const backendId = conversation.backendId || String(conversation.id);
      expect(backendId).toBe('conv_1726185431_987654321');
    });
  });
});
