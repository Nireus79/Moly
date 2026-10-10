# Investigation: Moly workflow and maturity (2026-10-09)

Status: findings only. No fixes applied in this document. Implementation is in `IMPLEMENTATION_PLAN.md`.

Evidence sources: server log `Christine_sub.txt` (conversation `conv_1791522035`, user `user_1a36e4fe72ea7a8`), the database (`~/.moly/moly.db`, read through the application's own open path), the specification `MOLY_11_LAYER_SYSTEM.md` and `MOLY_COMPLETE_VISION.md`, and temporary tests that were deleted after running. File references are to the state of the working tree on 2026-10-09.

---

## 1. What Moly should do (specification)

- Each user message is one circle of a loop. Extract who and what, load the accumulated context from earlier circles, re-evaluate it, then run the layers.
- Maturity measures how complete the context is. It is not a judgement about the message. A new user starts near 0.
- Maturity decides the behaviour: low maturity asks goal-aligned gaps, high maturity gives help with the accumulated context in the prompt.
- Gaps are what block the current goal. They are asked one at a time, and answered gaps are not asked again.
- Maturity must rise each circle. A flat score means the loop is stuck.
- Layer 2 (safety) runs before any reply. Obvious harm is denied at once.
- The user always gets a reply.

The specification contradicts itself on thresholds (see M7). It does not say whether maturity may fall.

## 2. What happened on the test message (Christine_sub.txt)

Message 1 ("Hello Moly. I want to talk to you about a girl I am interested to…", 381 characters) went through these steps:

| Step | Time | What the log shows |
|---|---|---|
| Handler entry | 08:00:35 | POST received |
| Extraction (LLM) | 08:00:35 → 08:03:28 | 2 min 53 s for the semantic extraction |
| Subject attribution (LLM) | → 08:04:06 | 38 s |
| Goal lock | 08:04:06 | `PRIMARY GOAL LOCKED: "initiate a conversation with a girl on fetlife"` (correct) |
| Layers 2, 3, 4, 5, 8, 9, 11 | 08:04:06 | each logged `(cached, duration=0.00s)` — none evaluated |
| Maturity | 08:04:06 | `score=0.88, phase=implementation` (from cache, see M3) |
| Gaps | 08:04:06 | strategy `gaps=0`; handler logs `gaps=3` (different lists, see M4) |
| Strategy | 08:04:06 | `acknowledge_and_guide` → placeholder text, no LLM call |
| Safety check | 08:04:06 | "Deferring safety evaluation: 3 gaps remain" |
| Risk assessment (LLM) | 08:04:06 → 08:04:52 | 46 s |
| Contact formatter | 08:04:52 | "Contact girl has low confidence (0.00)" → clarification branch |
| Save | 08:04:52 | assistant row saved = placeholder JSON, 181 bytes |
| Summary (LLM) | 08:04:52 → 08:05:56 | 64 s |

Total server time: about 5 min 20 s. The extension gave up at 190 s (08:03:45), so the user saw nothing live.

Database after the run (verified by reading):
- `chat_messages`: user row (381 chars) and assistant row whose content is `{"phase":"responding","response":"Based on what you've told me:…Let me help you achieve this..."}`.
- `contacts`: `girl`, confidence **0.9** (stored correctly).
- `conversation_context`: goal and 6 entities stored.
- `conversation_summaries`: version 1, message_count 1, `accumulated_entity_count` 0.
- `clarification_questions`: **empty**. No gap question was ever saved.
- No sent clarification question for the name.

## 3. Root causes: workflow

Each cause lists the observed symptom it explains.

### RC1 — Layers are short-circuited by a confidence shortcut (central)
- `agents/layer_adapters.go:422` and `:438`: Layer 3 returns "complete" maturity when the cached message summary's confidence is ≥ 0.85.
- `agents/layer4_gap_detector.go:76` and `:92`: Layer 4 returns **zero gaps** under the same condition.
- Layers 2, 5, 8, 9 and 11 use the same shortcut (`(cached, duration=0.00s)` in the log).
- The cache entry is written by the same request, a few milliseconds before the orchestrator runs (`MessageSummaryRepository` save at 08:04:06, before `Invoking UnifiedOrchestrator`). Each layer reads its own output.
- Explains: no name question, the placeholder reply, and the safety check being decided by a shortcut.

### RC2 — The FIX #11 cache is a performance shortcut that replaced evaluation
- Introduced for speed ("67% improvement"). It removes the layers' reasoning and keeps only a number.
- The speed concern (deferred by the user) and the correctness failure are the same design choice.

### RC3 — The placeholder reply is the default for "no gaps"
- `agents/response_strategy.go:70`: `confidence >= 0.85 && gapCount == 0` → `acknowledge_and_guide`.
- `agents/response_strategy.go:148` `BuildResponseFromExtraction`: returns a fixed template ending in "Let me help you achieve this…" and never calls the LLM.
- `DetermineStrategy` computes `maturity` and never uses it.

### RC4 — "Gap" means three different things
1. **Context-loader fields** missing: `aboutMe`, `contact`, `conversationHistory`, `userBehaviorProfile`, `relevantReflections`, `pastIntention`, `recentSafetyIncidents` (`main.go` ~2169–2205). "3 gaps" in the log is this list (`relevantReflections`, `pastIntention`, `recentSafetyIncidents`).
2. **Layer 4 goal gaps**: the questions the specification means.
3. **Persisted clarification questions**: rows in `clarification_questions`.

The safety deferral at `main.go:2516` uses list 1. A new user's empty fields therefore switch the input safety check off, which contradicts the specification ("Layer 2 runs first").

### RC5 — A decision is logged but not passed on
- `main.go:2775`: logs "Clarification mode (…) - ConversationAgent will ask questions, not give advice". The flag is not in the agent context. `ConversationAgent` decides alone from `layerCtx.Layer4.ShouldClarify`.
- `agents/conversation_agent.go:597`: "FIX #18: Layer 4 gate ShouldClarify=false - bypassing gap clarification". The cached Layer 4 sets `ShouldClarify: false` (`layer4_gap_detector.go:85`), so clarification is bypassed even when gaps exist.

### RC6 — Nothing could ask the name
- The LLM gap generator (`generateGoalAlignedGapsViaLLM`) runs in `Layer4.Process` after the cache check. When the cache hits, the generator never runs.
- Even when it runs, the extraction block told the LLM "ALREADY EXTRACTED — do NOT ask", which listed the contact as known (fixed in this session, prompt only).

### RC7 — The reply shown and the reply saved are different
- Live: `main.go:4005` replaces the agent response with the contact formatter's output. For a low-confidence contact that output is `{status: clarification_needed, phase: context_gathering, clarification: {...}}` with **no `response` field**. The sidebar (`ChatInterface.tsx` ~226) adds a message only when `data.response` exists, so nothing is shown and nothing reports an error.
- Saved: `main.go:3091` saves the agent response (placeholder), so a reload shows the placeholder JSON.
- Two sources of truth, and neither is the specification's single reply.

### RC8 — Contact confidence is never loaded (wiring)
- `database/contact_repository.go`: `Save` writes `confidence`; `GetByUserID` and `GetByID` do not select it. Every loaded contact has confidence 0.
- `agents/contact_response_formatter.go` `CheckIfClarificationNeeded`: a contact with confidence < 0.5 forces the clarification branch. So every message with a contact takes that branch (RC7).
- Verified: stored confidence 0.9, loaded 0.00.

### RC9 — Subject attribution defaulted to "user" (fixed in this session)
- `agents/extraction_phase.go` `determineSubjectViaLLM` matched the LLM answer with substring checks. Its contact branch never set the contact's name, and errors defaulted to `"user"`.
- Effect: a contact's trait could be stored as the user's, and the reply prompt listed it as "They've described themselves as".
- Fixed: JSON parsing, unknown instead of user, grouped prompt. Tests added. **Not yet verified live.**

### RC10 — The specification's loop is not implemented
The specification's own "Implementation Gaps" section lists five missing parts. I confirmed them in the code:
1. Re-evaluation of accumulated context — not implemented.
2. Accumulated context in the LLM prompt — partly (unordered lists; the placeholder path never uses it).
3. Maturity must rise each circle, with a circuit breaker — not implemented.
4. Layer-based updates to accumulated context — not implemented.
5. Clarification history — storage was added this session; the gate that decides "ask" is still RC1.

### RC11 — Other wiring faults found this session
- `main.go` context literal does not set `ConversationContext.Maturity` (the accomplishment object), so the agent's phase logic reads a nil object (`conversation_agent.go:1134`).
- Gap gate `conversation_agent.go:~1131` (`len(ctx.Gaps) >= 3`) runs only after the early return of the placeholder path, so it is effectively unreachable for these messages. Not yet confirmed by a log.
- An LLM call of 438 bytes logged at 08:04:06 after "Reading Layer 2 evaluation" has no completion line. Its caller is not yet identified.

## 4. Maturity: how it works now

### M1 — The accomplishment score is not persisted (verified by round trip)
- `database/schema.sql` `conversation_maturity` has `current_phase`, `maturity`, and `accomplishments_completed/total`. Only `current_phase` and `maturity` are written (`storage/maturity_service.go` `SaveMaturityContext`). The accomplishment counts are never written.
- `LoadOrCreateMaturityContext` builds a fresh `ConversationMaturity` and restores only the score and phase, not the accomplishments.
- Round-trip test (deleted after running): before saving, accomplishment maturity 0.417 (`gathering`); after reloading, **0.000 (`initial`)**. The stored score 0.51 survives.
- Effect: the value the agent receives (`ContextMaturity`, from `main.go:1082` and `:2531`) and the phase are rebuilt from the markers set in the **current** message only.

### M2 — The four-factor score (Layer 3 live path) is wrong in four ways
Code: `agents/layer_adapters.go` ~470–560, `tools/maturity_calculator.go`.
- **Profile factor is a constant 0.5** whenever any extraction exists (`CalculateProfileCompleteness` returns 0.5 as a placeholder). The profile is not read. A new user starts at 0.125.
- **Contact clarity is 0 once a conversation summary exists.** `clearContactCount` is only computed in the fallback branch; the summary branch sets only `contactCount`.
- **Entity count comes from the summary**, which resets to 0 each message (RC2 of the earlier report).
- **Depth** rises linearly over the first five messages. The choice is not derived from anything.
- Probe results on the same formula (temporary test, deleted):
  - Message 1 (no summary yet): 0.512.
  - Message 2 (summary exists): 0.379. A **drop of 0.13**. Layer 3 keeps the higher previous value (`layer_adapters.go` FIX #76), so the drop is hidden.
  - Message 2 with the clear-contact count computed: 0.629.
  - New user with nothing: 0.125.

### M3 — The 0.88 in the log came from the cache, not the formula
- The cached confidence (0.88) replaces the four-factor score (`layer_adapters.go:422`). The formula on message 1 gives about 0.51, not 0.88.

### M4 — Three maturity values reach the same request
| Value | Computed by | Used for |
|---|---|---|
| A. Four-factor score | `layer_adapters.go` Layer 3 | Layer 8 gates, Layer 3 labels, persisted to `maturity` |
| B. Accomplishment score | `models/phase_accomplishment.go` from markers in `main.go:2440–2500` | `ContextMaturity` to the agent, the phase |
| C. Cached confidence | `layer_adapters.go:422`, `:438` | Overrides A when ≥ 0.85 |

Two different "gap" lists are also in play (RC4).

### M5 — The accomplishment score can fall
- `models/phase_accomplishment.go` `CalculateOverallMaturity` averages phases with progress > 0.
- Probe: `initial` complete (1.000), then the first `gathering` marker → **0.583**. The agent receives this value with no guard.

### M6 — Decreases are hidden or never handled
- Layer 3 keeps `max(previous, new)` (FIX #76).
- `conversation_agent.go:547` also keeps the previous value (FIX #21).
- Nothing lowers maturity on a topic change, a contradiction, or a goal change.
- The stuck detector (`main.go:4210–4243`) computes a flat-score flag and writes metadata. Nothing reads the flag. The specification's "break the loop when stuck" is not implemented.

### M7 — Thresholds are inherited, inconsistent, and not derived
| Where | Values | Origin |
|---|---|---|
| Specification (Layer 3, flow) | 0.5, 0.8 | Spec (contradicts itself) |
| Vision doc (FIX #72) | 0.5 | Vision doc |
| Calculator constants | 0.5 ready, 0.9 complete, 0.2 warning | Labelled "from PoC" — **the PoC uses 20 / 100 / 10 on a 0–100 scale**, so these are not from the PoC |
| Layer 3 gate | 0.3, 0.7 | Code |
| Layer 3 quality label | 0.4, 0.7 | Code |
| Handler phases | 0.3, 0.6, 0.8 | `main.go:2551` |
| Accomplishment phases | 0.25, 0.5, 0.75 | `models/phase_accomplishment.go` |
| Agent `ContextMaturity` | 0.8 | `conversation_agent.go:1514` |
| Layer 8 | 0.3, 0.5, 0.6, 0.7 | `layer8_socratic_deepening.go` |
| Layer 5 | 0.2 | `layer5_unified_conflict.go:52` |
| Cache shortcut | 0.85 | `layer_adapters.go:422`, `layer4_gap_detector.go:76` |
| Constitutional comment | 0.5 | Stale: maturity gating was removed (FIX #74) |

The phase names also disagree. The reference calls 0.25–0.5 "analysis"; Moly's accomplishment model calls it "gathering" and calls 0.5–0.75 "analysis". The agent gates on `analysis` and `help` as "enough context to help", so it applies the band that the reference calls "design".

### M8 — Maturity categories are never fed
- `tools.MaturityCalculator` has eight categories with targets. `UpdateCategory` has no callers, so the categories stay at zero. `BuildPhaseMaturityWithFactors` calls `IdentifyWeakCategories`, which then marks every category weak.

### Other maturity defects
- `main.go:5005` `calculateContextMaturity` has **no callers** and its comment claims it gates safety. Its floor is 0.7, so it could never say "immature". Its SQL columns (`communication_style`, `tone_preference`, `goals`) do exist in `about_me`, so the function is valid code that is simply not called.
- Two summary writers update the same row: `main.go` (handler) and `tools/conversation_summary_manager.go`.

## 5. Reference logic (Socrates family)

Read from the local PyCharm projects (`/home/nireus79/PycharmProjects/`). Full comparison in `INVESTIGATION_REFERENCE_MATURITY.md`.

- **Accumulator (live path):** `socratic-agents/src/socratic_agents/quality_controller.py` `_update_maturity_after_response`. `answer_score = Σ value × confidence`; `score_after = min(100, score_before + answer_score)`, per phase. The comment states the purpose: incremental scoring prevents low-confidence specs from pulling down earlier answers' scores.
- Scores are stored in `project.phase_maturity_scores` and saved with the project object by the API (`projects_chat.py`).
- The overall score is the mean of phases with score > 0 (`models/project.py`). The docstring's own example (100 and 30 → 65) shows a drop when a new phase starts.
- The question loop does not read maturity (`socratic_counselor.py` `_generate_question`). Maturity is used only for a "phase complete" message at 100 and a confirmation prompt below 20 when advancing.
- Conflict resolution "replace" removes an old spec without updating the stored score.

## 6. Decisions made (2026-10-09, user)

1. Maturity is a persisted per-phase accumulator, following the reference (pillar 1).
2. Decreases happen only through explicit contradiction resolution that subtracts the replaced spec's contribution. Low-confidence noise never lowers a score (pillar 2).
3. Each facet is capped at a target, so one facet cannot dominate (pillar 3).
4. "Ready to help" requires the goal-blocking facets to be covered; the score is a secondary signal (pillar 4).

## 7. Open questions (not decided)

- The specification's thresholds. The plan uses facet coverage plus a score, so a single threshold is not required, but the score's "ready" value still has to be set.
- The meaning of "goal-blocking" for each goal type (communication goals only, or others).
- Whether the greeting and first message may produce a reply without any gap question.
