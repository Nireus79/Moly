# ConversationAnalyzer Implementation ✅

## Status: COMPLETE & TESTED

ConversationAnalyzer is the core extraction engine for Phase 1.2. It analyzes completed conversations and extracts structured insights.

## What It Does

**Input**: Raw conversation messages (user + assistant)  
**Process**: LLM extraction with structured prompts  
**Output**: Confidence-scored insights (AboutMe, patterns, contacts, goals)

```
Conversation Messages
    ↓
buildConversationText() - format for LLM
    ↓
extractInsights() - call LLM with structured prompt
    ↓
JSON parsing - convert LLM response to ExtractionResult
    ↓
validateResults() - filter by confidence, clamp values
    ↓
ExtractionResult (ready for ProfileUpdater)
```

## Architecture

### Types

**ExtractionResult**: Contains all extracted insights
```go
type ExtractionResult struct {
    AboutMeUpdates []AboutMeUpdate
    PatternDetections []PatternDetection
    ContactMentions []ContactMention
    GoalProgressUpdates []GoalProgressUpdate
    ConfidenceScore float64
    ExtractedAt int64
    ErrorMessage string
}
```

**AboutMeUpdate**: Single insight about user's communication
```go
type AboutMeUpdate struct {
    Key string // "communication_style", "value", "preference", "tone_preference"
    Value string
    Confidence float64 // 0-1
    Source string // where it came from
    Reasoning string // why
}
```

**PatternDetection**: Recurring communication behavior
```go
type PatternDetection struct {
    Pattern string // "avoids_conflict_then_over_explains"
    Category string // "avoidance", "assertiveness", "clarity"
    Confidence float64
    Evidence string // specific example
    IsGrowthArea bool // user working on this?
}
```

**ContactMention**: Person mentioned in conversation
```go
type ContactMention struct {
    Name string
    RelationshipType string // "professional", "family", "romantic", "friendship"
    ToneObserved string // "warm", "formal", "tense", "fearful"
    Context string
    MainTopics []string
    Frequency string // "daily", "weekly", "monthly", "rare"
    Confidence float64
}
```

**GoalProgressUpdate**: Progress on communication goals
```go
type GoalProgressUpdate struct {
    GoalDescription string
    Progress string // "in_progress", "made_progress", "struggling"
    Evidence string
    Confidence float64
}
```

### Main Methods

**AnalyzeConversation**
```go
func (ca *ConversationAnalyzer) AnalyzeConversation(
    ctx context.Context,
    userID string,
    conversationID string,
    messages []Message,
) (*ExtractionResult, error)
```
- Main entry point
- Handles error cases gracefully
- Returns confidence-scored result
- Never panics on empty input

**extractInsights**
- Calls LLM with structured extraction prompt
- Parses JSON response
- Validates results

**validateResults**
- Filters items with confidence < 0.5
- Clamps all confidence values to 0-1 range
- Adjusts overall confidence if no items extracted

**buildConversationText**
- Formats raw messages for LLM
- "User: message\nAssistant: response\n..."

## LLM Prompt

The extraction prompt is carefully designed for structured output:

1. **System prompt**: Explains the job (extract communication insights)
2. **User prompt**: Includes conversation text + JSON format specification
3. **Temperature**: 0.3 (low, for consistent structured output)
4. **Max tokens**: 2000

Example extraction from prompt:
```
Analyze this conversation and extract ONLY high-confidence insights about how the user communicates.

[conversation text]

Return JSON with AboutMe, Patterns, Contacts, Goals...
Only include if confidence > 0.5
```

## Tests (8/8 Passing)

| Test | What It Verifies |
|------|------------------|
| BasicExtraction | Can analyze a real conversation |
| EmptyConversation | Handles empty input gracefully |
| ConfidenceFiltering | Low-confidence items filtered out |
| ConfidenceNormalization | Confidence clamped to 0-1 |
| ConversationText | Message formatting works |
| JSONSerialization | Round-trip JSON encoding/decoding |
| MultipleContacts | Multiple people detected |
| EdgeCases | No panics on edge cases |

