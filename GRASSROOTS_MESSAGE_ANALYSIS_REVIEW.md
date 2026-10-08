# Grassroots Review: How Moly Analyzes Each Message

**Purpose:** Understand the complete pipeline from message input to information usage  
**Scope:** Data structures, transformations, storage, and later usage  
**Date:** October 8, 2026

---

## PHASE 1: MESSAGE INPUT

### What Enters the System
```
User message (raw string): 
"Here are some insights from my profile. [his kinks]
Here are some from her's: [her kinks]
[his communication strategy]
So, how do I start it?"
```

### Entry Point: `main.go` line 998+

**ExtractionPhaseInput created with:**
- `UserID` - who sent it
- `ConversationID` - which conversation
- `MessageID` - unique identifier
- `Message` - the raw text
- `MessageCount` - how many messages so far
- `RecentMessages` - previous messages (if available)
- `PreviousExtraction` - extracted data from Message 1
- `UserProfile` - stored AboutMe data
- `Cache` - LLM cache for performance

**Key Question:** What context is known at this point?
- ✓ Message 1 goal (if loaded)
- ✓ Previous contact names
- ? Previous characteristic tags

---

## PHASE 2: EXTRACTION (Layer 1)

### Where It Happens: `agents/extraction_phase.go`

### Step 2.1: Context Extractor Processes Message
**File:** `agents/context_extractor.go`  
**Function:** `Extract(ctx, userMessage)`

**CRITICAL:** Only receives the CURRENT message, nothing else!

**LLM Extraction Prompt asks for:**
```
Extract and return JSON with:
- contact: {name, relationship, traits[], confidence, evidence}
- intention: main goal/purpose
- intentionPrinciples: [principles engaged]
- userCharacteristics: [traits about USER]
- contactCharacteristics: {name: [traits]}
- entities: [{name, type, confidence, evidence}]
```

**What the prompt SAYS it wants:**
- Tag subject: "USER|trait|confidence"
- Only extract linked characteristics

**What actually gets returned (parsing issue):**
The JSON comes back with characteristics as plain strings:
```json
{
  "userCharacteristics": ["USER|dominant|0.9", "USER|98% dominant|0.95"],
  "contactCharacteristics": {
    "Christine_sub": ["CONTACT_Christine_sub|submissive|0.9"]
  }
}
```

### Step 2.2: Parsing the LLM Response
**Problem Location:** `agents/context_extractor.go` lines 220-226

```go
// Extract userCharacteristics
if chars, ok := response["userCharacteristics"].([]interface{}); ok {
	for _, c := range chars {
		if char, ok := c.(string); ok {
			extracted.UserCharacteristics = append(extracted.UserCharacteristics, char)
		}
	}
}
```

**What happens:**
- ✓ Receives: `"USER|dominant|0.9"`
- ✓ Stores in: `extracted.UserCharacteristics`
- ✗ Does NOT parse the tag format
- ✗ Stores literal string with pipe delimiters

**Data stored in ExtractedContext:**
```go
type ExtractedContext struct {
	UserCharacteristics []string  // ["USER|dominant|0.9", "USER|98% dominant|0.95"]
	ContactCharacteristics map[string][]string // {"Christine_sub": ["CONTACT_Christine_sub|submissive|0.9"]}
	// ... plus Intention, Goals, Contact, etc.
}
```

**Key Issue:** 
- No structured Subject field populated here
- Subject information is embedded in STRING as delimiters
- Will need to be parsed later

---

## PHASE 3: ENTITY CONVERSION & SUBJECT ATTRIBUTION

### Where It Happens: `agents/extraction_phase.go` line 131+

### Step 3.1: Extract Maps ExtractedContext → ExtractionArtifact

**Process:**
1. Takes the ExtractedContext from Step 2.2
2. Converts to ExtractionArtifact for storage
3. Creates Entity objects for each piece of data

**Step 3.2: Convert Characteristics → Entities**
**Location:** `agents/extraction_phase.go` lines 143-175

For each UserCharacteristic string:
```go
// Line: Takes string like "USER|dominant|0.9"
// Need to parse it somehow...
```

**Question:** How does the characteristic "USER|dominant|0.9" become an Entity?

