# Infrastructure Audit - What Exists vs What's Needed

**Date:** October 8, 2026  
**Status:** COMPREHENSIVE EXISTING INFRASTRUCTURE FOUND  
**Recommendation:** Build on existing foundations, don't reinvent

---

## SUMMARY

The codebase already has **significant infrastructure** for handling contacts, pronouns, and clarifications. The design I proposed can largely build on existing structures rather than starting from scratch.

---

## EXISTING INFRASTRUCTURE

### 1. CONTACT MANAGEMENT

**File:** `models/conversation_types.go`

**Contact Struct Fields:**
```go
type Contact struct {
  ID                       int64        // Unique ID
  UserID                   string       // Which user
  Name                     string       // ← ISSUE: Required (not optional for unnamed)
  Relationship             string       // romantic|professional|family|friend|other
  Age                      string
  Characteristics          []string     // Accumulated traits
  Interests                []string
  CommunicationPreferences string
  Notes                    string
  FirstMentionedAt         int64        // When first appeared
  LastMentionedAt          int64        // When last mentioned (Gap #1)
  ExtractionCount          int          // How many times mentioned
  Confidence               float64      // Extraction confidence
  CreatedVia               string       // "conversation", "manual", "import"
  Status                   string       // "active", "archived"
  Version                  int64        // Optimistic locking
  Reflections              []Reflection // Associated insights
  CreatedAt, UpdatedAt     int64
}
```

**ContactRepository Methods:**
- `Save()` - Create or update contact
- `GetByID()` - Retrieve by ID
- `GetByName()` - Retrieve by name
- `GetByUserID()` - Get all contacts for user
- `GetByRelationship()` - Filter by type
- `Update()` - Update existing
- `Delete()` - Soft delete
- `AddTrait()` - Add single trait
- `GetAll()` - Get all active
- `SaveExtractedContact()` - Save from extraction
- `RecordContactMention()` - Track mentions
- `UpdateFromClarification()` - Update from user correction
- `MarkExtractionSuperseded()` - Handle corrections

**Status:** ✅ GOOD - Has CRUD operations and mention tracking

