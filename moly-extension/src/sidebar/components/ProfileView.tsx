import React, { useState, useEffect } from 'react';

interface AboutMeProfile {
  communicationStyle?: string;
  tonePreference?: string;
  coreValues?: string[];
  preferences?: Record<string, any>;
  confidence?: number;
  extractedFromCount?: number;
}

interface ContactProfile {
  name: string;
  relationshipType: string;
  timesMentioned: number;
  communicationPatterns?: Array<{
    frequency: string;
    toneObserved: string;
    mainTopics?: string[];
  }>;
}

interface PatternProfile {
  pattern: string;
  category: string;
  observationCount: number;
  confidence: number;
  isActive: boolean;
  isGrowthArea: boolean;
}

interface GoalProfile {
  goal: string;
  category: string;
  status: string;
  progressNotes?: string;
  confidence: number;
}

interface LearningProfile {
  learningType: string;
  learningKey: string;
  learningValue: string;
  confidence: number;
  isConfirmed: boolean;
  isRejected: boolean;
}

interface ReflectionEntry {
  content: string;
  tags?: string[];
  entryType: string;
}

interface UserProfile {
  userId: string;
  aboutMe?: AboutMeProfile;
  contacts: ContactProfile[];
  patterns: PatternProfile[];
  goals: GoalProfile[];
  learnings: LearningProfile[];
  reflections: ReflectionEntry[];
  confidenceScores?: Record<string, any>;
  lastUpdated?: number;
}

interface ProfileViewProps {
  isOpen: boolean;
  onClose: () => void;
  userId?: string;
  backendUrl?: string;
}

const ConfidenceBadge: React.FC<{ confidence: number }> = ({ confidence }) => {
  const getColor = (conf: number) => {
    if (conf >= 0.9) return '#10b981'; // green
    if (conf >= 0.6) return '#f59e0b'; // amber
    return '#ef4444'; // red
  };

  const getLabel = (conf: number) => {
    if (conf >= 0.9) return 'High';
    if (conf >= 0.6) return 'Medium';
    return 'Low';
  };

  return (
    <span
      style={{
        display: 'inline-block',
        background: getColor(confidence),
        color: 'white',
        padding: '2px 8px',
        borderRadius: '12px',
        fontSize: '11px',
        fontWeight: 'bold',
      }}
    >
      {getLabel(confidence)} ({(confidence * 100).toFixed(0)}%)
    </span>
  );
};

