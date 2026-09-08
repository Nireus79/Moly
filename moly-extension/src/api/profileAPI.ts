/**
 * Phase 1.2 Profile API Client
 *
 * Handles all communication with the Phase 1.2 backend API:
 * - Fetching user profiles (AboutMe, patterns, goals, learnings, reflections)
 * - Confirming/rejecting learnings
 * - Creating reflections
 * - Updating goals
 */

export interface UserProfile {
  userId: string;
  aboutMe?: AboutMeProfile;
  contacts: ContactProfile[];
  patterns: PatternProfile[];
  goals: GoalProfile[];
  learnings: LearningProfile[];
  reflections: ReflectionEntry[];
  confidenceScores?: Record<string, ConfidenceStats>;
  lastUpdated?: number;
}

export interface AboutMeProfile {
  id?: number;
  communicationStyle?: string;
  tonePreference?: string;
  coreValues?: string[];
  preferences?: Record<string, any>;
  confidence?: number;
  extractedFromCount?: number;
  lastUpdated?: number;
}

export interface ContactProfile {
  id?: number;
  name: string;
  relationshipType: string;
  context?: string;
  firstMentioned?: number;
  timesMentioned: number;
  lastMentioned?: number;
  communicationPatterns?: CommPattern[];
}

export interface CommPattern {
  frequency?: string;
  toneObserved?: string;
  patterns?: string[];
  mainTopics?: string[];
  recentOutcome?: string;
  userNotes?: string;
}

export interface PatternProfile {
  id?: number;
  pattern: string;
  category: string;
  observationCount: number;
  isActive: boolean;
  isGrowthArea: boolean;
  confidence: number;
  firstObserved?: number;
  lastObserved?: number;
}

export interface GoalProfile {
  id?: number;
  goal: string;
  category: string;
  status: 'active' | 'achieved' | 'paused' | 'abandoned';
  startedAt?: number;
  targetDate?: number;
  progressNotes?: string;
  confidence: number;
  created?: number;
}

export interface LearningProfile {
  id?: number;
  learningType: 'about_me' | 'pattern' | 'contact' | 'goal';
  learningKey: string;
  learningValue: string;
  source: 'extraction' | 'user_input';
  confidence: number;
  isConfirmed: boolean;
  isRejected: boolean;
  createdAt?: number;
}

export interface ReflectionEntry {
  id?: number;
  content: string;
  tags?: string[];
  entryType: 'reflection' | 'learning' | 'breakthrough' | 'struggle';
  aboutContactId?: number;
  created?: number;
}

export interface ConfidenceStats {
  average: number;
  count: number;
  min: number;
  max: number;
}

export interface ApiResponse<T> {
  data?: T;
  message?: string;
  error?: string;
  code?: number;
  timestamp?: number;
}

class ProfileAPI {
  private baseUrl: string;
  private userId: string | null = null;

  constructor(baseUrl: string = 'http://localhost:8080') {
    this.baseUrl = baseUrl.replace(/\/$/, ''); // Remove trailing slash
  }

  /**
   * Set the user ID for API calls
   */
  setUserId(userId: string): void {
    this.userId = userId;
  }

  /**
   * Get the current user ID
   */
  async getUserId(): Promise<string> {
    if (this.userId) {
      return this.userId;
    }

    // Try to get from storage
    try {
      const result = await chrome.storage.local.get('userId');
      if (result.userId) {
        this.userId = result.userId;
        return result.userId;
      }
    } catch (err) {
      console.warn('[ProfileAPI] Failed to get userId from storage:', err);
    }

    // Fallback
    this.userId = 'unknown';
    return this.userId;
  }

