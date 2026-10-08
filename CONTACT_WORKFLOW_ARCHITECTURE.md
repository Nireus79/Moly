# Contact Workflow Architecture - Detailed Design

**Goal:** Create a workflow that properly handles contacts through real conversations  
**Scope:** Contact detection, tracking, disambiguation, clarification, naming  
**Status:** Design Phase - Before Implementation

---

## PHASE 1: CONTACT DETECTION

### When Do We Detect Contacts?

**Trigger 1: Explicit Contact Introduction**
```
"I want to talk about a girl"
"I'm dating this woman"
"There's a colleague who..."
"My boss has been..."

→ Detect: Contact exists
  Extract: Type (romantic/professional/friend/family/other)
           Pronouns (she/her, he/him, they/them)
           Status: UNNAMED
```

**Trigger 2: Pronoun Introduction**
```
"She likes BDSM"
"He's been flirting"
"They prefer..."

→ Detect: Contact referenced (but unnamed)
  Extract: Pronouns
           Implied type: "she" = likely female (not certain)
           Status: UNNAMED
```

**Trigger 3: Named Contact Introduction**
```
"I'm dating Emily"
"My colleague Marcus"
"This woman Sarah..."

→ Detect: Contact exists
  Extract: Name
           Type (if available)
           Pronouns (if stated)
           Status: NAMED
```

### Contact Detection Decision Tree

```
Message contains:
  │
  ├─ "I want to talk about [indefinite] [type]"
  │  └─ Create UNNAMED contact
  │
  ├─ "[Pronoun] [verb] [characteristic]"
  │  ├─ Is pronoun part of known contact? YES → Update contact
  │  └─ Is pronoun part of known contact? NO → Create UNNAMED or ASK
  │
  ├─ "[Name] [verb] [characteristic]"
  │  ├─ Is name known contact? YES → Update contact
  │  └─ Is name known contact? NO → Create NAMED contact
  │
  └─ "I/Me [verb]"
     └─ Update USER profile
```

---

## PHASE 2: CONTACT TRACKING

### Contact Data Structure

```go
type Contact struct {
  ID                string       // "contact_1", "contact_2"
  Name              *string      // null = unnamed
  Type              string       // "romantic", "professional", "family", "friend", "other"
  Status            string       // "unnamed", "named", "clarified"
  Pronouns          []string     // ["she", "her"]
  Characteristics   []string     // Accumulated from all messages
  LastMentioned     int64        // Message index
  CreatedAt         int64        // When first appeared
  UpdatedAt         int64        // When last updated
  IsActive          bool         // Currently being discussed?
  Confidence        float64      // How sure are we this is correct?
}
```

### Active Contacts Tracking Per Message

**Store with each message:**
```
Message {
  index: 2
  text: "She's submissive..."
  activeContacts: [
    {id: "contact_1", pronouns: ["she", "her"], confidence: 0.95}
  ]
  newContacts: []
  ambiguousContacts: []
}
```

### Pronoun Resolution Algorithm

**When pronoun appears (e.g., "she"):**

```
Step 1: Check if pronoun explicitly links to named contact
  Input: "She [Emily] is..."
  Result: contact = Contact("Emily") ✓

Step 2: Check if pronoun matches single unnamed contact
  Active: 1 unnamed female contact (contact_1)
  Input: "She is..."
  Result: contact = contact_1 ✓ (high confidence)

Step 3: Check if pronoun matches multiple contacts
  Active: 2 unnamed female contacts (contact_1, contact_2)
  Input: "She is..."
  Result: AMBIGUOUS - Need clarification ⚠️

Step 4: Check if pronoun matches last-mentioned contact
  Last mentioned: contact_1
  Other active: contact_2 (not recently mentioned)
  Input: "She is..."
  Result: contact = contact_1 (assume continuity)

Step 5: If no match, create new unnamed contact
  Input: "She is... [context suggests new person]"
  Result: NEW contact_3 + Ask: "Is this someone new?"
```

---

## PHASE 3: DISAMBIGUATION STRATEGY

### When Is There Ambiguity?

