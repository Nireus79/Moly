# Sidebar Integration Example

## How to Integrate MolyAgent into Sidebar

This shows exactly how to wire up the Go backend safety/ethics checks into the existing Sidebar component.

## Step 1: Import the Hook

```typescript
// In src/sidebar/Sidebar.tsx

import { useMolyAgent } from '@/hooks/useMolyAgent';
import { SafetyAlert } from './components/SafetyAlert';
```

## Step 2: Use the Hook

```typescript
export const Sidebar: React.FC = () => {
  // Existing state
  const [currentContact, setCurrentContact] = useState<Contact | null>(null);
  const [conversationMessages, setConversationMessages] = useState<Message[]>([]);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  
  // NEW: Add analysis hook
  const { analyze, safety, constitution, loading: analyzing } = useMolyAgent();

  // ... existing code ...

  const handleSendMessage = async (userMessage: string) => {
    if (!userMessage.trim()) return;

    // ... existing validation ...

    // NEW: Run backend analysis
    try {
      await analyze(
        userMessage,
        currentContact?.name,
        `Talking to ${currentContact?.name} on ${currentContact?.platform}`
      );
    } catch (error) {
      console.warn('Backend analysis not available:', error);
      // Continue without backend - extension still works
    }

    // Existing message sending flow
    const userMsg: Message = { /* ... */ };
    const updatedMessages = [...conversationMessages, userMsg];
    setConversationMessages(updatedMessages);
    
    // ... rest of existing code ...
  };

  return (
    <div className="sidebar-container">
      <div className="sidebar-header">
        <h2>Moly</h2>
        {/* ... existing header ... */}
      </div>

      <div className="sidebar-content">
        {/* NEW: Show safety alerts above everything else */}
        {safety && safety.alert_type !== 'none' && (
          <SafetyAlert 
            alert={safety} 
            onDismiss={() => clear()}
          />
        )}

        {/* Existing components */}
        <ContactSelector
          onSelectContact={setCurrentContact}
          currentContact={currentContact}
        />

        <ChatHistory
          messages={conversationMessages}
          onDeleteMessage={handleDeleteMessage}
          onExport={handleExportConversation}
        />

        {/* NEW: Show ethics warnings if constitution issues detected */}
        {constitution && constitution.violations && constitution.violations.length > 0 && (
          <div style={{
            padding: '12px',
            marginBottom: '12px',
            background: '#fef3c7',
            border: '1px solid #fcd34d',
            borderRadius: '6px',
            fontSize: '13px',
            color: '#78350f'
          }}>
            <strong>Ethics Check:</strong> {constitution.violations.length} concern(s)
            {constitution.violations.slice(0, 2).map((v) => (
              <div key={v.principle_id} style={{ marginTop: '6px', fontSize: '12px' }}>
                • {v.principle}: {v.description}
              </div>
            ))}
            {constitution.violations.length > 2 && (
              <div style={{ marginTop: '6px', fontSize: '12px', opacity: 0.8 }}>
                +{constitution.violations.length - 2} more...
              </div>
            )}
          </div>
        )}

        <MessageInput
          onSend={handleSendMessage}
          disabled={isLoading || analyzing}
          placeholder="Type a message or paste from chat..."
        />

        {/* Existing suggestions display */}
        {suggestions.length > 0 && (
          <Suggestions
            suggestions={suggestions}
            loading={isLoading}
            onCopy={handleCopySuggestion}
            error={error || undefined}
          />
        )}

        {/* Show loading state during analysis */}
        {analyzing && (
          <div style={{
            padding: '8px 12px',
            fontSize: '12px',
            color: '#6366f1',
            textAlign: 'center'
          }}>
            Analyzing message for safety & ethics...
          </div>
        )}

        {/* ... rest of existing code ... */}
      </div>
    </div>
  );
};
```

## Step 3: Handle Crisis Blocks

Optional: Block sending if crisis detected

```typescript
const handleSendMessage = async (userMessage: string) => {
  if (!userMessage.trim()) return;

  // Run analysis first
  try {
    await analyze(
      userMessage,
      currentContact?.name,
      `Talking to ${currentContact?.name}`
    );
  } catch (error) {
    console.warn('Backend analysis not available:', error);
  }

  // NEW: Block if crisis detected
  if (safety && safety.alert_type === 'crisis') {
    setError('Crisis detected. Resources provided above. Please reach out for help.');
    setIsLoading(false);
    return;
  }

  // Continue with normal message send
  // ... existing code ...
};
```

