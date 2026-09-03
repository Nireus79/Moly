import React, { useState } from 'react';
import './Sidebar.css';

export default function Sidebar({ selectedModel, provider, onModelChange }) {
  const [message, setMessage] = useState('');
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(false);
  const [context, setContext] = useState('friendly');
  const [mode, setMode] = useState('direct');

  function buildSystemPrompt() {
    const modeDesc = mode === 'socratic'
      ? 'Ask thoughtful questions to help them improve their message instead of providing direct suggestions.'
      : 'Provide direct, ready-to-use suggestions for improving the message.';

    const toneDesc = {
      friendly: 'Use a warm, approachable tone.',
      formal: 'Use a professional, formal tone.',
      playful: 'Use a playful, witty tone.'
    }[context] || 'Use a friendly tone.';

    return `You are Moly, an AI coach helping users craft better messages for communication on dating apps, social media, and messaging platforms.

Mode: ${modeDesc}
Tone: ${toneDesc}

Help the user improve their message by providing ${mode === 'socratic' ? 'insightful questions' : 'specific suggestions'}.`;
  }

  async function callClaudeAPI(userMsg, conversationHistory) {
    const apiKey = localStorage.getItem('moly-claude-key');
    if (!apiKey) throw new Error('Claude API key not configured in Settings');

    const messages_payload = [
      ...conversationHistory.map(msg => ({
        role: msg.type === 'user' ? 'user' : 'assistant',
        content: msg.text
      })),
      { role: 'user', content: userMsg }
    ];

    const response = await fetch('https://api.anthropic.com/v1/messages', {
      method: 'POST',
      headers: {
        'x-api-key': apiKey,
        'anthropic-version': '2023-06-01',
        'content-type': 'application/json'
      },
      body: JSON.stringify({
        model: 'claude-3-5-sonnet-20241022',
        max_tokens: 1024,
        system: buildSystemPrompt(),
        messages: messages_payload
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(`Claude API: ${error.error?.message || 'Unknown error'}`);
    }

    const data = await response.json();
    return data.content[0].text;
  }

  async function callOpenAIAPI(userMsg, conversationHistory) {
    const apiKey = localStorage.getItem('moly-openai-key');
    if (!apiKey) throw new Error('OpenAI API key not configured in Settings');

    const messages_payload = [
      ...conversationHistory.map(msg => ({
        role: msg.type === 'user' ? 'user' : 'assistant',
        content: msg.text
      })),
      { role: 'user', content: userMsg }
    ];

    const response = await fetch('https://api.openai.com/v1/chat/completions', {
      method: 'POST',
      headers: {
        'authorization': `Bearer ${apiKey}`,
        'content-type': 'application/json'
      },
      body: JSON.stringify({
        model: 'gpt-4',
        messages: [
          { role: 'system', content: buildSystemPrompt() },
          ...messages_payload
        ],
        max_tokens: 1024
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(`OpenAI API: ${error.error?.message || 'Unknown error'}`);
    }

    const data = await response.json();
    return data.choices[0].message.content;
  }

  async function callOllamaAPI(userMsg, conversationHistory) {
    const prompt = `${buildSystemPrompt()}\n\nConversation history:\n${
      conversationHistory.map(m => `${m.type === 'user' ? 'User' : 'Assistant'}: ${m.text}`).join('\n')
    }\n\nUser: ${userMsg}`;

    const response = await fetch('http://127.0.0.1:11434/api/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: selectedModel,
        prompt: prompt,
        stream: false
      })
    });

    if (!response.ok) throw new Error('Ollama not responding. Make sure Ollama is running: ollama serve');

    const data = await response.json();
    if (!data.response) throw new Error('No response from model');
    return data.response;
  }

  async function handleSendMessage() {
    if (!message.trim()) return;

    setLoading(true);
    const userMsg = message;
    setMessage('');
    setMessages(prev => [...prev, { type: 'user', text: userMsg }]);

    try {
      let response;
      const conversationHistory = messages;

      if (provider === 'claude') {
        response = await callClaudeAPI(userMsg, conversationHistory);
      } else if (provider === 'openai') {
        response = await callOpenAIAPI(userMsg, conversationHistory);
      } else if (provider === 'ollama') {
        response = await callOllamaAPI(userMsg, conversationHistory);
      } else {
        throw new Error('No provider configured');
      }

      setMessages(prev => [...prev, { type: 'assistant', text: response }]);
    } catch (error) {
      setMessages(prev => [...prev, { type: 'error', text: `Error: ${error.message}` }]);
    } finally {
      setLoading(false);
    }
  }

  function handlePasteMessage() {
    navigator.clipboard.readText().then(text => {
      setMessage(text);
    }).catch(err => {
      console.error('Paste error:', err);
      setMessage('[Paste failed - clipboard access denied]');
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
