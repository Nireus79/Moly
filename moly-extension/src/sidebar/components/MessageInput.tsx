import React, { useState, useRef } from 'react';

interface MessageInputProps {
  onSend: (text: string) => void;
  onPaste?: () => void;
  disabled?: boolean;
  placeholder?: string;
}

export const MessageInput: React.FC<MessageInputProps> = ({
  onSend,
  onPaste,
  disabled = false,
  placeholder = 'Type or paste a message...',
}) => {
  const [text, setText] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSend = () => {
    if (text.trim()) {
      onSend(text.trim());
      setText('');
      if (textareaRef.current) {
        textareaRef.current.focus();
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSend();
    }
  };

  const handlePaste = async () => {
    try {
      const clipboardText = await navigator.clipboard.readText();
      setText((prev) => (prev ? `${prev}\n${clipboardText}` : clipboardText));
      if (textareaRef.current) {
        textareaRef.current.focus();
      }
    } catch (err) {
      console.warn('Failed to read clipboard:', err);
      if (onPaste) {
        onPaste();
      }
    }
  };

  const handleClear = () => {
    setText('');
    if (textareaRef.current) {
      textareaRef.current.focus();
    }
  };

  const charCount = text.length;
  const maxChars = 5000;

  return (
    <div className="message-input-container">
      <textarea
        ref={textareaRef}
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        disabled={disabled}
        rows={3}
        maxLength={maxChars}
        className="message-textarea"
      />

      <div className="input-footer">
        <div className="char-counter">
          {charCount}/{maxChars}
        </div>

        <div className="input-buttons">
          {onPaste && (
            <button
              onClick={handlePaste}
              disabled={disabled}
              className="btn-secondary"
              title="Paste from clipboard"
            >
              Paste
            </button>
          )}
          <button
            onClick={handleClear}
            disabled={disabled || !text}
            className="btn-secondary"
            title="Clear input"
          >
            Clear
          </button>
          <button
            onClick={handleSend}
            disabled={disabled || !text.trim()}
            className="btn-primary"
            title="Send (Ctrl+Enter)"
          >
            Send
          </button>
        </div>
      </div>

      <div className="input-hint">
        Tip: Ctrl+Enter to send, paste from clipboard, or type freely
      </div>
    </div>
  );
};

export default MessageInput;
