import React, { useState, useEffect } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { getBackendManager } from '@/api/backendManager';
import './metrics-panel.css';

interface MetricsData {
  question_effectiveness?: {
    total_asked: number;
    total_reduced_ambiguity: number;
    total_insights_gained: number;
    total_depth_advanced: number;
    total_principles_clarified: number;
    ambiguity_reduction_rate: number;
    insight_generation_rate: number;
  };
  principle_violations?: {
    total_violations: number;
    critical: number;
    high: number;
    medium: number;
    resolved: number;
    resolution_rate: number;
  };
  principle_breakdown?: Array<{
    principle: string;
    severity: string;
    count: number;
    resolved: number;
  }>;
  approach_comparison?: Array<{
    approach: string;
    total_used: number;
    successful: number;
    success_rate: number;
    avg_depth_advancement: number;
    avg_insight_rate: number;
  }>;
  timestamp?: number;
}

export const MetricsPanel: React.FC = () => {
  const { session } = useAuth();
  const [metrics, setMetrics] = useState<MetricsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['effectiveness']));

  useEffect(() => {
    if (!session) return;

    const fetchMetrics = async () => {
      try {
        const apiBase = getBackendManager().getBackendUrl();
        const response = await fetch(`${apiBase}/api/v2/metrics`, {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${session.sessionId}`,
          },
        });

        if (!response.ok) {
          throw new Error(`Failed to fetch metrics: ${response.statusText}`);
        }

        const data = await response.json();
        setMetrics(data.data?.metrics || data.metrics || {});
        setError(null);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load metrics';
        setError(message);
        console.error('[MetricsPanel] Error:', message);
      } finally {
        setLoading(false);
      }
    };

    fetchMetrics();
  }, [session]);

  const toggleSection = (section: string) => {
    const newExpanded = new Set(expandedSections);
    if (newExpanded.has(section)) {
      newExpanded.delete(section);
    } else {
      newExpanded.add(section);
    }
    setExpandedSections(newExpanded);
  };

  const formatPercentage = (rate: number) => {
    return `${(rate * 100).toFixed(1)}%`;
  };

  const formatApproachName = (approach: string) => {
    return approach.split('_').map(word =>
      word.charAt(0).toUpperCase() + word.slice(1)
    ).join(' ');
  };

  if (loading) {
    return (
      <div className="metrics-panel">
        <div className="metrics-loading">
          <div className="spinner"></div>
          <p>Loading metrics...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="metrics-panel">
        <div className="metrics-error">
          <p>⚠️ {error}</p>
        </div>
      </div>
    );
  }

  if (!metrics) {
    return (
      <div className="metrics-panel">
        <div className="metrics-empty">
          <p>No metrics available yet</p>
          <p className="subtitle">Start using Socratic questions to see your learning progress</p>
        </div>
      </div>
    );
  }

  return (
    <div className="metrics-panel">
      <div className="metrics-header">
        <h3>📊 Learning Metrics</h3>
        <p className="metrics-subtitle">Track your understanding and growth</p>
      </div>

      {/* Question Effectiveness */}
      <div className="metrics-section">
        <button
          className="section-toggle"
          onClick={() => toggleSection('effectiveness')}
        >
          <span className="toggle-icon">
            {expandedSections.has('effectiveness') ? '▼' : '▶'}
          </span>
          <span className="section-title">📈 Question Effectiveness</span>
        </button>

        {expandedSections.has('effectiveness') && metrics.question_effectiveness && (
          <div className="section-content">
            <div className="metric-grid">
              <div className="metric-card">
                <div className="metric-label">Questions Asked</div>
                <div className="metric-value">{metrics.question_effectiveness.total_asked}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Ambiguity Reduced</div>
                <div className="metric-value">
                  {formatPercentage(metrics.question_effectiveness.ambiguity_reduction_rate)}
                </div>
                <div className="metric-detail">
                  {metrics.question_effectiveness.total_reduced_ambiguity} of {metrics.question_effectiveness.total_asked}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Insights Gained</div>
                <div className="metric-value">
                  {formatPercentage(metrics.question_effectiveness.insight_generation_rate)}
                </div>
                <div className="metric-detail">
                  {metrics.question_effectiveness.total_insights_gained} insights
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Depth Advanced</div>
                <div className="metric-value">{metrics.question_effectiveness.total_depth_advanced}</div>
                <div className="metric-detail">questions explored deeper</div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Approach Effectiveness */}
      {metrics.approach_comparison && metrics.approach_comparison.length > 0 && (
        <div className="metrics-section">
          <button
            className="section-toggle"
            onClick={() => toggleSection('approaches')}
          >
            <span className="toggle-icon">
              {expandedSections.has('approaches') ? '▼' : '▶'}
            </span>
            <span className="section-title">🎯 Socratic Approach Effectiveness</span>
          </button>

          {expandedSections.has('approaches') && (
            <div className="section-content">
              <div className="approaches-list">
                {metrics.approach_comparison.map(approach => (
                  <div key={approach.approach} className="approach-item">
                    <div className="approach-header">
                      <span className="approach-name">{formatApproachName(approach.approach)}</span>
                      <span className="approach-rate">
                        {formatPercentage(approach.success_rate)} effective
                      </span>
                    </div>
                    <div className="approach-stats">
                      <span className="stat">Used {approach.total_used}x</span>
                      <span className="stat">Success {approach.successful}/{approach.total_used}</span>
                      <span className="stat">Depth avg {approach.avg_depth_advancement?.toFixed(1) || 0}</span>
                    </div>
                    <div className="progress-bar">
                      <div
                        className="progress-fill"
                        style={{ width: `${approach.success_rate * 100}%` }}
                      ></div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Principle Violations */}
      {metrics.principle_violations && metrics.principle_violations.total_violations > 0 && (
        <div className="metrics-section">
          <button
            className="section-toggle"
            onClick={() => toggleSection('violations')}
          >
            <span className="toggle-icon">
              {expandedSections.has('violations') ? '▼' : '▶'}
            </span>
            <span className="section-title">⚠️ Principle Violations</span>
          </button>

          {expandedSections.has('violations') && (
            <div className="section-content">
              <div className="violation-stats">
                <div className="violation-summary">
                  <div className="summary-item critical">
                    <span className="label">Critical</span>
                    <span className="count">{metrics.principle_violations.critical}</span>
                  </div>
                  <div className="summary-item high">
                    <span className="label">High</span>
                    <span className="count">{metrics.principle_violations.high}</span>
                  </div>
                  <div className="summary-item medium">
                    <span className="label">Medium</span>
                    <span className="count">{metrics.principle_violations.medium}</span>
                  </div>
                </div>
                <div className="resolution-rate">
                  Resolution Rate: {formatPercentage(metrics.principle_violations.resolution_rate)}
                  <br />
                  <span className="detail">{metrics.principle_violations.resolved} of {metrics.principle_violations.total_violations} resolved</span>
                </div>
              </div>

              {metrics.principle_breakdown && metrics.principle_breakdown.length > 0 && (
                <div className="principle-breakdown">
                  <h4>By Principle</h4>
                  {metrics.principle_breakdown.map((item, idx) => (
                    <div key={idx} className="breakdown-item">
                      <span className="principle-name">
                        {item.principle.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')}
                      </span>
                      <span className={`severity-badge ${item.severity}`}>
                        {item.severity}
                      </span>
                      <span className="counts">
                        {item.resolved}/{item.count} resolved
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* No violations */}
      {(!metrics.principle_violations || metrics.principle_violations.total_violations === 0) && (
        <div className="metrics-section success-message">
          <p>✨ No principle violations detected</p>
          <p className="subtitle">Great job maintaining ethical standards!</p>
        </div>
      )}
    </div>
  );
};
