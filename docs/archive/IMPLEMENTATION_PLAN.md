# Implementation plan: workflow and maturity (revision 2, 2026-10-09)

Sources: `INVESTIGATION_WORKFLOW_AND_MATURITY.md` (root causes RC1–RC11, M1–M8), `INVESTIGATION_REFERENCE_MATURITY.md` (reference logic), `DEAD_AND_DUPLICATE_CODE.md` (removal list).

## 0. Decisions (user, 2026-10-09)

| # | Decision |
|---|---|
| D1 | Maturity is built from what Moly already extracts and stores. No invented targets. |
| D2 | Goals are compared with the existing goal-coherence logic (same, related subgoal, different). Consent is **not** a maturity item; safety stays in Layer 2. |
| D3 | Readiness is per phase. |
| D4 | Schema change is an additive upgrade. Existing data is kept. |
| D5 | Maturity rises when the user provides new data. Answered questions only control repetition (existing clarification tracking). Confidence is not a weight: an extracted item is accepted or rejected, and counts once. |
| D6 | Maturity is per conversation. Reuse the existing tables and fields. |
| D7 | A named person is required. The name can be anything the user chooses, including a private label such as "Girl from fet". When a person has no name, Moly interrupts and asks for it in chat before continuing. The name is kept on the person record, which lives beyond the conversation, so a later conversation can reuse it. |
| D8 | A fact that is resolved or clarified increases maturity. |
| D9 | Decrease rule: see section 3. Topic change does not decrease maturity. |

## 1. Rules for this work (wiring and timing)

R1. **One request, one ordered pipeline:** save the user message → extract → load persisted state → run the goal comparison and the person check → run layers → decide → build the reply → persist state and reply in one transaction → return the reply. A step may read only what an earlier step produced, or what was persisted before the request started.
R2. **No cache may stand in for evaluation.** A cache is allowed only if it returns the identical result for the identical input. No threshold may skip a layer.
R3. **One value, one writer.** Each persisted quantity has one named writer (section 4.2).
R4. **Decisions are passed, not logged.** "Ask for name", "ask gap", "help" are fields of the context the agent reads.
R5. **Every reply has text or is an explicit error.** The sent reply and the saved reply are the same object.
R6. **Each phase has contract tests and a live check** (section 7). A phase is not done until its live check passes.
R7. **No deletion in the same change as a behaviour change.** Dead code is removed in phase 8.
R8. **No new keyword or regular-expression rules.** Extraction stays LLM-based. Existing regex rules that change behaviour are removed, not extended.
R9. **Timing is part of the contract.** Any change that adds LLM calls states the added latency. The extension timeout is 20 minutes.

## 2. What already exists and will be reused

| Need | Existing element | Location | Note |
|---|---|---|---|
| Goal extraction and lock | `Layer1` reuse path, primary goal lock | `agents/layer_adapters.go` ~221 | Fixed this session: locks the first non-empty goal. |
| Goal comparison | `GoalCoherence` (`same`, `related_subgoal`, `different`) | `agents/goal_coherence_handler.go`, `models/goal_models.go` | Use as is. |
| Person records | `contacts` table: `name`, `status` (`active`, `secondary`, `archived`, `unnamed`), characteristics, WHAT columns | `database/contact_repository.go`, `models/conversation_types.go` | Per user, so it outlives a conversation. |
| Placeholder names | `GeneratePlaceholderName` | `models/conversation_types.go:296` | Gives generic names ("Contact", "Friend"). Use for the interim label only. |
| Name clarification | Contact clarification (A/B options) | `agents/contact_response_formatter.go` | Currently the only name question. Its text is not shown (phase 1). |
| Name from a later message | `ProgressiveNamingDetector` | `agents/progressive_naming.go` | Regex-based. **Replace** (R8). |
| Asked and answered questions | `clarification_questions`, `MarkConversationQuestionsAnswered`, `AnsweredQuestionTexts` | `database/clarification_*.go` | Added this session. Controls repetition only. |
| Maturity per conversation | `conversation_maturity` (`current_phase`, `maturity`) | `database/schema.sql` ~163 | Write `current_phase` and `maturity` only. Accomplishment columns are unused. |
| Counts per conversation | `conversation_summaries` accumulated fields (`accumulated_entity_count`, `accumulated_contact_count`, `accumulated_values`, `accumulated_characteristics`) | `database/conversation_summary_repository.go` | Fix the write bug (RC2) and reuse. |
| Conflict resolution | Layer 5 conflict detection and the "replace" resolution | `agents/layer5_*.go`, `agents/conversation_agent.go` | The only path that decreases counts. |
| Subject of a trait | `subject` on extracted entities (`user`, a name, a pronoun) | `models/agent_types.go` | Fixed this session (RC9). |

