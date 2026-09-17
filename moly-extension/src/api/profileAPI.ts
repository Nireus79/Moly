/**
 * Profile API Client - FIXED for v2 endpoints
 * 
 * Uses /api/v2/* endpoints with Bearer token authentication
 * Methods for endpoints that don't exist return empty/default data
 */

import { getBackendManager } from './backendManager';

function getBackendUrl(): string {
  return getBackendManager().getBackendUrl();
}

function getAuthToken(): string {
  try {
    // Read from new auth store (moly_session)
    const sessionStr = localStorage.getItem('moly_session');
    if (sessionStr) {
      const session = JSON.parse(sessionStr);
      if (session.sessionId && session.expiresAt) {
        // Check if token is expired
        if (session.expiresAt > Date.now()) {
          return session.sessionId;
        } else {
          console.error('[ProfileAPI] Token expired - clearing auth');
          localStorage.removeItem('moly_session');
          throw new Error('Auth token expired');
        }
      }
    }

    // Fallback to old auth method for backward compatibility
    const token = localStorage.getItem('authToken');
    if (token) {
      const expiresAtStr = localStorage.getItem('tokenExpiresAt');
      if (expiresAtStr) {
        const expiresAt = parseInt(expiresAtStr, 10);
        if (expiresAt > Date.now()) {
          return token;
        } else {
          console.error('[ProfileAPI] Token expired - clearing auth');
          localStorage.removeItem('authToken');
          localStorage.removeItem('tokenExpiresAt');
        }
      }
    }
  } catch (err) {
    console.warn('[ProfileAPI] Could not read auth token from localStorage:', err);
  }
  throw new Error('Auth token not available');
}

export interface AboutMeProfile {
  communicationStyle?: string;
  tonePreference?: string;
  coreValues?: string[];
  preferences?: { notes?: string };
  goals?: string[];
  patterns?: string[];
}

export interface ContactProfile {
  id?: string;
  name: string;
  relationship?: string;
  notes?: string;
}

export interface UserProfile {
  userId?: string;
  aboutMe?: AboutMeProfile;
  contacts: ContactProfile[];
}

