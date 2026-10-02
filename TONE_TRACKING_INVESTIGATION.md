# TONE TRACKING & COMMUNICATION ADVISORY INVESTIGATION
**Date**: October 2, 2026  
**Status**: Deep Analysis of Tone Detection Gaps  
**Commercial Application**: Moly as Communication Advisor

---

## EXECUTIVE SUMMARY

Moly has the **infrastructure** to detect tone but lacks the **commercial application logic** to:
1. Track tone changes over time for connections
2. Store tone trajectories systematically
3. Use tone trends in future decision-making
4. Inform users about meaningful tone changes

This investigation identifies the gaps and proposes a comprehensive solution.

---

## PART 1: CURRENT STATE ANALYSIS

### What Moly CAN Do Today

✅ **User Tone Preferences**:
- Stores in `about_me.tone_preference` (e.g., "warm and genuine")
- Uses in AnalysisContext for response generation
- Baseline for matching communication style

✅ **Emotional Tone Tracking**:
- Records `interactions.emotional_tone` for each message
- Extracts tone from message content ("vulnerable", "frustrated", "optimistic")
- LLM-based detection via MessageClarityAnalyzer

✅ **Contact Metadata**:
- Stores contact characteristics as JSON
- Can include `toneObserved` field in conversation_analyzer output
- Tracks relationship type and frequency

✅ **Conversation Analysis**:
- ConversationAnalyzer extracts tone observations during message processing
- Format: `"toneObserved":"warm|formal|tense|fearful|relaxed|casual"`
- Uses in AnalysisContext building

### What Moly CANNOT Do (Current Gaps)

❌ **Tone Change Tracking**:
- No systematic record of "tone was X, now is Y"
- No historical comparison for same contact
- No trending (improving, declining, volatile)
- Data is extracted but **NOT SAVED** for later use

❌ **Connection Tone Profiles**:
- No table for contact-level tone history
- No "this person is usually X but today was Y" detection
- No baseline per-contact tone establishment
- Tone observations disappear after single extraction

❌ **Tone Change Impact on Decision-Making**:
- Tone data not used to refine future advice
- No "connection seems upset, adjust approach" logic
- No "relationship tone improving, can ask deeper questions" gating
- Orchestrator layers don't receive tone trend data

❌ **User Feedback on Tone Changes**:
- Users don't see "I noticed they responded differently"
- No clarification questions about tone shifts
- No suggestions based on tone trajectory
- Tone insights hidden from user view

---

## PART 2: COMMERCIAL USE CASE DEEP DIVE

### The Advisory Workflow

**Current Flow**:
```
User: "Help me write a message to Sarah"
  ↓
[Moly helps draft message]
  ↓
User: "Sarah replied: [Sarah's message]"
  ↓
[Moly reads Sarah's tone - BUT doesn't track/use it]
  ↓
[System extracts tone as metadata, then loses it]
```

**Desired Flow**:
```
User: "Help me write a message to Sarah"
  ↓
[Moly drafts, aware of Sarah's baseline tone from history]
  ↓
User: "Sarah replied: [Sarah's message]"
  ↓
[Moly DETECTS tone: "formal" (unusual - Sarah is normally warm)]
  ↓
[Moly SAVES: Sarah's tone changed from warm→formal]
  ↓
[Moly INFORMS: "Sarah seemed different this time - more formal. Did something happen?"]
  ↓
[Moly USES this: "Given her tone change, I'd suggest..."]
  ↓
[Next time: Moly remembers "Sarah was recently more formal"]
```

### Why This Matters

**For User**:
- Realizes connection dynamics are shifting
- Gets insights about how their communication impacts others
- Can be proactive about relationship changes
- Learns communication patterns

**For Moly**:
- Richer context for advice
- Can detect relationship deterioration early
- Can suggest preventive communication
- Builds richer contact profiles

**Commercial Value**:
- User sees tangible value: "Moly noticed Sarah was upset"
- Differentiator: Most apps don't track this
- Retention: Tone tracking makes system more useful over time
- Upsell: Tone analytics, relationship insights

---

## PART 3: DATABASE CHANGES NEEDED

### New Table: `contact_tone_history`

