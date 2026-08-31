/**
 * Socratic Questions Component
 * Display guiding questions to help users think through their message
 */

import React from 'react';
import type { SocraticQuestion } from '@/api/modeAwareLLM';
import './modeDisplay.css';

interface SocraticQuestionsProps {
  questions: SocraticQuestion[];
  conversationContext: string;
  toneReminder: string;
}

export const SocraticQuestions: React.FC<SocraticQuestionsProps> = ({
  questions,
  conversationContext,
  toneReminder,
}) => {
  return (
    <div className="mode-display socratic">
      <div className="mode-header">
        <span className="mode-icon">S</span>
        <div className="mode-info">
          <h3>Socratic Mode</h3>
          <p className="mode-subtitle">Guiding Questions</p>
        </div>
      </div>

      <div className="socratic-context">
        <p className="context-label">Context:</p>
        <p className="context-text">{conversationContext}</p>
      </div>

      <div className="questions-container">
        {questions.map((q, idx) => (
          <div key={idx} className="question-card">
            <div className="question-number">Q{idx + 1}</div>
            <div className="question-content">
              <p className="question-text">{q.question}</p>
              <p className="question-purpose">{q.purpose}</p>
            </div>
          </div>
        ))}
      </div>

      <div className="tone-reminder">
        <strong>Tone Reminder:</strong> {toneReminder}
      </div>

      <div className="socratic-hint">
        <p>
          Take time to reflect on these questions. They'll help you craft a more authentic and
          meaningful message.
        </p>
      </div>
    </div>
  );
};

export default SocraticQuestions;
