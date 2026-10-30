import React, { useState } from 'react';
import { profileAPI } from '@/api/profileAPI';
import { Icon } from '@/components/Icon';
import './about-me.css';

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

      // Save to backend; the form closes only when the server took the profile (found by a tester: it closed on a refusal)
      const saved = await profileAPI.saveAboutMe({
        communicationStyle: communicationStyle,
        coreValues: prefs.coreValues,
        tonePreference: tonePreference,
        preferences,
      });
      if (!saved) {
        setError('Your choices could not be saved. Try again, or skip and set them later in Settings.');
        return;
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
    <div className="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="aboutme-title">
      <div className="modal-sheet">
        <h2 id="aboutme-title">Tell me about yourself</h2>
        <p className="modal-lead">Help me understand your communication style and preferences to give better advice. All of it is optional.</p>

        {error && <div className="notice danger"><Icon name="alert" /><span>{error}</span></div>}

        <div className="form-group">
          <label className="form-label" htmlFor="aboutme-style">How would you describe your communication style?</label>
          <textarea id="aboutme-style" className="textarea" rows={3} value={communicationStyle} onChange={(e) => setCommunicationStyle(e.target.value)} placeholder="For example: direct and honest, or thoughtful and careful" />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="aboutme-values">What are your core values?</label>
          <input id="aboutme-values" className="input" type="text" value={coreValues} onChange={(e) => setCoreValues(e.target.value)} placeholder="For example: honesty, respect, growth" />
          <p className="form-hint">Separate them with commas.</p>
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="aboutme-tone">Preferred tone for feedback</label>
          <select id="aboutme-tone" className="select" value={tonePreference} onChange={(e) => setTonePreference(e.target.value)}>
            <option value="friendly">Friendly and supportive</option>
            <option value="direct">Direct and honest</option>
            <option value="analytical">Analytical and logical</option>
            <option value="humorous">Humorous and witty</option>
            <option value="balanced">Balanced mix</option>
          </select>
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="aboutme-notes">Anything else I should know?</label>
          <textarea id="aboutme-notes" className="textarea" rows={2} value={preferences} onChange={(e) => setPreferences(e.target.value)} placeholder="Optional" />
        </div>

        <div className="modal-actions">
          <button className="btn btn-secondary" onClick={handleSkip} disabled={saving}>Skip for now</button>
          <button className="btn btn-primary" onClick={handleSave} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
        </div>
        <p className="modal-footnote">You can change this at any time in Settings.</p>
      </div>
    </div>
  );
};
