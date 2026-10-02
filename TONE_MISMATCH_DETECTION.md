# TONE MISMATCH DETECTION
**Purpose**: Inform user when their tone differs from connection's tone  
**Scope**: Current interaction only (no persistence)  
**Status**: Specification for implementation  
**Date**: October 2, 2026

---

## WHAT IT DOES

When analyzing messages between user and connection, detect and inform user of tone mismatches.

**Example**:
```
User message: "I'm so excited to meet you!" (enthusiastic)
Connection reply: "OK, that works for me" (neutral/reserved)

Moly: "You seem excited, but they're being pretty measured about it. 
       Don't be shocked if they're more casual than you expect."
```

---

## REQUIREMENTS

### Input
- User's message (extract tone)
- Connection's message (extract tone)
- Context (relationship, conversation history)

### Processing
1. Extract tone from user message
2. Extract tone from connection message
3. Compare tones
4. If significant difference → generate insight
5. Include in response to user

### Output
- Inform user of mismatch
- Brief insight on what it means
- Optional: suggestion on how to respond

---

## TONE VOCABULARY

Tones to detect:
- **Enthusiasm**: excited, energetic, enthusiastic
- **Warmth**: warm, friendly, affectionate, genuine
- **Formal**: professional, businesslike, formal
- **Neutral**: matter-of-fact, objective, neutral
- **Reserved**: cautious, guarded, hesitant, reserved
- **Dismissive**: dismissive, cold, indifferent, uninterested
- **Tense**: stressed, frustrated, tense, upset
- **Supportive**: supportive, encouraging, helpful

---

## MISMATCH TYPES

### Type 1: Enthusiasm Mismatch
```
User: Enthusiastic → Connection: Neutral/Reserved
Insight: "They're more measured than you about this"

User: Neutral → Connection: Enthusiastic  
Insight: "They're more excited than you seem to be"
```

### Type 2: Warmth Mismatch
```
User: Warm/Friendly → Connection: Formal/Professional
Insight: "They're being more formal than you"

User: Casual → Connection: Serious/Tense
Insight: "They seem stressed; adjust your tone"
```

### Type 3: Intent Mismatch
```
User: Seeking advice → Connection: Dismissive
Insight: "They don't seem interested in helping"

User: Making plans → Connection: Hesitant
Insight: "They're not as keen as you are"
```

---

## IMPLEMENTATION

### Location
- **Where**: ConversationAgent or ResponseGenerator
- **When**: After extracting both user and connection tones
- **How**: In response metadata, add tone mismatch info

### Code Pattern

```go
// Extract tones (already in system)
userTone := extractTone(userMessage)
connectionTone := extractTone(connectionMessage)

// Detect mismatch
mismatch := detectToneMismatch(userTone, connectionTone)
if mismatch != nil && mismatch.Significance >= "moderate" {
    // Add to response
    response.Metadata["toneMismatch"] = mismatch
    response.Metadata["toneInsight"] = generateInsight(mismatch)
}

// Include in response text if important
if mismatch.Significance == "significant" {
    response.Message += "\n\n" + formatToneMismatchNote(mismatch)
}
```

### Tone Extraction (Existing)
Already implemented via:
- `interactions.emotional_tone` field
- LLM-based tone detection
- ConversationAnalyzer

### Mismatch Detection (NEW - Simple)

```go
type ToneMismatch struct {
    UserTone           string    // "enthusiastic", "warm", etc.
    ConnectionTone     string    
    Significance       string    // "minor", "moderate", "significant"
    Type               string    // "enthusiasm", "warmth", "intent"
    Insight            string    // Human-readable explanation
    Recommendation     string    // Optional: how to adjust
}

func detectToneMismatch(userTone, connectionTone string) *ToneMismatch {
    if userTone == connectionTone {
        return nil  // No mismatch
    }
    
    // Classify mismatches
    if isOpposite(userTone, connectionTone) {
        return &ToneMismatch{
            UserTone:       userTone,
            ConnectionTone: connectionTone,
            Significance:   "significant",
            Type:           classifyType(userTone, connectionTone),
            Insight:        generateInsight(userTone, connectionTone),
        }
    }
    
    // Minor differences
    return &ToneMismatch{
        UserTone:       userTone,
        ConnectionTone: connectionTone,
        Significance:   "minor",
        Type:           classifyType(userTone, connectionTone),
        Insight:        generateInsight(userTone, connectionTone),
    }
}
```