## 3. What should decrease maturity

Principle: maturity measures how much Moly understands the user's situation. It drops only when something Moly understood is shown to be wrong. It does not drop because the situation changed.

**Decreases (the only cases):**
- The user corrects or retracts a fact, and the user's correction replaces it (Layer 5 "replace"). The fact's count is removed from the conversation.
- The user renames a person, or merges two people. The person count changes only by that correction, not by the rename itself.

**Does not decrease:**
- **Topic or goal change.** Moving to a new goal or person does not make the old one less understood. Each goal is a separate focus within the conversation. The active focus is the one being helped. Returning to an earlier goal finds its counts intact. Lowering the old focus on a topic change would punish the user for changing topic and would make a returning topic start from zero.
- Silence or no reply.
- Low-confidence noise. Confidence does not weight counts (D5).
- Time passing.

Implementation: each stored item carries the goal (focus) it belongs to, taken from `GoalCoherence`. Maturity is shown per focus and per phase. A focus with a "different" goal starts its own entries. The conversation's `maturity` value is the active focus's value.

## 4. Data model and writers

### 4.1 What is counted

Each item is counted once, per conversation, per focus, on the first time it is saved:

| Phase | Item | Counted when | Source |
|---|---|---|---|
| goal | goal (focus) | the primary goal is locked, or a related subgoal is accepted | Layer 1, `GoalCoherence` |
| person | person (contact) | a contact is saved for this conversation | `ContactRepository` save path |
| person | person has a name | the name is set from a non-placeholder value | name answer (phase 4) |
| person | person trait | a trait with subject = that person is saved | extraction, attribution |
| user | user trait, value, style | a characteristic, value or style with subject = user is saved | extraction, `about_me` |
| user | principle | an intention principle is saved | extraction |

Items are counted by a normalised key (person id, facet, value text). Repeats do not count.

### 4.2 Tables and writers

- **`conversation_maturity`** (existing): `current_phase`, `maturity` (display, derived). Writer: the maturity service only.
- **`maturity_items`** (new, additive, D4): `user_id, conversation_id, focus_id, phase, kind, item_key, created_at, PRIMARY KEY(user_id, conversation_id, focus_id, kind, item_key)`. Writer: maturity service. One row per counted item.
- **`maturity_events`** (new, additive): `id, user_id, conversation_id, focus_id, phase, delta, reason, source_message_id, created_at`. Writer: maturity service. History for the "must rise each circle" check and for the decrease audit.
- **`conversation_summaries`** accumulated fields (existing): written by the summary path only, after the reply is persisted (R1). Read by the next message.
- **`schema.sql`**: SchemaVersion 3, with `CREATE TABLE IF NOT EXISTS` in an upgrade like `schemaV1ToV2` in `database/db.go`. No data is changed.

### 4.3 Readiness (D3)

- `ready(phase, focus)` = the phase's required items are present for that focus. Required items per phase are listed in section 5.
- Display shows counts per phase, not one overall number.
- Advancing a phase never removes counts from earlier phases.

## 5. Required items and the person gate (D7)

Required before Moly helps with a goal that concerns a person:
1. The goal (focus) exists.
2. The person exists and has a name that is not a placeholder.
3. The user has at least one trait, value or style. (Existing `about_me`, or extraction. Required only once the goal is locked.)

Item 2 is the "who is who" gate. When it is missing, Moly does not give help. It asks for the name once, and says why (the name is needed so the person can be recognised in later conversations). The person record is marked `awaiting_name`. While that flag is set, Moly gives no advice. The next message is read as the answer, and the name is extracted by the LLM (R8), not by pattern. If the user gives no usable name, the label the user already used (for example "girl on fetlife") is stored as the name, and Moly does not ask again.

Resume: after the name is set, the goal continues with the same focus. The user's name choice is kept on the person record, so a later conversation with the same person starts with the name.

## 6. Phase order

### Phase 1 — Reply contract (RC7)
- One `Reply` value per request: `text`, `phase`, `metadata`, `clarificationQuestions`.
- The contact formatter returns metadata and an optional clarification. It never replaces `text`.
- The HTTP response includes `response` in every branch. The saved assistant row is the same value.
- Empty text is an HTTP 500, and nothing is saved as a reply.
- Tests: a low-confidence contact gives visible, saved text; empty agent reply gives an error and no saved reply.
- Live L1: message 1 shows a visible reply, and the saved text equals it.