class ProfileAPI {
  /**
   * Get AboutMe profile from /api/v2/about-me
   */
  async getAboutMe(): Promise<AboutMeProfile | null> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/v2/about-me`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to get about-me:', response.status);
        return null;
      }

      const data = await response.json();
      return data.profile || null;
    } catch (error) {
      console.error('[ProfileAPI] getAboutMe failed:', error);
      return null;
    }
  }

  /**
   * Save AboutMe profile to /api/v2/about-me
   */
  async saveAboutMe(profile: AboutMeProfile): Promise<boolean> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/v2/about-me`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
        body: JSON.stringify(profile),
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to save about-me:', response.status);
        return false;
      }

      return true;
    } catch (error) {
      console.error('[ProfileAPI] saveAboutMe failed:', error);
      return false;
    }
  }

  /**
   * Get contacts from /api/v2/contacts
   */
  async getContacts(): Promise<ContactProfile[]> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/v2/contacts`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to get contacts:', response.status);
        return [];
      }

      const data = await response.json();
      return data.contacts || [];
    } catch (error) {
      console.error('[ProfileAPI] getContacts failed:', error);
      return [];
    }
  }

  /**
   * Create/save contact to /api/v2/contacts
   */
  async saveContact(contact: ContactProfile): Promise<ContactProfile | null> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/v2/contacts`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
        body: JSON.stringify(contact),
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to save contact:', response.status);
        return null;
      }

      const data = await response.json();
      return data.contact || null;
    } catch (error) {
      console.error('[ProfileAPI] saveContact failed:', error);
      return null;
    }
  }

  /**
   * Get patterns from metrics endpoint
   */
  async getPatterns(): Promise<any[]> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/v2/metrics`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to get patterns:', response.status);
        return [];
      }

      const data = await response.json();
      // Extract patterns from metrics data structure
      const metrics = data.metrics || {};
      return Object.entries(metrics).map(([key, value]) => ({ key, value }));
    } catch (error) {
      console.error('[ProfileAPI] getPatterns failed:', error);
      return [];
    }
  }

  /**
   * Get goals from about-me endpoint
   */
  async getGoals(): Promise<any[]> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/v2/about-me`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to get goals:', response.status);
        return [];
      }

      const data = await response.json();
      const profile = data.profile || {};
      return profile.goals || [];
    } catch (error) {
      console.error('[ProfileAPI] getGoals failed:', error);
      return [];
    }
  }

  /**
   * Get messages for a conversation
   */
  async getMessages(conversationId: string): Promise<any[]> {
    try {
      const response = await fetch(
        `${getBackendUrl()}/api/v2/messages?conversationId=${encodeURIComponent(conversationId)}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${getAuthToken()}`,
          },
        }
      );

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to get messages:', response.status);
        return [];
      }

      const data = await response.json();
      return data.messages || [];
    } catch (error) {
      console.error('[ProfileAPI] getMessages failed:', error);
      return [];
    }
  }

  /**
   * Get learnings from interaction history
   */
  async getLearnings(): Promise<any[]> {
    try {
      // Learnings are derived from conversation messages
      const response = await fetch(`${getBackendUrl()}/api/v2/messages`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to get learnings:', response.status);
        return [];
      }

      const data = await response.json();
      // Extract learning insights from messages
      const messages = data.messages || [];
      return messages.filter((msg: any) => msg.metadata?.learning).slice(0, 20);
    } catch (error) {
      console.error('[ProfileAPI] getLearnings failed:', error);
      return [];
    }
  }

  /**
   * Get reflections from reflections endpoint
   */
  async getReflections(): Promise<any[]> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/v2/reflections`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to get reflections:', response.status);
        return [];
      }

      const data = await response.json();
      return data.reflections || [];
    } catch (error) {
      console.error('[ProfileAPI] getReflections failed:', error);
      return [];
    }
  }

  /**
   * Get conversations from /api/v2/conversations
   */
  async getConversations(): Promise<any[]> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/v2/conversations`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
      });

      if (!response.ok) {
        console.warn('[ProfileAPI] Failed to get conversations:', response.status);
        return [];
      }

      const data = await response.json();
      return data.conversations || [];
    } catch (error) {
      console.error('[ProfileAPI] getConversations failed:', error);
      return [];
    }
  }

  /**
   * Create a new conversation on /api/v2/conversations
   */
  async createConversation(
    name: string,
    type?: string,
    description?: string,
    purpose?: string,
    members?: any[],
    settings?: { mode?: string; context?: string; llmProvider?: string },
    notes?: string
  ): Promise<any> {
    if (!name || name.trim().length === 0) {
      throw new Error('Conversation name is required');
    }

    try {
      const requestBody = {
        name: name.trim(),
        type: type || 'generic',
        description: description || '',
        purpose: purpose || '',
        members: members || [],
        settings: settings || {},
        notes: notes || ''
      };
      console.log('[ProfileAPI] Creating conversation:', requestBody);

      const response = await fetch(`${getBackendUrl()}/api/v2/conversations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getAuthToken()}`,
        },
        body: JSON.stringify(requestBody),
      });

      console.log('[ProfileAPI] Create conversation response status:', response.status, response.statusText);

      if (!response.ok) {
        const errorText = await response.text();
        console.error('[ProfileAPI] Backend error response:', errorText);
        throw new Error(`Failed to create conversation: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();
      console.log('[ProfileAPI] Full response data:', data);
      console.log('[ProfileAPI] Extracted conversation:', data.conversation);

      const conversation = data.conversation || null;
      if (conversation) {
        console.log('[ProfileAPI] Conversation object keys:', Object.keys(conversation));
        console.log('[ProfileAPI] Conversation.id value:', conversation.id);
        console.log('[ProfileAPI] Conversation.ID value:', (conversation as any).ID);
      }

      return conversation;
    } catch (error) {
      console.error('[ProfileAPI] createConversation failed:', error);
      throw error;
    }
  }

  /**
   * Get complete profile
   */
  async getProfile(): Promise<UserProfile> {
    try {
      const [aboutMe, contacts] = await Promise.all([
        this.getAboutMe(),
        this.getContacts(),
      ]);

      return {
        aboutMe: aboutMe || undefined,
        contacts: contacts || [],
      };
    } catch (error) {
      console.error('[ProfileAPI] getProfile failed:', error);
      return { contacts: [] };
    }
  }
}

// Singleton instance
let instance: ProfileAPI | null = null;

export function getProfileAPI(): ProfileAPI {
  if (!instance) {
    instance = new ProfileAPI();
  }
  return instance;
}

// Export singleton instance for direct usage
export const profileAPI = getProfileAPI();

export default ProfileAPI;
