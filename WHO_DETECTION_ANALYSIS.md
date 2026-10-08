# WHO Detection Analysis - Where It Breaks

**Question:** How does the system detect WHO information is about?  
**Answer:** It tries to, but fails because it lacks essential context.

---

## THE EXTRACTION PROMPT'S "WHO" INSTRUCTION

**File:** `agents/context_extractor.go` lines 153-156

```
CRITICAL - SUBJECT TAGGING FOR ALL CHARACTERISTICS
When extracting any characteristic, preference, or value, ALWAYS tag the subject:
- If about USER (the person writing): tag as "USER|trait|confidence"
- If about CONTACT: tag as "CONTACT_[actual_name]|trait|confidence"
```

**The Assumption:** LLM can determine WHO by reading the message.

**The Reality:** LLM lacks the context needed to determine WHO.

---

## THE MISSING CONTEXT PROBLEM

### What the LLM Receives

**Extract Function Signature:**
```go
func (ce *ContextExtractor) Extract(ctx context.Context, userMessage string)
```

**Input to LLM:**
- ONLY the current message
- No previous messages
- No previous contacts
- No conversation history
- No message count
- No prior goals

### Example: Message 2 Input

**User Message:**
```
"Here are some insights from my profile. I'm 98% dominant, sadist.
Here are some from her's: She's submissive, good girl.
You can compare our interests.
So, how do I start the message?"
```

**What LLM Gets:**
- The text above (only this)
- No reference to Message 1
- No knowledge that "she" = "Christine_sub"
- No knowledge that this is a clarification response
- No knowledge of the primary goal ("craft_message")

---

## THE PRONOUNS PROBLEM

### Message Structure

Message 1:
```
"Help me craft a message to Christine_sub. 
Tell me about yourself and her interests."
```

Message 2:
```
"Here's my profile: I'm dominant...
Here's her's: She's submissive...
So how do I start it?"
```

### WHO Detection Attempts

**For "I'm dominant":**
- Pronoun: "I" = The person writing
- WHO: USER ✓
- LLM can tag: "USER|dominant|0.98" ✓

**For "She's submissive":**
- Pronoun: "She" = Some female person
- But WHO is "she"? 
  - Could be "Christine_sub" (from Message 1)
  - Could be someone else
  - Could be a generic reference
- Without Message 1 context, LLM cannot know
- LLM must either:
  - Guess: "CONTACT_christine|submissive" (if lucky)
  - Fail: "CONTACT_she|submissive" (wrong - "she" is not a name)
  - Guess: "CONTACT_unknown|submissive" (incomplete)

**Result:** WHO detection fails at the pronoun resolution stage

---

## THE MISSING INFORMATION CHAIN

### Information Available at Message 1
- Contact name: "Christine_sub" (explicitly mentioned)
- User pronouns/identity: "I"
- Goal: "Craft a message"

### Information Available at Message 2
- Pronouns: "I", "She", "our"
- NO contact names (only pronouns)
- NO reference to Message 1
- NO conversation history

### WHO Resolution Required for Message 2
```
"I'm dominant"  → WHO is "I"?
                → Need: Is "I" the user or a character?
                → Context: Message 1 says "me" = user
                → But Message 2 extraction can't access Message 1

"She's submissive" → WHO is "she"?
                  → Need: Previous contact from Message 1
                  → Context: Message 1 mentioned "Christine_sub"
                  → But Message 2 extraction can't access Message 1

"Our interests" → WHO is "our"?
                → Need: Both previous user AND contact
                → Message 2 extraction can't access Message 1
```

---

## THE DETECTION STRATEGIES (ALL FAIL)

### Strategy 1: Rely on Pronouns
```
"I'm X" → User
"She's Y" → Contact
"They're Z" → Ambiguous
```
**Problem:** Without conversation history, can't resolve pronouns  
**Result:** ❌ FAILS on Message 2

### Strategy 2: Rely on Named Entities
```
"John is dominant" → CONTACT_John
"I'm creative" → USER
```
**Problem:** Message 2 has pronouns, not names  
**Result:** ❌ FAILS on Message 2

### Strategy 3: Rely on Verb Patterns
```
"I prefer X" → USER preference
"She likes Y" → CONTACT preference
```
**Problem:** Could be any subject with any verb  
**Result:** ❌ FAILS without context

### Strategy 4: Rely on LLM Inference
```
LLM reads message in isolation
LLM tries to infer WHO based on context
```
**Problem:** Needs conversation history  
**Result:** ❌ FAILS when history is unavailable

---

## WHAT THE LLM ACTUALLY RETURNS

### For Message 2

**Best Case (LLM Gets Lucky):**
```json
{
  "userCharacteristics": ["USER|dominant|0.98", "USER|sadist|0.90"],
  "contactCharacteristics": {
    "christine": ["CONTACT_christine|submissive|0.95", "CONTACT_christine|good-girl|0.90"]
  }
}
```
- User characteristics tagged correctly ✓
- Contact characteristics guessed the name ⚠️
- By luck, "christine" matches "christine_sub" ⚠️