### Phase 2 — Remove the cache shortcut (RC1, RC2)
- Delete the cached-summary branches in layers 2, 3, 4, 5, 8, 9, 11 (`layer_adapters.go` ~420–445, `layer4_gap_detector.go` ~73–100, and the same pattern elsewhere).
- Delete the placeholder branch in `conversation_agent.go` ~622–660. It falls through to the LLM response.
- The summary for message N is written after the reply and read only by message N+1 (R1).
- Tests: no `(cached` path for any layer; the summary of message N is not read during N.
- Live L2: message 2 asks a gap question; no `(cached` lines; latency recorded.

### Phase 3 — Context-loader gaps and the safety check (RC4)
- Rename the loader's list to `missingContext` (`main.go` ~2169–2205). It is informational only.
- Remove the deferral at `main.go:2516` (`len(gaps) > 2`). Layer 2 runs on every message before the reply.
- Remove the clarification-mode log decision (`main.go` ~2775), or pass it as a field (R4).
- Tests: a new user's harmful test message is blocked by Layer 2; `missingContext` does not change Layer 2's result.
- Live L3: the harmful test message (a stand-in phrase agreed with the user) is blocked before the reply.

### Phase 4 — Maturity items, focus and readiness (D1, D2, D3, D5, D6, D8, section 3)
- Implement the item table in 4.1 with the writers in 4.2. Schema upgrade to version 3 (D4).
- Focus comes from `GoalCoherence`: `same` keeps the focus; `related_subgoal` adds to it; `different` starts a new focus (section 3).
- Remove the four-factor score (`layer_adapters.go` ~470–560), `calculateContextMaturity` (`main.go:5005`), the accomplishment markers (`main.go` ~2440–2500), and `storage/maturity_service.go` `MarkAccomplishment`, once the new path is live (phase 8).
- Decrease: only Layer 5 "replace" subtracts a count (section 3). Tests assert that no other path decreases a count.
- Tests: an item counts once; a repeat does not count; a topic change keeps the old focus's counts; a replace subtracts exactly one item; advancing keeps earlier phases.
- Persistence test: counts survive a reload (regression for M1).
- Live L4: the protocol in section 7, with counts printed per message.

### Phase 5 — Person gate and name answers (D7)
- Implement the gate in section 5. Replace `ProgressiveNamingDetector` regex rules (`progressive_naming.go`) with an LLM extraction of the name answer (R8).
- The name question is the reply's only content while the gate is open (R5, R4).
- Pseudonyms and labels such as "Girl from fet" are accepted without validation beyond non-empty.
- Tests: missing name blocks help; the answer sets the name; a placeholder is not counted as a name; a later conversation with the same person gets the name.
- Live L5: messages 2–3 of the protocol.

### Phase 6 — Gaps from missing items (RC5, RC6, RC10 items 1 and 5)
- Layer 4 builds gaps from the required items in section 5 that are missing for the active focus. The LLM phrases one question per gap. The prompt receives the missing item, the answered-question list (existing), and the no-echo rule (existing).
- Asked gaps are saved by text (existing path). A reply in the conversation marks them answered (existing `MarkConversationQuestionsAnswered`). Answered questions only stop repetition; they add no count (D5).
- `ShouldClarify` is derived from the missing items, not set independently (RC5).
- Tests: a missing required item produces one gap; a present item produces none; an answered question is not asked again.

### Phase 7 — Live protocol (used at L1–L5)
Scripted conversation, new conversation each run:
1. "Hi Moly" → greeting text; no focus; no counts.
2. "I want to write a first message to a girl I saw on fetlife." → focus set; person exists without a name; name question is the only content.
3. "Call her 'Girl from fet'." → person name set; the goal continues; counts for person-name and focus recorded.
4. "She is submissive, I am dominant." → person trait counted for her; user trait counted for the user; no trait assigned to the wrong subject.
5. A harmful test phrase agreed with the user (not a real harmful request) → blocked by Layer 2 before the reply.
6. Change the topic to an unrelated question → new focus; the first focus's counts are unchanged.

For each message record: reply text (first 200 characters), focus, counts by phase, gap asked, latency, and whether the reply was saved and shown.