Let me search for where this parsing should happen...

---

## PHASE 4: SUBJECT ATTRIBUTION (FIX #69)

### Where It Happens: `agents/extraction_phase.go` lines 464-493

**Function:** `determineSubjectViaLLM()`

**Purpose:** For each characteristic, determine WHO it's about

**Current Logic:**
```go
for i, entity := range artifact.Entities {
	if entity.Type == "characteristic" {
		attribution := ep.determineSubjectViaLLM(
			ctx, 
			entity.Value,      // The trait (e.g., "dominant")
			message,           // Original message
			knownContacts      // Who's being discussed
		)
		artifact.Entities[i].Subject = attribution.Subject  // "user", contact name, etc.
	}
}
```

**LLM Call for Subject Attribution:**
Asks: "In this message, who is described as [trait]?"

**Problem:**
- The LLM is getting ENTITY.VALUE (just "dominant")
- The context is: message + known contacts
- But the characteristics were already extracted with subject tags in Step 2!
- **This is redundant and error-prone**

**Example:**
- Entity.Value = "dominant" 
- Known contacts = "Christine_sub"
- Message = "Here's my profile: I'm 98% dominant. Here's her's: She's submissive"
- **LLM must re-infer:** "who is dominant?" 
- **Should have been clear:** Tag was "USER|dominant|0.9"

---

## PHASE 5: DATA FLOW TO STORAGE

### Where Characteristics End Up

After extraction_phase.go processes the message:

**Saved to database:**
- Entities → Stored with Subject field
- Characteristics → Stored with Subject field
- Contact info → Stored with relationships

**Accumulated in LayerContext:**
```go
lc.AccumulatedExtractedEntities = artifact.Entities  // All entities from all messages
```

**Problem with accumulation:**
- If Subject attribution was wrong (failed LLM call)
- The wrong subject gets accumulated
- Used by all downstream layers

---

## PHASE 6: USAGE IN LAYERS

### Layer 4: Gap Detection
Uses accumulated entities to detect missing information

### Layer 5: Conflict Detection  
**Critical Usage Point**

```go
func (cd *ConflictDetector) detectCrossMessageContradictions(
	currentEntities []models.ExtractedEntity,
	accumulatedEntities []models.ExtractedEntity,
) []ConflictDetectorResult {
	
	for _, currentEnt := range currentEntities {
		for _, prevEnt := range accumulatedEntities {
			// Check if same subject
			if currentEnt.Subject != prevEnt.Subject {
				continue  // Different subjects = OK
			}
			
			// Check if antonyms
			antonym, hasAntonym := cd.antonymMap[currentEnt.Value]
			if antonym == prevEnt.Value {
				// FLAG CONTRADICTION!
			}
		}
	}
}
```

**If Subject Attribution Failed:**
- currentEnt.Subject = "user" (correct)
- prevEnt.Subject = "user" (WRONG - should be "christine_sub")
- Both are "dominant" vs "submissive" (antonyms)
- **FALSE CONTRADICTION FLAGGED**

---

## COMPLETE DATA FLOW DIAGRAM

```
┌─────────────────────────────────────────────────────────────┐
│ INPUT: User Message (raw string)                            │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ STEP 1: Context Extractor (context_extractor.go)            │
│ Input: userMessage only                                      │
│ Output: ExtractedContext                                     │
│                                                              │
│ UserCharacteristics = ["USER|dominant|0.9"]                 │
│ ContactCharacteristics = {"Christine": ["CONTACT_...|sub"]}  │
│                                                              │
│ ⚠️ PROBLEM: Subject tags embedded in STRING, not parsed     │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ STEP 2: Extract → Artifact Conversion (extraction_phase.go) │
│ Input: ExtractedContext                                      │
│ Output: ExtractionArtifact with Entities                    │
│                                                              │
│ Questions:                                                   │
│ - How do characteristics strings become Entity objects?     │
│ - Are subject tags parsed here?                             │
│ - If not, when are they parsed?                             │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ STEP 3: Subject Attribution via LLM (FIX #69)               │
│ Input: Entity.Value (e.g., "dominant"), message, contacts   │
│ Output: Entity.Subject = "user" or contact name             │
│                                                              │
│ ⚠️ REDUNDANT: Subject was already in the tag!               │
│ ⚠️ ERROR-PRONE: LLM may get it wrong                         │
│ ⚠️ LATE: Attribution happens AFTER extraction               │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ STEP 4: Accumulation (Layer 1 → LayerContext)               │
│ Input: Entities with (hopefully correct) Subject            │
│ Output: AccumulatedExtractedEntities                         │
│                                                              │
│ Used by: Layers 4, 5, 6+ for gap/conflict detection        │
│                                                              │
│ ⚠️ PROPAGATION: Wrong Subject tagging propagates forward    │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ STEP 5: Layer 5 Conflict Detection                          │
│ Input: Current entities + accumulated entities              │
│ Compare: Subject matches? → Value antonyms?                 │
│                                                              │
│ If Subject attribution was wrong in Step 3:                 │
│ - "dominant" (from user) vs "submissive" (from contact)    │
│ - Both have Subject="user" (WRONG)                          │
│ - Both are antonyms                                          │
│ - FALSE CONTRADICTION FLAGGED ❌                            │
└─────────────────────────────────────────────────────────────┘
```

