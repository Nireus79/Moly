# FIX #75: Layer 4 - Goal-Aligned Gap Detection

## CURRENT PROBLEM

Layer 4 asks generic profile questions ABOUT extracted data:
- "No user profile data (communication style, values, goals)"
- "Communication style not defined"
- "Personal values not identified"  
- "No contacts identified"

This is WRONG per spec: "Don't ask about extracted data - use it as context instead"

### Example of Wrong Gap:
```
User says: "I want to write a smart first message to Christine_sub. 
            I focus on safety, consent, respectful communication."

Current system asks gap:
  "You mentioned safety and consent. How does this apply?"
  ← User JUST explained how it applies!
```

### What Should Happen:
```
User says: "I want to write a smart first message to Christine_sub.
            I have BDSM experience but only real-life. 
            I focus on safety, consent, respectful communication."

Goal-aligned gap to ask:
  "In your first message, should you be direct about your interest in BDSM, 
   or start more subtly and gauge her response?"
  ← This gap helps accomplish the goal (write message), not asks about extracted data
```

## ROOT CAUSES

1. **Gap Detection Sources Wrong** (Lines 321-407):
   - Checks for missing PROFILE DATA (profile.CommunicationStyle == "")
   - Checks for missing CONTACT INFO
   - Checks for EXTRACTION CONFIDENCE
   - All of these are "asking ABOUT extracted data"

2. **Goal-Aligned Filter is Broken** (Lines 449-494):
   - Function `filterGapsByGoal` exists but doesn't actually filter by goal
   - Just keeps high-severity gaps regardless of relevance to goal
   - Missing logic to match gap types to user's actual goal

3. **No Goal-Aligned Gap GENERATION**:
   - Only has hardcoded gap types (missing_profile, no_contacts, etc.)
   - Doesn't generate NEW gaps based on user's specific goal
   - Example: If user wants to "write message", should ask gaps specific to that task

## WHAT FIX #75 NEEDS TO DO

### PHASE 1: Stop Asking About Extracted Data
Remove/comment out:
- Lines 322-347: Missing profile gaps
- Lines 349-377: Missing contact gaps  
- Lines 389-397: Low extraction confidence gaps

Replace with: "We have this context, now what's needed to accomplish the goal?"

### PHASE 2: Extract User's Stated Goal
Already done - Line 103: `userGoal = lc.Layer1.ExtractedContext.Intention`

### PHASE 3: Generate Goal-Aligned Gaps
New function: `generateGoalAlignedGaps(userGoal, userValues, userMessage, extractedContext)`

Examples:
```
Goal: "write message to Christine_sub"
  → Ask: "Should you be direct about interests or gradual?"
  → Ask: "What tone - formal or playful?"
  → Ask: "Any boundaries she's mentioned you should know?"

Goal: "improve communication with girlfriend"
  → Ask: "What specifically isn't working in how you communicate?"
  → Ask: "When conflicts happen, what usually causes them?"
  → Ask: "What would better communication look like to you?"

Goal: "understand her better"
  → Ask: "What aspects of her are you trying to understand?"
  → Ask: "What would understanding look like in practice?"
```

### PHASE 4: Prioritize by Impact on Goal
Already attempted but broken - Lines 415-416 call functions that don't exist

Ranking:
- **Goal-blocking gaps** (HIGH) - Prevents achieving goal
- **Goal-supporting gaps** (MEDIUM) - Helps achieve goal  
- **Safety gaps** (ALWAYS HIGH) - Any concerns that matter
- **Context gaps** (LOW) - Nice-to-know

### PHASE 5: Ask ONE Gap, Wait for Answer
- Current: Generates list of gaps
- Should be: Generate list, then ask only the TOP-priority one
- After user answers: Re-evaluate maturity, re-rank remaining gaps

## IMPLEMENTATION NOTES

- Keep extracted data as CONTEXT (used in maturity calculation, response generation)
- Don't ask ABOUT extracted data
- Generate gaps that support user achieving their goal
- Ask ONE prioritized gap per message
- Re-assess after each answer

## TESTING

After FIX #75, test with:
```
User: "I want to write a message to Christine_sub. I have BDSM experience. 
       I focus on safety, consent, respectful communication."

Should ask gap that HELPS achieve "write message", not:
  ✗ "What's your communication style?"
  ✗ "Tell me about your values"
  
Should ask:
  ✓ "In this first message, should you be direct about your interests, or gradual?"
  ✓ "Any specific tone you want to set - formal or playful?"
```

## FILES TO MODIFY

- `agents/layer4_gap_detector.go` - Main gap detection
- `tools/layer_models.go` - Gap type definitions (if needed)

## ESTIMATED SCOPE

This is a substantial rewrite:
- Remove 60+ lines of wrong logic
- Add ~100-150 lines of goal-aligned gap generation
- Add ~50 lines of impact ranking

Total: ~40% rewrite of Layer 4 gap generation