**Gap:** ❌ Name field is required (can't store unnamed contacts)

---

### 2. CONTACT DEDUPLICATION

**File:** `database/contact_deduplicator.go`

**Handles:**
- Generic→Specific naming (Not specified→Kyle)
- Pronoun resolution (He→Kyle)
- Duplicate detection
- Merge decisions with confidence

**Methods:**
- `CheckForDuplicate()` - Detect duplicates
- `CheckForDuplicateWithConversation()` - With history context
- `handleGenericName()` - Handle unnamed→named
- `handleGenericNameWithConversation()` - With conversation context
- `resolveFromConversationHistory()` - Use prior messages
- `extractMentionedContactsFromMessages()` - Get all contacts from history
- `handleSpecificNameIdentifyingGeneric()` - Resolve by name
- `mergeContactsAndUpdate()` - Merge two contacts
- `calculateSimilarity()` - Score match confidence
- `calculateTraitOverlap()` - Compare characteristics
- `DeduplicateBySubject()` - Subject-based merging

**Status:** ✅ EXCELLENT - Already handles generic→specific naming and deduplication

**Gap:** ❌ No clarification modal for user approval when merging

---

### 3. PRONOUN RESOLUTION

**File:** `tools/pronoun_resolver.go`

**PronounReference Struct:**
```go
type PronounReference struct {
  Pronoun     string    // "she", "he", "they", "it", "both"
  PronounType string    // "personal", "demonstrative", "relative", "possessive"
  SentenceNum int       // Position in message
  Position    int       // Position in sentence
  Confidence  float64
}
```

**PronounResolution Struct:**
```go
type PronounResolution struct {
  ID              int64
  UserID          string
  ConversationID  string
  Pronoun         string
  PronounType     string
  AntecedentType  string // "name", "contact", "group", "concept", "unknown"
  AntecedentValue string // "Christine", "my boss", etc.
  AntecedentID    *int64 // FK to contacts
  MessageID       string
  Confidence      float64
  EvidenceText    string
  ResolutionMethod string // "linguistic_match", "llm_reasoning", "user_clarification", "context"
  ScopeStartSeq   int    // When valid from
  ScopeEndSeq     *int   // When invalid (NULL = ongoing)
  IsActive        bool
}
```

**PronounResolver Methods:**
- `DetectPronouns()` - Find all pronouns in message
- `ResolveAntecedent()` - Map pronoun to what it refers to
- `extractAntecedentFromSentence()` - Parse sentence context
- `extractAntecedentFromMessage()` - Use message context
- `isLikelyAntecedent()` - Score candidate antecedent
- `isFemaleName()`, `isMaleName()` - Check pronoun-name agreement
- `MarkScopeEnd()` - Mark when resolution becomes invalid
- `IsResolutionValid()` - Check if resolution still applies
- `SaveResolution()` - Store to database
- `GetResolutionsForPronoun()` - Retrieve saved resolutions
- `IsGroupPronoun()` - Detect plural pronouns
- `DetectGroupMembers()` - Find members of group reference

**Database Schema (migration 032):**
```sql
CREATE TABLE pronoun_resolutions (
  id, user_id, conversation_id,
  pronoun, pronoun_type,
  antecedent_type, antecedent_value, antecedent_id,
  message_id, sentence_position,
  confidence, evidence_text, resolution_method,
  scope_start_message_id, scope_start_seq,
  scope_end_message_id, scope_end_seq,
  is_active,
  created_at, updated_at,
  FOREIGN KEY (antecedent_id) REFERENCES contacts(id)
)
```

**Status:** ✅ EXCELLENT - Pronoun detection, resolution, scope tracking all in place

---

### 4. SUBJECT RESOLUTION

**File:** `agents/subject_resolver.go`

**Purpose:** Map subject references (pronouns, names) to specific entities

**Methods:**
- `ResolveAmbiguousPronoun()` - Convert pronoun to specific subject
- `ResolveSubject()` - Resolve any subject reference

**Status:** ✅ GOOD - Basic subject resolution exists

**Gap:** ❌ Doesn't use confidence scoring or clarification

---

### 5. CLARIFICATION CONTEXT

**File:** `models/agent_types.go`

**ClarificationContext Struct:**
```go
type ClarificationContext struct {
  Type                      string
  Confidence                float64
  RelatedPreviousExtraction *ExtractedEntity
  RequiresFollowUp          bool
  SuggestedFollowUpQuestion string
}
```

**Status:** ✅ FOUNDATION EXISTS - Basic clarification structure in place

**Gap:** ❌ Not wired to UI modal or user response handling

---

### 6. EXTRACTED CONTACT

**File:** `models/agent_types.go`

**ExtractedContact Struct:**
```go
type ExtractedContact struct {
  Name         string
  Relationship string
  Traits       []string
  Confidence   float64
  Evidence     string
}
```

**Status:** ✅ GOOD - Extraction result structure exists

**Gap:** ❌ No Pronouns field, no unnamed/progressive naming support

---

### 7. EXTRACTED ENTITY SUBJECT FIELD

**File:** `models/agent_types.go`

**ExtractedEntity has:**
```go
Subject string // WHO has this property: "user", contact name, or pronoun
```

**Status:** ✅ GOOD - Structure exists for subject tagging

**Gap:** ❌ Not being populated correctly (discovered earlier)

---

## WHAT'S MISSING OR INCOMPLETE

### Gap 1: UNNAMED CONTACT SUPPORT ❌

**Issue:** Contact.Name is required field, can't store unnamed contacts

**Needed:**
- Make Name nullable: `Name *string`
- Add placeholder names: "Unknown_Female_1", "Unnamed_Colleague"
- Track unnamed status
- Support renaming mid-conversation

**Effort:** Low - Schema change + nullable handling

---

### Gap 2: PRONOUNS FIELD IN CONTACT ❌

**Issue:** Contact model has no Pronouns field

**Needed:**
- `Pronouns []string` field in Contact
- Track: she/her, he/him, they/them, etc.
- Use for pronoun resolution confidence

**Effort:** Low - Add field + database migration

---

### Gap 3: PRE-LAYER-1 CONTACT WORKFLOW ❌

**Issue:** No orchestrated contact workflow before extraction

**Needed:**
- ContactContextBuilder component
- Orchestration logic: detect → resolve → disambiguate
- Integration point before Extract()

**Effort:** Medium - New layer, orchestration logic

---

### Gap 4: CONFIDENCE-BASED CLARIFICATION ❌

**Issue:** Pronoun resolution exists but doesn't trigger clarifications

**Needed:**
- Threshold logic (< 0.50 = ask)
- Clarification modal integration
- User response handling
- Resolution update from responses

**Effort:** Medium - Logic + UI integration

---

### Gap 5: MODAL UI FOR CLARIFICATIONS ❌

**Issue:** Clarification structures exist but no UI

**Needed:**
- Modal component (separate from main response)
- Display contact options
- Collect user selection
- Handle "skip" / "other"

**Effort:** Medium-High - UI/UX design + implementation

---

### Gap 6: PROGRESSIVE NAMING ❌

**Issue:** No system for updating contact names after creation

**Needed:**
- Detect "name is X" patterns
- Match to existing contact via pronouns
- Rename without losing data
- Update all pronoun resolutions

**Effort:** Low-Medium - Pattern matching + update logic

---

### Gap 7: ACTIVE CONTACTS TRACKING ❌

**Issue:** No per-message tracking of which contacts are being discussed

**Needed:**
- Track active contacts per message
- Update when new contacts introduced
- Use for pronoun resolution context
- Clear inactive contacts

**Effort:** Low - Tracking structure + updates

---

### Gap 8: CONFIDENCE SCORING FOR PRONOUN RESOLUTION ❌

**Issue:** Existing pronoun resolution has confidence but no scoring algorithm

**Needed:**
- Implement scoring formula (as designed)
- Base score, adjustments, threshold
- Update SaveResolution to use confidence tiers

**Effort:** Low - Algorithm implementation

---

## IMPLEMENTATION ROADMAP

### Phase 1: Schema & Data Model Updates (LOW EFFORT)
```
1. Make Contact.Name nullable
   ├─ Alter contacts table
   └─ Update Contact struct

2. Add Pronouns field to Contact
   ├─ Alter contacts table (add pronouns_json)
   ├─ Update Contact struct
   └─ Update ContactRepository

3. Add placeholder naming for unnamed
   └─ Helper function: GeneratePlaceholderName()
```

### Phase 2: Contact Context Layer (MEDIUM EFFORT)
```
1. Create ContactContextBuilder
   ├─ DetectContacts()
   ├─ ResolvePronouns()
   ├─ DetectAmbiguities()
   └─ BuildContextString()

2. Integrate into message processing
   ├─ Call before Extract()
   ├─ Pass context to Extract
   └─ Update extraction prompt
```

### Phase 3: Clarification Modal (MEDIUM-HIGH EFFORT)
```
1. Wire ClarificationContext to UI
   ├─ Modal component
   ├─ Options display
   └─ User response collection

2. Implement clarification workflow
   ├─ Check confidence thresholds
   ├─ Show modal if needed
   ├─ Update resolutions from response
   └─ Continue to main response
```

### Phase 4: Progressive Naming (LOW-MEDIUM EFFORT)
```
1. Detect naming patterns
   ├─ "name is X" detection
   ├─ Pronoun linking
   └─ Contact matching

2. Implement renaming
   ├─ Update Contact.Name
   ├─ Update pronoun resolutions
   ├─ Preserve all characteristics
   └─ Log the change
```

### Phase 5: Testing & Integration (MEDIUM EFFORT)
```
1. Test with real conversation patterns
   ├─ Unnamed → named
   ├─ Multiple unnamed contacts
   ├─ Pronoun disambiguation
   └─ Progressive naming

2. End-to-end testing
   ├─ Extract receives correct context
   ├─ Subject tagging works
   └─ No false contradictions
```

---

## RECOMMENDATION

**DO NOT START FROM SCRATCH**

The existing infrastructure is solid and covers:
- ✅ Contact storage and retrieval
- ✅ Deduplication and merging
- ✅ Pronoun detection and resolution
- ✅ Subject tracking
- ✅ Confidence scoring
- ✅ Database schema

**WHAT TO BUILD:**

1. **Short term:** Fix Contact model to support unnamed (nullable Name)
2. **Medium term:** Implement ContactContextBuilder orchestration layer
3. **Medium term:** Wire ClarificationContext to modal UI
4. **Short term:** Implement progressive naming
5. **Verification:** Test with real conversation patterns

**ESTIMATED EFFORT:**

- Schema updates: 2-3 hours
- ContactContextBuilder: 4-6 hours
- Clarification modal wiring: 6-8 hours
- Progressive naming: 2-3 hours
- Testing: 4-6 hours

**Total: 18-26 hours of implementation**

---

## FILES TO MODIFY

**High Priority:**
- `models/conversation_types.go` - Make Name nullable, add Pronouns
- `database/contact_repository.go` - Update CRUD operations
- `database/migrations/*.sql` - Schema changes

**Medium Priority:**
- `agents/context_extractor.go` - Update extraction prompt
- New file: `agents/contact_context_builder.go` - NEW layer
- UI files (TBD) - Clarification modal

**Low Priority (Existing):**
- `tools/pronoun_resolver.go` - Minor enhancements
- `database/contact_deduplicator.go` - Already good

---

## CRITICAL INSIGHT

The existing infrastructure is **80% of what we need**. The design I proposed can integrate with and enhance the existing system rather than replace it.

This is actually **better than starting fresh** because:
1. ✅ Database schema already proven
2. ✅ Pronoun resolution already implemented
3. ✅ Contact merging logic exists
4. ✅ Confidence scoring framework ready
5. ✅ Deduplication handles generic→specific

We just need to:
1. Wire it together with ContactContextBuilder
2. Add clarification modal UI
3. Support unnamed contacts (nullable Name)
4. Implement progressive naming
5. Test end-to-end