## Step 4: Display Questions for Context Building

Show generated questions to help user think through response:

```typescript
import { useState } from 'react';

export const Sidebar: React.FC = () => {
  const [showContextQuestions, setShowContextQuestions] = useState(false);
  const { analyze, questions, clear } = useMolyAgent();

  return (
    <div className="sidebar-container">
      {/* ... existing code ... */}

      {/* Show context questions if generated */}
      {questions && questions.questions.length > 0 && (
        <div style={{
          padding: '12px',
          marginBottom: '12px',
          background: '#dbeafe',
          border: '1px solid #93c5fd',
          borderRadius: '6px',
          fontSize: '13px'
        }}>
          <div style={{ fontWeight: '600', marginBottom: '8px' }}>
            Think About:
          </div>
          {questions.questions.map((q, i) => (
            <div key={i} style={{ 
              marginBottom: '6px', 
              paddingLeft: '16px',
              position: 'relative'
            }}>
              <span style={{
                position: 'absolute',
                left: '0',
                color: '#3b82f6'
              }}>•</span>
              {q}
            </div>
          ))}
          <button
            onClick={() => clear()}
            style={{
              marginTop: '8px',
              padding: '4px 8px',
              fontSize: '12px',
              background: '#93c5fd',
              border: 'none',
              borderRadius: '3px',
              cursor: 'pointer',
              color: '#1e40af'
            }}
          >
            Done thinking
          </button>
        </div>
      )}

      {/* ... rest of components ... */}
    </div>
  );
};
```

## Step 5: Handle Backend Unavailable

The MolyAgent gracefully handles backend not running:

```typescript
const { analyze, safety, error } = useMolyAgent();

// analyze() throws if backend not available
// But extension continues to work - just no backend checks
try {
  await analyze(message, contactName, context);
} catch (error) {
  console.warn('Backend analysis not available - using LLM suggestions only');
  // Extension works fine - user gets LLM suggestions but not safety/ethics checks
}
```

## Complete Example

Here's a minimal working integration:

```typescript
import React, { useState } from 'react';
import { useMolyAgent } from '@/hooks/useMolyAgent';
import { SafetyAlert } from './components/SafetyAlert';

export const Sidebar: React.FC = () => {
  const [message, setMessage] = useState('');
  const { analyze, safety, constitution, loading } = useMolyAgent();

  const handleSend = async () => {
    // Analyze message
    await analyze(message, 'John', 'Dating context');

    // If crisis, don't send
    if (safety?.alert_type === 'crisis') {
      alert('Please reach out for help!');
      return;
    }

    // Send message
    console.log('Sending:', message);
    setMessage('');
  };

  return (
    <div>
      {safety && <SafetyAlert alert={safety} />}
      {constitution?.violations.length > 0 && (
        <div>Ethics concerns: {constitution.violations[0].principle}</div>
      )}

      <textarea
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        placeholder="Type message..."
      />
      <button onClick={handleSend} disabled={loading}>
        {loading ? 'Analyzing...' : 'Send'}
      </button>
    </div>
  );
};
```

## Testing Integration

Once integrated, test by:

1. Start backend: `moly-go/moly`
2. Build extension: `npm run build`
3. Load extension in Chrome: `chrome://extensions` → Load unpacked → select `dist/`
4. Type safe message → should analyze without alerts
5. Type crisis message → should show SafetyAlert
6. Look at console → should see analyze calls

## Common Issues

### Backend not connecting
- Check if `moly-go/moly` is running
- Verify port 11436 is accessible: `curl http://127.0.0.1:11436/api/status`
- Look at Go backend logs for errors

### Analyze not called
- Verify `useMolyAgent` hook is imported
- Check if `analyze()` is called in `handleSendMessage`
- Look for console errors about async/await

### SafetyAlert not showing
- Verify `safety` state is updated after `analyze()`
- Check if `safety?.alert_type !== 'none'` condition is correct
- Verify SafetyAlert component imports correctly

### Performance issues
- First call takes longer due to backend startup
- Subsequent calls are faster
- Question generation (1-5s) uses LLM - check backend logs

---

**Status**: Ready to integrate  
**Next**: Add to Sidebar.tsx and test end-to-end
