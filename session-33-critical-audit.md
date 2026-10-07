# Session 33: Critical System Audit - Root Causes & Findings

**Date:** October 6, 2026  
**Status:** ⚠️ THREE CRITICAL ISSUES IDENTIFIED  
**Build:** Clean (401b0ed + latest fixes)

---

## ISSUE #1: MESSAGE_SUMMARIES SCHEMA MISMATCH ❌

### Root Cause
Database table schema is INCOMPLETE - missing 13 critical columns that code tries to persist.

### Details

**MessageSummary struct has 19 fields:**
```
ID, MessageID, UserID, ConversationID, MessageIndex, Role, MessageLength,
ExtractedEntities, EntityTypes, Intention, KeyPhrases, Tone, CommunicationStyle,
Confidence, ExtractionSource, TopicShift, HasClarification, ProcessedAt, CreatedAt, UpdatedAt
```

**message_summaries table only has 9 columns:**
```
id, user_id, conversation_id, message_id, entities_count, confidence,
extraction_source, summary_text, created_at
```

**Missing 13 columns (ALL ACTIVELY USED IN CODE):**
- MessageIndex (used 4x)
- Role (used 4x) 
- MessageLength (used 4x)
- ExtractedEntities (used 12x)
- EntityTypes (used 4x)
- Intention (used 7x)
- KeyPhrases (used 8x)
- Tone (used 9x)
- CommunicationStyle (used 5x)
- TopicShift (used 5x) 
- HasClarification (used 5x)
- ProcessedAt (used 5x)
- UpdatedAt (used 13x)

### Impact
Every message processed causes "no such column" errors:
- Line 273: Failed to save message summary
- Line 291: Failed to load message summary
- Line 640: Failed to save message summary (appears again)

### FIX REQUIRED
**Migration 034:** Add all 13 missing columns to message_summaries table

Column types:
```sql
ALTER TABLE message_summaries ADD COLUMN message_index INTEGER DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN role TEXT DEFAULT 'user';
ALTER TABLE message_summaries ADD COLUMN message_length INTEGER DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN extracted_entities TEXT; -- JSON
ALTER TABLE message_summaries ADD COLUMN entity_types TEXT; -- JSON
ALTER TABLE message_summaries ADD COLUMN intention TEXT DEFAULT '';
ALTER TABLE message_summaries ADD COLUMN key_phrases TEXT; -- JSON
ALTER TABLE message_summaries ADD COLUMN tone TEXT DEFAULT '';
ALTER TABLE message_summaries ADD COLUMN communication_style TEXT DEFAULT '';
ALTER TABLE message_summaries ADD COLUMN topic_shift BOOLEAN DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN has_clarification BOOLEAN DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN processed_at INTEGER DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN updated_at INTEGER DEFAULT 0;
```

---

## ISSUE #2: MATURITY SCORE RESETS TO 0.00 ❌

### Root Cause
UNCLEAR - Requires investigation of:
1. Where does maturityCalc get reset/overwritten?
2. What creates a new ConversationMaturity with 0 score?
3. Why does loaded value (e.g., 0.36) get saved as 0.00?

### Timeline from logs

**Message 1 (21:57:02):**
- Layer3 calculates: profile=0.50 + entities=0.93 → Overall=0.36
- Saves: 0.36 ✓

**Between messages (22:01:18):**
- Something saves maturity as 0.00 ⚠️

**Message 2 (22:07:16):**
- Layer3 loads: 0.00 (was reset!)
- Recalculates: 0.36
- Says "IMPROVED (0.00 → 0.36)" (but it was 0.36 before!)
- Saves: 0.36

### Code Flow

✓ **Line 857:** Loads maturityCalc from database
```go
maturityCalc, matErr = srv.maturityService.LoadOrCreateMaturityContext(userID, req.ConversationID)
```

✓ **Not reset between load and save** - no reassignments found

✗ **Line 3696:** Saves maturityCalc to database
```go
saveErr := srv.maturityService.SaveMaturityContext(userID, req.ConversationID, maturityCalc)
```

### Impact
- Maturity never accumulates
- Layer 3 always sees 0.00 from previous message
- User context quality never improves
- System stuck in "initial" phase

### Investigation Needed
1. Check if Layer 3 creates NEW ConversationMaturity instead of updating loaded one
2. Check if finalContextMaturity variable resets it
3. Check if orchestrator creates fresh maturity object
4. Trace where 0.00 comes from between load (0.36) and save (0.00)