**Ambiguity Type 1: Multiple Possible Contacts**
```
Active contacts: 
  - contact_1: unnamed female (girlfriend)
  - contact_2: unnamed female (colleague)

User says: "She's dominant"

Ambiguity: Which "she"?
Confidence: 50% (coin flip)
Action: ASK FOR CLARIFICATION
```

**Ambiguity Type 2: New vs Existing Contact**
```
Previous context: contact_1 (girlfriend)

User says: "I had this experience with a woman..."

Question: Same girlfriend or different woman?
Action: ASSUME same (pronoun/context continuity)
        OR ASK if context is unclear
```

**Ambiguity Type 3: User vs Contact Information**
```
"I had this experience with her. I felt vulnerable."

Question: "Vulnerable" = about user or about contact?
Action: ASK OR use verb patterns
        ("felt X" = user, "she felt X" = contact)
```

### Confidence Scoring

When determining pronoun resolution:

```
Confidence formula:
  Base: Number of possible contacts
    - 1 unnamed contact matching pronoun? 0.95
    - Multiple unnamed contacts? 0.40
    - Named contact explicitly referenced? 0.99
    - Generic pronoun with no context? 0.20

Adjustment:
  + Recent mention (within 1-2 messages): +0.15
  + Explicit type match ("colleague", "girlfriend"): +0.20
  + Pronouns previously used for this contact: +0.10
  - Conflicting type (colleague vs romantic): -0.30
  - Ambiguous pronouns (they/them): -0.10

Threshold:
  >= 0.80: High confidence - assume
  0.50-0.80: Medium - ask OR assume with note
  < 0.50: Low confidence - ASK FOR CLARIFICATION
```

---

## PHASE 4: CLARIFICATION STRATEGY

### When Should We Ask?

**Threshold 1: Ambiguous Pronouns (< 0.50 confidence)**
```
User: "She's different from her"
System confidence: 0.40

→ Ask clarification immediately
  "You mentioned a girlfriend and a colleague. 
   When you say 'she's different', do you mean:
   a) Your girlfriend
   b) Your colleague
   c) Someone else?"
```

**Threshold 2: Possible New Contact (0.50-0.70 confidence)**
```
User: "I had this experience with a woman"
System confidence: 0.60 (might be new person)

→ Ask OR note for later clarification
  "Are you talking about the same girl from before,
   or someone different?"
```

**Threshold 3: High Confidence (>= 0.80)**
```
User: "She's submissive" (only 1 unnamed female)
System confidence: 0.95

→ Assume, don't ask
  Update contact_1 with characteristic
```

### Clarification Modal - YES or NO?

**I recommend: YES - Use Modal (Separate Clarifications)**

**Reasoning:**

1. **Clear distinction between:**
   - Main conversation flow (response to user)
   - Clarifications (questions about understanding)

2. **Prevents signal loss:**
   - Clarifications don't get mixed into extracted data
   - Main analysis continues uninterrupted
   - Can revisit clarifications without re-processing main message

3. **Better UX:**
   - User sees: "I'm clarifying something... [MODAL]"
   - vs: "Wait, before I respond... [INLINE]"
   - Modal signals "this is a question about context, not about your goal"

4. **Architectural cleanness:**
   - Main workflow: Extract → Analyze → Respond
   - Side workflow: Clarification questions (modal)
   - Keep concerns separate

### Clarification Modal Structure

```
CLARIFICATION MODAL (appears BEFORE main response)
┌─────────────────────────────────────────────────┐
│ I need to clarify something                      │
├─────────────────────────────────────────────────┤
│ You mentioned:                                   │
│  • A girlfriend (unnamed)                        │
│  • A colleague (unnamed)                         │
│                                                  │
│ When you say "she's different", do you mean:    │
│  ○ Your girlfriend                              │
│  ○ Your colleague                               │
│  ○ Someone else?                                │
├─────────────────────────────────────────────────┤
│ [ Girlfriend ]  [ Colleague ]  [ Someone else ] │
└─────────────────────────────────────────────────┘

THEN MAIN RESPONSE FOLLOWS
┌─────────────────────────────────────────────────┐
│ Based on what you told me...                    │
│ [Main response about the goal]                  │
└─────────────────────────────────────────────────┘
```

### When NOT to Ask