### Phase 8 — Dead and duplicate code (after L1–L5 pass)
Follow `DEAD_AND_DUPLICATE_CODE.md`. One commit per group. Groups: maturity leftovers; workflow leftovers (greeting templates, placeholder reply, intent-detector extraction cluster, vague-name remnants, unused answered-type parameter, unused clarification helpers); duplicates (Layer 5 implementations, summary writers, answer writers, contact row scanning); candidate list in `DEAD_CODE_CANDIDATES.txt`; test helpers; the Socrates-family duplicates (needs your decision on the source copy). Each step runs `go build`, `go vet`, `go test`.

## 6a. Live findings, run of 2026-10-09 (10:00–10:32), after phase 1

Phase 1 result: L1 passes. The clarification reply is visible, and the saved rows (`chat_messages`) equal the text shown, for both messages.

Problems seen in the run, with the phase that owns each fix:

| ID | Observed | Cause (from log) | Owner |
|---|---|---|---|
| N1 | "Hi Moly" gets a withdrawal reply ("I notice you might not want to dive deep…") | Layer 11 denial check returns `denial=true` on the first message. It needs prior context (spec: short message after longer context). Its phase `denial` is not a known phase (`Unknown phase 'denial'`). | Phase 3 (denial gating) |
| N2 | Greeting becomes the primary goal ("Greeting"). The real goal is then "DIFFERENT", an intention conflict is queued, and the real intention is not saved. | Goal lock on the first message accepts a greeting. The extraction prompt change did not stop it. | Phase 4 (focus), with an LLM check "does this message state a goal?" (no keywords) |
| N3 | Gap question "Can you clarify your intentions and goals…" asks about extracted data. The question shown ("Have you ever reached out…") is not one of the gaps. | Gap wording is rewritten by the response generator, and the prompt rule against asking about extracted data is not enforced. | Phase 6 |
| N4 | Contact "girl" has confidence 0.00 on load, so the clarification "are you talking about girl?" is shown. | RC8 still present: `GetByUserID`/`GetByID` do not select `confidence`. | Proposed phase 1b (small wiring fix); needs approval |
| N5 | Message 2 took about 9.7 minutes: Layer 2 173 s, Layer 4 78 s, response 2 min 21 s. | LLM calls per layer (deferred speed work). | Recorded; speed deferred |

## 6b. Investigation: problems P1–P3 and the maturity value 1.00

**Approved (2026-10-09):** save the contact now as unnamed; the name stays empty until the user gives one.

### P1 — Should Moly interrupt to ask the name before saving the contact?

Current behaviour (run 2026-10-09 10:22): the extraction prompt asks for a contact `name` (`agents/context_extractor.go:138`). With no name in the message, the model writes the label "girl" into `name`. The contact is saved as `girl`, status `active`, so the system treats it as a named person. The clarification then asks "are you talking about girl?", using that label as if it were a name.

Answer: the contact should be saved **now**, but as an unnamed person. The name stays empty until the user gives one. Saving early loses nothing if the user stops. What must not happen is writing the label into `name`, because that marks a placeholder as a real name.

Coverage in the plan: **not covered.** Phase 5 says the gate blocks advice, but does not say what `name` holds before the answer. It must be added to phase 5 (see section 6c).

### P2 — What should the clarification ask?

Current behaviour: "Just to clarify, are you talking about girl? A) girl (other) B) Someone else?" comes from `ContactResponseFormatter` (`agents/contact_response_formatter.go`), and it treats the label as a choice.

**Approved (2026-10-09)** text for the name question, a fixed template (protocol, not content):
"Can you give me a name for the {contact} you mentioned?"
where `{contact}` is the user's own label (for example "girl"). No options.

No options. The formatter's "are you talking about X" and the A/B options are removed for the unnamed-person case.

Coverage in the plan: **partly.** Phase 5 says "the name question is the reply's only content" but has no text and no removal of the A/B formatter. It must be added (section 6c).

### P3 — Why Moly never greeted back

Chain in the log (run 10:00):
1. "Hi Moly" → extraction intention "Greeting" → the primary goal is locked as "Greeting" (`layer_adapters.go` lock, see N2).
2. Layer 4 asks "Who is the intended recipient of the greeting, Moly?".
3. Layer 11 returns `denial=true` on the first message and prevents response generation ("Preventing response generation, returning denial", `unified_orchestrator`). The reply is the withdrawal text (N1).
4. `ConversationAgent` runs (log lines 329–365), but its denial protocol is applied first ("Denial protocol will be applied"), so the withdrawal text is the reply. The routing step that would select the greeting guidance (`RouteResponse(IntentGreet)`, `conversation_agent.go:1567`, guidance at ~2545–2585) is never logged for this message. The greeting path exists, but the denial takes precedence over it.
5. The only greeting detection before the layers is `isGreetingOrSelfRef` (`main.go:2124`). It matches substrings such as "hi " and "hello". That is a keyword rule, against R8. It only skips gap questions. It does not produce a greeting.
6. The reply templates for greetings were seeded, but nothing reads them (D-W1).