```sql
CREATE TABLE contact_tone_history (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    contact_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    
    -- Tone observation
    observed_tone TEXT NOT NULL,      -- "warm", "formal", "tense", "cold", etc.
    confidence REAL,                   -- 0-1 confidence in tone detection
    
    -- Context
    message_content TEXT,              -- The message where tone was observed
    message_type TEXT,                 -- "user" or "connection"
    message_index INTEGER,             -- Position in conversation
    
    -- Trend data
    change_from_baseline TEXT,          -- "warm→formal", "friendly→distant", "normal"
    change_significance TEXT,           -- "minor", "moderate", "significant"
    
    -- Metadata
    detected_at INTEGER NOT NULL,       -- When tone detected (timestamp)
    created_at INTEGER NOT NULL,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (contact_id) REFERENCES contacts(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX idx_contact_tone_history_user_contact 
    ON contact_tone_history(user_id, contact_id);
CREATE INDEX idx_contact_tone_history_contact 
    ON contact_tone_history(contact_id, detected_at DESC);
```

### New Table: `contact_tone_baseline`

```sql
CREATE TABLE contact_tone_baseline (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    contact_id TEXT NOT NULL,
    
    -- Baseline tone profile
    primary_tone TEXT,                  -- Most common tone (e.g., "warm")
    secondary_tone TEXT,                -- Secondary common tone
    tone_variance TEXT,                 -- "stable", "varies", "volatile"
    
    -- Statistics
    observation_count INTEGER DEFAULT 0,  -- How many observations
    last_observation_at INTEGER,        -- Most recent observation
    
    -- Trend
    current_trend TEXT,                 -- "stable", "improving", "declining"
    recent_change_detected BOOLEAN,     -- True if significant recent change
    
    -- Metadata
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (contact_id) REFERENCES contacts(id),
    UNIQUE(user_id, contact_id)
);
```

### Modified Tables

**`contacts` table additions**:
```sql
ALTER TABLE contacts ADD COLUMN observed_tone TEXT;      -- Most recent tone
ALTER TABLE contacts ADD COLUMN tone_baseline TEXT;      -- Baseline/typical tone
ALTER TABLE contacts ADD COLUMN tone_change_detected BOOLEAN;  -- Flag for UI
```

**`interactions` table enhancement**:
- Already has `emotional_tone` ✅
- Add: `contact_mentioned TEXT` (which contact this interaction concerns)
- Add: `observed_contact_tone TEXT` (if connection replied)

---

## PART 4: LINGUISTIC ANALYSIS CHANGES NEEDED

### Current: Simple Tone Detection

**What it does**:
```go
// In ConversationAnalyzer
"toneObserved": "warm"  // One-time extraction
```

**Problem**: 
- No context
- No comparison
- No baseline
- No change detection

### Required: Contextual Tone Analysis

**New Component: `ToneChangeDetector`**

```
Input:
- Current message tone
- Previous messages from contact (last 10)
- Contact baseline tone
- User's current emotional state

Process:
1. Extract tone from current message
2. Compare to contact baseline
3. Detect if significant change
4. Classify change type: (warming, cooling, becoming formal, becoming casual, volatile)
5. Assess significance: (minor fluctuation, moderate shift, significant change)
6. Generate insight: "Usually warm, but today was formal"

Output:
- Current tone
- Change from baseline (if any)
- Significance level
- Trend direction (improving, declining, stable)
- Recommended response adjustment
```

### Implementation Pattern

```go
type ToneChangeDetector struct {
    llmClient        *LLMClient
    baselineStore    *ContactToneBaselineStore
    historyStore     *ContactToneHistoryStore
}

func (t *ToneChangeDetector) AnalyzeToneChange(
    ctx context.Context,
    userID string,
    contact Contact,
    currentMessage string,
    previousMessages []string,
) (*ToneChangeAnalysis, error) {
    // 1. Extract baseline
    baseline := t.baselineStore.Get(userID, contact.ID)
    
    // 2. Extract current tone
    currentTone := t.extractTone(currentMessage)
    
    // 3. Detect change
    if baseline != nil {
        change := t.detectChange(baseline, currentTone)
        significance := t.assessSignificance(change, previousMessages)
        insight := t.generateInsight(baseline, currentTone, significance)
        
        // 4. Save to history
        t.historyStore.Record(userID, contact.ID, currentTone, change, significance)
        
        // 5. Update baseline
        t.baselineStore.Update(userID, contact.ID, currentTone)
        
        return insight, nil
    }
    
    // First observation - just record
    t.historyStore.Record(userID, contact.ID, currentTone, "baseline", "initial")
    t.baselineStore.Create(userID, contact.ID, currentTone)
    
    return nil, nil
}
```

---