**Don't ask if:**
- Confidence >= 0.80 (high confidence in pronoun resolution)
- Only one possible contact (no ambiguity)
- User explicitly clarified in message ("I mean the colleague...")
- Context is completely clear

**Do ask if:**
- Multiple possible contacts with same pronoun
- New contact might be being introduced
- Statement could apply to user or contact
- High impact on understanding (goal/characteristics)

---

## PHASE 5: CONTACT NAMING

### Progressive Naming Flow

**Stage 1: Detection (Message 1)**
```
User: "I wanna talk about a girl I'm dating"
System: 
  Create contact_1
  name: NULL
  status: "unnamed"
  pronouns: ["she", "her"]
```

**Stage 2: Accumulation (Message 2-4)**
```
User: "She's submissive and likes..."
System:
  Resolve "she" → contact_1
  Add characteristics
  Still no name requested
```

**Stage 3: Naming Opportunity (Message 5)**
```
User: "Oh, her name is Emily by the way"
System detects:
  "name is [X]" pattern
  "her" → contact_1
  Update: contact_1.name = "Emily"
  Update: contact_1.status = "named"
  Link all previous "she" to now-named "Emily"
```

**Stage 4: Name Confirmation**
```
User: "Emily asked me..."
System:
  "Emily" explicitly referenced
  Confidence in contact_1 = 0.99
  Update lastMentioned timestamp
```

### Should We EVER Ask For Name?

**Only if:**
1. Multiple ambiguous unnamed contacts exist
2. AND we need to disambiguate between them
3. AND context doesn't resolve it

**Example where we ask:**
```
Active contacts:
  - contact_1: unnamed female
  - contact_2: unnamed female

User: "She said something about..."
System: Can't resolve

Ask: "You mentioned a girlfriend and a colleague.
     When you say 'she', which one?
     
     And do you mind if I call them by nicknames
     while you decide on names? Like 'girlfriend'
     and 'colleague'?"
```

**Best practice:** Let names arrive naturally, don't force them

---

## PHASE 6: COMPLETE WORKFLOW - DETAILED

### Entry Point: New Message Arrives

```
MESSAGE ARRIVES
  │
  ├─ Step 1: DETECT CONTACTS
  │  │
  │  ├─ Scan for explicit contact mentions
  │  │  ("girl I'm dating", "my colleague", etc.)
  │  │
  │  ├─ Scan for pronouns
  │  │  ("she", "he", "they")
  │  │
  │  └─ Scan for named references
  │     ("Emily", "Marcus", etc.)
  │
  ├─ Step 2: RESOLVE PRONOUNS TO CONTACTS
  │  │
  │  ├─ For each pronoun found:
  │  │  a) Is it explicitly linked to name? (Yes → High confidence)
  │  │  b) Does it match single active unnamed? (Yes → 0.95 confidence)
  │  │  c) Does it match multiple active unnamed? (Yes → Ambiguous)
  │  │  d) Does it continue last-mentioned contact? (Maybe → 0.70)
  │  │  e) Is it a new contact? (Maybe → Create new)
  │  │
  │  └─ Calculate confidence for each resolution
  │
  ├─ Step 3: DETECT AMBIGUITIES
  │  │
  │  ├─ Pronoun ambiguity (0.50 confidence or less)?
  │  │  → Mark for clarification
  │  │
  │  ├─ New vs existing contact?
  │  │  → Mark for potential clarification
  │  │
  │  └─ User vs contact information?
  │     → Mark for potential clarification
  │
  ├─ Step 4: GATHER CLARIFICATIONS NEEDED
  │  │
  │  ├─ Collect all ambiguous resolutions
  │  ├─ Calculate if clarification is REQUIRED (< 0.50 confidence)
  │  ├─ Calculate if clarification is OPTIONAL (0.50-0.80 confidence)
  │  │
  │  └─ Determine clarification priority
  │     (Must ask now vs can ask later)
  │
  ├─ Step 5: SHOW CLARIFICATION MODAL (if needed)
  │  │
  │  ├─ IF clarification needed (<0.50 confidence):
  │  │  Show modal with options
  │  │  Wait for user response
  │  │  Update contact resolutions based on answer
  │  │
  │  └─ IF only optional clarifications (0.50-0.80):
  │     Add to Layer 4 gaps (ask in context of main response)
  │
  ├─ Step 6: EXTRACT WITH KNOWN CONTACTS
  │  │
  │  ├─ Now extraction has:
  │  │  - List of active contacts (resolved)
  │  │  - Their names (if known) or IDs (if unnamed)
  │  │  - Confidence levels
  │  │
  │  ├─ Pass to extraction:
  │  │  "Previous contacts: contact_1 (emily, romantic),
  │  │   contact_2 (marcus, professional, unnamed)"
  │  │
  │  └─ LLM extracts with context of WHO information is about
  │
  ├─ Step 7: UPDATE CONTACTS
  │  │
  │  ├─ For each characteristic extracted:
  │  │  Link to specific contact (from Step 6)
  │  │
  │  ├─ Check for new names:
  │  │  "Her name is Sarah" → Update contact_X.name
  │  │
  │  ├─ Check for type updates:
  │  │  "Actually, we're not dating anymore" → Update type?
  │  │
  │  └─ Store to database
  │
  └─ Step 8: CONTINUE TO MAIN RESPONSE
     Now all contacts are properly identified
     All characteristics linked to correct contacts
     No ambiguity about WHO
     Proceed with Layers 1-11 normally
```