export const ProfileView: React.FC<ProfileViewProps> = ({
  isOpen,
  onClose,
  userId = '',
  backendUrl = 'http://localhost:8080',
}) => {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'patterns' | 'contacts' | 'goals' | 'learnings' | 'reflections'>('overview');

  useEffect(() => {
    if (isOpen && userId) {
      loadProfile();
    }
  }, [isOpen, userId]);

  const loadProfile = async () => {
    try {
      setLoading(true);
      setError(null);

      // Get user ID from storage if not provided
      let userToFetch = userId;
      if (!userToFetch) {
        const result = await chrome.storage.local.get('userId');
        userToFetch = result.userId || 'unknown';
      }

      const response = await fetch(`${backendUrl}/api/v2.1/profile`, {
        headers: {
          'X-User-ID': userToFetch,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to load profile: ${response.statusText}`);
      }

      const data = await response.json();
      setProfile(data.data);
    } catch (err) {
      console.error('[ProfileView] Failed to load profile:', err);
      setError(err instanceof Error ? err.message : 'Failed to load profile');
    } finally {
      setLoading(false);
    }
  };

  const handleConfirmLearning = async (learningId: number) => {
    try {
      let userToUse = userId;
      if (!userToUse) {
        const result = await chrome.storage.local.get('userId');
        userToUse = result.userId || 'unknown';
      }

      await fetch(`${backendUrl}/api/v2.1/learnings/${learningId}/confirm`, {
        method: 'POST',
        headers: {
          'X-User-ID': userToUse,
          'Content-Type': 'application/json',
        },
      });

      // Reload profile
      await loadProfile();
    } catch (err) {
      console.error('[ProfileView] Failed to confirm learning:', err);
    }
  };

  const handleRejectLearning = async (learningId: number) => {
    try {
      let userToUse = userId;
      if (!userToUse) {
        const result = await chrome.storage.local.get('userId');
        userToUse = result.userId || 'unknown';
      }

      await fetch(`${backendUrl}/api/v2.1/learnings/${learningId}/reject`, {
        method: 'POST',
        headers: {
          'X-User-ID': userToUse,
          'Content-Type': 'application/json',
        },
      });

      // Reload profile
      await loadProfile();
    } catch (err) {
      console.error('[ProfileView] Failed to reject learning:', err);
    }
  };

  if (!isOpen) return null;

  const containerStyle: React.CSSProperties = {
    position: 'fixed',
    inset: 0,
    background: 'rgba(0, 0, 0, 0.7)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 10001,
    pointerEvents: 'auto',
  };

  const modalStyle: React.CSSProperties = {
    background: 'white',
    borderRadius: '8px',
    padding: '20px',
    maxWidth: '600px',
    width: '90%',
    maxHeight: '85vh',
    overflow: 'auto',
    boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)',
  };

  const headerStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
    borderBottom: '1px solid #e5e7eb',
    paddingBottom: '12px',
  };

  const tabsStyle: React.CSSProperties = {
    display: 'flex',
    gap: '8px',
    marginBottom: '16px',
    borderBottom: '1px solid #e5e7eb',
    overflow: 'auto',
  };

  const tabButtonStyle = (active: boolean): React.CSSProperties => ({
    padding: '8px 12px',
    border: 'none',
    background: 'transparent',
    borderBottom: active ? '2px solid #3b82f6' : '2px solid transparent',
    color: active ? '#3b82f6' : '#666',
    cursor: 'pointer',
    fontSize: '13px',
    fontWeight: active ? '600' : '500',
    whiteSpace: 'nowrap',
  });

  return (
    <div style={containerStyle} onClick={onClose}>
      <div style={modalStyle} onClick={(e) => e.stopPropagation()}>
        <div style={headerStyle}>
          <h2 style={{ margin: 0, fontSize: '18px', fontWeight: 'bold' }}>
            Your Profile
          </h2>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '24px',
              cursor: 'pointer',
              color: '#999',
              padding: '0',
            }}
          >
            ✕
          </button>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '40px 20px', color: '#999' }}>
            Loading your profile...
          </div>
        ) : error ? (
          <div style={{ padding: '16px', background: '#fee2e2', borderRadius: '4px', color: '#991b1b' }}>
            {error}
          </div>
        ) : !profile ? (
          <div style={{ padding: '16px', color: '#666' }}>
            No profile data available yet. Start chatting with Moly to build your profile.
          </div>
        ) : (
          <>
            <div style={tabsStyle}>
              <button style={tabButtonStyle(activeTab === 'overview')} onClick={() => setActiveTab('overview')}>
                Overview
              </button>
              <button style={tabButtonStyle(activeTab === 'patterns')} onClick={() => setActiveTab('patterns')}>
                Patterns ({profile.patterns.length})
              </button>
              <button style={tabButtonStyle(activeTab === 'contacts')} onClick={() => setActiveTab('contacts')}>
                Contacts ({profile.contacts.length})
              </button>
              <button style={tabButtonStyle(activeTab === 'goals')} onClick={() => setActiveTab('goals')}>
                Goals ({profile.goals.length})
              </button>
              <button style={tabButtonStyle(activeTab === 'learnings')} onClick={() => setActiveTab('learnings')}>
                Learnings ({profile.learnings.length})
              </button>
              <button style={tabButtonStyle(activeTab === 'reflections')} onClick={() => setActiveTab('reflections')}>
                Reflections ({profile.reflections.length})
              </button>
            </div>

            {/* Overview Tab */}
            {activeTab === 'overview' && (
              <div>
                {profile.aboutMe && (
                  <div style={{ marginBottom: '20px' }}>
                    <h3 style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '10px' }}>
                      About You
                    </h3>
                    {profile.aboutMe.communicationStyle && (
                      <p style={{ margin: '8px 0', fontSize: '13px' }}>
                        <strong>Communication Style:</strong> {profile.aboutMe.communicationStyle}
                      </p>
                    )}
                    {profile.aboutMe.tonePreference && (
                      <p style={{ margin: '8px 0', fontSize: '13px' }}>
                        <strong>Tone Preference:</strong> {profile.aboutMe.tonePreference}
                      </p>
                    )}
                    {profile.aboutMe.coreValues && profile.aboutMe.coreValues.length > 0 && (
                      <p style={{ margin: '8px 0', fontSize: '13px' }}>
                        <strong>Core Values:</strong> {profile.aboutMe.coreValues.join(', ')}
                      </p>
                    )}
                    {profile.aboutMe.confidence !== undefined && (
                      <div style={{ marginTop: '8px' }}>
                        <ConfidenceBadge confidence={profile.aboutMe.confidence} />
                      </div>
                    )}
                  </div>
                )}

                {profile.confidenceScores && (
                  <div style={{ background: '#f3f4f6', padding: '12px', borderRadius: '4px' }}>
                    <h4 style={{ margin: '0 0 8px 0', fontSize: '13px', fontWeight: 'bold' }}>
                      Profile Confidence
                    </h4>
                    {Object.entries(profile.confidenceScores).map(([category, stats]: [string, any]) => (
                      <div key={category} style={{ fontSize: '12px', marginBottom: '6px' }}>
                        <strong>{category}:</strong> {(stats.average * 100).toFixed(0)}% ({stats.count} items)
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Patterns Tab */}
            {activeTab === 'patterns' && (
              <div>
                {profile.patterns.length === 0 ? (
                  <p style={{ color: '#999', fontSize: '13px' }}>No patterns observed yet.</p>
                ) : (
                  <div>
                    {profile.patterns.map((pattern, idx) => (
                      <div key={idx} style={{ marginBottom: '12px', padding: '10px', background: '#f9fafb', borderRadius: '4px' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', marginBottom: '6px' }}>
                          <strong style={{ fontSize: '13px' }}>{pattern.pattern}</strong>
                          <ConfidenceBadge confidence={pattern.confidence} />
                        </div>
                        <p style={{ margin: '4px 0', fontSize: '12px', color: '#666' }}>
                          Category: {pattern.category} • Observed {pattern.observationCount} times
                        </p>
                        {pattern.isGrowthArea && (
                          <span style={{ display: 'inline-block', background: '#fef08a', color: '#92400e', padding: '2px 8px', borderRadius: '12px', fontSize: '11px', marginTop: '4px' }}>
                            Growth Area
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Contacts Tab */}
            {activeTab === 'contacts' && (
              <div>
                {profile.contacts.length === 0 ? (
                  <p style={{ color: '#999', fontSize: '13px' }}>No contacts tracked yet.</p>
                ) : (
                  <div>
                    {profile.contacts.map((contact, idx) => (
                      <div key={idx} style={{ marginBottom: '12px', padding: '10px', background: '#f9fafb', borderRadius: '4px' }}>
                        <strong style={{ fontSize: '13px' }}>{contact.name}</strong>
                        <p style={{ margin: '4px 0', fontSize: '12px', color: '#666' }}>
                          {contact.relationshipType} • Mentioned {contact.timesMentioned} times
                        </p>
                        {contact.communicationPatterns && contact.communicationPatterns.length > 0 && (
                          <div style={{ fontSize: '12px', color: '#666', marginTop: '6px' }}>
                            <strong>Patterns:</strong> {contact.communicationPatterns.map(p => p.frequency).join(', ')}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Goals Tab */}
            {activeTab === 'goals' && (
              <div>
                {profile.goals.length === 0 ? (
                  <p style={{ color: '#999', fontSize: '13px' }}>No goals set yet.</p>
                ) : (
                  <div>
                    {profile.goals.map((goal, idx) => (
                      <div key={idx} style={{ marginBottom: '12px', padding: '10px', background: '#f9fafb', borderRadius: '4px' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', marginBottom: '6px' }}>
                          <strong style={{ fontSize: '13px' }}>{goal.goal}</strong>
                          <span style={{ background: '#dbeafe', color: '#1e40af', padding: '2px 8px', borderRadius: '12px', fontSize: '11px' }}>
                            {goal.status}
                          </span>
                        </div>
                        <p style={{ margin: '4px 0', fontSize: '12px', color: '#666' }}>
                          {goal.category}
                        </p>
                        {goal.progressNotes && (
                          <p style={{ margin: '6px 0', fontSize: '12px', fontStyle: 'italic', color: '#555' }}>
                            "{goal.progressNotes}"
                          </p>
                        )}
                        <ConfidenceBadge confidence={goal.confidence} />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Learnings Tab */}
            {activeTab === 'learnings' && (
              <div>
                {profile.learnings.length === 0 ? (
                  <p style={{ color: '#999', fontSize: '13px' }}>No learnings yet.</p>
                ) : (
                  <div>
                    {profile.learnings.map((learning, idx) => (
                      <div key={idx} style={{ marginBottom: '12px', padding: '10px', background: learning.isRejected ? '#fee2e2' : learning.isConfirmed ? '#dcfce7' : '#f9fafb', borderRadius: '4px' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', marginBottom: '6px' }}>
                          <strong style={{ fontSize: '13px' }}>{learning.learningKey}</strong>
                          <ConfidenceBadge confidence={learning.confidence} />
                        </div>
                        <p style={{ margin: '4px 0', fontSize: '12px', color: '#666' }}>
                          {learning.learningValue}
                        </p>
                        <div style={{ marginTop: '8px', display: 'flex', gap: '8px' }}>
                          {!learning.isConfirmed && !learning.isRejected && (
                            <>
                              <button
                                onClick={() => handleConfirmLearning(idx)}
                                style={{
                                  padding: '4px 12px',
                                  background: '#10b981',
                                  color: 'white',
                                  border: 'none',
                                  borderRadius: '4px',
                                  fontSize: '12px',
                                  cursor: 'pointer',
                                }}
                              >
                                Confirm
                              </button>
                              <button
                                onClick={() => handleRejectLearning(idx)}
                                style={{
                                  padding: '4px 12px',
                                  background: '#ef4444',
                                  color: 'white',
                                  border: 'none',
                                  borderRadius: '4px',
                                  fontSize: '12px',
                                  cursor: 'pointer',
                                }}
                              >
                                Reject
                              </button>
                            </>
                          )}
                          {learning.isConfirmed && (
                            <span style={{ fontSize: '11px', color: '#10b981', fontWeight: 'bold' }}>✓ Confirmed</span>
                          )}
                          {learning.isRejected && (
                            <span style={{ fontSize: '11px', color: '#ef4444', fontWeight: 'bold' }}>✗ Rejected</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Reflections Tab */}
            {activeTab === 'reflections' && (
              <div>
                {profile.reflections.length === 0 ? (
                  <p style={{ color: '#999', fontSize: '13px' }}>No reflections yet.</p>
                ) : (
                  <div>
                    {profile.reflections.map((reflection, idx) => (
                      <div key={idx} style={{ marginBottom: '12px', padding: '10px', background: '#f9fafb', borderRadius: '4px' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', marginBottom: '6px' }}>
                          <span style={{ background: '#e0e7ff', color: '#3730a3', padding: '2px 8px', borderRadius: '12px', fontSize: '11px' }}>
                            {reflection.entryType}
                          </span>
                        </div>
                        <p style={{ margin: '6px 0', fontSize: '13px', lineHeight: '1.5' }}>
                          "{reflection.content}"
                        </p>
                        {reflection.tags && reflection.tags.length > 0 && (
                          <div style={{ marginTop: '8px', display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
                            {reflection.tags.map((tag, tagIdx) => (
                              <span key={tagIdx} style={{ background: '#e5e7eb', color: '#374151', padding: '2px 6px', borderRadius: '4px', fontSize: '11px' }}>
                                #{tag}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default ProfileView;