## PART 5: LLM LEVEL CHANGES NEEDED

### Current: Single Prompt for Tone

**Problem**: No comparative analysis, no context awareness

### Required: Comparative Tone Analysis Prompt

```
SYSTEM PROMPT:
You are analyzing how a person's communication tone has changed.

You have:
- Their baseline tone (from previous interactions)
- Their recent tone trajectory (last 3-5 messages)
- Their current message
- Context about the relationship

Your task:
1. Identify the current tone of their message
2. Compare to their baseline
3. Identify any significant shifts
4. Assess what might have caused the shift
5. Suggest how to respond

Tone vocabulary: warm, formal, distant, dismissive, interested, engaged, tense, 
relaxed, enthusiastic, reluctant, supportive, critical, playful, serious, etc.

---

USER MESSAGE HISTORY:
[Previous messages from contact]

BASELINE TONE:
[Their typical tone - e.g., "usually warm and engaged"]

CURRENT MESSAGE:
[The new message to analyze]

---

Provide analysis as JSON:
{
  "currentTone": "tone_word",
  "confidence": 0.9,
  "baselineComparison": {
    "changed": true/false,
    "changeType": "warming|cooling|becoming_formal|becoming_casual|volatile",
    "significance": "minor|moderate|significant"
  },
  "possibleCause": "brief observation",
  "recommendedResponse": "brief suggestion"
}
```

### Integration into Orchestrator

**New Layer (Insert between Layer 5 & 6)**:

```
Layer 5.5: Contact Tone Change Assessment

When connection's message is analyzed:
1. Extract tone of connection's message
2. Compare to baseline
3. If significant change detected:
   - Save to contact_tone_history
   - Flag for Layer 6+ (adjust strategy)
   - Queue user notification
```

---

## PART 6: USER COMMUNICATION STRATEGY

### How to Inform Users About Tone Changes

#### Option A: Clarification Question (Current Model)

**When**: Tone change detected during message analysis

**Format**:
```
Moly: "I noticed Sarah's tone was different this time - more formal than usual. 
       Is something different between you two, or is she just having a day?"
```

**Pros**:
- Consistent with existing clarification pattern
- Non-intrusive
- User stays in control

**Cons**:
- Might feel like Moly is making assumptions
- Delays processing

#### Option B: Insight Notification (New)

**When**: Significant tone change detected

**Format**:
```
Moly: "📊 Insight: Sarah's recent messages seem more formal than her baseline.
       Her tone has shifted from 'warm & engaged' to 'more business-like'.
       
       Would you like to:
       - Adjust your approach?
       - Talk about what might be different?
       - Just acknowledge for now?"
```

**Pros**:
- Educational
- Helps user learn communication patterns
- Optional action

**Cons**:
- More UI complexity
- Could feel like Moly is overanalyzing

#### Option C: Integrated Advice (Recommended)

**When**: User asks for help drafting next message

**Format**:
```
User: "Help me write a follow-up to Sarah"

Moly: "I can help! Quick context - Sarah's last few messages have been more 
       formal than usual. She's normally warmer with you. So I'd suggest:
       
       - Keep it brief (she might be busy)
       - Be specific (she seems to want facts right now)
       - Add warmth gently (let her set the tone back)
       
       Here's a draft..."
```

**Pros**:
- Naturally integrated
- Actionable
- Helps user improve communication
- Teaches pattern recognition

**Cons**:
- Requires message drafting trigger

---

## PART 7: IMPLEMENTATION ROADMAP

### Phase 1: Infrastructure (2-3 days)
- ✅ Create `contact_tone_history` table
- ✅ Create `contact_tone_baseline` table
- ✅ Create migration script
- ✅ Update Contact model with tone fields
- ✅ Create repositories for tone data

### Phase 2: Tone Change Detection (2-3 days)
- ✅ Build `ToneChangeDetector` component
- ✅ Integrate with LLM for comparative analysis
- ✅ Wire into extraction pipeline
- ✅ Add tone change storage after each extraction

### Phase 3: Orchestrator Integration (1-2 days)
- ✅ Add Layer 5.5: Contact Tone Assessment
- ✅ Pass tone trends to response generation
- ✅ Update ConversationAgent to use tone context

### Phase 4: User Communication (2-3 days)
- ✅ Implement clarification questions for tone changes
- ✅ Add insight notifications
- ✅ Update response generation to mention tone adjustments

