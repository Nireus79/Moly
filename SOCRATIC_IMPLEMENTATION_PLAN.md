# Socratic Integration Implementation Plan

## Project Overview
**Goal:** Transform Moly from keyword-based safety checking and template-based questions into a context-aware, principle-driven system that uses Socratic dialogue to understand user intent before responding.

**Duration:** 4 phases, estimated 4-6 weeks  
**Key Success Metric:** False positive rate on safety checks drops to <5%

---

## Phase 0: Foundation & Planning (1 week)

### 0.1: Constitution Definition
**What:** Define Moly's explicit constitutional principles  
**Why:** Everything else builds on this - questions, safety checks, responses all align to principles  
**Output:** `moly-go/config/constitution.yaml`

**Key Deliverables:**
- Define 6 supreme principles (User Autonomy, Stakeholder Consideration, Harm Prevention, Transparency, Consent, Growth)
- Define 4 ethical frameworks (Kantian, Utilitarian, Virtue ethics, Rights-based)
- Create YAML schema and document principle → message-type mappings
- Review with user for clarity

**Dependencies:** None (independent)

---

### 0.2: Question Library Design
**What:** Catalog all Socratic questions Moly will use  
**Why:** These are the source of truth for intelligent questioning  
**Output:** `moly-go/config/questions/` (5 YAML files, ~40 questions total)

**Key Deliverables:**
- Define question structure (ID, text, approach, category, principle, framework, insights, depth, follow-ups)
- Create 8 stakeholder questions (identifying stakeholders approach)
- Create 8 consequence questions (exploring consequences)
- Create 8 principle questions (testing universality)
- Create 8 assumption questions (revealing assumptions)
- Create 8 alternative questions (exploring alternatives)
- Validate all 40 questions for clarity and uniqueness

**Dependencies:** 0.1

---

### 0.3: Database Schema Updates
**What:** Add columns to support Socratic tracking  
**Why:** Track questions and measure effectiveness  
**Output:** Migration script + SQL changes

**Key Deliverables:**
- Add 6 columns to `clarification_questions` table (approach, principle, framework, insights, depth, follow-ups)
- Create `principle_violations` table (tracks when responses violate principles)
- Create `question_effectiveness` table (tracks if questions help)
- Create migration script with rollback plan

**Dependencies:** None

---

## Phase 1: Core Infrastructure (1.5 weeks)

### 1.1: Models & Types
**Output:** `moly-go/models/socratic.go`

**Key Deliverables:**
- `Constitution` struct (principles + frameworks)
- `Principle` struct (name, severity, description, domains)
- `Framework` struct (name, principle, key questions)
- `SocraticQuestion` struct (as designed in 0.2)
- `QuestionLibrary` struct with helper methods

**Dependencies:** 0.2, 0.1

---

### 1.2: Configuration Loading
**Output:** `moly-go/config/loader.go`

**Key Deliverables:**
- `LoadConstitution()` - reads YAML, returns Constitution struct
- `LoadQuestionLibrary()` - reads all question files, builds searchable maps
- Wire into `main.go` initialization
- Add unit tests (constitution loads, questions validate, no duplicates)

**Dependencies:** 1.1, 0.1, 0.2

---

### 1.3: Question Selection Engine
**Output:** `moly-go/agents/socratic_question_selector.go`

**Key Deliverables:**
- `SocraticQuestionSelector` struct (holds library + constitution refs)
- `IdentifyAmbiguity()` - analyze context → outputs ambiguity types
- `MapAmbiguityToPrinciples()` - ambiguity → affected principles
- `GetUncoveredCategories()` - track which categories already asked
- `SelectSocraticApproach()` - ambiguity type → Socratic approach
- `SelectNextQuestion()` - orchestrator method
- Unit tests with >80% coverage

**Key Logic:**
```
IdentifyAmbiguity(context, message)
  ↓
MapAmbiguityToPrinciples(ambiguities)
  ↓
GetUncoveredCategories(affected_principles, previous_questions)
  ↓
SelectSocraticApproach(ambiguity_type)
  ↓
FindQuestion(approach, category)
  ↓
Return best question for situation
```

**Dependencies:** 1.1, 1.2, 0.2

---

## Phase 2: Integration with Existing Systems (1.5 weeks)

### 2.1: Update SafetyChecker
**Output:** Modified `moly-go/safety/checker.go`

**Key Changes:**
- Keep ONLY obvious crisis keywords (high-confidence only)
  - Suicide: "kill myself", "suicide", "want to die"
  - Threats: "bomb", "kill someone", "hurt someone"
  - Self-harm: "hurt myself", "harm myself"
- Remove aggressive LLM classification entirely
- Add logging for auditing
- Update `CheckMessage()` to call `detectObviousCrisisOnly()`

**Key Logic:**
```
CheckMessage(text)
  ├─ Obvious keywords? → Block + provide resources
  └─ No keywords? → Pass through
      (ambiguous cases handled by ConversationAgent → clarification → HarmAnalyzer)
```

**Testing:** 50 messages (10 obvious crises → all blocked, 40 innocent → all pass)

**Dependencies:** 1.2

---

### 2.2: Integrate into ConversationAgent
**Output:** Modified `moly-go/agents/conversation_agent.go`

**Key Changes:**
- Add `selector *SocraticQuestionSelector` field
- Modify `Run()` to call `selector.SelectNextQuestion()` instead of template-based generation
- Track covered question categories in Context
- Attach metadata (approach, principle, framework) to response
- Pass context + previous questions to selector