---

## CRITICAL BREAKDOWN POINTS

### Breakdown #1: Subject Tag Parsing
**Location:** `context_extractor.go` line 220-226  
**Issue:** Characteristic string `"USER|dominant|0.9"` is stored as-is, not parsed  
**Impact:** Subject information lost before Step 3

### Breakdown #2: Redundant Subject Attribution  
**Location:** `extraction_phase.go` line 479-481  
**Issue:** LLM re-determines subject when it was already tagged in Step 1  
**Impact:** Introduces LLM uncertainty into something that should be deterministic

### Breakdown #3: Subject Attribution Timing
**Location:** After entities are created, before storage  
**Issue:** Subject attribution is a post-processing step  
**Impact:** If it fails, all downstream layers get wrong data

### Breakdown #4: No Subject Validation
**Location:** After subject attribution  
**Issue:** No check that Subject makes sense given the message and context  
**Impact:** False contradictions go undetected

---

## DATA STRUCTURE MISMATCH

### What's Sent to Extraction LLM
```json
{
  "userCharacteristics": ["USER|dominant|0.9", "creative|0.85"],
  "contactCharacteristics": {
    "Christine": ["submissive|0.95", "CONTACT_Christine|good-girl|0.90"]
  }
}
```

### What Should Be Stored
```go
type CharacteristicEntity struct {
  Value string      // "dominant"
  Subject string    // "user" or "christine"
  Confidence float64 // 0.9
  Evidence string   // Quote from message
}
```

### What Actually Gets Stored
```go
ExtractedContext.UserCharacteristics []string
// = ["USER|dominant|0.9", "creative|0.85"]
// Just strings, no structure
```

---

## WHERE WHO IS LOST

### Message 1 Input
```
"Help me craft a smart message to Christine. 
I'm 98% dominant. She's submissive and likes good girls."
```

### What Should Be Extracted
```
Characteristic {
  value: "dominant"
  subject: "user"      ← WHO has this trait
  confidence: 0.98
}

Characteristic {
  value: "submissive"
  subject: "christine" ← WHO has this trait
  confidence: 0.95
}
```

### What Actually Happens
```
Step 1: LLM returns string "USER|dominant|0.98"
Step 2: Parser stores as string (no parsing)
Step 3: LLM asked "who is dominant?" (redundant)
Step 4: Attributes subject = "user" (hopefully correct)
Step 5: Stored in DB with Subject field
Step 6: Later used in contradiction detection

If Step 3 fails:
- Subject could be wrong
- Contradictions are false positives
```

---

## THE REAL QUESTION

**How should characteristics flow through the system?**

### Current Path (BROKEN)
```
LLM extraction → String with tag → Parse lost → Re-determine subject → 
Hopefully correct → Store → Use in conflicts
```

### What Should Happen
```
LLM extraction → Structured data with subject → Validate subject → 
Store correctly → Use with confidence
```

---

## INVESTIGATION QUESTIONS TO ANSWER

