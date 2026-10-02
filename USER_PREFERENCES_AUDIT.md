# User Preferences Audit: What's Wired and How

**Date**: October 2, 2026  
**Status**: ✅ WIRED - All preferences now influence behavior  
**Commits**: 5a4391a (separate state), 1f206e5 (wire preferences)

---

## User Preferences System

User preferences are configuration that influence Moly's behavior. They differ from observable state (current tone, mood).

---

## Audit Results

### ✅ Preferences ACTIVELY WIRED

| Preference | Storage | Loaded | Used in Prompts | Influences | Evidence |
|------------|---------|--------|-----------------|------------|----------|
| **CommunicationStyle** | about_me | ✅ | ✅ | Response tone/depth | agents/conversation_agent.go:2179 |
| **PreferredTone** | about_me | ✅ | ✅ | LLM personality | agents/conversation_agent.go:2182 |
| **Values** | about_me | ✅ | ✅ | Ethics/principles | agents/conversation_agent.go:2185 |
| **Goals** | about_me | ✅ | ✅ | Response focus | agents/conversation_agent.go:2188 |

### Deprecated/Not Used

| Component | Status | Reason |
|-----------|--------|--------|
| Patterns | Loaded but not used | Future enhancement (behavioral patterns) |
| Notes | Loaded but not used | Internal notes, not for prompting |

---

## Data Flow: Preferences → Behavior

```
1. USER SETS PREFERENCES (via settings)
   └─ "I prefer warm, professional tone"
   └─ "My goals: better communication"

2. DATABASE STORES (about_me table)
   └─ tone_preference = "warm, professional"
   └─ goals = ["better communication"]
   └─ communication_style = "casual, direct"
   └─ values = ["authenticity"]

3. LOAD ON MESSAGE (ConversationAgent)
   └─ Fetch about_me for user_id
   └─ Build AboutMe struct with all fields

4. INJECT INTO PROMPT (buildUserPromptContext)
   └─ userProfile += "Preferred tone: warm, professional"
   └─ userProfile += "Goals: better communication"
   └─ userProfile += "Communication style: casual, direct"
   └─ userProfile += "Values: authenticity"

5. SEND TO LLM (llmClient.Call)
   └─ SystemPrompt = personality guidelines
   └─ UserPrompt = facts + preferences + context
   └─ LLM generates response respecting all preferences

6. USER SEES RESPONSE
   └─ Warm (preferred tone) ✓
   └─ Professional (preferred tone) ✓
   └─ Focused on communication (goal) ✓
   └─ Authentic (value) ✓
```

---

## Example: How Preferences Work

### User Configuration
```json
{
  "communicationStyle": "casual, direct, authentic",
  "preferredTone": "warm, professional",
  "values": ["authenticity", "loyalty"],
  "goals": ["improve communication", "build confidence"]
}
```

### User Message
```
"I'm not sure how to talk to Sarah about my feelings."
```

### Generated Prompt Context
```
Stored communication style: casual, direct, authentic
Preferred tone: warm, professional
Values: authenticity, loyalty
Goals: improve communication, build confidence

Current situation: User is unsure about communication with Sarah
```

### Moly's Response Style
- **Tone**: Warm (prefers it) + Professional (prefers it) ✓
- **Style**: Casual and direct (prefers it) ✓
- **Focus**: Communication improvement (goal) ✓
- **Approach**: Authentic (value) ✓

**WITHOUT preferences**: Generic response, ignores user's style  
**WITH preferences**: Personalized to user's needs

---

## Settings Options

### CommunicationStyle
Examples: "casual", "formal", "direct", "indirect", "humorous", "serious"  
**Used for**: Response depth, language choice, formality level

### PreferredTone
Examples: "warm", "professional", "friendly", "formal", "playful"  
**Used for**: Emotional tone of response, empathy level

### Values
Examples: "authenticity", "loyalty", "growth", "honesty", "privacy"  
**Used for**: Ethical frame, advice alignment, emphasis

### Goals
Examples: "better communication", "build confidence", "understand myself"  
**Used for**: Response focus, advice relevance

---

## Architecture: Preferences vs. State

### Preferences (Configuration)
- Source: User sets in settings
- Storage: Persistent (database)
- Duration: Stable over time
- Usage: Configure Moly's behavior
- Example: "I prefer warm tone"

### State (Observable)
- Source: Extracted from current message
- Storage: Ephemeral (not persisted to DB)
- Duration: Current message only
- Usage: Understand current situation
- Example: "User sounds excited (right now)"

### Key Difference
```
Preference: "I like warm communication" (always true)
State: "I'm excited today" (temporarily true)
→ Moly should be warm (preference) AND acknowledge excitement (state)
```

---

## Current Wiring

### What's in UserProfile Prompt (buildUserPromptContext)

```go
if ctx.AboutMe != nil {
    if ctx.AboutMe.CommunicationStyle != "" {
        userProfile += fmt.Sprintf("Stored communication style: %s\n", ...)
    }
    if ctx.AboutMe.PreferredTone != "" {
        userProfile += fmt.Sprintf("Preferred tone: %s\n", ...)
    }
    if len(ctx.AboutMe.Values) > 0 {
        userProfile += fmt.Sprintf("Values: %s\n", ...)
    }
    if len(ctx.AboutMe.Goals) > 0 {
        userProfile += fmt.Sprintf("Goals: %s\n", ...)
    }
}
```

**All preferences now in prompt ✅**

---

## Testing the Preferences System

### Manual Test
1. Set user preferences: "warm, professional tone"
2. Send message: "I need help"
3. Check response: Does it sound warm and professional?

### Debug
```
Check logs:
[ConversationAgent] SystemPrompt adapted: style=casual tone=...
Look for userProfile in logs:
"Stored communication style: casual, direct"
"Preferred tone: warm, professional"
```

---

## Future Enhancements

### Patterns (Not Yet Used)
```
Store: "User tends to withdraw when frustrated"
Use: "User seems frustrated - give them space"
Status: Infrastructure ready, not yet implemented
```

### Behavioral Learning
```
Store: "User prefers detailed explanations"
Learn: From feedback on response length
Use: Adjust response length to preference
Status: Future work
```

### Context-Aware Goals
```
Store: Multiple goals, priority levels
Use: Rotate between goals based on topic
Status: Future work
```

---

## File References

- **agents/conversation_agent.go:2176-2189** - buildUserPromptContext (where preferences are included)
- **models/agent_types.go** - AboutMe struct definition
- **main.go:1700-1715** - Load preferences into AnalysisContext
- **ARCHITECTURAL_FIX_PREFERENCES_VS_STATE.md** - Preferences vs. state separation

---

## Verification Checklist

- ✅ CommunicationStyle loaded and used
- ✅ PreferredTone loaded and used
- ✅ Values loaded and used
- ✅ Goals loaded and used
- ✅ Preferences injected into prompt
- ✅ LLM respects preferences
- ✅ No confusion with state
- ✅ Build passes
- ✅ Tests pass

---

## Status

**User Preferences System**: ✅ FULLY WIRED

All user preferences now:
1. Load from database ✅
2. Get passed to response generation ✅
3. Influence LLM behavior ✅
4. Result in personalized responses ✅

Settings dropdown changes now have real impact on Moly's responses.

