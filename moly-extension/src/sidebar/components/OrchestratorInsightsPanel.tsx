import React from 'react';
import { OrchestratorInsights } from '../../types/apiTypes';

interface OrchestratorInsightsPanelProps {
  insights: OrchestratorInsights | null;
  loading?: boolean;
}

export const OrchestratorInsightsPanel: React.FC<OrchestratorInsightsPanelProps> = ({
  insights,
  loading = false,
}) => {
  if (loading) {
    return (
      <div className="orchestrator-panel loading">
        <p>Analyzing message...</p>
      </div>
    );
  }

  if (!insights) {
    return (
      <div className="orchestrator-panel empty">
        <p>No orchestrator insights available</p>
      </div>
    );
  }

  return (
    <div className="orchestrator-panel">
      <div className="orchestrator-header">
        <h3>System Insights</h3>
      </div>

      {/* Extraction Confidence */}
      {insights.extractionConfidence !== undefined && (
        <div className="insight-section">
          <h4>Extraction Confidence</h4>
          <div className="confidence-bar">
            <div
              className="confidence-fill"
              style={{
                width: `${insights.extractionConfidence * 100}%`,
              }}
            />
          </div>
          <span className="confidence-value">
            {(insights.extractionConfidence * 100).toFixed(0)}%
          </span>
        </div>
      )}

      {/* Maturity Score */}
      {insights.maturityScore !== undefined && (
        <div className="insight-section">
          <h4>Context Maturity</h4>
          <div className="maturity-display">
            <span className="score">
              {(insights.maturityScore * 100).toFixed(0)}%
            </span>
            <span className="quality">
              {insights.contextQuality || 'evaluating'}
            </span>
          </div>
        </div>
      )}

      {/* Detected Gaps */}
      {insights.detectedGaps && insights.detectedGaps.length > 0 && (
        <div className="insight-section">
          <h4>Detected Gaps ({insights.detectedGaps.length})</h4>
          <ul className="gaps-list">
            {insights.detectedGaps.map((gap, idx) => (
              <li key={idx} className={`gap-${gap.severity}`}>
                <span className="gap-type">{gap.type}</span>
                <span className="gap-severity">{gap.severity}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Detected Conflicts */}
      {insights.detectedConflicts && insights.detectedConflicts.length > 0 && (
        <div className="insight-section">
          <h4>Detected Conflicts ({insights.detectedConflicts.length})</h4>
          <ul className="conflicts-list">
            {insights.detectedConflicts.map((conflict, idx) => (
              <li key={idx} className={`conflict-${conflict.severity}`}>
                <span className="conflict-type">{conflict.type}</span>
                <span className="conflict-severity">{conflict.severity}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Ambiguity Detection */}
      {insights.isAmbiguous && (
        <div className="insight-section ambiguous">
          <h4>⚠️ Message is Ambiguous</h4>
          {insights.ambiguousElements && insights.ambiguousElements.length > 0 && (
            <ul className="ambiguous-elements">
              {insights.ambiguousElements.map((element, idx) => (
                <li key={idx}>{element}</li>
              ))}
            </ul>
          )}
        </div>
      )}

      {/* Principle Violations */}
      {insights.principleViolations && insights.principleViolations.length > 0 && (
        <div className="insight-section violation">
          <h4>⚠️ Principle Violations</h4>
          <ul className="violations-list">
            {insights.principleViolations.map((violation, idx) => (
              <li key={idx}>{violation}</li>
            ))}
          </ul>
        </div>
      )}

      {/* Topic Shifts */}
      {insights.topicShifts && insights.topicShifts.length > 0 && (
        <div className="insight-section">
          <h4>Topic Shifts ({insights.topicShifts.length})</h4>
          <ul className="shifts-list">
            {insights.topicShifts.map((shift, idx) => (
              <li key={idx} className={`shift-${shift.severity}`}>
                <span className="shift-type">{shift.type}</span>
                <span className="shift-severity">{shift.severity}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Socratic Questions */}
      {insights.socraticQuestions && insights.socraticQuestions.length > 0 && (
        <div className="insight-section">
          <h4>Suggested Questions ({insights.socraticQuestions.length})</h4>
          <ul className="questions-list">
            {insights.socraticQuestions.map((question, idx) => (
              <li key={idx}>{question}</li>
            ))}
          </ul>
        </div>
      )}

      <style>{`
        .orchestrator-panel {
          background: #f5f5f5;
          border: 1px solid #ddd;
          border-radius: 4px;
          padding: 12px;
          margin: 8px 0;
          font-size: 12px;
        }

        .orchestrator-panel.empty,
        .orchestrator-panel.loading {
          text-align: center;
          color: #999;
          padding: 16px;
        }

        .orchestrator-header h3 {
          margin: 0 0 12px 0;
          font-size: 13px;
          font-weight: 600;
          color: #333;
        }

        .insight-section {
          margin: 8px 0;
          padding: 8px;
          background: white;
          border-radius: 3px;
          border-left: 3px solid #0066cc;
        }

        .insight-section h4 {
          margin: 0 0 6px 0;
          font-size: 12px;
          font-weight: 600;
          color: #333;
        }

        .insight-section.violation {
          border-left-color: #cc3300;
          background: #fff5f5;
        }

        .insight-section.ambiguous {
          border-left-color: #ff9900;
          background: #fff9f5;
        }

        .confidence-bar {
          background: #e0e0e0;
          border-radius: 2px;
          height: 16px;
          overflow: hidden;
          margin: 4px 0;
        }

        .confidence-fill {
          background: linear-gradient(90deg, #0066cc, #00cc66);
          height: 100%;
          transition: width 0.3s ease;
        }

        .confidence-value {
          font-size: 11px;
          font-weight: 600;
          color: #0066cc;
        }

        .maturity-display {
          display: flex;
          align-items: center;
          gap: 12px;
        }

        .maturity-display .score {
          font-size: 18px;
          font-weight: 700;
          color: #0066cc;
        }

        .maturity-display .quality {
          font-size: 11px;
          color: #666;
          text-transform: capitalize;
        }

        .gaps-list,
        .conflicts-list,
        .shifts-list,
        .violations-list,
        .questions-list,
        .ambiguous-elements {
          list-style: none;
          padding: 0;
          margin: 4px 0;
        }

        .gaps-list li,
        .conflicts-list li,
        .shifts-list li,
        .violations-list li,
        .questions-list li,
        .ambiguous-elements li {
          padding: 4px 6px;
          margin: 2px 0;
          background: #fafafa;
          border-radius: 2px;
          font-size: 11px;
          color: #333;
        }

        .gap-critical,
        .conflict-critical,
        .shift-high {
          border-left: 2px solid #cc3300;
          background: #fff5f5;
        }

        .gap-high,
        .conflict-high,
        .shift-medium {
          border-left: 2px solid #ff9900;
          background: #fff9f5;
        }

        .gap-medium,
        .conflict-medium,
        .shift-low {
          border-left: 2px solid #0066cc;
          background: #f5f9ff;
        }

        .gap-low,
        .conflict-low {
          border-left: 2px solid #00cc66;
          background: #f5fff9;
        }

        .gap-type,
        .conflict-type,
        .shift-type {
          font-weight: 600;
          margin-right: 8px;
        }

        .gap-severity,
        .conflict-severity,
        .shift-severity {
          font-size: 10px;
          text-transform: uppercase;
          font-weight: 600;
          color: #999;
        }

        .questions-list li {
          padding: 6px 8px;
          background: #f0f4ff;
          border-left: 2px solid #0066cc;
          line-height: 1.4;
        }

        .violations-list li,
        .ambiguous-elements li {
          padding: 6px 8px;
          background: #fff5f5;
          border-left: 2px solid #cc3300;
          line-height: 1.4;
        }
      `}</style>
    </div>
  );
};
