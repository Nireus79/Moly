# Immediate Fixes Required (Sep 7, 2026)

## Status: 35% V2-compliant, Critical work remaining

---

## PRIORITY 1: Database Integration (BLOCKING)

### What to do:
1. **File: `moly-go/agents/v2_agents.go`**
   - Add `db *Database` parameter to NewAgentSystem()
   - Pass db to NewContextManager(), NewLearningAgent()
   - Update agent constructors to accept db

2. **File: `moly-go/agents/context_manager.go`**
   - Add `db *Database` field to contextManager struct
   - Implement GetAboutMe() to query: `db.getAboutMe(userID)`
   - Implement SetAboutMe() to call: `db.saveAboutMe(userID, aboutMe)`
   - Implement GetContact() to query: `db.getContact(userID, contactID)`
   - Implement GetContacts() to query: `db.getContacts(userID)`
   - Implement GetRelevantContext() to load AboutMe + Contacts + History

3. **File: `moly-go/agents/learning_agent.go`**
   - Add `db *Database` field
   - Implement RecordSuggestionChoice() to save to: `db.recordSuggestionChoice()`
   - Implement GetUserProfile() to load from: `db.getUserProfile(userID)`

4. **File: `moly-go/v2_handlers.go`**
   - Update getAgentSystem() to pass: `db` to NewAgentSystem()
   - Currently: `agents.NewAgentSystem(llm, userID)`
   - Should be: `agents.NewAgentSystem(llm, userID, srv.database)`

---

## PRIORITY 2: Use Loaded Context in Suggestions (1 hour)

**File: `moly-go/agents/conversation_agent.go`**

Current:
```go
// Build personalized suggestions based on context
suggestions := generateContextualSuggestions(aboutMe, contact, userMessage, intention)
```

Problem: aboutMe and contact are loaded but generateContextualSuggestions() doesn't actually USE them.

Fix:
- Pass loaded aboutMe/contact to LLM prompt
- Use user's actual communication style in suggestions
- Reference actual contact name/characteristics
- Make suggestions truly personalized not generic

---

## PRIORITY 3: Real LLM Integration (3-4 hours)

**File: `moly-go/tools/llm_client.go`**

Current:
```go
func (lc *LLMClient) CallClaude(prompt string) (string, error) {
    // TODO: Implement actual Anthropic SDK call
    return "Claude response placeholder", nil
}
```

Needs:
- Import Anthropic SDK: `github.com/anthropics/anthropic-sdk-go`
- Implement actual API call with CLAUDE_API_KEY
- Pass system prompts for each use case
- Parse structured responses

---

## How to Test After Fixes

```bash
# 1. Create test user data in extension
# 2. Send message through sidebar
# 3. Check database was queried:
#    - AboutMe loaded?
#    - Contacts loaded?
#    - Suggestion choice saved?

# 4. Send another message
# 5. Verify previous choice was stored and considered
```

---

## Files That Must Change

| File | TODOs | Status |
|------|-------|--------|
| context_manager.go | 10 | ❌ Not implemented |
| learning_agent.go | 5 | ❌ Not implemented |
| v2_agents.go | 0 | ⚠️ Needs db param |
| v2_handlers.go | 0 | ⚠️ Needs db pass-through |
| conversation_agent.go | 2 | ⚠️ Doesn't use context |
| llm_client.go | 1 | ❌ Placeholder only |
| tools/* | Many | ❌ Return placeholders |

---

## Why This Matters

**Current behavior:**
```
User sends message
→ Handler calls agent
→ Agent returns generic context-aware suggestions
→ (But has NO idea who the user is)
→ Suggestions saved nowhere
→ Next user message: same generic suggestions again
```

**After fixes:**
```
User sends message
→ Handler loads user's AboutMe from DB
→ Handler loads contact profiles from DB
→ Agent personalizes suggestions to THIS USER'S style
→ Agent records which suggestions were chosen
→ Next message: suggestions consider past choices + learned patterns
```

---

## Token Estimate

- Database integration: 50-66 hours
- LLM integration: 40-50 hours  
- Testing & refinement: 10-15 hours

**Critical path: 50-66 hours to get to MVP**

---

**Next session should start with Priority 1 above.**

