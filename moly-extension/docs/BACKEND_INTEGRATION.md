# TypeScript ↔ Go Backend Integration Guide

## Overview

The Moly extension now connects to the Go backend (moly-go) via HTTP API. This enables advanced features:

- **Safety Checking**: Detects crisis/self-harm and illegal activity language
- **Ethics Evaluation**: Assesses messages against 10 communication principles
- **Mode Transition Analysis**: Evaluates relationship shifts (e.g., Professional → Romantic)
- **Question Generation**: Creates contextual questions to help users think through responses

## Architecture

```
Extension UI (React/TypeScript)
        ↓
MolyAgent Service (molyAgent.ts)
        ↓
Go Backend HTTP API (127.0.0.1:11436)
        ↓
Go Systems:
  - SafetyChecker
  - ConstitutionEvaluator
  - ModeTransitionEngine
  - QuestionAgent
```

## Getting Started

### 1. Start the Go Backend

```bash
cd moly-go
go run .
```

The server will start on `127.0.0.1:11436`.

### 2. Use in React Components

```typescript
import { useMolyAgent } from '@/hooks/useMolyAgent';

export const MyComponent: React.FC = () => {
  const { analyze, safety, constitution, loading, error } = useMolyAgent();

  const handleAnalyze = async (message: string) => {
    try {
      await analyze(message, 'John', 'We met at work');
    } catch (err) {
      console.error('Analysis failed:', err);
    }
  };

  return (
    <div>
      <button onClick={() => handleAnalyze('Hey!')}>
        Analyze
      </button>

      {loading && <p>Analyzing...</p>}
      {error && <p>Error: {error}</p>}

      {safety && <SafetyAlert alert={safety} />}
      {constitution && (
        <div>
          <h3>Ethics Assessment</h3>
          {constitution.violations.length > 0 && (
            <ul>
              {constitution.violations.map((v) => (
                <li key={v.principle_id}>{v.principle}</li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
};
```

## API Reference

### MolyAgent Methods

#### `analyzeMessage(message, contactName?, context?)`
Runs complete analysis pipeline (safety → constitution → questions).

**Returns:**
```typescript
{
  safety: SafetyCheckResult;
  constitution?: ConstitutionalAnalysis;
  questions?: QuestionGeneratorResult;
}
```

#### `checkSafety(message)`
Detects crisis/self-harm and illegal activity language.

**Returns:** `SafetyCheckResult`

#### `evaluateConstitution(message)`
Evaluates message against 10 ethical principles.

**Returns:** `ConstitutionalAnalysis`

#### `analyzeModeShift(currentMode, proposedMode, context)`
Analyzes relationship mode shifts with risk assessment.

**Returns:** `ModeShiftAnalysis`

#### `generateQuestions(contactName, context)`
Generates contextual questions based on contact and situation.

**Returns:** `QuestionGeneratorResult`

#### `getPrinciples()`
Gets all 10 communication principles with descriptions.

**Returns:** `CommunicationPrinciple[]`

#### `isAvailable()`
Non-throwing check if backend is accessible.

**Returns:** `Promise<boolean>`

## Integration Points

### In Message Input Flow

```typescript
const handleSend = async (userMessage: string) => {
  // 1. Run safety check first
  const safety = await molyAgent.checkSafety(userMessage);

  // 2. If crisis detected, show alert and block send
  if (safety.alert_type !== 'none') {
    showSafetyAlert(safety);
    return;
  }

  // 3. Evaluate ethics
  const constitution = await molyAgent.evaluateConstitution(userMessage);
  if (constitution.overall_risk_level === 'high') {
    showEthicsWarning(constitution);
  }

  // 4. Generate contextual questions if needed
  if (needsContext) {
    const questions = await molyAgent.generateQuestions(
      contactName,
      conversationHistory
    );
    displayQuestions(questions);
  }

  // 5. Send message and get suggestions
  const suggestions = await llmProvider.generateSuggestions(userMessage);
};
```

### In Sidebar Component

```typescript
export const Sidebar: React.FC = () => {
  const { analyze, safety, constitution, questions, clear } = useMolyAgent();
  const [showAnalysis, setShowAnalysis] = useState(false);

  const handleMessageSend = async (message: string) => {
    // ... existing logic ...

    if (showAnalysis) {
      await analyze(message, currentContact?.name, context);
    }
  };

  return (
    <div>
      <ChatHistory messages={conversationMessages} />

      {showAnalysis && safety && <SafetyAlert alert={safety} />}

      {/* ... other components ... */}

      <MessageInput onSend={handleMessageSend} />

      <label>
        <input
          type="checkbox"
          checked={showAnalysis}
          onChange={(e) => setShowAnalysis(e.target.checked)}
        />
        Enable safety & ethics checks
      </label>
    </div>
  );
};
```

## Error Handling

The bridge includes graceful fallback:

```typescript
try {
  await molyAgent.checkSafety(message);
} catch (error) {
  // Backend unavailable
  console.warn('Backend not available:', error);
  // Continue without safety check
  // Or show user-friendly message
}
```

If the Go backend is not running:
- `molyAgent.isAvailable()` returns `false`
- Calls throw an error with message: "Moly backend is not running..."
- Extension continues to function with LLM providers only

## Response Types

### SafetyCheckResult
```typescript
{
  alert_type: 'crisis' | 'illegal' | 'none';
  severity: 'immediate' | 'high' | 'warning';
  title: string;
  message: string;
  indicators: string[];
  resources: CrisisResource[];
  recommendations: string[];
}
```

### ConstitutionalAnalysis
```typescript
{
  analyzed_action: string;
  violations: PrincipleViolation[];
  aligned_principles: string[];
  overall_risk_level: string;
  critical_concerns: string[];
  recommendations: string[];
  is_constitutional: boolean;
}
```

### ModeShiftAnalysis
```typescript
{
  relationship_modes: { current: string; proposed: string };
  phase: string;
  risk_level: number;
  indicators: {
    clear_indicators: string[];
    potential_concerns: string[];
  };
  mitigation_strategies: string[];
  questions_for_reflection: string[];
}
```

### QuestionGeneratorResult
```typescript
{
  questions: string[];
  context: string;
  reasoning: string;
}
```

## Development

### Testing the Bridge

```bash
# Terminal 1: Start Go backend
cd moly-go
go run .

# Terminal 2: Build extension
cd moly-extension
npm run dev

# Terminal 3: Check bridge is working
curl http://127.0.0.1:11436/api/status
```

### Adding New Endpoints

1. Add handler in `moly-go/main.go`
2. Add method to `MolyAgent` class in `molyAgent.ts`
3. Add hook method to `useMolyAgent.ts`
4. Use in components

## Performance Notes

- Safety checks: <50ms
- Constitution evaluation: <100ms
- Mode analysis: <200ms
- Question generation: Depends on LLM (1-5s typically)

For local models (Ollama), all operations run locally without network latency.

## Troubleshooting

### Backend not connecting
```bash
# Check if backend is running
curl http://127.0.0.1:11436/api/status

# Restart backend
cd moly-go
go run .
```

### Responses are empty
- Ensure Go backend is running
- Check backend logs for errors
- Verify JSON request format matches API

### High latency
- For cloud LLMs: Normal, depends on provider
- For Ollama: Check model size and hardware
- Consider using smaller models (mistral vs llama2)

## Future Extensions

- WebSocket for real-time analysis
- Batch analysis for multiple messages
- Custom analysis pipelines
- Streaming responses for long analyses