1. **String Parsing:** Where (if at all) are strings like "USER|dominant|0.9" parsed?
2. **Entity Creation:** How do characteristics become Entity objects? When does Subject get set?
3. **Storage:** When entities are saved to DB, what Subject value is used?
4. **Accumulation:** What Subject values are in AccumulatedExtractedEntities?
5. **Conflicts:** When contradiction detection runs, what subjects does it see?

---

## NEXT STEPS

To properly understand the WHO problem, need to:

1. Trace string "USER|dominant|0.9" through the entire codebase
2. Find where (if anywhere) it gets parsed
3. Understand the exact data flow from extraction to conflict detection
4. Verify Subject field is correct at each stage
5. Identify where the breakdown actually occurs

Then we can properly fix subject attribution instead of just disconnecting things.


---

## VERIFIED FINDINGS

### What I Found

**The Subject Tag Format IS Intended**
- File: `models/agent_types.go` line 16  
- Comment: "FIX #5: Tagged characteristics about the user (format: "USER|trait|confidence")"
- Status: INTENDED but NEVER PARSED

**The Data Flow**

1. **LLM Extraction** returns JSON:
```json
{
  "userCharacteristics": ["USER|dominant|0.9", "creative|0.85"],
  "contactCharacteristics": {
    "Christine": ["CONTACT_Christine|submissive|0.9"]
  }
}
```

2. **Parsing** (context_extractor.go lines 220-226):
```go
// Just stores the STRING as-is
extracted.UserCharacteristics = ["USER|dominant|0.9", "creative|0.85"]
```
✗ Does NOT parse "USER|" tag  
✗ Does NOT extract subject  
✗ Does NOT extract confidence value  

3. **Entity Creation** (extraction_phase.go lines 187-192):
```go
for _, c := range extractedCtx.UserCharacteristics {
	artifact.Entities = append(artifact.Entities, models.ExtractedEntity{
		Type:       "characteristic",
		Value:      c,  // ← ENTIRE STRING "USER|dominant|0.9"
		Confidence: 0.85,  // ← OVERRIDDEN! Lost the 0.9
	})
}
```
✗ Stores entire tag string as Value  
✗ Loses extracted confidence  
✗ Subject field not set  

4. **Subject Attribution** (extraction_phase.go lines 464-493):
```go
attribution := ep.determineSubjectViaLLM(
	ctx,
	entity.Value,  // ← Passing "USER|dominant|0.9"!
	message,       // ← Original message
	knownContacts
)
```
✗ LLM gets malformed entity value  
✗ Re-determines subject redundantly  
✗ May fail or return wrong answer  

5. **Storage & Usage**:
Entity with Subject = (result of re-determination, possibly WRONG)

### The Damage

When subject attribution fails:
- "USER|dominant|0.9" → Subject determined as "user" or "christine" (guessed wrong)
- Message 2: "She's submissive" → "CONTACT_christine|submissive|0.9" → Subject guessed wrong
- Both characteristics end up with Subject="user"
- Contradiction detector sees: dominant (user M1) vs submissive (user M2)
- FALSE CONTRADICTION FLAGGED

---

## ROOT CAUSE SUMMARY

### Where WHO Is Lost

**Level 1: Intent (Broken)**
- Extraction prompt asks for subject tags ✓
- Parser ignores the tags ✗
- Subject information discarded ✗

**Level 2: Recovery Attempt (Unreliable)**
- Later, LLM is asked "who is this about?"
- LLM tries to re-infer from limited context
- Often gets it wrong
- Can't recover information already discarded

**Level 3: Cascade (Damage Spreads)**
- Wrong subject propagates through system
- Conflict detection uses wrong subjects
- False contradictions cascade through layers
- User sees redundant questions

---

## THE ARCHITECTURAL ERROR

The system was built with:
1. **Good intent:** Tag subject in extraction
2. **Missing implementation:** Parse the tags
3. **Bad recovery:** Re-determine subject later
4. **False confidence:** System thinks it knows WHO

This explains ALL the noise:
- Tone detection (false) - different people's tones conflicting
- Autonomy detection (false) - analyzing wrong person's autonomy
- Preferences extraction (false) - context about others extracted as preferences
- Contradiction detection (false) - characteristics of different people treated as contradictions

They're ALL rooted in the same problem: **WHO IS LOST AT PARSING TIME**

