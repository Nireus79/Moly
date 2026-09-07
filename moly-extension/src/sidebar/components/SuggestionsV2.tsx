/**
 * Enhanced Suggestions Component with V2 Agent Status
 * Displays suggestions and shows whether they came from V2 agents or fallback
 */

import React, { useState } from 'react';
import './suggestions-v2.css';

interface SuggestionsV2Props {
  suggestions: string[];
  onCopy?: (suggestion: string, index: number) => void;
  isLoading?: boolean;
  error?: string | null;
  usingV2?: boolean;
  v2Fallback?: boolean;
  processingTimeMs?: number;
  provider?: string;
}

export const SuggestionsV2: React.FC<SuggestionsV2Props> = ({
  suggestions,
  onCopy,
  isLoading,
  error,
  usingV2 = false,
  v2Fallback = false,
  processingTimeMs = 0,
  provider = 'Unknown',
}) => {
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);

  const handleCopy = async (suggestion: string, index: number) => {
    try {
      await navigator.clipboard.writeText(suggestion);
      setCopiedIndex(index);

      // Reset "copied" indicator after 2 seconds
      setTimeout(() => setCopiedIndex(null), 2000);

      if (onCopy) {
        onCopy(suggestion, index);
      }
    } catch (error) {
      console.error('[SuggestionsV2] Failed to copy:', error);
    }
  };

  if (isLoading) {
    return (
      <div className="suggestions-container loading">
        <div className="suggestions-header">
          <h3>Generating suggestions...</h3>
        </div>
        <div className="suggestions-loader">
          <div className="spinner"></div>
          <p>Moly is thinking...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="suggestions-container error">
        <div className="suggestions-header">
          <h3>Error generating suggestions</h3>
        </div>
        <p className="error-message">{error}</p>
        <p className="error-hint">Please try again or check your settings.</p>
      </div>
    );
  }

  if (!suggestions || suggestions.length === 0) {
    return (
      <div className="suggestions-container empty">
        <p>No suggestions yet. Send a message to get started!</p>
      </div>
    );
  }

  return (
    <div className="suggestions-container">
      <div className="suggestions-header">
        <div className="header-content">
          <h3>Response Suggestions</h3>
          <div className="badge-row">
            {usingV2 && (
              <span className="badge v2-badge">
                <span className="badge-icon">🤖</span>
                V2 Agent
              </span>
            )}
            {v2Fallback && (
              <span className="badge fallback-badge">
                <span className="badge-icon">⚡</span>
                Fallback Mode
              </span>
            )}
            {!usingV2 && !v2Fallback && (
              <span className="badge v1-badge">
                <span className="badge-icon">✨</span>
                Classic Mode
              </span>
            )}
            {processingTimeMs > 0 && (
              <span className="badge time-badge">
                {processingTimeMs}ms
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="suggestions-list">
        {suggestions.map((suggestion, index) => (
          <div key={index} className="suggestion-item">
            <div className="suggestion-number">{index + 1}</div>
            <div className="suggestion-content">
              <p className="suggestion-text">{suggestion}</p>
            </div>
            <button
              className={`copy-button ${copiedIndex === index ? 'copied' : ''}`}
              onClick={() => handleCopy(suggestion, index)}
              title="Copy to clipboard"
              aria-label={`Copy suggestion ${index + 1}`}
            >
              {copiedIndex === index ? (
                <>
                  <span className="copy-icon">✓</span>
                  <span className="copy-label">Copied!</span>
                </>
              ) : (
                <>
                  <span className="copy-icon">📋</span>
                  <span className="copy-label">Copy</span>
                </>
              )}
            </button>
          </div>
        ))}
      </div>

      <div className="suggestions-footer">
        <p className="provider-info">
          Powered by {usingV2 ? 'V2 Agents' : provider}
          {processingTimeMs > 0 && ` • Generated in ${processingTimeMs}ms`}
        </p>
      </div>
    </div>
  );
};

export default SuggestionsV2;