---

## PHASE 7: ARCHITECTURAL LAYERS

### Where Does This Fit in Existing System?

```
Current:
┌─────────────────────┐
│ Message Arrives     │
└────────────┬────────┘
             │
             ▼
┌─────────────────────┐
│ Layer 1: Extract    │ ← Broken (no contact context)
└─────────────────────┘

NEW:
┌─────────────────────┐
│ Message Arrives     │
└────────────┬────────┘
             │
             ▼
┌─────────────────────────────────────┐
│ PRE-LAYER-1: Contact Workflow       │  ← NEW LAYER
│ 1. Detect contacts                  │
│ 2. Resolve pronouns                 │
│ 3. Disambiguate                     │
│ 4. Ask clarifications (modal)       │
│ 5. Update contact DB                │
│ 6. Build contact context            │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│ Layer 1: Extract (WITH CONTEXT)     │ ← Now has contact info
│ "Previous contacts were: [list]"    │
└─────────────────────────────────────┘
```

### New Component: Contact Context Builder

```go
type ContactContextBuilder struct {
  activeContacts []Contact
  knownNames map[string]Contact
  pronounMap map[string][]Contact
  ambiguities []AmbiguityMarker
  
  methods:
    DetectContacts(message string) []Contact
    ResolvePronoun(pronoun string) (Contact, Confidence)
    DetectAmbiguities() []Ambiguity
    GetClarifications() []ClarificationQuestion
    BuildContextString() string // "Previous contacts were..."
}
```

---

## PHASE 8: DECISION TREE - YES/NO ANSWERS

### Should We Use Modal for Clarifications?