So Moly never greeted back because no step makes "greet" the outcome. Layer 11 and the goal lock take the message first, and the greeting logic that exists comes too late.

Coverage in the plan: **not covered.** Phase 3 covers Layer 2 and `missingContext`. Layer 11 is listed in the findings table as "Phase 3" but the phase text does not include it. There is no greeting phase at all. Required (section 6c).

### Maturity = 1.00 in the logs

There are three separate sources, and the 1.00 is not a measurement:

| Log line | Source | Value |
|---|---|---|
| `ConstitutionalEvaluator ... maturity: 1.00` | `layer_adapters.go:344` calls the deprecated `EvaluateWithAnalysisContext`, which passes a **hard-coded** `1.0` (`tools/constitutional_evaluator.go:156`) | constant |
| `PRIMARY safety evaluation ... maturity 0.00 → 0.33` | `main.go` ~2519: `contextFieldsLoaded / contextFieldsTotal`. For a greeting, `contextFieldsTotal` is 0, so the value is +Inf and is clamped to 1.0; it is then replaced by the accomplishment score | intermediate, then overwritten |
| `4-Factor maturity ... score=0.24` and `Updated summary ... maturity=0.24` | four-factor score (Layer 3) and accomplishment score | the values being replaced in phase 4 |

Will it be fixed later? **Partly.** Phase 4 removes the four-factor score and the accomplishment flags. It does **not** list the hard-coded `1.0` in Layer 2, the deprecated wrappers (`tools/constitutional_evaluator.go` lines 56, 63, 156), or the division in `main.go` ~2519. Those must be added to phase 4 (section 6c). Note that Layer 2 no longer gates on maturity (FIX #74), so the honest fix may be to remove the parameter, not to feed it a real value.

## 6d. What happens after a new goal is extracted (run 2, message "I want to write a first message…", 10:22)

Timeline from the log (9 min 20 s in total):

1. **Extraction is correct.** Goal "write a first message to the girl on fetlife", confidence 0.90, 7 entities. The contact "girl" is extracted with confidence 0.60 and saved (`main.go` extraction, 10:25:35).
2. **Three question sources start at once.**
   - `ConfidenceBasedClarifications` generates 3 questions ("contact confidence low", "style confidence low", "no values"). They are wired to Layer 4 and do not appear in the final reply.
   - Layer 4 generates 2 goal gaps. Its goal-aware filter first returns 0, then the LLM generates 2 anyway. One says "Can you clarify your intentions and goals…", which asks about data already stated.
   - `ConversationAgent` adds 2 more ("What do you hope to achieve…"), making 4 gaps.
3. **Strategy goes to clarification for the wrong reason.** `Layer1.Confidence` is set from `Contact.Confidence` (`layer_adapters.go:~189`), and is 0 when the analysis context has no contact ("Extracted 0 relevant contacts"). The strategy then reads 0.00 and chooses `clarify_extraction`. The extraction confidence was 0.90, so the strategy's input is the wrong number.
4. **The reply is an LLM rewrite, not one of the gaps.** The top gap is chosen (4 → 1), and an LLM writes the question. It produced "Have you ever reached out to someone on Fetlife before…", which is not among the gaps. The 2 minutes 21 seconds for this step is part of the slowness you asked me to leave for now.
5. **The goal change is detected but not handled.** "Goal changed to DIFFERENT (primary Greeting → write a first message…)". Layer 4 filters by goal target and then generates gaps anyway.
6. **The new intention is not saved.** "Different intention in same context general" → queued for approval → "Skipping intention save". So the stored intention does not change.
7. **The primary goal stays "Greeting".** The saved context shows goal "write a first message…" with primary "Greeting". Every later message will again be "different" from the primary.

Coverage in the plan: items 2 and 4 fall under phase 6 (one question owner, gaps from missing items). Item 3 is a wiring error not in the plan: `Layer1.Confidence` must mean extraction confidence, and the strategy must read it. Item 6 falls under phase 4 (focus). Item 7 falls under phase 4 (a primary goal that a greeting does not set, and a goal change that starts a new focus). None of these is covered by a phase yet, so item 3 is added to phase 6 as a prerequisite.

**Decision (2026-10-09):** a greeting is a message intent, not a goal. The goal stays blank.

What this changes in code, for phase 4 and phase 3:
- Extraction returns a message intent from the existing enum (`IntentGreet`, `IntentAsk`, `IntentShare`, …) chosen by the LLM. No keywords (R8). A greeting gets intent `greeting` and an empty goal.
- The primary-goal lock reads only the goal. An empty goal never locks, so "Greeting" cannot become the primary goal.
- `GoalCoherence` has nothing to compare when the goal is empty, so no "different goal" path runs.
- Layer 4 aims goal gaps only when a goal exists. A greeting produces no gaps.
- Intention conflicts and saved intentions are written only when the goal is non-empty.
- The greeting reply path (phase 3) is selected by the intent `greeting`.

Programmatically this removes three branches (lock, coherence, gaps) that currently treat every message as a goal. The current code has one string for both, `Intention`, and a `Goals` list that the lock and coherence never use (every log line shows `goals=0`).

**Proposal (2026-10-09, pending approval):** goal gate. When a conversation has no goal, Moly asks once for one ("What would you like to work on?" or a message-specific form), sets `awaiting_goal`, and gives no advice or gap questions until it has an answer. If the user gives no goal, the conversation continues as general chat and Moly does not ask again. This uses the same mechanism as the name gate (section 5): one rule for any missing required item. The greeting reply is "greet, then ask what the user wants to work on". Product choice to confirm: is a goal required before advice, or may a user chat without one?

Risk: if the LLM misclassifies a message as a goal, the same path runs again. Tests must cover greeting, question, and goal messages with a fixed set of expected intents.

## 6c. Additions to the phases (required by 6b)

- **Phase 3 (safety, denial, greeting):**
  - Layer 11 may deny only with prior context (the spec's rule: a short message after a longer one). A first message never triggers denial. Fix the `denial` phase name too (N1).
  - Add a greeting outcome. A message that is only a greeting gets a short reply from the LLM greeting guidance, with no goal lock, no gap questions, no denial, and no maturity change.
  - Greeting detection is an LLM intent check (`IntentGreet`), not the substring list in `main.go:2124` (R8). Remove the substring list.
- **Phase 4 (maturity):** remove the hard-coded `1.0` path (`layer_adapters.go:344`, `constitutional_evaluator.go` 56/63/156) and the division in `main.go` ~2519. Layer 2 receives no maturity value unless a real one is needed.
- **Phase 5 (person gate):**
  - Extraction writes the user's words into `label`, and writes `name` only when the user gave a name. The contact is saved with status `unnamed`.
  - The name question uses the fixed template in P2. The A/B formatter is not used for an unnamed person.
  - While the gate is open, no gap questions are asked (the gate has priority, so Problem 2's "Have you ever reached out…" cannot appear before the name).
- **Phase 1b (proposed, needs approval):** RC8 — select `confidence` in `GetByUserID` and `GetByID`. Without it, every contact with a name is still treated as low confidence.

## 6e. Phase 3 status (2026-10-09): code and tests done, live check L3 pending

Done:
- **Layer 11 denial** needs prior context: at least two user messages (the current one is already in the recent messages, `main.go` prepends it). A first message and a greeting are never denials. Test: `TestLayer11FirstShortMessageIsNotDenial`, `TestLayer11GreetingIsNotDenial`. Interim rule: still "under 10 characters" after prior context; an LLM check replaces it later.
- **Greeting by LLM intent:** the phrase lists (`main.go` substring check and `detectGreeting` in `intent_detector.go`) are removed. The intent is decided once, before the layers, and the agent reuses it (`ConversationContext.MessageIntent`) instead of deciding again. Net LLM calls per message are unchanged, but the call now happens before the layers (latency to measure).
- **A greeting carries no goal:** goal entities and the extraction intention are removed for a greeting, so the primary goal cannot be locked as "Greeting". Test: `TestWithoutGoalEntitiesKeepsOtherEntities`.
- **Layers 4–10 skip a greeting** (`CanSkip` on `LayerContext.IsGreeting`). Test: `TestLayersSkipForGreeting`.
- **Safety is never deferred** by missing context fields (`main.go`, the `len(gaps) > 2` deferral is removed). The maturity division is guarded for totals of 0.
- **Context-loader list renamed** to `MissingContext`. `ConversationContext.Gaps` now holds only Layer 4, conflict and clarification questions.
- **The "Clarification mode" log decision is removed.** Four orchestrator flags (`shouldAskClarification`, `shouldHandleAmbiguity`, `shouldHandleViolation`, `shouldHandleConflict`) are still computed and not yet passed to the agent. They are marked in code and belong to phase 4 / phase 6 wiring.

Not done in phase 3 (moved):
- Gate thresholds at `conversation_agent.go` ~1131 and ~1440 (`len(ctx.Gaps) >= 3`) were tuned when the loader fields were counted. They now count questions only, so the thresholds must be re-chosen in phase 6.
- The `Layer 2` call in the handler still uses a maturity value; that is phase 4 (6b).
- The goal-gate and name-gate are phases 4 and 5.

Problem 2 root cause (run 2026-10-09 10:22, derived from the counts): `ctx.Gaps` began with the loader fields (`relevantReflections`, `recentSafetyIncidents`) and then appended the two Layer 4 gaps. The gap gate asked about the first item (`relevantReflections`), and the LLM turned that field name into "Have you ever reached out to someone on Fetlife before…". Fixed by the rename above. The gate itself still needs phase 6.

Live check L3 (needs the server restarted with the new binary): send "Hi Moly" as the first message. Expected: a greeting reply, no withdrawal text, no goal locked, no gap question.

## 6f. Phase 2 status (2026-10-09): code and tests done, live check L2 pending

Done:
- The cached-summary shortcut is removed from layers 2, 3, 4, 5, 6, 7, 8, 9, 10 and 11 (`HasMessageSummary` blocks). Each layer evaluates the current message.
- The Layer 1 extraction-cache read (fixed confidence 0.95) is removed. Reuse of this message's own extraction (`Reusing extraction from AnalysisContext`) stays: it is the real result of this message, not a shortcut.
- The orchestrator no longer builds the message-summary cache for the layers (`unified_orchestrator.go`).
- The `acknowledge_and_guide` case no longer returns the fixed template. It falls through to the normal response path.
- Contract test: `TestLayer11IgnoresCachedSummaryOfCurrentMessage` (a 0.95 cached summary does not decide Layer 11).
- All packages pass `go test`; `gofmt` is clean.

Left for phase 8 (dead after phase 2): `buildMessageSummaryCache`, the `MessageSummaryCache` field and its accessors in `tools/layer_context.go`, `BuildResponseFromExtraction` and `ValidateResponseFitsContext` in `response_strategy.go`, and the `l1.cache` writes in `layer_adapters.go`.

Not yet confirmed: the message summary for message N is still written before the layers run (FIX #10). No layer reads it for message N any more, so it is not a correctness issue, but R1 asks for it to be written after the reply. This moves to phase 4, where the maturity items are persisted.

Live check L2 (needs the server restarted from `bin/moly`): send "Hi Moly", then "I want to write a first message to a girl I saw on fetlife." Expected: Layer 2 and Layer 11 log "(evaluated)" lines, no "(cached" lines, and a gap question for the missing item.

## 6g. Question ordering (2026-10-09): code and tests done, live check pending

Rule (user decision): open missing items are asked first; reflective (Socratic) questions come only after the deepening gates pass.

Done:
- The Socratic question is asked only when the deepening gates allowed it (`shouldDeepen`). The `ack_socratic` branch used to call the question selector without checking the gates ("Attempting Socratic deepening … shouldDeepen=false" in the log). Rule: `socraticAllowed`, tests in `agents/ordering_test.go`.
- The gap gate asks any open gap (was: three or more). The two Layer 4 gaps in the goal run were never asked because of the threshold.
- Gate 2 blocks deepening while any gap is open (was: three or more).

Not done (needs phase 5 and 6):
- The name question does not come first yet. The name gate (phase 5) is not built, so the first question asked is the first open Layer 4 gap. Those gap texts were not reliable in the last run (the goal run asked about interests and empathy); phase 6 fixes their wording and ownership.
- The phase used by the gates is still the accomplishment phase, which the agent never receives (`ConversationContext.Maturity` is nil). Phase 4 replaces it.

Live check (needs the server restarted from `bin/moly`): send the goal message after the greeting. Expected: no Socratic question while a gap is open; the first question is a gap question; the log shows "Attempting Socratic deepening" only after the gates pass.

## 6h. Name gate (phase 5), built 2026-10-09: code and tests done, live check pending

Built (approved on 2026-10-09: save as unnamed; question "Can you give me a name for the {contact} you mentioned?"):
- Extraction returns the user's own words as `label` and a real `name` only when the user gave one (`NameKnown`). Prompt and parser: `context_extractor.go`.
- A person without a given name is saved with `name_status = 'unnamed'`. The state is stored in a new column `contacts.name_status` (values `named`, `unnamed`, `asked`). Schema v3, additive upgrade from v2 (`database/db.go`), tested with a real version 2 layout.
- After the contact is saved, the person is asked for a name once: status becomes `asked`, and the reply is only the question (`ConversationAgent`, name gate after the Layer 2 check).
- The next message answers it: a given name renames the same row (`ApplyNameAnswer`, `RenameContact`), so no second contact is created. A message with no name keeps the label as the name and is not asked again.
- A message that refers to a person without a name is matched to the only named person of that relationship, if there is exactly one; otherwise a new unnamed person is created.

Deviations from section 6c, to confirm:
1. The label is stored in the `name` column while the person is unnamed (the column is required and unique per user). The `name_status` column says it is not a real name. The plan said "name stays empty"; that is not possible with the current schema without more change.
2. The rule that matches a nameless mention to the only named person of that relationship is a stop-gap. The extraction does not see the known people. The durable fix is to give the extractor the known people (phase 6).

Bugs found and fixed on the way:
- `GetByID` selected 21 columns (four repeated) and scanned 17: every call failed. It is used by contact archiving, traits and progressive naming.
- `GetByName` selected 17 columns and scanned 13: every call failed. Its callers ignore the error, so the handler treated each mention as a new person, and `INSERT OR REPLACE` overwrote the existing row. This is the same class as RC1.
- All single-contact reads now use one column list and one decoder (`contactColumns`, `scanContact`).
- The second contact-save path (raw SQL in `main.go`) now writes `name_status`.

Live check (needs the server restarted from `bin/moly`): new conversation. (1) "Hi Moly". (2) "I want to write a first message to a girl I saw on fetlife." Expected: the reply is only "Can you give me a name for the girl you mentioned?"; the contact is saved as unnamed. (3) "Her name is Christine." Expected: the same contact is renamed to Christine (one row); the goal continues. Check the database for one contact.

## 6i. Live findings, run 2026-10-09 12:04–12:52 (after the name gate)

Fixed in this round:
- Confidence was saved but never read back, so every contact looked low-confidence (0.00). That sent every message with a contact to the old "are you talking about X?" formatter. `contactColumns` and `scanContact` now include `confidence`.
- The second early-save path (`main.go` ~1297) wrote contacts without a name status, so the name question could never fire. It now writes `contactNameStatus`.
- Layer 1 re-extracted a greeting and locked "Greeting" as the primary goal (the first extraction had confidence 0.00, so the reuse path was skipped). A greeting now skips extraction and never locks a goal (`layer_adapters.go`, test `TestLayer1SkipsGreeting`).

Found, not fixed (phase 6 / phase 4 / dead-code phase):
1. **The user is saved as a contact.** The greeting produced contact "user" (confidence 1.00, name given). The prompt says OMIT when the user is discussing themselves; the model did not follow it. Needs a prompt rule plus a check.
2. **A pasted profile becomes the user's goal.** The profile text ("Looking for long-term relationship, dominant") was extracted as the user's intention "Establish a long-term relationship with a dominant partner". The profile belongs to the other person. Needs subject attribution for pasted third-party text (phase 4).
3. **Two early-save paths for contacts** (`main.go` ~1141 and ~1297) do nearly the same work with different rules. Keep one (dead-code phase).
4. **A second profile, "Christine_sub", was created as a new contact** instead of renaming "girl on fetlife". The rename only happens when the name question was asked, and that did not happen in this run (the name status fix came too late for it). The next run should show the rename.
5. The gap questions (`ask_goal_aligned_gaps`) now run, as intended, but their wording still covers interests and approach (phase 6).

## 7. Risks

- **Latency.** Phase 2 adds LLM calls. Measure at L2 and L4. The 20-minute timeout is the current limit.
- **Existing conversations** have no items. Their maturity starts at zero, and the live check should say so.
- **Name extraction** is an LLM call per person-related message while the gate is open. It is needed for correctness, so the cost is accepted, but it is measured.
- **Focus detection** depends on `GoalCoherence`, which has not been tested live as a focus switch. Phase 4 tests it before relying on it.
- **Speed work** (per-phrase attribution, message saving before analysis) stays deferred, as decided.

## 8. Open questions for the user

1. Resolved: the name is asked once, with the reason. Advice is blocked only while `awaiting_name` is set. Whether the question sits alone or with an acknowledgement does not change the logic.
2. Resolved: no repeat question. The user's own label is used if no name is given.
3. Whether a focus that is abandoned (not returned to) should show as "paused" in the UI. Not implemented in this plan.