---

## USER COMMUNICATION

### In Response Text (Significant Mismatches)
```
"One thing to note: You're enthusiastic about this, 
but they seem more cautious. Don't let that dampen your energy,
but be prepared for them to be slower to commit."
```

### In Metadata (Moderate Mismatches)
```json
{
  "toneMismatch": {
    "userTone": "warm",
    "connectionTone": "professional",
    "significance": "moderate",
    "insight": "They're being more formal than you're being warm"
  }
}
```

### Not Shown (Minor Mismatches)
```
Just store in metadata, don't disrupt response
```

---

## EXAMPLES

### Example 1: Enthusiasm Gap
```
User: "I can't wait to see you this weekend! It's going to be amazing!"
(Tone: enthusiastic, excited)

Connection: "Sure, that works for me"
(Tone: neutral, matter-of-fact)

Mismatch detection:
- Type: Enthusiasm mismatch
- Significance: Significant (enthusiastic vs neutral is opposite)
- Insight: "You're much more excited than they are about this"

Response includes:
"You're clearly looking forward to this, which is great. 
But they seem to be going along with it rather than being excited.
Set expectations accordingly."
```

### Example 2: Warmth Gap
```
User: "Hey! How have you been? I miss you!"
(Tone: warm, affectionate)

Connection: "I'm fine. What did you need?"
(Tone: formal, businesslike)

Mismatch detection:
- Type: Warmth mismatch  
- Significance: Significant
- Insight: "They're being formal while you're being warm"

Response includes:
"You're reaching out warmly, but they're being more businesslike.
They might be busy or in work mode. Keep it brief and to the point."
```

### Example 3: No Mismatch
```
User: "Are you free Thursday?"
(Tone: neutral, straightforward)

Connection: "Yes, Thursday works"
(Tone: neutral, straightforward)

Mismatch detection: None
Response: Proceeds normally
```

---

## TESTING CHECKLIST

- [ ] Detect enthusiasm mismatch correctly
- [ ] Detect warmth mismatch correctly
- [ ] Detect formal/casual mismatch correctly
- [ ] Generate appropriate insights
- [ ] Don't flag minor differences as significant
- [ ] Include in response when significant
- [ ] Store in metadata when moderate
- [ ] Don't show for minor
- [ ] Provide useful recommendations
- [ ] Don't interrupt response flow

---

## EFFORT ESTIMATE

- **Analysis**: 2-4 hours (understand existing tone extraction)
- **Implementation**: 4-6 hours (mismatch detection logic)
- **Testing**: 2-3 hours (verify accuracy)
- **Total**: 8-13 hours

---

## ROLLOUT

### Phase 1: Basic Detection (4-6 hours)
- Implement mismatch detection
- Add to metadata
- Test accuracy

### Phase 2: User Communication (2-3 hours)
- Format insights for display
- Include in response text
- Test UX

### Phase 3: Refinement (2-3 hours)
- Tune significance thresholds
- Improve insight generation
- Handle edge cases

---

## SUCCESS CRITERIA

✅ Detects tone mismatches accurately (>85% accuracy)  
✅ Generates useful insights (users find them helpful)  
✅ Doesn't interfere with response quality  
✅ Doesn't require new data persistence  
✅ No additional LLM calls needed (uses existing extraction)  

---

## NO DATABASE CHANGES

This feature requires:
- ❌ No new tables
- ❌ No persistence changes
- ❌ No schema migrations
- ✅ Just: Use existing tone detection + add simple comparison logic

