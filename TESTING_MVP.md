# MVP Testing Guide

**Status**: V2 Implementation Complete ✅  
**All 4 Priorities Implemented**

---

## Quick Test (Local)

### 1. Backend API Test
```bash
curl -X POST http://127.0.0.1:11436/api/v2/conversation/generate \
  -H "Content-Type: application/json" \
  -d '{
    "conversationId":"test1",
    "userId":"user1",
    "userMessage":"Got promoted! Telling Sarah"
  }' | jq .suggestions
```

**Expected**: 3 suggestions about celebrating promotion

---

### 2. Extension UI Test

#### Setup
1. Chrome: Go to `chrome://extensions`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select `/Moly/moly-extension/dist/`
5. Extension icon appears in toolbar

#### Test Flow
1. **Click Moly icon** → Sidebar opens
2. **Type message**: "I got promoted! Want to tell Sarah"
3. **Watch for**:
   - ✅ Loading spinner
   - ✅ Suggestions appear (3 options)
   - ✅ 🤖 V2 Agent badge (blue)
   - ✅ Processing time shows
4. **Click suggestion** → Copy button works
5. **Send another message**: "Actually, I'm worried she'll be jealous"
6. **Verify**: Different suggestions (apologize intent)

---

## What's Actually Working

### Database ✅
- AboutMe loaded on request
- Contacts loaded from database
- Conversation history stored

### Learning ✅
- Suggestion choices recorded
- Patterns analyzed
- User profile built

### LLM ✅
- Real Claude API calls (when CLAUDE_API_KEY set)
- Falls back to context-aware suggestions without API
- Generates different suggestions per intention

### V2 Agents ✅
- ConversationAgent analyzes message intent
- ContextManager loads user data
- LearningAgent records choices
- All working together

---

## With Claude API (Optional)

To test with real Claude:

```bash
export CLAUDE_API_KEY='sk-ant-...'
cd moly-go && ./moly-backend
```

Then:
- Suggestions use real Claude analysis
- More intelligent personalization
- Better Socratic questions

Without API key:
- Suggestions still context-aware
- Pattern detection works
- Learning loop functions
- Full fallback support

---

## Database Verification

Check that data is being saved:

```bash
# View conversation history
sqlite3 ~/.config/moly/moly.db "SELECT * FROM interactions LIMIT 5;"

# View contacts
sqlite3 ~/.config/moly/moly.db "SELECT name, communication_style FROM contacts;"

# View suggestion choices
sqlite3 ~/.config/moly/moly.db "SELECT * FROM interactions WHERE topic='suggestion_choice';"
```

---

## Success Criteria

✅ Extension sidebar opens and accepts messages  
✅ Suggestions appear with V2 badge  
✅ Different suggestions for different intentions  
✅ Processing time shows (even if ~0ms without LLM)  
✅ Database stores data  
✅ Second message shows learned patterns  
✅ Copy-to-clipboard works  
✅ No console errors (F12)

---

## Known Limitations (MVP)

- ⚠️ Risk detection: Not yet integrated
- ⚠️ Socratic questions: Basic implementation
- ⚠️ Safety checking: Placeholder only
- ⚠️ OpenAI: Not yet supported (Claude only)

These are Phase 2 enhancements, not blocking MVP.

---

## Next: Phase 2 (Polish)

- [ ] Real risk pattern detection
- [ ] Enhanced Socratic questions
- [ ] Full safety checking with crisis resources
- [ ] OpenAI provider support
- [ ] Performance optimization
- [ ] Comprehensive error handling

---

**Status**: Ready for testing ✅

