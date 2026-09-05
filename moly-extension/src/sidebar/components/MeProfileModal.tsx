import React, { useState, useEffect } from 'react';

interface MeProfile {
  id: string;
  name: string;
  description: string;
  communicationStyle?: string;
  values?: string;
  goals?: string;
  patterns?: string;
  notes?: string;
}

interface MeProfileModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (profile: MeProfile) => void;
}

export const MeProfileModal: React.FC<MeProfileModalProps> = ({
  isOpen,
  onClose,
  onSave,
}) => {
  const [profile, setProfile] = useState<MeProfile>({
    id: 'me',
    name: 'Me',
    description: '',
    communicationStyle: '',
    values: '',
    goals: '',
    patterns: '',
    notes: '',
  });
  const [saved, setSaved] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (isOpen) {
      loadProfile();
    }
  }, [isOpen]);

  const loadProfile = async () => {
    try {
      setLoading(true);
      const result = await chrome.storage.local.get('meProfile');
      if (result.meProfile) {
        setProfile(result.meProfile);
      }
    } catch (err) {
      console.error('[MeProfileModal] Failed to load profile:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      await chrome.storage.local.set({ meProfile: profile });
      onSave(profile);
      setSaved(true);
      setTimeout(() => {
        setSaved(false);
        onClose();
      }, 800);
    } catch (err) {
      console.error('[MeProfileModal] Failed to save profile:', err);
    }
  };

  if (!isOpen) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.7)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 10001,
        pointerEvents: 'auto',
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: 'white',
          borderRadius: '8px',
          padding: '24px',
          maxWidth: '500px',
          width: '90%',
          maxHeight: '90vh',
          overflow: 'auto',
          boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h2 style={{ marginTop: 0, marginBottom: '16px', fontSize: '18px' }}>
          About Me
        </h2>

        <p style={{ fontSize: '12px', color: '#666', marginBottom: '16px' }}>
          Help Moly understand you better. This helps personalize suggestions for how YOU like to communicate.
        </p>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '20px', color: '#999' }}>
            Loading profile...
          </div>
        ) : (
          <>
            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
                Communication Style
              </label>
              <textarea
                value={profile.communicationStyle || ''}
                onChange={(e) =>
                  setProfile({ ...profile, communicationStyle: e.target.value })
                }
                placeholder="e.g., 'I prefer direct, honest feedback' or 'I'm naturally sarcastic'"
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '12px',
                  fontFamily: 'inherit',
                  boxSizing: 'border-box',
                  minHeight: '60px',
                  resize: 'vertical',
                }}
              />
            </div>

            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
                Values
              </label>
              <textarea
                value={profile.values || ''}
                onChange={(e) => setProfile({ ...profile, values: e.target.value })}
                placeholder="e.g., 'I value authenticity' or 'I prioritize kindness over efficiency'"
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '12px',
                  fontFamily: 'inherit',
                  boxSizing: 'border-box',
                  minHeight: '60px',
                  resize: 'vertical',
                }}
              />
            </div>

            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
                Goals & Interests
              </label>
              <textarea
                value={profile.goals || ''}
                onChange={(e) => setProfile({ ...profile, goals: e.target.value })}
                placeholder="e.g., 'Building stronger relationships' or 'Learning to be more confident'"
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '12px',
                  fontFamily: 'inherit',
                  boxSizing: 'border-box',
                  minHeight: '60px',
                  resize: 'vertical',
                }}
              />
            </div>

            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
                Patterns I Notice
              </label>
              <textarea
                value={profile.patterns || ''}
                onChange={(e) => setProfile({ ...profile, patterns: e.target.value })}
                placeholder="e.g., 'I tend to overthink things' or 'I'm great with technical details but bad with emotions'"
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '12px',
                  fontFamily: 'inherit',
                  boxSizing: 'border-box',
                  minHeight: '60px',
                  resize: 'vertical',
                }}
              />
            </div>

            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
                Notes
              </label>
              <textarea
                value={profile.notes || ''}
                onChange={(e) => setProfile({ ...profile, notes: e.target.value })}
                placeholder="Anything else Moly should know about you..."
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '12px',
                  fontFamily: 'inherit',
                  boxSizing: 'border-box',
                  minHeight: '60px',
                  resize: 'vertical',
                }}
              />
            </div>

            <div style={{ display: 'flex', gap: '8px' }}>
              <button
                onClick={handleSave}
                disabled={saved}
                style={{
                  flex: 1,
                  padding: '10px',
                  background: saved ? '#10b981' : '#6366f1',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: saved ? 'default' : 'pointer',
                  fontSize: '14px',
                  fontWeight: '600',
                }}
              >
                {saved ? '✓ Saved' : 'Save Profile'}
              </button>
              <button
                onClick={onClose}
                style={{
                  flex: 1,
                  padding: '10px',
                  background: '#e5e7eb',
                  color: '#333',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontSize: '14px',
                  fontWeight: '600',
                }}
              >
                Close
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
};
