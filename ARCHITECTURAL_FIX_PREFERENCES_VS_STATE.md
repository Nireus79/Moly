# ARCHITECTURAL FIX: Preferences vs. Observable State

**Date**: October 2, 2026  
**Status**: ✅ FIXED  
**Impact**: Ensures clean separation between user configuration and observed behavior

---

## The Problem

**Architectural Confusion**: Observable tone (current state) was being saved to `tone_preference` (user configuration).

### Example of the Bug
```
User message: "I'm excited!"
System extracts: Style.Tone = "excited" (observable state THIS message)
System saved to: tone_preference (user's tone preference)
Database result: "User prefers excited tone"

This is WRONG because:
- User's mood changes with context
- Preference is stable across time
- Current tone ≠ Preference
```

### Where It Happened
1. **main.go:2722** - Style extraction saves observed tone to tone_preference
2. **main.go:3451** - Clarification processing saves context to tone_preference

---

## The Fix

### Architectural Principle
```
PREFERENCES (Configuration):
- What user WANTS/CONFIGURED
- Stable over time
- Examples: "formal", "warm and professional", "direct"
- Source: User explicitly sets via settings or clarifications
- Storage: about_me.tone_preference (protected from automatic updates)

OBSERVABLE STATE (Current):
- What user IS DOING right now
- Changes with mood, context, topic
- Examples: "excited", "frustrated", "formal"
- Source: Extracted from current message via LLM
- Storage: ephemeral (analyzed, not persisted)
```

### Changes Made

**1. Line 2722 (Style Extraction)**
```go
// BEFORE: Saved observed tone to preference
`, userID, extractedContext.Style.Style, valuesJSON, extractedContext.Style.Tone, now, now)

// AFTER: Only save communication_style and values, NOT tone
`, userID, extractedContext.Style.Style, valuesJSON, now, now)
```

**2. Line 3451 (Clarification Processing)**
```go
// BEFORE: Saved context data to all preference fields
INSERT INTO about_me (..., tone_preference, ...)
VALUES (..., contextStr, ...)

// AFTER: Only save communication_style and patterns
INSERT INTO about_me (user_id, communication_style, patterns, ...)
VALUES (userID, contextStr, patternsJSON, ...)
```

### Key Changes
- ✅ Removed `extractedContext.Style.Tone` from INSERT statements
- ✅ Removed tone_preference from clarification processing
- ✅ Added explanatory comments about architectural principle
- ✅ tone_preference now ONLY updated by explicit user preferences

---

## How It Works Now

### User Settings (Preferences)
```
User configures in settings:
"I prefer warm, direct communication"
→ Stored as tone_preference = "warm, direct"
→ Passed to orchestrator as PreferredTone
→ Influences Moly's response style (Layer 8, Layer 6)
```

### Observable Tone (State)
```
User message: "I'm excited about this!"
→ LLM extracts Style.Tone = "excited" (observable, ephemeral)
→ Used for tone tracking (future feature)
→ NOT saved to database
→ NOT confused with preferences
```

### Orchestrator Integration
```
Layer 1-2: Extract context and preferences separately
Layer 3-5: Use PreferredTone for context awareness
Layer 6: Consider both preference + observed tone for tone shifts (future)
Layer 8: Adapt response depth based on PreferredTone
```

---

## Data Isolation

### about_me Table Structure
```
- communication_style: User's preferred communication style (e.g., "casual, direct")
- tone_preference: User's preferred tone (e.g., "warm, professional")
- core_values: User's stated values
- patterns: Detected behavioral patterns
- preferences: Other user preferences

NOTE: These are ALL configuration, never overwritten by observed data
```

### What's NOT Persisted
- Current tone/mood (ephemeral - analyzed per message, not stored)
- Message-specific emotional state
- Temporary tone fluctuations

These are tracked in-memory via Layer 1 extraction but never saved as preferences.

---

## Impact

### Before Fix
- ❌ Observed tone overwrites preferences
- ❌ Moly's behavior influenced by user's current mood (bad)
- ❌ Can't distinguish "how user usually talks" from "how user talks today"
- ❌ Preferences get corrupted after a few messages

### After Fix
- ✅ Preferences protected from observation
- ✅ Moly uses stable preferences for behavior
- ✅ Current tone tracked separately (future: for tone shifts)
- ✅ Clean separation: preferences ≠ state

---

## Future Enhancement

Once tone tracking is re-implemented properly (with database support):
```
NEW tables would store:
- tone_observations: Historic tone per message
- tone_patterns: Recurring tone patterns
- tone_trends: User's tone trajectory

These would be SEPARATE from:
- tone_preference: User's configured preference

Usage: "User prefers formal, but is currently frustrated"
→ Moly adapts: respect formality preference while acknowledging frustration
```

---

## Files Modified

- `main.go`: Lines 2722, 3451-3471 (removed tone parameter from INSERT statements)

## Testing

- ✅ Build: Clean compile, zero warnings
- ✅ Tests: 100% pass (8/8 packages)
- ✅ No regressions detected

## Commits

- Commit message includes architectural principle explanation
- Code comments explain the fix

---

## Verification

### Checklist
- ✅ Observed tone no longer saved to preferences
- ✅ Preferences only updated by explicit user configuration
- ✅ AnalysisContext correctly receives PreferredTone (not observed tone)
- ✅ Orchestrator layers use preferences for behavior
- ✅ Build and tests pass

### No Breaking Changes
- ✅ API contract unchanged
- ✅ Database schema unchanged (no migration needed)
- ✅ Response format unchanged
- ✅ Existing preferences preserved

---

## Architectural Principle

**"Preferences configure behavior. State informs understanding."**

- Preferences come from user (settings, clarifications)
- State comes from observation (current message, current context)
- Never confuse the two
- Never let observation overwrite configuration
- Let both inform Moly's understanding of the user

