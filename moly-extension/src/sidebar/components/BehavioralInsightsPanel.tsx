import React, { useState, useEffect } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { getBackendManager } from '@/api/backendManager';
import './behavioral-insights.css';

interface ContextProfile {
  style: string;
  confidence: number;
  sampleSize?: number;
}

interface CommunicationProfile {
  dominant_style?: string;
  frequency?: Record<string, unknown>;
  contexts?: Record<string, ContextProfile>;
}

interface UserBehavioralProfile {
  userId: string;
  communicationProfile: CommunicationProfile;
  communicationGoals?: Record<string, number>;
  successMetrics?: Record<string, unknown>;
  emergingPersonality?: string[];
  growthTrajectory?: Record<string, unknown>;
  confidence: number;
  createdAt: number;
  updatedAt: number;
}

export const BehavioralInsightsPanel: React.FC = () => {
  const { session } = useAuth();
  const [profile, setProfile] = useState<UserBehavioralProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!session) return;

    const fetchProfile = async () => {
      try {
        const apiBase = getBackendManager().getBackendUrl();
        const response = await fetch(`${apiBase}/api/v2/metrics`, {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${session.sessionId}`,
          },
        });

        if (!response.ok) {
          throw new Error(`Failed to fetch insights: ${response.statusText}`);
        }

        const data = await response.json();
        // Handle both possible response structures
        const profileData = data.data?.profile || data.profile || data;
        setProfile(profileData);
        setError(null);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load insights';
        setError(message);
        console.error('[BehavioralInsightsPanel] Error:', message);
      } finally {
        setLoading(false);
      }
    };

    fetchProfile();
  }, [session]);

  if (loading) {
    return (
      <div className="behavioral-insights-panel loading">
        <div className="loading-spinner"></div>
        <p>Learning about you...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="behavioral-insights-panel error">
        <p className="error-message">⚠️ {error}</p>
        <p className="error-hint">Try refreshing or chat more for better insights</p>
      </div>
    );
  }

  if (!profile || profile.confidence < 0.5) {
    return (
      <div className="behavioral-insights-panel insufficient-data">
        <p className="info-message">📚 Still learning about you</p>
        <p className="info-hint">Resolve more conflicts in chat to unlock behavioral insights</p>
      </div>
    );
  }

  const communicationStyle = profile.communicationProfile?.dominant_style || 'developing';
  const contexts = profile.communicationProfile?.contexts || {};
  const confidence = Math.round(profile.confidence * 100);
  const growthData = profile.growthTrajectory as Record<string, unknown> || {};
  const successMetrics = profile.successMetrics as Record<string, unknown> || {};

  return (
    <div className="behavioral-insights-panel">
      <div className="insights-container">
        {/* Section 1: Current Communication Style */}
        <div className="insight-section">
          <h3 className="section-title">Your Communication Style</h3>
          <div className="insight-card main-style">
            <div className="style-content">
              <div className="style-name">{communicationStyle}</div>
              <div className="confidence-display">
                <span className="confidence-text">{confidence}% confident</span>
                <div className="confidence-bar">
                  <div className="confidence-fill" style={{ width: `${confidence}%` }}></div>
                </div>
              </div>
              <div className="style-description">
                {getStyleDescription(communicationStyle)}
              </div>
            </div>
            <div className="card-actions">
              <button className="action-btn accurate" title="This is accurate">✓</button>
              <button className="action-btn incorrect" title="Not quite right">✗</button>
            </div>
          </div>
        </div>

        {/* Section 2: Context Profiles */}
        {Object.keys(contexts).length > 0 && (
          <div className="insight-section">
            <h3 className="section-title">How You Adapt to Different Contexts</h3>
            <div className="context-profiles">
              {Object.entries(contexts).map(([context, data]) => {
                const contextData = data as ContextProfile;
                const contextConfidence = Math.round((contextData.confidence || 0.5) * 100);
                return (
                  <div key={context} className="context-card">
                    <div className="context-icon">{getContextIcon(context)}</div>
                    <div className="context-info">
                      <div className="context-label">{formatContextLabel(context)}</div>
                      <div className="context-style">{contextData.style}</div>
                      <div className="confidence-display">
                        <span className="confidence-text">{contextConfidence}%</span>
                        <div className="confidence-bar small">
                          <div className="confidence-fill" style={{ width: `${contextConfidence}%` }}></div>
                        </div>
                      </div>
                    </div>
                    <button className="action-btn small" title="Approve">✓</button>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Section 3: Growth & Evolution */}
        {growthData.trend && (
          <div className="insight-section">
            <h3 className="section-title">Your Evolution</h3>
            <div className="insight-card growth">
              <div className="growth-content">
                <div className="growth-trend">
                  <span className="trend-badge">{growthData.trend}</span>
                  <span className="trend-label">
                    {growthData.trend === 'growing'
                      ? `Shifting from ${growthData.early_style} to ${growthData.recent_style}`
                      : growthData.trend === 'stable'
                      ? `Consistently ${growthData.recent_style}`
                      : 'Exploring different styles'}
                  </span>
                </div>
                {growthData.confidence_trend && (
                  <div className="confidence-trend">
                    Confidence improving over time
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Section 4: Key Insights */}
        <div className="insight-section">
          <h3 className="section-title">Key Insights</h3>
          <div className="insights-list">
            {generateKeyInsights(profile).map((insight, idx) => (
              <div key={idx} className="insight-item">
                <span className="insight-bullet">•</span>
                <span className="insight-text">{insight}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Last Updated */}
        <div className="last-updated">
          Updated {formatDate(profile.updatedAt)}
        </div>
      </div>
    </div>
  );
};

// Helper functions
function getStyleDescription(style: string): string {
  const descriptions: Record<string, string> = {
    formal: 'You communicate in a structured, professional manner with clear reasoning',
    casual: 'You prefer relaxed, natural conversation with flexibility',
    playful: 'You bring humor and lightness to your interactions',
    authentic: 'You value genuine, honest expression in your communication',
    direct: 'You get straight to the point without unnecessary elaboration',
    diplomatic: 'You consider others while expressing your views thoughtfully',
    developing: 'Your style is still being discovered through our conversations',
  };
  return descriptions[style] || 'Your unique communication style is emerging';
}

function getContextIcon(context: string): string {
  const icons: Record<string, string> = {
    work: '🏢',
    home: '🏠',
    social: '👥',
    general: '💭',
  };
  return icons[context] || '•';
}

function formatContextLabel(context: string): string {
  const labels: Record<string, string> = {
    work: 'At Work',
    home: 'At Home',
    social: 'Social Settings',
    general: 'General',
  };
  return labels[context] || context.charAt(0).toUpperCase() + context.slice(1);
}

function generateKeyInsights(profile: UserBehavioralProfile): string[] {
  const insights: string[] = [];
  const comm = profile.communicationProfile;
  const style = comm.dominant_style || 'developing';
  const confidence = Math.round(profile.confidence * 100);
  const metrics = profile.successMetrics as Record<string, unknown> || {};
  const modRate = metrics.modification_rate as string || '0%';
  const growth = profile.growthTrajectory as Record<string, unknown> || {};

  // Add main style insight
  insights.push(`You typically communicate in a ${style} way (${confidence}% confidence)`);

  // Add consistency insight
  const consistency = metrics.consistency as string;
  if (consistency) {
    insights.push(`You maintain consistency in your approach ${consistency} of the time`);
  }

  // Add flexibility insight
  insights.push(`You change your approach ${modRate} of the time (balanced flexibility)`);

  // Add evolution insight
  if (growth.trend === 'growing') {
    insights.push(`You've evolved from ${growth.early_style} toward ${growth.recent_style}`);
  } else if (growth.trend === 'stable') {
    insights.push(`You've maintained a stable ${growth.recent_style} style over time`);
  }

  // Add context insight
  const contexts = comm.contexts || {};
  const contextCount = Object.keys(contexts).length;
  if (contextCount > 1) {
    insights.push(`You show distinct communication patterns across ${contextCount} different contexts`);
  }

  return insights.slice(0, 5); // Return top 5 insights
}

function formatDate(timestamp: number): string {
  if (!timestamp) return 'just now';
  const date = new Date(timestamp * 1000);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 60) return `${diffMins} minute${diffMins !== 1 ? 's' : ''} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
  if (diffDays < 7) return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;

  return date.toLocaleDateString();
}