**Likely Case (LLM Can't Determine):**
```json
{
  "userCharacteristics": ["USER|dominant|0.98"],
  "contactCharacteristics": {
    "unknown": ["CONTACT_unknown|submissive|0.75"],
    "she": ["CONTACT_she|good-girl|0.70"]
  }
}
```
- User characteristics tagged correctly ✓
- Contact characteristics use pronouns/generic ❌
- WHO is ambiguous or wrong ❌

**Worst Case (LLM Misattributes):**
```json
{
  "userCharacteristics": ["USER|dominant|0.98", "USER|submissive|0.60"],
  "contactCharacteristics": {}
}
```
- Mixed subjects together ❌
- Can't distinguish user vs contact ❌
- Some characteristics lost ❌

---

## THE REAL PROBLEM

**The extraction prompt ASKS the LLM to do something impossible:**

"Determine WHO each trait is about... without giving the LLM the context needed to do so"

### What Should Happen
```
LLM gets:
1. Current message
2. Previous messages (to resolve pronouns)
3. Known contacts (to resolve names)
4. Conversation history (to understand context)

Then LLM can:
- Resolve "she" → "christine_sub"
- Resolve "I" → user
- Tag characteristics with confidence
```

### What Actually Happens
```
LLM gets:
1. Only current message

LLM must:
- Guess who "she" is (often wrong)
- Hope pronouns are clear (often ambiguous)
- Tag characteristics without context (often fails)
```

---

## THE DATA FLOW OF FAILURE

```
Message 2: "Here's my profile... She's submissive..."
    │
    ▼
Extract(userMessage only) ← NO CONTEXT
    │
    ▼
LLM tries: "Who is 'she'?"
    │
    ├─ Strategy 1: Check message for names
    │  → Only finds pronouns
    │  → Can't determine contact name
    │
    ├─ Strategy 2: Check message structure
    │  → See "Here's mine... Here's hers"
    │  → Infer "hers" = some female contact
    │  → But which one? No way to know
    │
    └─ Strategy 3: Guess/Default
       → Tag as "CONTACT_unknown"
       → Or omit the contact characteristics
       → Or guess wrong
    │
    ▼
Return: "CONTACT_unknown|submissive" or "USER|submissive" (WRONG)
    │
    ▼
Parser stores: String with wrong subject tag
    │
    ▼
Subject attribution tries to recover (TOO LATE)
    │
    ▼
Conflicts detected with wrong WHO
    │
    ▼
FALSE CONTRADICTIONS
```

---

## WHY PREVIOUS EXTRACTION HAD SUBJECTS RIGHT

**On Message 1:**
```
"Help me craft a message to Christine_sub.
I'm 98% dominant. She's submissive."
```

- "I'm dominant" → USER (clear pronoun) ✓
- "She's submissive" → Need to know who "she" is
  - But Message 1 just said "Christine_sub" explicitly
  - LLM can reference back within the message
  - Can tag as "CONTACT_christine_sub|submissive" ✓

**On Message 2 (ISOLATED):**
```
"Here are some from my profile... 
Here are some from her's..."
```

- "my profile" → USER (clear pronoun) ✓
- "her's" → Who is "her"?
  - No names in Message 2
  - Can't reference Message 1
  - LLM must guess ❌

---

## THE SOLUTION REQUIREMENTS

To properly detect WHO, extraction needs:

1. **Previous Contact Names**
   - "Previous contacts were: Christine_sub, Marcus, ..."
   - Can resolve pronouns like "she", "he", "they"

2. **Message Count**
   - Know if this is Message 1, 2, 3+
   - Message 1 is self-contained
   - Later messages may reference previous

3. **Previous Goals**
   - Understand if clarification or new request
   - Affects WHO interpretation

4. **Conversation Context**
   - What was established in previous messages
   - What pronouns refer to
   - What topics are ongoing

---

## CURRENT ARCHITECTURE MISMATCH

| Need | Required For | Currently Provided | Status |
|------|--------------|-------------------|--------|
| Contact names | Resolve "she", "he" | ❌ NO | BROKEN |
| Message count | Detect if clarification | ❌ NO | BROKEN |
| History | Resolve pronouns | ❌ NO | BROKEN |
| Goals | Understand intent | ❌ NO | BROKEN |
| Previous messages | Full context | ❌ NO | BROKEN |

**Extract function provides:** ONLY userMessage  
**What's needed:** userMessage + context  

---

## CONCLUSION

**WHO is not properly detected because:**

1. LLM is asked to identify WHO without adequate context
2. Extract function provides only current message
3. No access to previous contacts, names, or pronouns
4. No conversation history to resolve ambiguities
5. Fallback to guessing when context unavailable
6. Wrong WHO tags propagate through system

**This is the FOUNDATIONAL issue** all other problems flow from.

The system cannot properly understand "Who is this information about" because it isolates each message from the conversation context that would make WHO clear.

