# User Preference Isolation Architecture

## Problem
User preferences (communication style, tone, values) were leaking into:
- Constitutional evaluation
- Intent detection  
- Clarity analysis
- Intention detection

This biased independent analyses and broke objectivity.

## Solution: Clear Architectural Boundaries

### Layer 1: Analysis Layer (NO preferences)
Analyzes user input objectively, independent of preferences.

**Functions that should NEVER receive user preferences:**
- detectTopic(message)
- detectIntent(message, history)
- analyzeSafety(message, context)
- analyzeClarity(message, history)
- detectEntities(message)
- calculateMaturity(message, context)

**What they receive:**
- Message content
- Conversation history
- Safety/ethics principles (objective)
- Conversation context
- NOT user preferences

### Layer 2: Response Generation Layer (YES preferences)
Uses analysis results + user preferences to generate response.

**Functions that SHOULD receive user preferences:**
- buildAdaptiveSystemPrompt(style, tone, topic, ...)
- generateResponse(analysis, userPreferences)
- adaptToneToPreferences(response, style)

**Data flow:**
```
User Message
    ↓
[Analysis Layer] → Topics, Intent, Safety, Entities
    ↓
[Response Layer] + UserPreferences → Moly's Response
```

## Enforcement Strategy

### 1. Code Organization
```
agents/
  ├── analysis/           ← NO user preferences
  │   ├── intent_detector.go
  │   ├── topic_detector.go
  │   ├── clarity_analyzer.go
  │   └── safety_evaluator.go
  │
  └── response/           ← YES user preferences
      ├── response_generator.go
      ├── system_prompt_builder.go
      └── tone_adapter.go
```

### 2. Function Signature Contract
```go
// ANALYSIS LAYER - No preferences parameter
func (ca *conversationAgent) detectTopic(message string) string
func (id *LLMIntentDetector) DetectIntentWithLLM(message string, history []models.Message) IntentAnalysis
func (ce *ConstitutionalEvaluator) Evaluate(message string, context *models.AnalysisContext) EvaluationResult

// RESPONSE LAYER - Gets preferences
func buildAdaptiveSystemPrompt(style string, tone string, topic string, ...) string
func (ca *conversationAgent) generateResponse(analysis AnalysisResult, preferences *models.AboutMe) string
```

### 3. LLM Prompt Segregation
**Analysis prompts MUST NOT include:**
```go
// ❌ WRONG - Never do this in analysis prompts
prompt.WriteString(fmt.Sprintf("User style: %s\n", userProfile.CommunicationStyle))
prompt.WriteString(fmt.Sprintf("Preferred tone: %s\n", userProfile.PreferredTone))
```

**Response prompts CAN include:**
```go
// ✅ CORRECT - Only in response generation
systemPrompt := fmt.Sprintf("Adapt your tone to: %s\n", communicationStyle)
systemPrompt += fmt.Sprintf("Use style: %s\n", emotionalTone)
```

### 4. Code Review Checklist
When reviewing analysis functions:
- [ ] Function does NOT receive `AboutMe` parameter
- [ ] Function does NOT receive `UserProfile` parameter
- [ ] Function does NOT reference `CommunicationStyle`
- [ ] Function does NOT reference `PreferredTone`
- [ ] LLM prompt does NOT include user preferences
- [ ] Analysis receives ONLY message + objective context

When reviewing response functions:
- [ ] Function receives `AboutMe` or user preferences
- [ ] Preferences used ONLY for tone/style adaptation
- [ ] No analytical decisions based on preferences

### 5. Documentation Pattern
```go
// ANALYSIS: Topic detection (objective, preference-independent)
// Input: message text only
// Output: topic classification
// RULE: Must NOT include user preferences or communication style
func (ca *conversationAgent) detectTopic(lowerMsg string) string {
```

vs

```go
// RESPONSE: Build system prompt (subjective, preference-aware)
// Input: analysis results + user preferences
// Output: system prompt for LLM
// RULE: Can use user preferences for tone/style
func (ca *conversationAgent) buildAdaptiveSystemPrompt(..., style, tone, ...) string {
```

## Validation Rules

### Rule 1: Analysis functions are pure
```
Input: Message + Objective Context
Output: Analysis Result
No Side Effects: No access to user state
```

### Rule 2: Response functions compose analysis + preferences
```
Input: Analysis Result + User Preferences
Output: Moly's Response
Transformation: Use preferences only for tone/style
```

### Rule 3: Data never flows backward
```
❌ WRONG:  Response → Analysis (preferences leak)
✅ CORRECT: Analysis → Response (unidirectional)
```

## Implementation Checklist

- [ ] Separate `analysis/` package from `response/` package
- [ ] Document preference isolation principle in each file
- [ ] Add lint rule to catch preference access in analysis functions
- [ ] Code review template with preference-isolation questions
- [ ] Add test cases that verify analysis ignores preferences
- [ ] Document AnalysisContext to exclude preferences
- [ ] Document AboutMe is only for response generation

## Testing

```go
// Test: Changing user preferences should NOT affect analysis
func TestAnalysisIgnoresPreferences(t *testing.T) {
    msg := "Hello Moly"
    
    // Analyze with professional style
    result1 := detectTopic(msg)
    
    // Analyze with casual style
    result2 := detectTopic(msg)
    
    // Results must be identical
    assert.Equal(t, result1, result2)
}
```

## Files Fixed (Sept 29, 2026)

1. **constitutional_evaluator.go** - Removed USER PROFILE from safety evaluation
2. **intent_detector.go** - Removed USER COMMUNICATION STYLE from intent analysis
3. **message_clarity_analyzer.go** - Removed USER STYLE and PREFERRED TONE
4. **intention_detector.go** - Removed communication style context

All analysis systems now work independently of user preferences.