**Key Logic:**
```
ConversationAgent.Run()
  ├─ Generate response
  ├─ Select Socratic question (instead of deterministic)
  │   └─ selector.SelectNextQuestion(ctx, message, prevQuestions)
  ├─ Check response with HarmAnalyzer
  └─ Return response + question + metadata
```

**Testing:** End-to-end flow with real selector + context

**Dependencies:** 1.3, 2.1

---

### 2.3: Enhance HarmAnalyzer
**Output:** Modified `moly-go/tools/harm_analyzer.go`

**Key Changes:**
- Add `CheckPrinciples()` method (takes response + constitution)
- Implement principle-specific checks (autonomy, stakeholder, harm, transparency, consent)
- Log violations to `principle_violations` table
- Integrate into `AnalyzeResponse()` return value
- Keep existing harm detection logic, add principle layer on top

**Key Logic:**
```
AnalyzeResponse(response, context, constitution)
  ├─ Existing: Check for direct harm
  └─ NEW: Check against 6 principles
      ├─ User Autonomy: Does response pressure user?
      ├─ Stakeholder: Consider others' perspectives?
      ├─ Harm: Could response cause indirect harm?
      ├─ Transparency: Explain limitations?
      ├─ Consent: Assume consent?
      └─ Growth: Support flourishing?
```

**Testing:** 20+ response scenarios covering each principle

**Dependencies:** 1.1, 0.1

---

## Phase 3: Learning & Refinement (1 week)

### 3.1: Question Effectiveness Tracking
**Key Deliverables:**
- Track when user answers a question:
  - Did it reduce ambiguity?
  - Did it surface the expected insight?
  - Should depth progress to next level?
- Store in `question_effectiveness` table
- Use feedback to avoid repeating ineffective questions

**Dependencies:** 0.3, 2.2

---

### 3.2: Follow-Up Question Generation
**Key Deliverables:**
- `GenerateFollowUp()` - given previous question + answer, select/generate next
- Implement depth progression (1-5 levels)
- Test that follow-ups are contextual and don't repeat categories

**Dependencies:** 1.3

---

### 3.3: Frontend Updates
**Output:** Modified `moly-extension/src/sidebar/components/ChatInterface.tsx`

**Key Deliverables:**
- Display question metadata (targets_principle, socratic_approach)
- Show expected_insights before asking ("This helps me understand...")
- Display principle violations as badges (collapsible)
- Test responsive design and mobile

**Dependencies:** 2.2

---

## Phase 4: Monitoring & Iteration (1 week)

### 4.1: Metrics & Monitoring
**Key Deliverables:**
- Track safety accuracy: % of ambiguous messages correctly passed, % of crises correctly blocked, false positive rate
- Track question effectiveness: which questions helped most, which ambiguities hardest to resolve
- Track principle violations: most common violations, which principles hardest to satisfy
- Create monitoring queries for weekly reports

**Dependencies:** 3.1

---

### 4.2: Refinement & Iteration
**Key Deliverables:**
- Analyze metrics data
- Refine ambiguity detection based on failures
- Update question library (replace ineffective questions, adjust phrasing)
- Tune principle checks (reduce false positives)

**Dependencies:** 4.1

---

## Critical Path & Dependencies

```
Phase 0
├── 0.1 Constitution ──┐
├── 0.2 Questions ─────├→ Phase 1
├── 0.3 Database ──────┤   ├── 1.1 Models ──┐
                        │   ├── 1.2 Loader ─├→ Phase 2
                        └→  └── 1.3 Selector┤
                                            ├→ 2.1 SafetyChecker ──┐
                                            ├→ 2.2 ConversationAgent ──→ Phase 3 & 4
                                            └→ 2.3 HarmAnalyzer ────┘
```

**Critical Path:** 0.1 → 0.2 → 1.1 → 1.2 → 1.3 → 2.1 → 2.2 → 2.3 → 3.1 → 4.1 → 4.2

**Parallel Work:** 0.3 (database) can happen during Phase 0/1 without blocking others

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Questions too generic | Review with users, iterate |
| Selection logic broken | Extensive unit tests before integration |
| False positives increase | Keep obvious-keyword checks, monitor metrics |
| Principles too rigid | Start permissive, tighten based on data |
| Performance impact | Selector is lightweight, no external calls |

---

## Success Criteria

**End of Phase 2:**
- ✅ SafetyChecker false positive rate <10%
- ✅ Socratic questions display correctly
- ✅ Ambiguous message → question flow works end-to-end
- ✅ All existing tests still pass

**End of Phase 4:**
- ✅ False positive rate <5%
- ✅ 80%+ of questions marked as "helpful"
- ✅ Users understand why questions are asked
- ✅ System learns from patterns over time

---

## Backward Compatibility

**What stays unchanged:**
- ConversationAgent's response generation logic
- HarmAnalyzer's core harm-detection algorithm
- Existing database tables (only adding columns/tables)
- API response envelope structure

**What changes:**
- SafetyChecker's message analysis (keyword only, no LLM)
- Question generation (deterministic → Socratic selection)
- HarmAnalyzer output (adds principle violations)

---

## Implementation Approach

1. **Phase 0 (Foundation):** Complete before coding - these are design decisions
2. **Phase 1 (Infrastructure):** Build types and loading, no integration yet
3. **Phase 2 (Integration):** Wire together, test end-to-end
4. **Phase 3 (Learning):** Add effectiveness tracking
5. **Phase 4 (Monitoring):** Measure and iterate

Each phase has clear deliverables and dependencies. Only proceed to next phase when current phase is complete and tested.

