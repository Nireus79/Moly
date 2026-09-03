import React, { useState } from 'react';
import './Sidebar.css';

export default function Sidebar({ selectedModel, provider, onModelChange }) {
  const [message, setMessage] = useState('');
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(false);
  const [context, setContext] = useState('friendly');
  const [mode, setMode] = useState('direct');

  async function handleSendMessage() {
    if (!message.trim()) return;

    setLoading(true);
    const userMsg = message;
    setMessage('');
    setMessages(prev => [...prev, { type: 'user', text: userMsg }]);

    try {
      const response = await fetch('http://127.0.0.1:11435/api/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: selectedModel,
          prompt: userMsg,
          stream: false
        })
      });

      const data = await response.json();
      if (data.response) {
        setMessages(prev => [...prev, { type: 'assistant', text: data.response }]);
      }
    } catch (error) {
      setMessages(prev => [...prev, { type: 'error', text: `Error: ${error.message}` }]);
    } finally {
      setLoading(false);
    }
  }

  function handlePasteMessage() {
    navigator.clipboard.readText().then(text => {
      setMessage(text);
    });
  }

  return (
    <div className="sidebar">
      <div className="chat-container">
        <div className="messages-area">
          {messages.length === 0 && (
            <div className="welcome-message">
              <h2>Welcome to Moly</h2>
              <p>Start typing or paste a message to get suggestions.</p>
            </div>
          )}
          {messages.map((msg, i) => (
            <div key={i} className={`message message-${msg.type}`}>
              {msg.text}
            </div>
          ))}
        </div>

        <div className="controls">
          <div className="model-select">
            <label>Model:</label>
            <select value={selectedModel} onChange={e => onModelChange(e.target.value)}>
              <option value="mistral">Mistral</option>
              <option value="neural-chat">Neural Chat</option>
              <option value="llama2">Llama 2</option>
            </select>
          </div>

          <div className="mode-select">
            <label>Mode:</label>
            <select value={mode} onChange={e => setMode(e.target.value)}>
              <option value="direct">Direct (Ready-to-use)</option>
              <option value="socratic">Socratic (Questions)</option>
            </select>
          </div>

          <div className="context-select">
            <label>Tone:</label>
            <select value={context} onChange={e => setContext(e.target.value)}>
              <option value="friendly">Friendly</option>
              <option value="formal">Formal</option>
              <option value="playful">Playful</option>
            </select>
          </div>
        </div>

        <div className="input-area">
          <textarea
            value={message}
            onChange={e => setMessage(e.target.value)}
            onKeyPress={e => e.key === 'Enter' && !e.shiftKey && handleSendMessage()}
            placeholder="Type a message or paste incoming text..."
            disabled={loading}
          />
          <div className="input-buttons">
            <button onClick={handlePasteMessage} disabled={loading}>
              📋 Paste
            </button>
            <button onClick={handleSendMessage} disabled={loading || !message.trim()}>
              {loading ? 'Thinking...' : 'Send'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
