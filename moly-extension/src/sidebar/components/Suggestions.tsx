import React, { useState, useEffect } from 'react';

interface SuggestionsProps {
  suggestions: string[];
  loading?: boolean;
  onCopy: (text: string) => void;
  onRefine?: (suggestion: string) => void;
  error?: string;
}

export const Suggestions: React.FC<SuggestionsProps> = ({
  suggestions,
  loading = false,
  onCopy,
  onRefine,
  error,
}) => {
  const [copiedId, setCopiedId] = useState<number | null>(null);

  useEffect(() => {
    if (copiedId !== null) {
      const timer = setTimeout(() => setCopiedId(null), 2000);
      return () => clearTimeout(timer);
    }
  }, [copiedId]);

  const handleCopy = (text: string, index: number) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedId(index);
      onCopy(text);
    });
  };

  if (loading) {
    return (
      <div className="suggestions-container loading">
        <div className="loading-state">
          <div className="spinner"></div>
          <p>Generating suggestions...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="suggestions-container error">
        <div className="error-state">
          <p className="error-message">{error}</p>
        </div>
      </div>
    );
  }

  if (suggestions.length === 0) {
    return (
      <div className="suggestions-container empty">
        <div className="empty-state">
          <p>No suggestions yet</p>
          <p className="hint">Start a conversation to get suggestions</p>
        </div>
      </div>
    );
  }

  return (
    <div className="suggestions-container">
      <div className="suggestions-header">
        <h3>Suggested Responses</h3>
      </div>

      <div className="suggestions-list">
        {suggestions.map((suggestion, index) => (
          <div key={index} className="suggestion-item">
            <div className="suggestion-number">
              {index + 1}
            </div>

            <div className="suggestion-content">
              <p className="suggestion-text">{suggestion}</p>
            </div>

            <div className="suggestion-actions">
              <button
                onClick={() => handleCopy(suggestion, index)}
                className={`btn-copy ${copiedId === index ? 'copied' : ''}`}
                title="Copy to clipboard"
              >
                {copiedId === index ? 'Copied!' : 'Copy'}
              </button>

              {onRefine && (
                <button
                  onClick={() => onRefine(suggestion)}
                  className="btn-refine"
                  title="Refine this suggestion"
                >
                  Refine
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default Suggestions;
