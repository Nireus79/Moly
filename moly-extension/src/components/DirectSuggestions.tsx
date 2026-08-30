/**
 * Direct Suggestions Component
 * Display ready-to-use message suggestions
 */

import React from 'react';
import type { MessageSuggestion } from '@/types';
import './modeDisplay.css';

interface DirectSuggestionsProps {
  suggestions: MessageSuggestion[];
  onCopyMessage?: (text: string) => void;
}

export const DirectSuggestions: React.FC<DirectSuggestionsProps> = ({ suggestions, onCopyMessage }) => {
  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    onCopyMessage?.(text);
  };

  return (
    <div className="mode-display direct">
      <div className="mode-header">
        <span className="mode-icon">⚡</span>
        <div className="mode-info">
          <h3>Direct Mode</h3>
          <p className="mode-subtitle">Message Suggestions</p>
        </div>
      </div>

      <div className="suggestions-container">
        {suggestions.length === 0 ? (
          <div className="empty-state">
            <p>No suggestions generated yet. Try sending a message!</p>
          </div>
        ) : (
          suggestions.map((suggestion, idx) => (
            <div key={suggestion.id} className="direct-suggestion-card">
              <div className="suggestion-header">
                <span className="suggestion-number">Option {idx + 1}</span>
                <div className="suggestion-badges">
                  <span className="badge tone">{suggestion.tone}</span>
                  <span className="badge confidence">
                    {Math.round(suggestion.confidence * 100)}% match
                  </span>
                </div>
              </div>

              <div className="suggestion-text">{suggestion.text}</div>

              <div className="suggestion-footer">
                <p className="suggestion-reasoning">{suggestion.reasoning}</p>
                <button
                  className="copy-suggestion-btn"
                  onClick={() => handleCopy(suggestion.text)}
                  title="Copy to clipboard"
                >
                  📋 Copy
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      <div className="direct-hint">
        <p>Pick the suggestion that feels most authentic to you, or use them as inspiration!</p>
      </div>
    </div>
  );
};

export default DirectSuggestions;