**YES - Use Modal because:**
1. ✓ Clear separation of concerns (context vs goal)
2. ✓ Don't lose clarifications in main response
3. ✓ Can revisit for future disambiguations
4. ✓ UX clarity (user knows what's being asked)
5. ✓ Prevents data contamination (main analysis stays clean)

### Should We Ask for Names?

**NO - Let them come naturally because:**
1. ✓ Reduces friction
2. ✓ Real conversations don't force names
3. ✓ Can identify contacts by ID/pronouns
4. ✓ Names arrive progressively anyway
5. ✗ Only ask if truly ambiguous (multiple unnamed contacts)

### Should We Ask for Pronouns?

**RARELY because:**
1. ✓ Usually implied by language ("a girl" = she/her)
2. ✗ Assuming pronouns can be wrong (non-binary)
3. ✓ Ask only if ambiguous AND matters for analysis

**When to ask:**
- "They mentioned..." (could be singular they or plural)
- Contact type unclear (could be any pronouns)

### Should We Create New Contact or Update Existing?

**Decision:**
1. Explicit name match → Update existing
2. Pronoun match to single unnamed → Update existing
3. Pronoun match to multiple unnamed → Ask clarification
4. New type introduced → Create new
5. "And also..." or "I also have..." → Create new

---

## PHASE 9: ERROR RECOVERY

### What If We Guess Wrong?

**User corrects us:**
```
Moly: "So your girlfriend..."
User: "Actually, that's my colleague"

System:
  1. Detect correction
  2. Swap contact attribution
  3. Update ALL characteristics for both contacts
  4. Log the correction
  5. Increase confidence that this is the right contact going forward
```

### What If Contact Type Changes?

**User reveals new information:**
```
User: "Actually, we broke up"

System:
  1. Update contact_1.type from "romantic" to "past_romantic" or "friend"
  2. Note when change occurred (which message)
  3. Treat characteristics differently if needed
  4. Don't re-ask old questions about current status
```

### What If Contact Gets Named Late?

**Message 10 arrives:**
```
User: "By the way, his name is Michael"

System:
  1. Detect "his name is Michael"
  2. Match "his" to current context (last-discussed male contact)
  3. Update contact_2.name = "Michael"
  4. Verify: Update all "he" references to "Michael"
  5. Update database
  6. No need to re-analyze previous messages
```

---

## PHASE 10: WORKFLOW PSEUDOCODE

```python
def process_message_with_contact_workflow(message):
    # Pre-Layer-1: Contact Workflow
    
    # Step 1: Detect possible contacts
    possible_contacts = detect_contacts(message)
    active_contacts = get_active_contacts_from_db()
    
    # Step 2: Resolve pronouns to contacts
    contact_resolutions = {}
    for pronoun in message.pronouns:
        contact, confidence = resolve_pronoun(
            pronoun, 
            active_contacts, 
            message
        )
        contact_resolutions[pronoun] = (contact, confidence)
    
    # Step 3: Detect ambiguities
    ambiguities = detect_ambiguities(contact_resolutions)
    clarifications_needed = [a for a in ambiguities if a.confidence < 0.50]
    
    # Step 4: Handle required clarifications
    if clarifications_needed:
        clarification_answers = show_clarification_modal(clarifications_needed)
        update_contact_resolutions(contact_resolutions, clarification_answers)
    
    # Step 5: Check for naming opportunities
    name_updates = detect_name_updates(message, contact_resolutions)
    for contact_id, new_name in name_updates.items():
        update_contact_name(contact_id, new_name)
    
    # Step 6: Build context for extraction
    context_string = build_contact_context(active_contacts)
    
    # Step 7: Extract with contact context
    extraction_input = ExtractionPhaseInput(
        message=message,
        contact_context=context_string,
        active_contacts=active_contacts
    )
    
    # Step 8: Update contacts with extracted info
    extracted_context = extract(extraction_input)
    for contact_id, characteristics in extracted_context.contact_characteristics.items():
        update_contact_characteristics(contact_id, characteristics)
    
    # Now proceed to Layer 1+ with proper contact understanding
    return proceed_to_main_layers(...)
```

---

## SUMMARY: Key Architectural Decisions

| Decision | Choice | Reasoning |
|----------|--------|-----------|
| **Clarification UX** | Modal (separate) | Clear distinction, no data loss |
| **Force Name Asking** | NO - natural only | Reduces friction, names come naturally |
| **Pronoun Resolution** | Confidence-based | Handle ambiguity intelligently |
| **New vs Existing** | Explicit detection + ask | Prevent contact duplication |
| **Where in flow** | PRE-Layer-1 | Extract gets contact context |
| **Contact storage** | By ID + name | Support unnamed and progressive naming |
| **Ambiguity threshold** | 0.50 confidence | Below = must ask, above = can assume |

---

## NEXT STEPS

1. **Implement Contact Context Layer** (PRE-Layer-1)
2. **Build Contact Resolution Engine** (pronoun/name matching)
3. **Create Clarification Modal UI** (separate from main response)
4. **Update Extract() to receive contact context**
5. **Test with real conversation patterns** (unnamed → named → multiple contacts)

This workflow solves:
- ✓ WHO detection
- ✓ Pronoun resolution
- ✓ Contact disambiguation
- ✓ Progressive naming
- ✓ False contradiction prevention
- ✓ Multiple contact handling

