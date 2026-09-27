import React, { useState } from 'react';
import { profileAPI } from '@/api/profileAPI';

interface AboutMeModalProps {
  onClose?: () => void;
  onSkip?: () => void;
  onSave?: (preferences: AboutMePreferences) => void;
}

interface AboutMePreferences {
  communicationStyle: string;
  coreValues: string[];
  tonePreference: string;
  preferences: string;
}

export const AboutMeModal: React.FC<AboutMeModalProps> = ({ onClose, onSkip, onSave }) => {
  const [communicationStyle, setCommunicationStyle] = useState('');
  const [coreValues, setCoreValues] = useState('');
  const [tonePreference, setTonePreference] = useState('friendly');
  const [preferences, setPreferences] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSave = async () => {
    try {
      setSaving(true);
      setError('');

      const prefs: AboutMePreferences = {
        communicationStyle,
        coreValues: coreValues.split(',').map(v => v.trim()).filter(v => v),
        tonePreference,
        preferences,
      };

      // Save to backend
      try {
        await profileAPI.saveAboutMe({
          communicationStyle: communicationStyle,
          coreValues: prefs.coreValues,
          tonePreference: tonePreference,
          preferences: { notes: preferences },
        });
        console.log('[AboutMeModal] ✓ Preferences saved');
      } catch (backendErr) {
        console.warn('[AboutMeModal] Backend save failed:', backendErr);
        setError('Failed to save to backend, but will continue');
      }

      onSave?.(prefs);
      onClose?.();
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      setError(`Error: ${errMsg}`);
      console.error('[AboutMeModal] Save error:', err);
    } finally {
      setSaving(false);
    }
  };

  const handleSkip = () => {
    console.log('[AboutMeModal] User skipped setup');
    onSkip?.();
    onClose?.();
  };

  return (
    <div style={{
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      backgroundColor: 'rgba(0, 0, 0, 0.6)',
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      zIndex: 1000,
      padding: '20px',
    }}>
      <div style={{
        backgroundColor: '#1f2937',
        borderRadius: '12px',
        padding: '2rem',
        maxWidth: '500px',
        width: '100%',
        boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.3)',
        maxHeight: '90vh',
        overflowY: 'auto',
      }}>
        <h2 style={{
          margin: '0 0 0.5rem 0',
          fontSize: '24px',
          color: '#fff',
        }}>
          ✨ Tell me about yourself
        </h2>

        <p style={{
          color: '#9ca3af',
          marginBottom: '1.5rem',
          fontSize: '14px',
        }}>
          Help me understand your communication style and preferences to give better advice.
        </p>

        {error && (
          <div style={{
            backgroundColor: '#7f1d1d',
            color: '#fecaca',
            padding: '10px',
            borderRadius: '6px',
            marginBottom: '1rem',
            fontSize: '13px',
          }}>
            {error}
          </div>
        )}

        {/* Communication Style */}
        <div style={{ marginBottom: '1.5rem' }}>
          <label style={{
            display: 'block',
            marginBottom: '6px',
            color: '#e5e7eb',
            fontSize: '14px',
            fontWeight: '500',
          }}>
            How would you describe your communication style?
          </label>
          <textarea
            value={communicationStyle}
            onChange={(e) => setCommunicationStyle(e.target.value)}
            placeholder="e.g., Direct and honest, or thoughtful and careful..."
            style={{
              width: '100%',
              minHeight: '80px',
              padding: '10px',
              backgroundColor: '#111827',
              borderRadius: '6px',
              border: '1px solid #374151',
              color: '#e5e7eb',
              fontFamily: 'inherit',
              fontSize: '13px',
              resize: 'vertical',
            }}
          />
        </div>

        {/* Core Values */}
        <div style={{ marginBottom: '1.5rem' }}>
          <label style={{
            display: 'block',
            marginBottom: '6px',
            color: '#e5e7eb',
            fontSize: '14px',
            fontWeight: '500',
          }}>
            What are your core values? (comma-separated)
          </label>
          <input
            type="text"
            value={coreValues}
            onChange={(e) => setCoreValues(e.target.value)}
            placeholder="e.g., honesty, respect, growth"
            style={{
              width: '100%',
              padding: '10px',
              backgroundColor: '#111827',
              borderRadius: '6px',
              border: '1px solid #374151',
              color: '#e5e7eb',
              fontFamily: 'inherit',
              fontSize: '13px',
              boxSizing: 'border-box',
            }}
          />
        </div>

        {/* Tone Preference */}
        <div style={{ marginBottom: '1.5rem' }}>
          <label style={{
            display: 'block',
            marginBottom: '6px',
            color: '#e5e7eb',
            fontSize: '14px',
            fontWeight: '500',
          }}>
            Preferred tone for feedback
          </label>
          <select
            value={tonePreference}
            onChange={(e) => setTonePreference(e.target.value)}
            style={{
              width: '100%',
              padding: '10px',
              backgroundColor: '#111827',
              borderRadius: '6px',
              border: '1px solid #374151',
              color: '#e5e7eb',
              fontFamily: 'inherit',
              fontSize: '13px',
              boxSizing: 'border-box',
            }}
          >
            <option value="friendly">Friendly & supportive</option>
            <option value="direct">Direct & honest</option>
            <option value="analytical">Analytical & logical</option>
            <option value="balanced">Balanced mix</option>
          </select>
        </div>

        {/* Additional Preferences */}
        <div style={{ marginBottom: '1.5rem' }}>
          <label style={{
            display: 'block',
            marginBottom: '6px',
            color: '#e5e7eb',
            fontSize: '14px',
            fontWeight: '500',
          }}>
            Any other preferences or notes?
          </label>
          <textarea
            value={preferences}
            onChange={(e) => setPreferences(e.target.value)}
            placeholder="Optional: Anything else I should know..."
            style={{
              width: '100%',
              minHeight: '60px',
              padding: '10px',
              backgroundColor: '#111827',
              borderRadius: '6px',
              border: '1px solid #374151',
              color: '#e5e7eb',
              fontFamily: 'inherit',
              fontSize: '13px',
              resize: 'vertical',
            }}
          />
        </div>

        {/* Buttons */}
        <div style={{
          display: 'flex',
          gap: '10px',
          marginTop: '2rem',
        }}>
          <button
            onClick={handleSkip}
            disabled={saving}
            style={{
              flex: 1,
              padding: '10px',
              backgroundColor: 'rgba(255, 255, 255, 0.1)',
              color: '#e5e7eb',
              border: '1px solid rgba(255, 255, 255, 0.2)',
              borderRadius: '6px',
              fontWeight: '500',
              cursor: saving ? 'not-allowed' : 'pointer',
              opacity: saving ? 0.5 : 1,
              fontSize: '14px',
            }}
          >
            Skip for now
          </button>

          <button
            onClick={handleSave}
            disabled={saving}
            style={{
              flex: 1,
              padding: '10px',
              backgroundColor: '#667eea',
              color: 'white',
              border: 'none',
              borderRadius: '6px',
              fontWeight: '500',
              cursor: saving ? 'not-allowed' : 'pointer',
              opacity: saving ? 0.7 : 1,
              fontSize: '14px',
            }}
          >
            {saving ? '💾 Saving...' : '✓ Save'}
          </button>
        </div>

        <p style={{
          marginTop: '1rem',
          fontSize: '12px',
          color: '#6b7280',
          textAlign: 'center',
        }}>
          You can update this anytime in Settings
        </p>
      </div>
    </div>
  );
};