**Run tests**:
```bash
go test -v ./agents -run ConversationAnalyzerGate
```

## Design Decisions

### ✅ Why LLM-Based Extraction
- More accurate than keyword matching
- Contextual understanding (tone, relationship type)
- Handles nuance (sarcasm, implicit communication)
- Extensible (easy to add new insight types)

### ✅ Why Confidence Scores
- No threshold = every extraction claimed
- With threshold (0.5) = system only updates notes on strong signals
- User can see confidence when viewing profile

### ✅ Why Validation
- LLM sometimes returns invalid JSON
- Filters low-confidence items before storage
- Clamps values to valid ranges
- Graceful degradation on errors

### ✅ Why Not Store Everything
- Low-confidence extractions often wrong
- Pollutes user profile with noise
- Better to be conservative and improve over time

## Usage Example

```go
// Create analyzer
analyzer := NewConversationAnalyzer(llmClient, db)

// Analyze a conversation
messages := []Message{
    {Role: "user", Content: "I need to tell my boss she's unfair"},
    {Role: "assistant", Content: "That sounds challenging. How are you thinking about it?"},
    {Role: "user", Content: "I want to be direct but I'm scared"},
}

result, err := analyzer.AnalyzeConversation(
    ctx,
    userID,
    conversationID,
    messages,
)

if err != nil {
    log.Fatalf("Extraction failed: %v", err)
}

// Result contains:
// - AboutMe updates (communication style, values, etc.)
// - Pattern detections (avoids conflict, etc.)
// - Contact mentions (boss, tone = fearful, etc.)
// - Goal progress (working on assertiveness, etc.)

// Next: Pass to ProfileUpdater to apply to permanent notes
```

## Integration Points

### Receives From:
- **EphemeralConversationManager**: Raw messages, conversation ID

### Sends To:
- **ProfileUpdater**: ExtractionResult with confidence scores
- **ExtractionQueue**: Serialized result for storage

### Dependencies:
- **tools.LLMProvider**: LLM client (real or mock)
- **database.Database**: For logging/stats (optional)

## Next Components

1. ✅ **ConversationAnalyzer** (THIS)
2. → **ProfileUpdater** - applies insights to permanent notes
3. → **EphemeralConversationManager** - lifecycle management
4. → **ProfileService** - read API for profile data
5. → **HTTP Handlers** - expose profile endpoints

## Error Handling

ConversationAnalyzer is defensive:

| Error | Handling |
|-------|----------|
| Empty conversation | Returns low-confidence result, no error |
| LLM call fails | Returns error, caller decides retry |
| Invalid JSON response | Returns error, logs response for debugging |
| Missing fields in JSON | Fills with defaults, continues |
| Confidence out of range | Clamps to 0-1 automatically |

## Performance Notes

- **LLM call**: ~1-2 seconds per conversation (typical)
- **JSON parsing**: < 10ms
- **Validation**: < 5ms
- **Total**: Dominated by LLM latency

For 100 conversations/day: ~200 seconds of LLM time (runs async in background)

## Future Improvements

1. **Incremental extraction**: Extract as user types, not after conversation ends
2. **Confidence tuning**: Learn which confidence thresholds work best
3. **Pattern merging**: Detect when two pattern descriptions mean the same thing
4. **Contact deduplication**: "Mom", "Mother", "my mother" → same person
5. **Contradiction handling**: User said "I'm direct" then "I avoid conflict" → merge intelligently

## Files

- `agents/conversation_analyzer.go` - Main implementation (170 lines)
- `agents/conversation_analyzer_test.go` - Tests (200+ lines)
- `PHASE_1_2_ARCHITECTURE.md` - System design
- `PHASE_1_2_SCHEMA_REDESIGN.md` - Database schema