### Phase 5: Testing & Refinement (1-2 days)
- ✅ Test tone detection accuracy
- ✅ Test tone comparison logic
- ✅ Verify tone trends track correctly
- ✅ User experience testing

**Total Effort**: 8-13 days

---

## PART 8: DATA FLOW DIAGRAM

```
Message from Connection
    ↓
[Extract content & tone]
    ↓
[Load contact baseline from DB]
    ↓
[Compare: current tone vs baseline]
    ↓
[Detect change significance]
    ↓
[Save to contact_tone_history]
    ↓
[Update contact_tone_baseline if needed]
    ↓
[If significant change:]
  ├─ Flag for user notification
  ├─ Pass to orchestrator layers
  └─ Use in response generation
    ↓
[Generate response]
    ├─ Include tone change insight?
    ├─ Adjust strategy based on tone?
    └─ Suggest communication adjustment?
    ↓
[Return response with context to user]
```

---

## PART 9: EXAMPLE SCENARIOS

### Scenario 1: Warming Tone

```
History:
- Message 1: "OK, I'm interested" (neutral)
- Message 2: "Yes, let's meet" (neutral)
- Message 3: "I'm really looking forward to it! 😊" (warm)

System detects:
- Baseline: neutral
- Current: warm
- Change: warming
- Significance: moderate

User sees:
"Sarah seems excited now! She went from neutral to really enthusiastic. 
 Keep up the positive energy in your response."
```

### Scenario 2: Cooling Tone

```
History:
- Message 1: "I'd love to! When?" (enthusiastic)
- Message 2: "That sounds fine" (neutral)
- Message 3: "OK" (dismissive)

System detects:
- Baseline: enthusiastic
- Current: dismissive
- Change: cooling
- Significance: significant

User sees:
"Sarah's tone has changed. She's normally enthusiastic with you, but her last 
 few messages seem less interested. Did something happen? Consider asking if 
 everything's okay before suggesting more plans."
```

### Scenario 3: Becoming Formal

```
History:
- Message 1: "Hey! How's it going?" (warm, casual)
- Message 2: "I'd appreciate your input on..." (formal)
- Message 3: "I need your thoughts on the project." (formal)

System detects:
- Baseline: warm, casual
- Current: formal
- Change: becoming formal
- Significance: moderate

User sees:
"Your manager has shifted to a more formal tone. They're probably switching 
 into work-mode. Keep your response professional and concise, but you can 
 still be friendly."
```

---

## PART 10: SUCCESS METRICS

### What Success Looks Like

✅ **System Tracks Tone**:
- Baseline established after 3+ messages from contact
- Changes detected within 1 message
- Historical data persists across sessions

✅ **System Uses Tone**:
- Tone change impacts response generation
- Advice adapts based on contact's tone trend
- User sees pattern in multiple interactions

✅ **User Learns**:
- Understands connection's communication patterns
- Sees how their messages impact tone
- Recognizes when relationships are shifting

✅ **Commercial Impact**:
- Users stay longer (feature is useful)
- Users take action based on tone insights
- Word-of-mouth: "Moly noticed my friend's tone changed"

---

## PART 11: RISKS & MITIGATIONS

### Risk 1: False Positives
**Problem**: System detects tone change when there isn't one  
**Mitigation**: 
- Require 2+ consistent observations before flagging change
- Use confidence thresholds
- Clarify with user instead of assuming

### Risk 2: Over-Confidence
**Problem**: User trusts Moly's tone analysis too much  
**Mitigation**:
- Frame as observations, not certainties
- Always include "I could be wrong" language
- Encourage user to verify with actual conversation

### Risk 3: Privacy Concerns
**Problem**: Users uncomfortable with tone tracking  
**Mitigation**:
- Be transparent: "I track tone changes to help you communicate better"
- Let users opt-out of tone tracking per contact
- Don't expose tone data externally

### Risk 4: Complexity
**Problem**: System becomes harder to maintain  
**Mitigation**:
- Clear separation of concerns (detector, storage, use)
- Well-documented LLM prompts
- Comprehensive tests

---

## CONCLUSION

**Current State**: Moly can detect tone but doesn't track it meaningfully

**Gap**: Infrastructure exists, but data isn't persisted or used

**Opportunity**: Add 4 new components to unlock commercial value

**Timeline**: 8-13 days to full implementation

**Impact**: Moly becomes communication advisor, not just clarification system

This transforms Moly from "Moly helped me think" to "Moly noticed my friend's tone changed and helped me respond better."

