# INVESTIGATION: Multi-Person Message Tracking Failure

**Date**: Sept 29, 2026  
**Issue**: Moly fails to recognize WHO IS WHO in multi-person messages  
**Status**: Root causes identified, ready for fix planning  

---

## Table of Contents

1. [Chat Evidence](#chat-evidence)
2. [Server Logs Analysis](#server-logs-analysis)
3. [Failure Points](#failure-points)
4. [Root Causes](#root-causes)
5. [Architecture Gap](#architecture-gap)
6. [Recommendations](#recommendations)

---

## Chat Evidence

### User Message 1 (First Contact)
```
05:37 PM
Hey Moly, what's up. I 'd like to talk to you about a girl. I saw her 
profile on fetlife and I think we have some common interests. Her profile 
name is Christine. She is 39, female submissive. I want to write a message 
to her to start a chat, but I don't know what to say. I want something 
playful and smart. Not a simple hello, like the dozens she takes every 
month. Can you help me make the message?
```

**Moly's Response** (Asking for clarification):
```
05:49 PM
Have you had any previous interactions with Christine on Fetlife or is 
this your first time reaching out to her? It would be helpful to know if 
there's any prior context between the two of you.
```

### User Message 2 (Providing Context + Christine's Profile)
```
05:55 PM
No, we had no prior interaction. We have some things in common. Take a 
look on what she has on her profile. 

Genders: FemaleCisgender
Roles: submissiveExploringGood Girl
Orientation: BicuriousBisexualSapiosexual
Pronouns: She/Her
Active: Curious And Want To Try
Looking for: Long-term Relationship, Relationship, Dominant, Friendship, Mentor / Teacher

About: 
Hello!! I'm a quiet, bi curvy girl , wanting to explore my sub nature 
with a caring Dom mentor/partner...
I'm a "good girl", who blooms with honesty, praise and clear, direct words.
I like taking my time, 'cause my submission is precious to me.

!! => NOT INTERESTED IN: !!
• LONG DISTANCE
• MALE SUBS/SWITCHES
• CASUAL SEX
• CYBERFUN

Fetishes Into: 69 (everything to do with it), Affectionate Domination 
(receiving), Aftercare (everything to do with it), Aftercare Cuddles, 
Anal Sex (receiving), Being called a good girl (receiving), 
Being Overpowered (receiving), Bisexuality (everything to do with it), 
Biting (everything to do with it), Blindfolds (receiving), Bondage 
(receiving), Boss/Secretary, Business Suits (watching), Candle Wax 
(everything to do with it), Collars (wearing), Control (receiving), 
Costumes/Dressing-up, Cuddling Naked (everything to do with it), 
Cuddling with benefits (everything to do with it), Community.

I am male dominant and I am also not interested in casual sex.
```

**Critical Observation**: This message contains TWO DIFFERENT PEOPLE'S DATA:
1. **Christine's profile**: Genders (Female), Roles (submissive), etc.
2. **User's statement**: "I am male dominant" and "not interested in casual sex"

Moly should recognize:
- Christine: female, submissive, bisexual, interested in aftercare, bondage, etc.
- User: male, dominant, NOT interested in casual sex

### Moly's Response (Generic Acknowledgement)
```
Hello! It's great to see you reaching out about your search for a 
compatible mentor/partner. From your message, I can see that you're a 
curious and open-minded individual seeking a dominant partner who shares 
similar interests. You've provided a comprehensive list of what you're 
looking for and what you're not interested in. This helps narrow down 
potential matches. Since you've mentioned that you're looking for a 
long-term relationship, it might be beneficial to focus on that aspect 
first. Let's discuss your preferences and needs in the context of a 
long-term relationship, and then we can explore how to find someone who 
aligns with those preferences. Would that be alright with you?
```

**Critical Problem**: Moly treats Christine's preferences as the USER's preferences:
- "you're looking for a dominant partner" ← This is CHRISTINE's need, not the user's
- User is male dominant, so he's looking for a female submissive (which Christine is)
- Moly failed to distinguish who wants what

---

## Server Logs Analysis

### Timeline of Processing

#### Phase: Entity Extraction (Message 1)
```
[17:46:04] [IntentDetector] Extracting entities with semantic classification
[17:46:04] [LLMClient] Adaptive timeout: prompt_len=2677 bytes -> timeout=15m25s
[17:46:04] [Ollama] Calling mistral at http://127.0.0.1:11434 with temp=0.3
[17:46:04] (6 minutes of processing)
[17:46:04] [IntentDetector] Entity validated: value=Moly type=self_reference subject=self_reference confidence=1.00
[17:46:04] [IntentDetector] Entity validated: value=user type=self_reference subject=self_reference confidence=1.00
[17:46:04] [IntentDetector] Entity validated: value=girl type=topic subject=ambiguous confidence=1.00
[17:46:04] [IntentDetector] Entity validated: value=Christine type=contact subject=contact confidence=1.00
[17:46:04] [IntentDetector] Entity validated: value=female submissive type=contact subject=contact confidence=1.00
```

**SUCCESS**: Entities extracted WITH subject field:
- `Christine` → subject=contact ✓
- `female submissive` → subject=contact ✓
- `girl` → subject=ambiguous ✓

#### Phase: Contact Storage (Message 1)
```
[17:40:45] [MessageProcessor] ✓ Extracted contact: Christine (other, confidence=1.00)
[17:40:45] [V2] ContactRepository: saving contact Christine for user user_d73d9eee9f0f6c0
[17:40:45] [V2] ContactRepository: saved contact Christine (id=14)
```

**SUCCESS**: Christine saved to database with id=14

---

### Timeline: Message 2 Processing

#### Phase: Context Extraction
```
[17:56:28] [ContextExtractor] Extracting context from message: 
           No, we had no prior interaction. We have some things in common...
[17:56:28] [LLMClient] Adaptive timeout: prompt_len=4230 bytes -> timeout=15m40s
[17:56:28] (2 minutes 25 seconds of processing)
[17:58:53] [MessageProcessor] ✓ Extracted contact: the girl (potential romantic, confidence=0.90)
[17:58:53] [V2] ContactRepository: saving contact the girl for user user_d73d9eee9f0f6c0
[17:58:53] [V2] ContactRepository: saved contact the girl (id=16)
```

**PROBLEM 1**: ContextExtractor creates new contact "the girl" instead of recognizing "Christine"
- First message: Christine (id=14)
- Second message: the girl (id=16)
- Same person, two database records

#### Phase: Entity Extraction with Subject Tracking (Message 2) ❌ TIMEOUT
```
[17:58:53] [IntentDetector] Extracting entities with semantic classification
[17:58:53] [LLMClient] Adaptive timeout: prompt_len=3634 bytes -> timeout=15m35s
[17:58:53] [LLMClient] Calling ollama with prompt length=1878, retries=1, timeout=15m35s
[17:58:53] [Ollama] Calling mistral at http://127.0.0.1:11434 with temp=0.3

(15 MINUTES 35 SECONDS PASS)

[18:14:28] [Ollama] Request failed: Post "http://127.0.0.1:11434/api/generate": 
           context deadline exceeded
[18:14:28] [LLMClient] Attempt 1 failed: Ollama request failed
[18:14:28] [LLMClient] Failed after 1 attempts
[18:14:28] [IntentDetector] Entity extraction LLM call failed: failed after 1 attempts
```

**CRITICAL FAILURE**: Phase 1 (ExtractEntitiesWithClassification) times out
- LLM call started: 17:58:53
- LLM timeout error: 18:14:28
- Duration: 15 minutes 35 seconds
- Expected timeout: 15m35s (935 seconds)
- Actual duration: ~935 seconds (matches but call still failed)

**Evidence of Timeout Calculation Issue**:
```
Prompt length: 3634 bytes (message + Christine's full profile)
Adaptive timeout: 15m35s
LLM call duration: 16+ minutes
Result: Context deadline exceeded
```

The timeout appears to be hit right at the edge or slightly exceeded.

#### Phase: Fallback to Clarification Detection ❌ NO SUBJECT TRACKING
```
[18:14:28] [MessageProcessor] Layer 3: Detected likely clarification response
[18:14:28] [ClarificationCapture] Attempting to detect if message is clarification response
[18:14:28] [ClarificationCapture] Message: No, we had no prior interaction. We have some things in common...
[18:14:28] [ClarificationCapture] Found 1 pending clarifications
[18:14:28] [ClarificationCapture] ✓ Detected clarification response to: gap_clarif_q_1790693301115672001
[18:14:28] [MessageProcessor] Layer 3: ✓ Matched to question: gap_clarif_q_1790693301115672001
[18:14:28] [ClarificationCapture] Detecting clarification type for message
[18:14:28] [ClarificationCapture] Detected: subject_clarification
[18:14:28] [MessageProcessor] Layer 3: Clarification type detected: subject_clarification (confidence: 0.85)
```

**PROBLEM 2**: System falls back to clarification processing
- Detects: "this is a clarification response"
- Type: "subject_clarification"
- But: No subject parsing logic in fallback path

#### Phase: Clarification Processing ❌ WRONG SUBJECT ATTRIBUTION
```
[18:14:28] [ClarificationCapture] Processing clarification type: subject_clarification
[18:14:28] [ClarificationCapture] Handling subject clarification
[18:14:28] [V2] ContextAttributeRepository: SAVE START - subject_clarification=
           No, we had no prior interaction. We have some things in common. Take a look on what she has on her profile.  Genders
           FemaleCisgender
           Roles
           submissiveExploringGood Girl
           Orientation
           BicuriousBisexualSapiosexual
           Pronouns
           She/Her
           Active
           Curious And Want To Try
           Looking for
           Long-term Relationship
           Relationship
           Dominant
           Friendship
           Mentor / TeacherAbout
           Hello!!
           I'm a quiet, bi curvy girl , wanting to explore my sub nature with a caring Dom mentor/partner...
           I'm a "good girl", who blooms with honesty, praise and clear, direct words.
           I like taking my time, 'cause my submission is precious to me.
           !! => NOT INTERESTED IN: !!
           •LONG DISTANCE
           •MALE SUBS/SWITCHES
           •CASUAL SEX
           •CYBERFUN. Fetishes
           Into: 69 (everything to do with it), Affectionate Domination (receiving), Aftercare (everything to do with it)...
           Community. I am male dominant and I am also not interested in casual sex. 
           for user_confirmed (conf=1.00)

[18:14:28] [V2] ContextAttributeRepository: ✓ SAVED id=21 user=user_d73d9eee9f0f6c0 
           type=subject_clarification value=[ENTIRE MESSAGE BLOB]
           subject=user_confirmed context=general
```

**CRITICAL PROBLEM 3**: The entire message (Christine's profile + user's statement) is saved as ONE attribute with subject=user_confirmed

**What Should Happen**:
```
Subject: user
  - role: male dominant
  - interests: NOT casual sex

Subject: Christine
  - gender: Female
  - role: submissive
  - orientation: Bisexual, Sapiosexual
  - interests: Affectionate domination (receiving), aftercare, bondage, etc.
```

**What Actually Happens**:
```
Subject: user_confirmed
  - value: [entire profile blob]
```

#### Phase: Contact Characteristic Check
```
[18:34:00] [MessageProcessor] ✓ Saved extracted contact to contacts: the girl (potential romantic)
[18:34:00] [MessageProcessor] ✓ Saved contact: the girl (potential romantic) with 0 characteristics
```

**PROBLEM 4**: "the girl" contact saved with ZERO characteristics
- Christine's profile data exists in message
- But none of it was parsed into contact characteristics
- Profile data stored only as unstructured blob in context_attributes

---

## Failure Points

### Failure Point 1: Phase 1 LLM Timeout ❌

**Location**: agents/intent_detector.go line 812 (ExtractEntitiesWithClassification)

**Condition**: Message with large profile data (3634 bytes)

**Evidence**:
```
Prompt length: 3634 bytes
Timeout calculated: 15m35s (935 seconds)
Actual request time: 16+ minutes
Result: context deadline exceeded
```

**Impact**: No entity extraction with subjects
- Expected: 10 entities with subject field populated
- Actual: Zero entities returned, function fails

### Failure Point 2: No Fallback for Subject Tracking ❌

**Location**: main.go line 612-650 (when Phase 1 fails)

**Condition**: When ExtractEntitiesAndAnalyzeIntent fails

**Current Code Flow**:
```go
intents, extractErr := srv.intentDetector.ExtractEntitiesAndAnalyzeIntent(context.Background(), req.Message)
if extractErr != nil {
    log.Printf("[MessageProcessor] ⚠ Entity extraction failed: %v - continuing without entity data", extractErr)
    // System just logs error and continues
    // No fallback entity extraction
    // No subject parsing
}
```

**Impact**: When Phase 1 times out, subject tracking stops
- Fallback is: Treat message as clarification
- No subject parsing in fallback path

### Failure Point 3: Clarification Without Subject Parsing ❌

**Location**: database/clarification_capture.go line 327 (ProcessClarification)

**Function**: ProcessClarification

**Current Code**:
```go
case "subject_clarification":
    log.Printf("[ClarificationCapture] Handling subject clarification")
    attr := &ContextAttribute{
        ID:             0,
        UserID:         userID,
        ConversationID: conversationID,
        FactType:       "subject_clarification",
        FactValue:      message,  // ← Stores entire message as-is!
        AttributedTo:   "user_confirmed",
        Context:        "general",
        Confidence:     1.0,
        Source:         "clarification_subject",
        Evidence:       message,
        Version:        1,
        CreatedAt:      time.Now().Unix(),
    }
    err := cc.contextAttrRepo.Save(attr)
```

**Problem**: 
- Takes entire message blob
- Stores with subject=user_confirmed
- Never parses "I am male dominant" vs "She is submissive"

**Impact**: Profile data saved as blob, not structured attributes

### Failure Point 4: Contact Name Confusion ❌

**Location**: Multiple places

**Problem**:
- Message 1: "Christine" extracted by Phase 1
- Message 2: "the girl" extracted by ContextExtractor (crude extraction)
- Phase 1 supposed to refine/override but times out
- ContactDeduplicator never runs because Phase 1 failed
- Two separate contacts in database

**Evidence**:
```
[17:40:45] ✓ Extracted contact: Christine (other, confidence=1.00)  [id=14]
[17:58:53] ✓ Extracted contact: the girl (potential romantic, confidence=0.90)  [id=16]
```

**Impact**: Same person represented twice, no link between them

### Failure Point 5: Profile Data Not Parsed ❌

**Location**: database/clarification_capture.go (ProcessClarification)

**Expected**: Parse Fetlife profile into structured attributes
```
Genders: Female → contact.gender = "Female"
Roles: submissive → contact.role = "submissive"
Into: X, Y, Z → contact.interests = ["X", "Y", "Z"]
```

**Actual**: Stored as unstructured text blob

**Evidence**:
```
[18:34:00] ✓ Saved contact: the girl (potential romantic) with 0 characteristics
```

Zero characteristics despite having full profile data in message

---

## Root Causes

### Root Cause 1: LLM Timeout on Long Messages

**Why**: 
- Message contains 3634 bytes (user context + full Fetlife profile)
- Adaptive timeout calculates: 15m35s
- Ollama/Mistral takes longer than expected
- Request exceeds timeout

**Why It Matters**:
- Multi-person messages with profile data are naturally longer
- Timeout applies to ALL uses of ExtractEntitiesWithClassification
- No retry logic for timeouts
- No fallback extraction

**Code Location**: tools/llm_client.go (adaptive timeout calculation)

### Root Cause 2: Incomplete Fallback Path

**Why**:
- Phase 1 is designed to work with LLM
- When LLM fails (timeout), no alternative strategy exists
- System falls back to clarification detection instead
- Clarification path has no subject-parsing logic

**Why It Matters**:
- Multi-person tracking depends entirely on Phase 1 LLM succeeding
- If LLM fails, multi-person tracking completely stops
- No graceful degradation

**Code Location**: main.go line 612-650, agents/intent_detector.go

### Root Cause 3: Subject Parsing Missing in Clarification Flow

**Why**:
- ProcessClarification type-switches on clarification type
- For "subject_clarification", just saves entire message
- No logic to parse "I am X" vs "She is Y"
- No logic to identify user vs contact

**Why It Matters**:
- When Phase 1 times out, system uses clarification path
- Clarification path doesn't parse subjects
- All data gets subject=user_confirmed (wrong)

**Code Location**: database/clarification_capture.go line 327-390 (ProcessClarification)

### Root Cause 4: Contact Deduplication Doesn't Run When Phase 1 Fails

**Why**:
- ContactDeduplicator depends on Phase 1 entity results
- When Phase 1 fails, deduplicator never sees entity data
- ContextExtractor creates "the girl" independently
- No merge between first message "Christine" and second message "the girl"

**Why It Matters**:
- Same person gets two database records (id=14, id=16)
- No link between them
- Characteristics can't be consolidated

**Code Location**: main.go (no call to deduplicator in Phase 1 failure path)

### Root Cause 5: No Profile Data Parser

**Why**:
- Fetlife profiles have structured format (Genders, Roles, Into, etc.)
- No code exists to parse this format
- Profile data stored as unstructured blob
- Can't distinguish user properties from contact properties

**Why It Matters**:
- Profile data is rich, structured, valuable
- Currently treated as unstructured text
- No way to query "what are Christine's interests?"

**Code Location**: database/clarification_capture.go (missing ParseFetlifeProfile function)

---

## Architecture Gap

### The Gap: Phase 1 Has No Fallback

**Designed For**:
```
Phase 1 (ExtractEntitiesWithClassification)
  ├─ LLM call succeeds → Extract entities with subjects ✓
  └─ LLM call fails → ??? (missing)

Current Fallback:
  └─ Fall back to clarification detection
      ├─ Detect clarification type ✓
      ├─ Save message as blob ❌
      └─ No subject parsing ❌
```

**Should Be**:
```
Phase 1 (ExtractEntitiesWithClassification)
  ├─ LLM call succeeds → Extract entities with subjects ✓
  └─ LLM call fails → Fallback extraction
      ├─ Try keyword-based subject parsing
      ├─ Try pronoun parsing ("I", "She", "He")
      ├─ Try name-based parsing ("Christine is...", "User said...")
      └─ Return partial entities with subjects

Phase 2 (Clarification Processing)
  ├─ If subject_clarification type:
  │   ├─ Parse subjects from message
  │   ├─ Extract properties per subject
  │   └─ Save per-subject attributes
  └─ If correction type:
      └─ Update database with correction

Phase 3 (Question Filtering)
  └─ Filter redundant questions

Plus:
  ├─ ContactDeduplicator runs even when Phase 1 fails (merge "Christine" + "the girl")
  ├─ ProfileParser extracts structured data from Fetlife format
  └─ SubjectAttributor maps properties to user vs contact
```

---

## Recommendations

### Immediate Priorities (To Fix Subject Tracking)

**Priority 1**: Add fallback subject parsing when Phase 1 LLM times out
- Implement keyword/pronoun-based parser
- Run when ExtractEntitiesWithClassification fails
- Returns partial entities for downstream use

**Priority 2**: Add subject parsing to clarification flow
- Extract "I am..." vs "She is..." patterns
- Parse named subjects ("Christine is...")
- Map properties to subjects instead of storing as blob

**Priority 3**: Add Fetlife profile parser
- Parse "Genders: Female" → contact.gender
- Parse "Into: X, Y, Z" → contact.interests
- Return structured contact attributes

**Priority 4**: Trigger ContactDeduplicator even when Phase 1 fails
- Run deduplicator to merge "Christine" (msg 1) with "the girl" (msg 2)
- Use name similarity matching as fallback
- Consolidate characteristics under one contact record

**Priority 5**: Fix or increase LLM timeout for Phase 1
- Current: Adaptive timeout based on prompt length
- Problem: Still timing out on long messages
- Options:
  - A) Increase timeout multiplier
  - B) Use keyword-based extraction for very long messages
  - C) Chunk large messages and extract separately

### Investigation Needed

1. **Why is timeout still exceeded?**
   - Calculated: 15m35s
   - Actual: exceeded
   - Is timeout calculation off by ~1 minute?
   - Is Ollama/Mistral just slow?

2. **Can Phase 1 LLM be optimized?**
   - Reduce prompt size?
   - Use simpler model?
   - Batch entity extraction?

3. **Should clarification ever be priority over Phase 1?**
   - Current: Phase 1 times out → fallback to clarification
   - Alternative: Only use clarification if message matches pending question
   - Alternative: Run Phase 1 and clarification detection in parallel

### Success Criteria for Fix

✓ Message 2 should extract:
- User: male, dominant, NOT interested in casual sex
- Christine: Female, 39, submissive, bisexual, interested in aftercare, bondage, etc.

✓ Moly should respond to Message 2 with:
- Recognition that Christine is looking for a dominant partner
- User IS a dominant partner who is NOT interested in casual sex
- These are highly compatible preferences
- Clear mention of Christine's characteristics (not confusing them with user's)

✓ Database should have:
- One contact "Christine" (id=14) with all characteristics
- All profile data parsed into attributes, not blobs
- Clear subject attribution on all facts

---

## Appendix: Database State at Failure Point

### Contacts Table
```
id=14: Christine (relationship: other)
id=16: the girl (relationship: potential romantic)
```

### Context Attributes Table
```
id=21: user=user_d73d9eee9f0f6c0
       type=subject_clarification
       value=[ENTIRE PROFILE BLOB]
       subject=user_confirmed
       
id=22: user=user_d73d9eee9f0f6c0
       type=user_interest_alignment
       value=[ENTIRE PROFILE BLOB]
       subject=user_confirmed
```

### Expected After Fix
```
id=21: user=user_d73d9eee9f0f6c0
       type=user_role
       value=male dominant
       subject=user

id=22: user=user_d73d9eee9f0f6c0
       type=user_interest
       value=NOT casual sex
       subject=user

id=23: user=user_d73d9eee9f0f6c0
       type=contact_gender
       value=Female
       subject=Christine

id=24: user=user_d73d9eee9f0f6c0
       type=contact_role
       value=submissive
       subject=Christine

id=25: user=user_d73d9eee9f0f6c0
       type=contact_orientation
       value=Bisexual, Sapiosexual
       subject=Christine
       
[etc for all Christine's interests]
```

---

## Document Metadata

- **Created**: Sept 29, 2026
- **Investigation Date**: Sept 29, 2026  
- **User**: user_d73d9eee9f0f6c0
- **Conversation**: conv_1790693164
- **Chat Session**: 05:37 PM - 05:55 PM
- **Server Logs**: Full logs from 17:31:59 to 18:34:00
- **Status**: Root causes identified, ready for solution planning