  /**
   * Fetch complete user profile
   */
  async getProfile(): Promise<UserProfile> {
    const userId = await this.getUserId();

    const response = await fetch(`${this.baseUrl}/api/v2.1/profile`, {
      method: 'GET',
      headers: {
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch profile: ${response.statusText}`);
    }

    const data: ApiResponse<UserProfile> = await response.json();
    if (!data.data) {
      throw new Error('Invalid response format');
    }

    return data.data;
  }

  /**
   * Fetch AboutMe profile only
   */
  async getAboutMe(): Promise<AboutMeProfile | null> {
    const userId = await this.getUserId();

    const response = await fetch(`${this.baseUrl}/api/v2.1/profile/about-me`, {
      method: 'GET',
      headers: {
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch about-me: ${response.statusText}`);
    }

    const data: ApiResponse<AboutMeProfile> = await response.json();
    return data.data || null;
  }

  /**
   * Fetch contacts
   */
  async getContacts(): Promise<ContactProfile[]> {
    const userId = await this.getUserId();

    const response = await fetch(`${this.baseUrl}/api/v2.1/profile/contacts`, {
      method: 'GET',
      headers: {
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch contacts: ${response.statusText}`);
    }

    const data: ApiResponse<ContactProfile[]> = await response.json();
    return data.data || [];
  }

  /**
   * Fetch patterns
   */
  async getPatterns(): Promise<PatternProfile[]> {
    const userId = await this.getUserId();

    const response = await fetch(`${this.baseUrl}/api/v2.1/profile/patterns`, {
      method: 'GET',
      headers: {
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch patterns: ${response.statusText}`);
    }

    const data: ApiResponse<PatternProfile[]> = await response.json();
    return data.data || [];
  }

  /**
   * Fetch goals
   */
  async getGoals(): Promise<GoalProfile[]> {
    const userId = await this.getUserId();

    const response = await fetch(`${this.baseUrl}/api/v2.1/profile/goals`, {
      method: 'GET',
      headers: {
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch goals: ${response.statusText}`);
    }

    const data: ApiResponse<GoalProfile[]> = await response.json();
    return data.data || [];
  }

  /**
   * Fetch learnings
   */
  async getLearnings(): Promise<LearningProfile[]> {
    const userId = await this.getUserId();

    const response = await fetch(`${this.baseUrl}/api/v2.1/profile/learnings`, {
      method: 'GET',
      headers: {
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch learnings: ${response.statusText}`);
    }

    const data: ApiResponse<LearningProfile[]> = await response.json();
    return data.data || [];
  }

  /**
   * Fetch reflections
   */
  async getReflections(): Promise<ReflectionEntry[]> {
    const userId = await this.getUserId();

    const response = await fetch(`${this.baseUrl}/api/v2.1/profile/reflections`, {
      method: 'GET',
      headers: {
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch reflections: ${response.statusText}`);
    }

    const data: ApiResponse<ReflectionEntry[]> = await response.json();
    return data.data || [];
  }

  /**
   * Confirm a learning
   */
  async confirmLearning(learningId: number): Promise<void> {
    const userId = await this.getUserId();

    const response = await fetch(
      `${this.baseUrl}/api/v2.1/learnings/${learningId}/confirm`,
      {
        method: 'POST',
        headers: {
          'X-User-ID': userId,
          'Content-Type': 'application/json',
        },
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to confirm learning: ${response.statusText}`);
    }
  }

  /**
   * Reject a learning
   */
  async rejectLearning(learningId: number): Promise<void> {
    const userId = await this.getUserId();

    const response = await fetch(
      `${this.baseUrl}/api/v2.1/learnings/${learningId}/reject`,
      {
        method: 'POST',
        headers: {
          'X-User-ID': userId,
          'Content-Type': 'application/json',
        },
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to reject learning: ${response.statusText}`);
    }
  }

  /**
   * Add a reflection
   */
  async addReflection(
    content: string,
    entryType: 'reflection' | 'learning' | 'breakthrough' | 'struggle',
    tags?: string[],
    aboutContactId?: number
  ): Promise<{ reflectionId: number }> {
    const userId = await this.getUserId();

    const response = await fetch(
      `${this.baseUrl}/api/v2.1/reflections`,
      {
        method: 'POST',
        headers: {
          'X-User-ID': userId,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          content,
          entryType,
          tags: tags || [],
          aboutContactId: aboutContactId || null,
        }),
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to add reflection: ${response.statusText}`);
    }

    const data: ApiResponse<{ reflectionId: number }> = await response.json();
    if (!data.data) {
      throw new Error('Invalid response format');
    }

    return data.data;
  }

  /**
   * Update goal progress
   */
  async updateGoalProgress(
    goalId: number,
    progressNotes: string,
    status: 'active' | 'achieved' | 'paused' | 'abandoned'
  ): Promise<void> {
    const userId = await this.getUserId();

    const response = await fetch(
      `${this.baseUrl}/api/v2.1/goals/${goalId}`,
      {
        method: 'PATCH',
        headers: {
          'X-User-ID': userId,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          progressNotes,
          status,
        }),
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to update goal: ${response.statusText}`);
    }
  }

  /**
   * Check if backend is reachable
   */
  async healthCheck(): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseUrl}/health`, {
        method: 'GET',
      });
      return response.ok;
    } catch {
      return false;
    }
  }
}

// Export singleton instance
export const profileAPI = new ProfileAPI();

// Also export class for testing
export { ProfileAPI };
