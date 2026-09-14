/**
 * useAboutMe - Hook for managing user's About Me profile
 * Syncs with backend and falls back to localStorage
 */

import { useState, useEffect } from 'react';
import type { AboutMeProfile } from '@/types';
import { useAuth } from '@/hooks/useAuth';
import { getBackendManager } from '@/api/backendManager';

const STORAGE_KEY = 'moly_about_me_profile';

export function useAboutMe() {
  const { session } = useAuth();
  const [profile, setProfile] = useState<AboutMeProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Load from backend on mount or when session changes
  useEffect(() => {
    const loadProfile = async () => {
      setIsLoading(true);
      try {
        // Try to fetch from backend first
        if (session) {
          const apiBase = getBackendManager().getBackendUrl();
          const response = await fetch(`${apiBase}/api/v2/about-me`, {
            method: 'GET',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${session.sessionId}`,
            },
          });

          if (response.ok) {
            const data = await response.json();
            if (data.profile) {
              setProfile(data.profile);
              // Cache to localStorage
              try {
                localStorage.setItem(STORAGE_KEY, JSON.stringify(data.profile));
              } catch (e) {
                console.warn('[useAboutMe] Failed to cache profile:', e);
              }
              setIsLoading(false);
              return;
            }
          }
        }

        // Fallback to localStorage if backend not available or session missing
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
          const parsed = JSON.parse(stored);
          setProfile(parsed);
        }
      } catch (error) {
        console.error('[useAboutMe] Failed to load profile:', error);
        // Still try localStorage as final fallback
        try {
          const stored = localStorage.getItem(STORAGE_KEY);
          if (stored) {
            const parsed = JSON.parse(stored);
            setProfile(parsed);
          }
        } catch (e) {
          console.error('[useAboutMe] Failed to load from storage:', e);
        }
      } finally {
        setIsLoading(false);
      }
    };

    loadProfile();
  }, [session]);

  const saveProfile = async (newProfile: AboutMeProfile) => {
    try {
      // Save to localStorage first
      localStorage.setItem(STORAGE_KEY, JSON.stringify(newProfile));
      setProfile(newProfile);
      console.info('[useAboutMe] Profile saved locally:', newProfile);

      // Save to backend if session exists
      if (session) {
        const apiBase = getBackendManager().getBackendUrl();
        const response = await fetch(`${apiBase}/api/v2/about-me`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${session.sessionId}`,
          },
          body: JSON.stringify(newProfile),
        });

        if (!response.ok) {
          console.error('[useAboutMe] Failed to save to backend:', response.statusText);
        } else {
          console.info('[useAboutMe] Profile saved to backend');
        }
      }
    } catch (error) {
      console.error('[useAboutMe] Failed to save profile:', error);
    }
  };

  const clearProfile = () => {
    try {
      localStorage.removeItem(STORAGE_KEY);
      setProfile(null);
    } catch (error) {
      console.error('[useAboutMe] Failed to clear profile:', error);
    }
  };

  return {
    profile,
    isLoading,
    saveProfile,
    clearProfile,
    hasProfile: !!profile && (
      profile.communicationStyle ||
      profile.coreValues?.length > 0 ||
      profile.goals?.length > 0 ||
      profile.patterns?.length > 0
    ),
  };
}