---

## ISSUE #3: GOAL EXTRACTION TOO SIMPLISTIC ❌

### Root Cause
Linguistic extraction only uses verb + next word, losing critical context and producing meaningless goals.

### Current Implementation (BROKEN)

**File:** `agents/context_extractor.go` lines 258-265

```go
goal := verb + " " + sent.Object  
// Example: "I want to talk about a girl"
// Extracts: verb="want", object="to" 
// Result: goal="want to" ← USELESS
```

### Problem
- Extracts only first verb found
- Takes only first word as "object" (actually preposition "to")
- Loses subject, full verb phrase, and actual goal
- Ignores available data: SentenceText, full context

### Available Data IGNORED
ExtractionOrchestrator returns full SentenceAnalysis with:
- `SentenceText` - Full sentence (e.g., "I want to write a smart message")
- `Subject` - Who is doing action (e.g., "user")
- `Verb` - Action verb (e.g., "want")
- `Object` - Direct object (e.g., "to") ← Being used INCORRECTLY
- `Confidence` - How confident is parse (0.0-1.0)

### Example of Failure
```
User message: "I want to write a message to Christine_sub that is smart and playful"

Current extraction:
  verb = "want"
  object = "to" (wrong - "to" is a preposition, not object!)
  goal = "want to" ← SEMANTICALLY USELESS

Should extract:
  goal = "write a message to Christine_sub" 
  OR: "write a smart playful message"
  OR: "craft an engaging first message"
```

### FIX REQUIRED
**Enhance linguistic goal extraction to:**

1. **Use full sentence when verb+object is weak**
   - If confidence < 0.7 OR object length < 3
   - Extract meaningful phrase from SentenceText

2. **Extract actual action phrase, not just verb**
   - "I want to [WRITE A MESSAGE TO CHRISTINE]"
   - Not just "want" + "to"

3. **Fall back to LLM if linguistic confidence is low**
   - Don't return empty string and skip LLM
   - Pass low-confidence extraction to LLM as hint

4. **Use subject context for clarity**
   - Goal: "user wants to write a message to Christine_sub"
   - Not: "want to"

### Implementation Approach
```go
// Pseudocode for better extraction
func extractGoalLinguistically(msg string) string {
    analysis = parseWithExtractionOrchestrator(msg)
    
    for sentence in analysis.SentenceAnalyses {
        verb = sentence.Verb
        if verb not in actionVerbs { continue }
        
        // BETTER: Extract full action phrase from sentence
        // "want to write a message" → extract "write a message"
        goalPhrase = extractGoalPhrase(sentence.SentenceText, verb)
        
        if confidence < 0.7 {
            // Don't return weak result
            // Let LLM handle it with hint
            return ""  // Fall back to LLM
        }
        
        // Return meaningful phrase (5-20 words)
        if len(goalPhrase) > 3 {
            return goalPhrase  // "write a message to Christine"
        }
    }
    
    return ""  // Fall back to LLM
}
```

### Why This Matters
- **Goal** drives everything: Layer 4 gap detection, Layer 7 principle evaluation, response strategy
- Truncated goal = wrong context for all downstream layers
- "want to" tells us NOTHING about user's actual need

---

## Summary Table

| Issue | Type | Severity | Root Cause | Status |
|-------|------|----------|-----------|--------|
| #1: Message Summaries | Schema Mismatch | 🔴 CRITICAL | 13 missing columns | Needs Migration 034 |
| #2: Maturity Reset | Logic Bug | 🔴 CRITICAL | Unknown reset point | Under Investigation |
| #3: Goal Extraction | Algorithm | 🔴 CRITICAL | Verb+word too simple | Needs Redesign |

---

## Previous Session Work (Session 32)
✅ Clarification tracking enum values fixed (cfb00fa, a5f27db, 9dc5a47, a64da43)  
✅ Principle-violations div removed from frontend (c84af2c)  
✅ All 4 clarificationType enum mismatches resolved  

---

## Next Steps (Priority Order)
1. **Debug FIX #2:** Trace where maturityCalc becomes 0.00
2. **Implement FIX #1:** Create migration 034 for message_summaries columns
3. **Redesign FIX #3:** Implement smarter linguistic goal extraction

---

**Session ended:** October 6, 2026  
**Issues documented:** 3 CRITICAL  
**Requiring fixes:** All 3
