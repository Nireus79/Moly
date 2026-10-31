/**
 * Profile API client: the About Me profile, with Bearer token authentication.
 * A failed save returns false (the form shows the error); a failed read returns null.
 */

import { getBackendManager } from './backendManager';
import { getSessionToken } from '../stores/authStore';

function getBackendUrl(): string {
  return getBackendManager().getBackendUrl();
}

function getAuthToken(): string {
  const t = getSessionToken();
  if (!t) throw new Error('Auth token not available');
  return t;
}

export interface AboutMeProfile {
  communicationStyle?: string;
  tonePreference?: string;
  coreValues?: string[];
  preferences?: string;
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
   * Get AboutMe profile from /api/about-me
   */
  async getAboutMe(): Promise<AboutMeProfile | null> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/about-me`, {
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
   * Save AboutMe profile to /api/about-me
   */
  async saveAboutMe(profile: AboutMeProfile): Promise<boolean> {
    try {
      const response = await fetch(`${getBackendUrl()}/api/about-me`, {
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
