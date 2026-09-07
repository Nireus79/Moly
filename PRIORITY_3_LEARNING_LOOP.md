# Priority 3: Learning Loop (Next Session - 1 hour)

**Database context loading: DONE ✅**

## Implement Learning Agent Database Methods

File: `moly-go/agents/learning_agent.go`

Replace 5 TODOs with database operations:

### 1. RecordSuggestionChoice (Line 55)
```go
// Save when user picks a suggestion
// SQL: INSERT INTO interactions (conversation_id, topic, ai_summary)
//      VALUES (?, 'suggestion_choice', json with choice data)
```

### 2. GetUserProfile (Line 32)
```go
// Load user's behavioral profile from database
// SQL: SELECT * FROM behavior_patterns LIMIT 1
// Return as UserBehavioralProfile
```

### 3. BuildUserProfile (Line 81)
```go
// Analyze past suggestion choices to build profile
// SQL: SELECT ai_summary FROM interactions WHERE topic='suggestion_choice'
// Extract patterns: which suggestions chosen, which worked
// Store updated patterns back
```

### 4. GetUserPatterns (Line 111)
```go
// Return patterns learned from past interactions
// SQL: Build query to aggregate suggestion choices
// Calculate success rates
```

### 5. RecordInteraction (Line 68)
```go
// Save interaction to database
// SQL: INSERT INTO interactions (conversation_id, topic, ai_summary, user_notes)
```

---

## What This Enables

Once learning loop works:
1. User picks suggestion → saved to database
2. User sends new message → agent loads past choices
3. System personalizes based on what user has chosen before
4. Suggestions improve over time (if using LLM)

---

## After Priority 3

Move to Priority 4: Real LLM Integration (3-4 hours)
- File: `moly-go/tools/llm_client.go`
- Implement actual Anthropic SDK calls
- Add proper error handling

