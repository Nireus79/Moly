import React, { useEffect, useState } from 'react';
import { useSettingsStore, initializeSettings } from '@/stores/settingsStore';
import { useAuth } from '@/hooks/useAuth';
import { useAboutMe } from '@/hooks/useAboutMe';
import { ChatInterface, AboutMeModal } from './components';
import { Settings } from '@/settings/Settings';
import { chatStarted } from './chatSession';
import './sidebar.css';

// "Skip for now" on the profile form is respected for a week: neither the start-up check nor the late backend answer reopens it.
const ABOUT_ME_DISMISSED_KEY = 'moly_aboutme_dismissed_until';
const aboutMeDismissed = (): boolean => {
  if (chatStarted()) return true; // never over a conversation that has begun
  try {
    return Number(localStorage.getItem(ABOUT_ME_DISMISSED_KEY) || 0) > Date.now();
  } catch {
    return false;
  }
};
const dismissAboutMe = (): void => {
  try {
    localStorage.setItem(ABOUT_ME_DISMISSED_KEY, String(Date.now() + 7 * 24 * 3600 * 1000));
  } catch {
    /* storage unavailable: the form may come back on the next start */
  }
};

export const Sidebar: React.FC = () => {
  const [showSettings, setShowSettings] = useState(false);
  const [showAboutMeModal, setShowAboutMeModal] = useState(false);

  const { loadSettings } = useSettingsStore();
  useAuth(); // loads the saved session and keeps it valid; the chat reads it from the store
  const { hasProfile, isLoading: profileIsLoading, saveProfile } = useAboutMe();

  useEffect(() => {
    const initializeApp = async () => {
      try {
        initializeSettings();
        await loadSettings();
      } catch (err) {
        console.error('[Sidebar] Failed to initialize app:', err);
      }
    };

    initializeApp();
  }, [loadSettings]);

  // Show About Me modal on first load if user has no profile
  useEffect(() => {
    if (!hasProfile && !profileIsLoading && !aboutMeDismissed()) {
      console.info('[Sidebar] ✓ About Me profile missing - showing modal on startup');
      setShowAboutMeModal(true);
    }
  }, [hasProfile, profileIsLoading]);

  const handleOpenSettings = () => {
    setShowSettings(!showSettings);
  };

  return (
    <>
      <ChatInterface onSettingsClick={handleOpenSettings} />
      {showAboutMeModal && (
        <AboutMeModal
          onClose={() => setShowAboutMeModal(false)}
          onSkip={() => { dismissAboutMe(); setShowAboutMeModal(false); }}
          onSave={(prefs) => {
            void saveProfile(prefs); // keeps the local copy and the in-memory profile in step with what the form saved
            setShowAboutMeModal(false);
          }}
        />
      )}
      {showSettings && (
        <div className="sidebar-overlay">
          <Settings onClose={() => setShowSettings(false)} />
        </div>
      )}
    </>
  );
};

export default Sidebar;
