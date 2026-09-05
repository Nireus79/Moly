/**
 * Conversation API Client
 * Handles communication with Go backend for conversation CRUD operations
 */

const BACKEND_URL = 'http://127.0.0.1:11436';

export interface ConversationCreateRequest {
  name: string;
  type: 'single' | 'group' | 'generic';
  purpose: string;
  notes: string;
  contact_ids: number[];
}

export interface ConversationContextRequest {
  conversation_id: number;
  include_history?: boolean;
}

export interface ConversationContextResponse {
  success: boolean;
  conversation: {
    id: number;
    name: string;
    type: string;
    purpose: string;
    notes: string;
  };
  members: Array<{
    id: number;
    name: string;
    relationship: string;
    platform: string;
    notes?: string;
  }>;
  recent_interactions?: Array<{
    date: string;
    topic: string;
    sentiment: string;
    summary: string;
  }>;
}

export class ConversationAPI {
  /**
   * Create a new conversation on backend
   */
  static async createConversation(
    request: ConversationCreateRequest
  ): Promise<{ success: boolean; conversation: any }> {
    try {
      const response = await fetch(`${BACKEND_URL}/api/conversations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('[ConversationAPI] Create failed:', error);
      throw error;
    }
  }

  /**
   * Get full context for a conversation
   */
  static async getConversationContext(
    conversationId: number,
    includeHistory: boolean = false
  ): Promise<ConversationContextResponse> {
    try {
      const response = await fetch(
        `${BACKEND_URL}/api/conversations/context?id=${conversationId}&include_history=${includeHistory}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('[ConversationAPI] Get context failed:', error);
      throw error;
    }
  }

  /**
   * Sync local conversation to backend
   * (Called when extension wants to persist conversation to database)
   */
  static async syncConversationToBackend(
    conversationData: any
  ): Promise<{ success: boolean }> {
    try {
      // Extract contact IDs from members
      const contactIds = conversationData.members?.map((m: any) => m.id) || [];

      const response = await fetch(`${BACKEND_URL}/api/conversations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: conversationData.name,
          type: conversationData.type,
          purpose: conversationData.purpose,
          notes: conversationData.notes,
          contact_ids: contactIds,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('[ConversationAPI] Sync failed:', error);
      // Return success even if backend is unavailable
      // (local storage is the source of truth for Phase 1)
      return { success: true };
    }
  }

  /**
   * Check if backend is available
   */
  static async isBackendAvailable(): Promise<boolean> {
    try {
      const response = await fetch(`${BACKEND_URL}/api/status`, {
        method: 'GET',
        signal: AbortSignal.timeout(2000),
      });
      return response.ok;
    } catch {
      return false;
    }
  }
}
