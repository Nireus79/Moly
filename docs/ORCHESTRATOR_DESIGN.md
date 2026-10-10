# Moly orchestrator: diagnosis, design and migration (2026-10-09)

Status: proposal. No code changed by this document. Decisions taken so far are in `IMPLEMENTATION_PLAN.md` (D1–D9) and the name-gate decisions (section 6h).

## 1. Diagnosis: why the code works in parts and fails as a whole

Measured in `moly-go` (not estimated):

| Measure | Value | What it means |
|---|---|---|
| `MessageProcessorHandler` | 3,618 lines | One function owns the whole pipeline |
| State writes inside it | 34 | Many writers, no single commit point |
| `return` exits | 20 | Many ways to finish, each with its own rules |
| Reads of `extractedContext` | 121 | Everything depends on one object whose timing varies |
| Contact save paths | 5 | Two of them create contacts with different rules |
| Reply overrides in the handler | 13 | The reply is changed after the agent has chosen it |
| `ConversationAgent` | 4,153 lines | Decides the reply and writes the text |
| Response exits in the agent | 17 | Each exit is a decision made locally |
| Gate/deepen decisions in the agent | 26 | Decisions spread across the file |
| Workflow switch cases | 39 | The reply type is a long chain of cases |
| `FIX #` comments | 147 across 23 distinct numbers | Fixes were added on top of fixes |

The unit tests pass (about 350 tests). They test parts: a parser, a repository, a layer. The failures in the logs are all at the joints: a value read before it is produced, a decision made after the thing it decides is already written, two writers for one fact. No test runs the whole pipeline in order, so no test can see a joint.

Concrete examples from the logs:
- The name answer ran before extraction, so it always saw an empty name.
- The greeting decision came after the contact was saved, so a greeting saved "the user" as a contact.
- The contact formatter replaced the agent's reply, so the agent's text was never shown.
- Layers 2, 3, 4, 5, 8, 9 and 11 returned a cached result from the same message, so the safety check did not run.

Diagnosis: there is no orchestrator. There is a long handler that calls components in an order that was built up over time, and each component decides for itself what happens next.

## 2. Is an agentic orchestrator the answer?

No, not for this part of Moly. Reasons:

1. **The flow is mostly fixed by the specification.** Safety first, then understanding, then identity, then goal, then clarification, then help. An agent that chooses its own next step would sometimes skip or reorder these, and that is the failure we are trying to remove.
2. **Safety has to be guaranteed, not likely.** The rule "Layer 2 runs before any reply" must hold on every path. A planner can skip it.
3. **Latency.** Each planner decision is an LLM call. The current message already takes 5 to 12 minutes on the local model.
4. **Reproducibility.** The same message must give the same decision, so that a test can assert it.
5. **The LLM is good at leaves.** Intent, extraction, goal wording, and the wording of questions are classification or phrasing tasks. They should stay inside stages, with a fixed contract, and never choose the next stage.

Recommendation: a **deterministic staged pipeline with explicit state** (a state machine). The LLM is called inside a stage. The stages and their order are code. An agentic planner can be considered later for a specific open-ended task, but not for the core flow.

So the answer to "do we need a different orchestrator": yes, a new one, and it should be a plain ordered pipeline, not an agent.

## 3. Principles the orchestrator must keep (invariants)

- **I1 Safety first.** No reply is produced before the safety stage has run on this message. A safety failure denies, it does not pass (fail closed).
- **I2 One reply decision per message.** A single function chooses the reply kind from explicit facts. Nothing downstream changes the kind.
- **I3 Sent equals saved.** The text that is returned and the text that is stored are the same value.
- **I4 Order is explicit.** A stage reads only what an earlier stage produced, or what was stored before the request.
- **I5 One writer per fact.** Each stored fact (contact, name status, goal, focus, maturity item, reply) has exactly one stage that writes it.
- **I6 Commit once.** State changes are written in one transaction at the end of the decision stage, except for the user's own message (saved first, so history includes it).
- **I7 The LLM classifies and phrases; it does not route.** Routing is code, driven by the classified facts.
- **I8 Every path ends in a reply or an explicit error.** An empty reply is an error, not a silent success.

## 4. The pipeline

Each stage: named inputs, named outputs, a precondition, and a failure policy.

| # | Stage | Inputs | Output | Writes | Failure policy |
|---|---|---|---|---|---|
| 1 | Ingest | request | `MessageContext` (user, conversation, text, history) | user message row | reject request |
| 2 | Intent | text | `intent` (greeting, asking, sharing, …), confidence | — | unknown intent, not greeting |
| 3 | Extract | text, intent | extracted goal, contact, traits, values, style | — (not stored yet) | no extraction, continue with empty |
| 4 | Identity | extraction, intent | `person` (none, named, unnamed, asked), `nameAnswered` | rename, reuse or create the person; set status | do not create a person if the write fails |
| 5 | Focus | goal, previous focus | `focus` (same, related, new), primary goal | goal and focus rows | keep previous focus |
| 6 | Safety | everything above | `verdict` (allow, deny) | — | **fail closed** on error: deny with the safe text |
| 7 | Assessment | focus, person, items | missing items (goal-blocking first), maturity counts | maturity items | counts stay as they were |
| 8 | ReplyPolicy | verdict, intent, person, missing items, focus | `ReplyPlan{kind, reason}` | — | pure function, cannot fail |
| 9 | Generate | `ReplyPlan`, context | reply text | — | fixed text fallback per kind |
| 10 | Commit | all of the above | — | one transaction: person, goal, focus, items, reply row, execution state | rollback, return error, send nothing |
| 11 | Post | committed state | — | summary, learning, analysis logs | never blocks the reply |

Stages 2 to 3 use the LLM. Stage 6 uses the constitutional evaluator and Layer 11 (rule-based denial, with the context rule). Stage 7 uses the maturity model. Stage 8 is the single decision point.

### ReplyPolicy precedence (stage 8)

Evaluated top to bottom; the first match decides:

1. Deny (safety verdict deny) → deny text.
2. Greeting (intent greeting) → greeting reply, nothing else.
3. Person unnamed and not yet asked → **name question** (fixed text).
4. Person asked and this message is the answer → apply the answer, continue (no question).
5. Goal missing (see D9, goal gate) → **goal question** (or general chat, as decided).
6. Goal-blocking item missing for the focus → **gap question** (one, Layer 4 wording).
7. Deepening gates passed → **Socratic question**.
8. Otherwise → **help** (LLM, with accumulated facts in the prompt).

Every kind is decided by this table, so a test can assert each row.

## 5. Exceptions and paths (all known from the logs)

| Path | Trigger | Handling |
|---|---|---|
| Greeting | intent greeting | no goal, no person, greeting reply |
| Harmful | safety deny | deny reply, nothing saved except the user message |
| Denial/withdrawal | Layer 11 with prior context | withdrawal reply; not on a first message |
| Name given | asked person, extraction gives a name | rename the same row |
| Name not given | asked person, no name | keep the label, status named, no question |
| Person without name, not yet asked | new unnamed person | name question |
| Same person mentioned again | label matches a stored person | reuse; no new person |
| Pronoun reference | no name, one named person of that relationship | reuse that person |
| Goal change | new goal, related or unrelated | new focus; old focus unchanged |
| Pasted third-party text | profile text | attribute to the person, not the goal (not implemented) |
| Clarification answer (A/B) | answer to a clarification question | apply to the referenced item |
| Conflict answer | answer to a pending conflict | resolve through conflict handler |
| Retry | same message sent twice | execution state: skip finished stages |
| Empty reply | agent produced nothing | error response, nothing saved |
| LLM failure | timeout or error in stages 2, 3, 6, 9 | 2/3: continue degraded; 6: deny (fail closed); 9: fixed text |

## 6. Mapping: where today's code goes

- Stage 1: auth, validation, user message save, conversation creation (handler lines ~880–1000).
- Stage 2: `DetectIntentWithLLM` (currently called from the handler after extraction; moved, done).
- Stage 3: `ExtractionPhase.Run` (done; unchanged).
- Stage 4: `ApplyNameAnswer`, the contact save paths (5 of them, to be reduced to 1), the name-question marking (moved after the save, done).
- Stage 5: `UnifiedOrchestrator` Layer 1 lock and `GoalCoherence` (lock on first real goal, done; the focus concept is not built).
- Stage 6: Layer 2 (`ConstitutionalEvaluator`), Layer 11 denial. The handler's second safety call at ~2500 is removed into this stage.
- Stage 7: Layers 3, 4 (gaps from missing items), maturity model (to build, phase 4).
- Stage 8: the 17 exits and 26 gates of `ConversationAgent` become one function, `DecideReply`.
- Stage 9: the text-writing parts of `ConversationAgent` (27 places) become one `Generate(kind)` with one prompt per kind. The 13 handler overrides are removed.
- Stage 10: the 34 writes are collected and committed once.
- Stage 11: summary, learning, message summaries, analysis logs.

## 7. Migration: strangler, one stage at a time

The old handler keeps working while the new pipeline is built beside it. Switch over stage by stage.

**M0 — Scripted end-to-end harness (first, before any stage moves).**
- A test that calls `NewAPIServer(scriptedLLM, tempDB)` and runs the handler over HTTP, with a session created in the test.
- `scriptedLLM` answers by prompt type (intent, extraction, gap, reply) with fixed JSON or text.
- Assertions for the three-message scenario: reply text, contact rows and status, primary goal, saved reply equals sent reply.
- Expected result today: failing, with the failing assertion naming the broken joint. This turns the live runs (5 to 12 minutes each) into a test that runs in seconds.

**M1 — ReplyPolicy as a pure function.** Extract the precedence table from the agent into `DecideReply(facts) ReplyPlan` with a table test per row. Behaviour changes only where the table differs from today's code, and each difference is listed.

**M2 — Identity and Focus stages.** Move name answer, person resolution and focus into one stage function with its own tests. Remove the 4 other contact-save paths.

**M3 — Safety stage.** One safety stage, fail closed, called once. Remove the second call and the cache paths (already removed for layers).

**M4 — Commit stage.** Collect the writes into one transaction. Test rollback.

**M5 — Generate stage.** One generator per reply kind. Remove the 13 handler overrides and the 27 text-setting places in the agent.

**M6 — Remove the old paths** and the dead code listed in `DEAD_AND_DUPLICATE_CODE.md`.

Each milestone ends with the harness green. Live checks happen once per milestone, not once per fix.

## 8. What this changes about how we work

- No more single-patch fixes to the handler or the agent. A fix is a stage change with a test row in M0 or M1.
- Before any change, the harness states the expected result. If the change does not make the harness pass, it is not done.
- The live server is used to confirm a milestone, not to find bugs.

## 9. Decisions needed

1. Approve the approach: a deterministic staged pipeline, not an agentic orchestrator.
2. Approve M0 first (scripted harness). It is the only way to verify the whole pipeline in seconds.
3. Confirm the open product choices that the stages depend on: D9 (goal gate: is a goal required before advice, or may a user chat without one?) and whether a pasted third-party profile should be stored as traits of the person (my recommendation).

## 10. Orchestration change plan (ordered; nothing is removed until step 6)

Rule for every step: the scripted end-to-end test (step 0) must pass the scenario that the step touches, and the full suite must stay green. No live run until step 7.

### Step 0 — Scripted end-to-end test (no behaviour change)
- Create `moly-go/orchestration_test.go`: build `NewAPIServer(scriptedLLM, tempDB)`, create a user and a session, call `MessageProcessorHandler` over HTTP, read the reply and the database.
- `scriptedLLM` answers by prompt type: intent (greeting / asking / sharing), extraction (JSON with contact name or label), gap questions, reply text. The answers are fixed per scenario.
- Scenario S1 (the three messages from the logs), with expected results written first:
  1. "Hi Moly": reply is a greeting; no contact row; primary goal empty.
  2. "I want to write a first message to a girl I saw on fetlife.": reply is exactly "Can you give me a name for the girl you mentioned?"; one contact, name_status `asked`, name "girl".
  3. "Her name is Christine.": same contact row renamed to Christine, name_status `named`; no second contact; reply continues the goal (not the name question).
- Expected: S1 fails today, and the failing assertion names the broken step.
- Acceptance: the test exists and runs in under 30 seconds.

### Step 1 — ReplyPolicy as a pure function (table test)
- New file `agents/reply_policy.go`: `DecideReply(facts ReplyFacts) ReplyPlan`. `ReplyFacts` holds: deny, greeting, person state (none, unnamed, asked, named), name answered this message, goal present, goal-blocking item missing, deepening allowed, has clarification. `ReplyPlan` holds: kind, reason.
- Precedence as in section 4 of this document. One table test row per kind and per precedence conflict.
- The agent calls `DecideReply` once, and the old gates are left in place for now but must agree with it. A test asserts agreement on S1.
- Acceptance: table tests green; S1 step 2 expects kind NameQuestion.

### Step 2 — Identity stage (one place for the person)
- New file `agents/identity_stage.go`: `ResolvePerson(repo, userID, extraction, intent) PersonResult`. It does, in order: apply the name answer (rename), reuse the person by label or by the single named person of that relationship, create an unnamed person otherwise, and return the person state. It is the only function that writes contact rows.
- Replace the four contact-save paths in `main.go` (the one after extraction and the three duplicates) with one call. Remove the pre-extraction block (already moved), and the `SaveExtractedContact` path if unused after this step.
- Acceptance: unit tests for each person path (rename, reuse, unnamed create, pronoun match); S1 steps 2 and 3 pass the database assertions.

### Step 3 — Intent and goal stage
- Intent is decided once, right after extraction (already done); move the call and the greeting cleanup into `agents/intent_stage.go` with the identity stage's inputs.
- Goal lock and focus: use `GoalCoherence` only when a goal exists. A greeting never reaches the goal logic.
- Acceptance: S1 step 1 has no goal, step 2 locks the goal once.

### Step 4 — Safety stage, one call, fail closed
- Remove the second safety evaluation in the handler (around the former line 2696) and the skip of Layer 2 for clarification answers (`skipLayer2OnClarification`). Layer 2 runs once, in the orchestrator, before the reply.
- Layer 2 error: deny with the safe text (no pass-through).
- Acceptance: a test with a scripted harmful message is denied and nothing else is generated; a test with a Layer 2 error is denied.

### Step 5 — Commit stage: one transaction, one reply
- The handler collects the writes (person, goal, focus, reply row, execution state) into one transaction committed after the reply is resolved (`resolveReplyText`, already in place).
- Remove the 13 reply overrides in the handler. The formatter (`ContactResponseFormatter`) may add metadata or the clarification text only through `ComposeReplyWithClarification`.
- Acceptance: a test forces a commit failure and asserts that no reply is returned and nothing is saved.

### Step 6 — Generate stage, dead code removal
- One generator per reply kind. The name question is fixed text (already in place). The agent's 27 text-setting places are replaced by the generator for the kind chosen in step 1.
- Only now: remove the dead code that the steps made unreachable: the handler and agent items listed in `DEAD_AND_DUPLICATE_CODE.md`, and the 26 skipped candidates from the earlier scan (`/tmp` list). The removal tool and the candidate list are in the scratchpad.
- Acceptance: S1 passes; full suite green; `go vet` clean.

### Step 7 — One live check
- Restart the server from `bin/moly`, run S1 once in a new conversation, compare with the scripted result. Any difference is a bug in the harness or in the live model, and is recorded as such.

### Risks and order dependencies
- Steps 1 to 3 change behaviour; step 0 must exist first, or the change cannot be checked.
- The intent is decided without history (step 3) until the history is loaded earlier. Accepted for now; recorded.
- Latency is not changed by this plan. Each live message still takes minutes on the local model.
- The dead-code removal is deferred to step 6 because the handler and the agent are being rewritten in steps 1–5.

### Step 0 status (2026-10-09): done
- `moly-go/orchestration_test.go` runs the real handler with a scripted model and a temporary database. Scenario S1 passes in under a second.
- Before the test passed, it found a second writer for the name: the regex naming detector (`ProgressiveNamingDetector`) matched "name is Christine" and set the lifecycle `status` to "named". Its call in the handler is removed; names now come only through the name gate. The detector file is for the dead-code step.
- Limit: the scripted model is cooperative and the scenario is narrow. It proves the joints for these three messages, not the live model's behaviour. Step 7 (one live run) still applies.


## Step 1 status (2026-10-09)

Done: the agent's reply kinds are one decision. `DecideReply` (agents/reply_policy.go) owns the precedence:
deny, safety alert, greeting, name question, spent questions, clarity, gap, principle, persistent, topic shift,
contact question, intent check, Socratic, help. Facts are found by lazy probes (`decideByProbes`), which stop at the first
deciding fact, so a decided reply costs no extra model calls. The question kinds are rendered in agents/reply_exits.go.
Removed: the strategy gap path, the workflow gap and intent branches, the contact and saturation blocks, and the
keyword fallbacks for preferences, values, style, contact and intention.

Left: the conflict question inside the default help branch, the end-of-Run response validation and re-writing,
and the maturity phase inputs (`ctx.Maturity` is never set, so the phase checks always read "initial").

## Step 2 status (2026-10-09): done
`agents.ResolvePerson` (agents/identity_stage.go) is the only message-path writer of contact rows. In order: apply a name answer;
reuse the person by exact name, or by the only named person of that kind when the message gives no name; create an unnamed or named
person otherwise; fill blank relationship and traits (never overwrite); merge traits given by exact name or label; mark an unnamed
person as asked (not in the message that answered). A greeting writes nothing.
Removed from main.go: the early save, the post-extraction save, the name-answer and name-marker blocks, the raw SQL insert after the
conflict check, the dead agent-contact save (the agent always set an empty name), and the `LIKE '%name%'` trait overwrite.
Unchanged by design: user-initiated writes (contacts API, conflict resolution after the user approves).
Tests: 9 unit tests with an in-memory repo (agents/identity_stage_test.go) plus the scenarios.
Open: S5 (a refused message still saves a person) needs the safety stage before this one (step 4).
Dead and left for step 6: agents/contact_manager.go, agents/clarification_response_handler.go, storage/dataflow_capture.go.

## Step 3 status (2026-10-09): done
- `agents.DecideIntent` (agents/intent_stage.go): the intent is decided once, right after extraction. No detector or no message gives "unknown", never a greeting. `StripGreeting` removes the goal and the person from a greeting; `WithoutGoalEntities` removes goal entities.
- `agents.ResolveGoal` (agents/goal_stage.go), pure and table-tested: no stated goal continues the locked one; the first goal stated is locked; a later different goal is reported as changed and does not replace the lock. Layer 1 calls it.
- Removed: the duplicate greeting cleanup in the handler, the unused `IsMessageOne` flag, the inline lock logic in Layer 1.
- Scenarios: S1 asserts no goal after the greeting, the goal locked once after message 2, and unchanged after message 3. S11 asserts a later goal does not replace the lock.
- Still open: the intent is decided without conversation history (accepted, recorded in the risks above); `GoalCoherence` is still not wired in; the "restore original intention" block in the handler (clarification-answer path) overlaps with ResolveGoal and is left for step 5.

## Step 4 status (2026-10-09): done
- `agents.EvaluateSafety` (agents/safety_stage.go) is the one safety call for a message. It runs in the handler after the conversation history is loaded and before extraction, so before anything is saved from the message. Any failure returns `ErrSafetyUnavailable` and the handler answers 503 with no reply and no assistant row (fail closed).
- `agents.SeverityGate` maps prior maturity to the blocking severity (named bands 0.3 / 0.5 / 0.7, table-tested). It uses the maturity before this message, not the one the message produces.
- A refused message (obvious harm) saves no person: the identity stage is skipped. Scenario S5 passes with no skip.
- Layer 2 reads the verdict from `AnalysisContext.SafetyVerdict` and does not ask again. Without a precomputed verdict it evaluates; if that fails it returns `ErrSafetyUnavailable` (it used to fail open with `Allowed: true`), and the handler answers 503.
- Removed: the handler's second ("PRIMARY") evaluation, the `deferredSafetyCheck` flag, and `skipLayer2OnClarification` (Layer 2 is now cheap, so it always runs). The maturity and phase persistence that shared the old guard now runs whenever there is a message.
- S12 asserts one evaluation of each user message, including a message that answers a question. S2 asserts the fail-closed 503.
- Kept on purpose: the agent's check of its own reply (`conversation_agent.go`, an evaluation of Moly's output, not of the message).
- Open: the evaluator itself still contains word-based filters (`isContextualViolation`: platform names, common context words, single-word evidence, action patterns). They decide which model-reported violations count. That conflicts with the no-hardcoding rule and needs a decision before it is replaced.

## Step 5 status (2026-10-09): done
- `turnCommit` (moly-go/turn_commit.go) is the commit stage. It has the same `Exec` shape as `*sql.DB`. The ten writes that make up a turn are buffered during the turn and run in one transaction after the reply is resolved: the user row, the assistant row, the reflection, style, intention, goals, values, principles, characteristics and system-feedback upserts, the locked goal / working state (`savePreviousExtractionWith`), and the "asked" mark on a person who is being asked for a name. If any write fails the whole transaction rolls back, the after-commit actions (the in-memory goal cache) do not run, and the handler answers 500 with no reply.
- Repository methods `SetNameStatusWith` and `SaveWith` take a `database.Executor`, so they can join a transaction.
- Because the "asked" mark is part of the commit, a retry after a failed turn finds the person still unnamed and asks the name question again. Scenario S13 asserts: a forced commit failure returns no reply, saves no user or assistant row, locks no goal; the retry asks the name question and saves exactly one user and one assistant row.
- A reply asks at most one thing: when the agent's reply already asks a question (`agents.AskedQuestion`), the contact formatter's A/B clarification is dropped. The formatter's "Got it, so your …" prefix was removed (it only applied to `Status == "named"`, which no contact ever has) with the tests that described it.
- Test seam: `APIServer.beforeCommit` lets a test force the commit to fail.
- Not in the commit, on purpose (derived or best-effort): conversation summaries, interaction log rows, the learning agent, safety incident rows, execution-state updates.
- The identity stage writes the person before the commit (later stages read the saved person). A failed turn can therefore leave a person row; it stays unnamed, so the retry is correct.
- Open: a "crisis" result from the parallel risk monitor blanks the reply (`agentResp.Response = ""`), which then fails "empty reply" and returns a 500 with no support message. A defined crisis reply is a product decision. The risk monitor is a third safety evaluation; it should be merged into the safety stage once the crisis reply is decided.

## Step 6 status (2026-10-09): dead code removed; generators not converted
Removed (backup of the state before this step: /home/nireus79/moly-backup-2026-10-09-step6):
- Every function with no reference anywhere, and every function referenced only by tests, together with those tests. The scan strips strings and comments; it was run in rounds until nothing more fell out. Also the aggregate "gate" tests that only called deleted tests.
- 11 shell files left empty by that: crypto.go, agents/progressive_naming.go, agents/subject_resolver.go, auth/session_repository.go, database/phase4_validator.go, database/context_loader.go, config/container.go, config/logging.go, storage/dataflow_capture.go, storage/batch_writer.go, tools/session_manager.go; 7 test files with no tests left.
- 75 unused types, in three rounds.
- Non-test code: 55,569 lines before the first removal, 49,029 at the start of this step, 40,151 now. Tests: 390 test functions before this step, 272 now (the 118 removed tested removed code).
- Kept deliberately: `savePreviousExtraction` and `ContactRepository.GetByID` (tests of real behaviour use them).
- Test infrastructure fix found on the way: `database.Init` is a process-wide singleton, so the first test to call it fixed the database path. Tests in package main now share one database directory created in `TestMain`.
Not done (needs a decision): "one generator per reply kind". The question kinds are rendered in agents/reply_exits.go, but their wording is fixed text: the topic-shift question, the principle questions (one template per principle), the persistent-question templates, two different denial texts, and the "I'm listening." fallback. Several of those ask two or three questions in one reply, which breaks "one question at a time". Converting them to model-written text with a contract (exactly one question, naming the subject) and a fixed fallback changes how Moly sounds and adds a model call per question.

## Step 7 status (live run, 2026-10-09)

Run: `live_check.sh` against `bin/moly` (started from `moly-go/`, because `config/constitution.yaml` is read by a relative path) with the local `mistral` model. Result: joints hold, 1 check failed, 2 live-model faults found.

- Message 1 (greeting): passed. Greeting reply, no person, no name question.
- Message 2: the policy chose the name question correctly. The check failed because the live model named the person "girl on fetlife", so the question read "...a name for the girl on fetlife you mentioned?" The check expected "girl". Harness fault (check too strict); the label comes from the model.
- Message 3: person renamed to Christine, one row, no second name question. Policy chose `gap_question` (gap probe), which is the rule. LIVE-MODEL FAULT: Layer 4 generated "What is the full real name of the person you have referred to as Christine?" right after the user gave the name, so Moly questioned an answer it had just asked for. The goal in the strategy log was "Identifying a third person", not the locked goal; not yet checked against the stored lock.
- Reply text came back wrapped in quotation marks (model output; not stripped).
- Latency per message: Layer 4 63s, reply about 2 to 3 minutes. Deferred by decision.
- `live_check.sh` prints `nameStatus` as empty (the contacts API uses another key); the contact check by name works.

Open from this run: (1) gap generation must be told the name was just answered and must not ask about it; (2) confirm the locked goal survives message 3 in the live run; (3) strip quotation marks around the reply.

### Step 7 follow-up (same day)
- Fixed: Layer 4 gap prompt now carries a "people in this conversation" block from the stored contacts (named = answered, never asked about; the name exception applies only to a person without a name). Unit test `TestPeopleStatusBlock`.
- Fixed: a reply wrapped whole in quotation marks is unwrapped (`unwrapQuotedReply`, tested).
- Check: the stored goal survived message 3 (log: primary goal saved unchanged). The `live_check.sh` message 2 check now accepts the model's label.
- Found, not fixed: for "Her name is Christine." the extractor invented the goal "Identifying a third person"; Layer 1 reported "goal changed" and Layer 4 filtered gaps by it. A name answer states no goal. This is an extraction prompt contract: a message that only answers a question has no goal. Needs a decision on how to word it (and a scripted scenario for it).

### Step 7 second live run
- All checks passed. Message 3 still asked "physical features... that help you identify Christine" because the extractor again invented a goal ("Identifying a person"), Layer 1 reported "goal changed" and Layer 4 asked for it. Locked goal unchanged.
- Fix: the extraction prompt's `intention` field now says to leave it empty when the message only answers a question just asked (a name, a detail). Not provable by the scripted model; needs a third live run.

### Step 7 answer-only decision (built)
- Third live run: the prompt rule did not stop the extractor inventing a goal ("Introducing Christine to the conversation") from a bare name answer.
- Built: the intent call (same call, no extra model call) now sees the conversation history and returns `answersQuestion`. `DecideIntent(src, message, history)` sets `AnswerOnly` only when the model claims it AND Moly's message is the last one in the history (the claim alone is not enough; a greeting is never an answer). `StripAnswerGoal` empties the stated goal and keeps the person. The handler then treats the goal like a greeting's: goal entities are removed, so the locked goal continues and no layer sees a "goal changed".
- This also ends the "intent decided without history" limitation for the intent call.
- Test: S14 (the scripted extractor invents "identify who Christine is" from the name answer; no later prompt may carry it; the lock stays). Mutation check: with the decision disabled, S14 fails. Unit test `TestDecideIntentAnswerOnly`. Harness fix: the scripted intent matches the current-message line only, because the prompt now carries history.
- Fourth live run pending.

### Step 6 completed: one question per reply, crisis reply (2026-10-10)
- Crisis reply (decided by the documented behaviour, README "Crisis Language" and MOLY_11_LAYER_SYSTEM "Layer 2 Blocking"): `agents.CrisisReply`, fixed text with resources and no question. A crisis result from the risk monitor now returns it with status 200 (before: empty reply, 500). Scenario S15.
- One question per reply: instead of converting each template to a model prompt, every question reply passes through one function, `singleQuestion` (`agents/single_question.go`): it already meets the contract (exactly one "?" and at most 30 words) -> unchanged, no model call. Otherwise the model rewrites it once as one short question; if that fails the contract or the call fails, the first question of the original is used. Applied to: contact, intent check, clarity, gap, principle, persistent, Socratic (both paths). The name question is already one question. Tests: unit tests, scenario S16 (a three-question wording yields one).
- Not changed: the topic-shift question (already one question under 30 words).
- Risk monitor still a third safety call per message. Evaluator word filters: see the explanation given to the user; undecided.

### Evaluator word filters removed; refusal wording changed (2026-10-10, user decision)
- Removed `isContextualViolation`, `isMultiwordHarmPhrase`, `reasoningExplainsViolation` (platform names, common words, single-word rule, harm-word auto-accept, action and principle keyword lists) and the `strings.Contains(evidence, "myself")` branch in `ToSafetyAlert`.
- Replaced by `confirmViolation`: one model call per claimed violation, asking whether the evidence is a real violation in the context of the whole message. If the call fails or is unreadable the first judgement stands. The evaluator's JSON now carries `harm_kind` (`self_harm` | `illegal` | `none`); the alert type and the refusal follow it.
- Refusals carry no help lines: self-harm -> "I can't help with that. Please consider asking advice from a specialist."; illegal -> "... a law expert."; other kinds keep the short principle-specific refusal. `CrisisReply` (risk monitor) is the self-harm text. The reply is the alert message alone (the duplicated title prefix is gone).
- Principle: an unclear message is never a crisis. Only explicit statements are; unclear ones get clarification and Socratic questions first.
- Tests: `tools/constitutional_confirm_test.go`; scripted model answers the confirmation call.

### Product decisions of 2026-10-10 (user) and their state
1. Goal before advice: not required. A greeting gets "I am Moly. How can I help you?" (written by the model in the user's language, under the one-question contract; fixed English text as fallback). DONE (`agents/greeting_reply.go`; the greeting kind now has its own reply instead of falling into the help text).
2. Mixed information: Moly does not take it all at once. It may say in one short sentence what it understood, then asks about one thing; the rest waits. DONE in the rewrite prompt and the gap prompt; the contract allows 45 words (one "?"). NOT DONE: "when in doubt, ask before saving" (a doubtful contact or fact should be confirmed before it is stored).
3. Maturity counts logical gaps: answered questions / (answered + open). DONE as the one decision value `agents.GapMaturity` (feeds the deepening gate and the safety severity gate). The accomplishment-based value remains only for phase labels in metadata; its removal is still to do. Open: the saturation exit (stop asking after 3 questions) contradicts "ask all gaps first".
4. Safety is not strict; the real fallback is the questions. Merge the risk monitor into the safety stage: BLOCKED by the permission classifier on 2026-10-10 (it removes a safety check); waiting for the user to allow it.
- Found and fixed while doing this: `detectRepeatedConcern` decided "the user persists" by words ("but", "actually", "still") and by words in Moly's last question ("how", "what"); any earlier Moly question plus one such word triggered deep principle questions. Now: only after a raised principle concern (stored reply kind), judged by the model. Scenario S17; S9 exposed it.

### Decisions of 2026-10-10 (second round) and what was built
- Risk monitor merged and removed (user: a monitor that sees one raw sentence gives false positives; merging is the lower-complexity choice and the evaluator already turns an unclear violation into a clarification question). The self-harm refusal comes from the one safety stage (`harm_kind`), also used by the agent's early exit, so there is one refusal text per kind. Dead `LastRiskAssessment` branches, the `RiskMonitor` field/interface and unused types removed. Scenario S15 rewritten.
- Goal shifts: a shift is "different" only by model judgement (`JudgeGoalRelation`: same / refinement / substep / different); text is compared only to skip the call when identical. A failed judgement is "same". `ResolveGoal(locked, stated, relation)`. NOT DONE: the lock is still never replaced, even if the user confirms a real switch.
- Readiness replaces the saturation counter: the gap prompt now says a message can be clear and returns `[]`; each gap must carry the `assumption` it prevents, or it is dropped. The saturation probe, `KindBestEffort`, `QuestionsSpent` and `isGapQuestionLLM` are deleted. Scenario S18.
- Per-fact confidence (`agents/fact_confidence.go`): min of the model's number, a basis ceiling (stated / implied 0.7 / guessed 0.4) and whether the evidence is in the message; no number from the model counts 0.5, not high. The constants 0.90 (intention) and 0.85 (contact) are gone. A fact under 0.6 is in doubt: a person is not saved, a goal is not locked; the reply is one confirming question (`KindConfirm`, after the name question). Scenarios S19, S20. The unread `lowConfidence` flag is removed.
- Also: `detectRepeatedConcern` is model-judged and only after a raised principle concern (S17); greeting reply is its own kind.
- Not done / next: replace the lock after a confirmed switch; `ConfidenceBasedClarifications` (generic questions at a fixed 0.80) should be checked for removal; the help prompt for the "mental_health" topic still says to suggest professional support (against the no-help-lines rule); live run of all of this.

### The three open items (2026-10-10): done
- Goal lock replacement: a goal the model judges different is not taken over. Layer 1 sets `GoalSwitch`; the reply is one confirming question (`KindConfirm`); the proposal is saved as `PendingGoal` and lasts one turn. Next message: `JudgeSwitchAnswer` (model) reads the user's answer; only a clear yes replaces the locked goal (`PrimaryGoal`), no or unclear keeps it. Scenario S21, unit tests.
- `ConfidenceBasedClarifications` removed: it generated generic questions at a fixed 0.80 every message, wired them into Layer 4, where the layer's own result overwrote them. The whole chain (generator, extraction output field, handler capture, orchestrator wiring, context field) is deleted. Backup before: /home/nireus79/moly-backup-2026-10-10-cbc.
- The "mental_health" topic guidance no longer tells the model to suggest professional support: it listens, validates, does not diagnose and does not point to help lines unless asked.
- Still open: a live run of everything since step 7 (`live_check.sh`), and a second `live_check.sh` scenario for goal switch and doubt (the script only covers the three S1 messages).

## Live check 2, rounds 2-3 (2026-10-10)
- The extraction prompt example "girl on fetlife" leaked into messages with no content ("Yes, ..."). The example is gone; the prompt says to omit the contact when the message names no third person, and to leave the intention empty for a bare yes/no.
- The safety verdict failed closed (503) because the model wrote `"is_direct_harm": "false"` as a string. `tools.LenientBool` reads strings and numbers; an unreadable value is false. A missing verdict still fails closed.
- The output gate (Moly's own draft flagged by the evaluator) no longer swaps in a canned refusal: the input already passed safety, so the draft is dropped and Moly asks one question (`replacementForBlockedReply`). Refusals belong to the safety stage only.
- Gap prompt: a gap must change the answer (wrong, unsafe or useless without it); a detail that only makes the answer more personal is not a gap.
- Logs: `[GapMaturity] answered= open= -> x` and `[FactConfidence] goal|person: ...` (no message text, no names).
- Scenarios S22 (gap maturity 0 -> 1 when the gap is answered) and S23 (a guessed person is held, saved once stated).
- `live_check2.sh` takes sections: `./live_check2.sh C E` (B is added automatically when C is asked).
- Round 3 finding: "Yes, that is what I want to work on now." made the extractor produce a person labelled "that", and Moly asked "Can you give me a name for the that you mentioned?". The intent stage had already marked the message answer-only. `PersonInput.AnswerOnly`: an answer may carry a name or point to a known person, but never creates a new unnamed one.

## The user's way out (2026-10-10)
Decision: Moly's questions are good, but the user may bypass them and take the result with whatever is known, then correct it afterwards by giving information. This ends the endless question loop without a static count: the user decides, the model judges it, and risk still stops it. Over time AboutMe holds enough that Moly needs fewer questions.
- Judged by the intent model: `wantsResultNow` (the user says to go ahead / just do it / does not want more questions, or pushes for the result instead of answering). Not set for a greeting or a plain answer. `IntentDecision.WantsResultNow` -> `Context.ResultNow` -> `ReplyFacts.Bypass`.
- Bypass skips only what improves the result: the name question, an open gap, a topic-shift question, an unclear-contact question, the Socratic question. It never skips: deny, safety alert, a doubtful fact, an unclear message (Clarity), an unclear intent, an engaged principle, a raised concern (persistent). Those are risk, or the reply cannot be made.
- The reply is generated with `resultNowGuidance`: give the result, no questions, one short sentence on what was assumed, invite correction. Token limit 400 for that reply.
- The questions that were open are marked `skipped`, not `answered`: maturity does not rise (it counts answered over answered+open), and they are not asked again (`SkipQuestionsAnsweredAt`; the answered-texts filter includes skipped).
- Tests: policy table rows, `TestDecideIntentWantsResultNow`, S24 (gap open, user takes the result: no question, maturity stays 0; mutation checked), S25 (a principle concern still asks). Live: section G of `live_check2.sh`.
- Open: every next message marks the open questions answered at the top of the request, before the model has judged it. A reply that does not answer (a new topic) still counts, so maturity can be inflated. The bypass corrects this for one case only.

### The way out is closed at maturity 0 (2026-10-10)
Decision: the user cannot bypass from the beginning. While gap maturity is exactly 0 (no question answered yet), Moly pushes back and asks for a minimum; after one answer the way out opens.
- `WayOutOpen(maturity) = maturity > 0`. The check runs once the intent model has said `wantsResultNow`, after the questions the reply just marked answered are turned into `skipped` (so the user's "just write it" is not counted as an answer).
- Refused with a question open: the skipped questions are reopened (active) and the most important one is asked by a new exit, `KindMinimum` ("I need at least this to get started: ..."). It sits after the name question and a doubtful fact, before clarity and the gaps. Gap detection cannot do this itself: it filters out gaps whose question is still active.
- Refused with nothing open (first message): the way out is simply not used; the normal flow asks the gap, or helps if there is none. A clear message with no gaps still gets its answer.
- Questions are now tracked by id (`MarkConversationQuestionsAnswered` returns ids; `SkipQuestions`, `ReopenSkipped`). A first version matched them by the answer timestamp in seconds, which also caught the previous answer when two messages came in the same second (found by S24).
- Tests: policy rows, `WayOutOpen`, `minimumQuestion` contract, database skip/reopen (same-second case), S24 (one answered, one open, way out taken), S26 (refused at 0, question stands, an answer raises maturity). Live: section G, four messages.

### The @go tag and a stricter spoken way out (2026-10-10)
- `@go` (`agents.WayOutTag`, one constant) is an explicit command, like a slash command, not a keyword list: the one certain way out. `StripWayOutTag` finds it as a whole word in any letter case (`bob@go.example` and `@goish` do not count), removes it before anything else reads the message, and a message holding only the tag becomes "Go ahead with what you have.". The tag wins even over an answer-only message. It does not open the way at maturity 0.
- When the way is refused at maturity 0, the pushback adds one statement after the question: "You can write @go if you want me to go ahead with what I have." (the one-question rule counts question marks, so it still holds).
- The spoken way out ("just write it") is kept but needs the model to quote the user's words (`wantsResultNowEvidence`, which must be in the message) and a confidence of at least 0.6 (`wantsResultNowConfidence`; none given counts as unsure). Otherwise it is not taken and Moly goes on asking: the safe side.
- Tests: `TestStripWayOutTag`, `TestDecideIntentWantsResultNow` (no quote, invented quote, low and missing confidence, greeting, answer), S27 (tag takes the result; the tag never reaches a prompt), S28 (tag refused at maturity 0, with the hint). Live: section H.
- Not done: telling the user about the tag after several questions in a row (only the maturity-0 pushback mentions it).

## The skip button (2026-10-10) - replaces the earlier ways out
Decision (user): the way out of Moly's questions is a button, not words. The spoken route ("just write it", judged by the model), the `@urgent`/`@go` tag and the maturity-0 pushback (`KindMinimum`) were removed; the three sections above ("The user's way out", "The way out is closed at maturity 0", "The @urgent tag ...") describe code that no longer exists, except where this section repeats it. The reason: a click is certain, discoverable, and needs no model judgement.
- **When it is shown:** the server puts `metadata.canSkip` on a reply. It is true only if the reply is a question the user may skip (name, gap, topic-shift, contact, Socratic: `SkippableKind`), gap maturity is above 0 (`WayOutOpen`: at least one question answered), and no conflict waits for confirmation. It is never shown under a doubtful-fact, unclear-message, unclear-intent, principle, persistent-concern, safety or denial reply.
- **When it is pressed:** the extension sends the fixed message "Go ahead with what you have." with `skipQuestions: true`. The server does not trust the flag: it checks `WayOutOpen` against the maturity as it was before this message marked anything. A press at maturity 0 (a stale or forged request) is ignored and the message is handled as a normal reply.
- **What a press does:** the questions the message just marked answered are set to `skipped` (not answered: maturity does not rise; not asked again). The press is handled like an answer-only message (no goal, no new unnamed person). `ResultNow` -> `ReplyFacts.Bypass` skips only what improves the result; the risk probes (principle, persistent concern) and safety, doubtful facts and unclear messages still run and still stop it. The reply is written with `resultNowGuidance` (give the result, no questions, one sentence on what was assumed, invite correction; 400 tokens).
- **Tests:** `TestCanSkip`; S24 (no button at maturity 0, button after one answer, press gives the result, skipped gap is neither answered nor open), S25 (a principle concern still asks after a press), S26 (a forced press at maturity 0 is ignored), S27 (no button under a principle question), database skip test (the earlier answer in the same second stays answered). S24, S25 and S27 were mutation-checked (button always shown; skip removing the principle concern).
- **Extension:** `ChatInterface.tsx` shows "Skip questions" under Moly's last message when `metadata.canSkip` is true and sends `skipQuestions: true`. The extension must be rebuilt (`npm run build` in `moly-extension/`); `tsc` has older errors unrelated to this change.
- **Not done:** the hint about the button needs no text (the button is its own hint). The assumption sentence in the result is still only an instruction to the model.
- After a skip (2026-10-10): only the questions Moly actually asked are stored (the top gap of a reply); the other detected gaps are not saved, they are detected again on every message. A skipped question is `skipped`: it is not pending (pending means `active`), it does not count toward maturity, and it is not asked again. Bug found by S28 and fixed: the exact-text filter (`HasQuestionTextBeenAnswered`) counted only `answered`, so a skipped question came back when the gap detector produced the same text; it now counts `skipped` too (the gap prompt already listed skipped questions as answered). Matching is by exact text; a reworded version of a skipped question relies on the model reading the "already answered" list in the gap prompt. A follow-up such as "it is good but too formal" is a normal message: new gaps are detected from it, a genuinely new gap is asked (S28), and the earlier tone question is not.

## Pending questions and gap ranking (2026-10-10) - replaces "a skipped question is not asked again"
Decision (user): the skip button is always shown, faded when not allowed; a skipped question is saved as pending; when the user resumes, pending questions are evaluated together with any new gaps.
- **Button:** `ChatInterface.tsx` shows "Skip questions" next to the input at all times; it is enabled only when the last Moly reply has `metadata.canSkip` (the server rule is unchanged).
- **Pending = status `skipped`.** It is an unresolved gap: `ConversationGapMaturity` counts it as open (after one answered and one skipped, maturity is 0.5, not 1). It is no longer filtered by the exact-text "already answered" filters (they count `answered` only).
- **Each later message (Layer 4, `mergePendingSkipped`):** the model judges which pending questions the message answers (`JudgePendingAnswered`, one call only when pending questions exist; unreadable or failed means none closed). Answered ones become `answered` (maturity rises). The others join the newly detected gaps as candidates (`Type: "pending"`, severity from the stored priority). The press of the skip button itself is never judged (`AnalysisContext.SkipPressed`).
- **Ranking (the answer to "is there a mechanism that picks the gap first"):** before this change, no. `rankGapsByImpact` ranked only the old static gaps by hardcoded type names; the model's gaps were never ranked and the reply asked whichever the model listed first. Now `rankGapsBySeverity` ranks the whole candidate list (new and pending) by the severity the model gives each gap against the goal (high: the answer would be wrong, unsafe or useless without it; medium: clearly worse; low: nice to know), then confidence; the gap prompt asks for this rating and for the most blocking gap first. At equal severity a new gap goes before a pending one. The reply asks the first. `rankGapsByImpact` and `calculateGapImpact` are still called by the static analyzer (almost empty since FIX #75) and could be deleted.
- **Asking a pending question again** reactivates its row (`ReactivateSkippedByText`, answer time 0 because the reader cannot scan NULL), so there is no duplicate; the gap's severity is stored as the row priority (1 high, 2 medium, 3 low) so it comes back at the same weight.
- **Goal switch:** when the user confirms a different goal, the pending questions of the old goal are `cancelled` (`CancelPending`).
- **Tests:** S28 (pending returns, same row active, maturity 0.5), S29 (answering message closes it, maturity 1), S30 (a high pending question goes before a new medium gap), S31 (a goal switch cancels pending), database tests (pending list, reactivation readable, cancel), `TestCanSkip`. S28-S31 mutation-checked. S24 now expects maturity 0.5 after a skip.
- **Cost and limits:** one extra model call per message while pending questions exist. The judge sees only the current message, not the history. After a result, a message like "too formal" does not answer a skipped question about her interests, so the question is asked again if it is still the most important gap; the user can press skip again (maturity stays above 0). A skipped question is per conversation. Matching of a re-listed pending question to its row is by exact text.

### Live run of section G, round 1 (2026-10-10) - three findings
- **The result of a skip had no request to work from (fixed).** The reply prompt only said "you have prior conversation history"; it did not contain the history. After the press the model saw "Go ahead with what you have." and wrote text about a friendship instead of the thank-you note. S24 passed because the scripted reply never looked at the prompt. Now `resultNowRequestBlock` puts the last 10 turns (not the press) in the prompt and the guidance says the request is in them. S24 requires the original request in the prompt (mutation-checked).
- **A pronoun or words about the text became a person (fixed).** "she" and "a bit too formal" were saved as unnamed people and Moly asked "Can you give me a name for the she you mentioned?". Before saving a new unnamed person the model now judges whether the words refer to a person (`JudgePersonLabel`, one small call, only at that moment; unreadable or failed counts as a person). A named person is never judged. S32, mutation-checked.
- **Goals are still invented from messages with no goal (not fixed).** "Warm and informal, we are friends and she is always helping me." answered Moly's question, but the extractor read "To continue with some ongoing matter" and the relation judge called it different, so Moly proposed a goal switch. The intent model did not mark the message as an answer. A pressed skip with nothing answered yet (an API-only case, the button is disabled) is handled as an ordinary message and produced a meaningless question.

### Live run of section G, round 2 (2026-10-10)
- **The press was judged as an unclear message (fixed).** After the press Moly asked "Could you please share a bit more about what you're hoping to achieve..." (`intent_check_question: what the user wants is unclear`): the clarity and intent checks read the bare "Go ahead with what you have." and found it unclear. A press has no content of its own, so with the skip (`Bypass`) those two checks no longer apply; the request is in the conversation, and the button is only offered after questions Moly asked because it understood the request. Without a press they are unchanged (table rows). S24's scripted intent model now finds the press unclear (confidence 0.3); mutation-checked.
- **"she" was saved as a new person even after the yes/no judge (fixed differently).** The weak model said "she" was a person. `JudgePersonReference` replaces it: the model is given the people the user already has and says whether the words refer to one of them (a pronoun then means that person), to a new person, or to no person. Unreadable or failed means a new person. S32 (words about the text) and S33 ("She is always helping me" after "Anna is my colleague": no new person, no name question) are mutation-checked.
- **Feedback on a draft is not a new goal (prompt only).** "It is good but a bit too formal." produced the goal "expressing discontent with formality", judged different, so Moly proposed a goal switch. The extractor prompt now says a comment, correction or adjustment of something Moly just wrote leaves the intention empty. A prompt rule alone did not hold for "answer-only" earlier; this one is untested live.
- **Still open:** goals invented from messages with no goal at 0.7 "implied" confidence ("Complete a task"): a quoted-evidence rule for the goal, as for persons, is proposed and waits for a decision.

### Live run of section G, round 3: investigation (2026-10-10)
Run: first message got a real note (no question, `help` exit); the follow-up "Warm and informal, we are friends and she is always helping me." was read as new information, not an answer (nothing was open), so maturity stayed 0 and the button correctly stayed hidden. Three problems were investigated.
1. **Two goal judges disagreed (fixed).** The log showed `Goal relation judged: refinement` and then `Goal changed to DIFFERENT`. Root cause: `AnalyzeGoalCoherence` (FIX #72, still driving Layer 4's gap filtering and the response strategy) compared goal TEXT (lower-case equality, then a hardcoded subgoal table, then "different"), so any rewording counted as a new goal. That contradicted the decision that goal shifts are judged by the model. Now Layer 1 stores the judged relation (`LayerContext.GoalRelation`) and `AnalyzeGoalCoherence` maps it (different -> "different", refinement/substep -> "related_subgoal", otherwise "same"); no judgement means "same". `isSubgoal` and its table are deleted. Test: `TestGoalCoherenceFollowsTheJudgedRelation`.
2. **The question reply spoke as the user (prompt fix, untested live).** "Hey there! I'm so grateful for all the help Anna has been providing..." Root cause: `buildGapClarificationPrompt` never said who speaks; it passed the gap as "missing context about: ..." and asked for a warm question, so the model took the user's voice from the message. The prompt now says "You are Moly, speaking TO the user. Never write as the user...". Test: prompt contract test.
3. **The note had `[Your Name]` and an invented detail (prompt fix, untested live).** Root cause: the base reply prompt had no rule for writing on the user's behalf and said "1-3 sentences". It now has a WRITING FOR THE USER rule (ready to send, only facts the user gave, no placeholders, leave an unknown name out). Test: prompt contract test. The assumption sentence after a skip result is still only an instruction.
- **Still open:** the goal "Maintaining friendship and seeking assistance" was extracted from a message that states no goal. Root cause: `applyFactConfidence` gives a goal `FactConfidence(1, basis, true)`: the model's number is not asked for and no evidence is checked, so basis "implied" gives 0.70, above the 0.6 doubt line. For persons the evidence quote is grounded in the message; for goals nothing is. The "feedback on a draft is not a goal" prompt rule probably covers this message ("Warm and informal" adjusts a draft) but is untested live.

### Decision: a clear request is answered at maturity 0 (2026-10-10)
Question to the user: must a user answer at least one question before any result, however clear the request? Decision: no (option 2, keep current). A request with a clear goal, no gap and no risk is answered at maturity 0; only the skip button needs maturity above 0. If the goal is not detected, Moly already asks first (clarity and intent checks). Consequence for tests: the realistic live test of the skip is a request that is missing something; section G of `live_check2.sh` is now adaptive (vague request, brief answers for up to 4 rounds, a press as soon as the button is offered, then checks the result has no placeholder and no question). If the model finds no gap, the script says so and ends: that is not a failure.

### Risk is checked however clear the request is (2026-10-10)
Decision (user): option 2 stays (a clear request is answered at maturity 0), but a crystal-clear request can still be harmful (a bomb, financial fraud): risk must always be checked first.
- The safety stage runs on every message before extraction, whatever the clarity, the gaps or the maturity; the answer-at-once path is only reached after it. S34 shows it: a clear request with no gap and a stated goal is refused with the law-expert wording (and live section F covers the real model with a clear, gap-free request).
- **Found by S35 and fixed:** a skip press right after a refusal reached the result path with the refused request in the prompt; only the output check stopped it (it replaced the draft by a question), one safeguard depending on the evaluator finding the harm in the text. Now the server remembers whether the last reply of a conversation offered the button (`takeSkipOffer` / `setSkipOffer`, in memory, cleared at the start of every message) and accepts a press only right after a reply that carried it. A refusal, an error or an early exit leaves it off. After a server restart a press is ignored until the next reply (safe; the user just asks again). S35 fails without the check (mutation-checked).
- S25 was restructured: the principle concern now arises at the press, after a skippable question carried the button, so it tests the risk logic and not the new offer check (mutation-checked).

## One maturity, and replies no longer count as answers (2026-10-10)
Two defects from the open list were fixed together: they were the same root problem (maturity that does not measure what was answered).
- **Defect: every reply counted as an answer.** The handler marked all open questions answered at the top of each request, before anything was judged. A reply on another topic raised maturity. **Now:** the open questions (asked and waiting, or skipped) are listed (`OpenQuestions`) and one model call (`JudgePendingAnswered`, only while questions are open) says which the message answers. Answered ones close (maturity rises); a question asked and not answered becomes pending (`skipped`) and competes again as a candidate gap with the new gaps (`mergePendingSkipped`, which no longer judges: the judgement happens once, on arrival). A skip press judges nothing: the waiting questions become pending. `MarkConversationQuestionsAnswered` and `SkipQuestions` are gone. If the judgement is unreadable no question closes (conservative; a question the user did answer may be asked once more).
- **Defect: the layers decided with an accomplishment score.** Layer 3 computes contacts + message count + entities, which rises with the length of the chat: with four messages and nothing answered it was already 0.47 (S36 mutation). It fed the conflict layer (skip below 0.2), the ambiguity layer (skip at 0.7), the Socratic layer (needs 0.5), a "severity gate" that dropped every gap with confidence below 0.3 / 0.5 / 0.7 / 1.0 as that score rose (so a long chat silently stopped asking, the static saturation again), and a rule that the score could never fall (FIX #21). **Now:** right after Layer 3 the orchestrator replaces the score with the gap maturity (`ConversationGapMaturity`) and rebuilds the Layer 3 result from it (`layer3FromScore`); the severity-gate filter, its two context fields and the never-falls rule are deleted. The old score is still computed and saved, for the phase labels in the metadata only.
- **Phase gates removed from the reply exits.** The gap, principle, persistent-concern and topic-shift probes were skipped when the old phase was "analysis" or "help" ("skipping to provide help"). They are no longer gated: whether Moly asks depends on the open gaps and the risk checks. `gatheringPhase` became `phaseLabel` (metadata only). This matters for safety: a principle concern is now evaluated in every phase.
- **Tests:** S36 (an unrelated reply keeps the question open: one row, maturity 0, Layer 3 score equals gap maturity; an answer closes it; mutation-checked both ways). The database tests were rewritten for `OpenQuestions` / `SetQuestionStatus`.
- **Still using the old phase (not decided here):** the Socratic deepening gates in `deepeningAllowed` (phase initial/gathering/help prevents a Socratic question) and the `ConversationPhase` gate; they only keep a Socratic question away, they do not skip a risk check. They should read the gap maturity instead.

### The goal needs a quote (2026-10-10, user decision)
A goal was given 0.70 confidence from its "implied" label alone, so goals were invented from messages that state none ("Maintaining friendship and seeking assistance", "Complete a task"), which led to goal-switch questions. Now, as for persons, the extractor must give `intentionEvidence`: the exact words of the message that state the goal, and the prompt says the intention must be empty if no such words exist. `applyFactConfidence` checks the quote against the message (`QuoteInMessage`): a goal without a grounded quote is capped as a guess (0.4), is in doubt, and is held and confirmed with one question (`HoldDoubtfulGoal`), never locked. Tests: `TestApplyFactConfidenceGoalNeedsAGroundedQuote`, S37 (mutation-checked); the scripted extractor now supplies the message as the quote unless a scenario sets its own.
Risk to watch in the live run: the local model may leave the quote out or paraphrase it, which would make real goals be confirmed too often. The `[FactConfidence] goal: basis=... grounded=...` log line shows how often.

## Live run 2026-10-10 (server26) and follow-ups
- Result: all rules passed. The goal quote was grounded in every logged line.
- Fixed: a skip press that is not allowed (maturity 0, not offered) is treated as answer-only, so its words never become a goal or a goal-switch question. A greeting reply over 25 words falls back to "I am Moly. How can I help you?".
- Not a code defect, test by hand: weak questions from the model; a draft written to the wrong person when the manager has no name; the skip button end to end (section G's canned answers do not answer Moly's questions, so maturity stays 0).

## Socratic deepening reads the gap maturity (2026-10-10, user decision)
- `deepeningAllowed` no longer reads the old phase (`ctx.Maturity.EstimateCurrentPhase`, the gathering-gaps gate, `ConversationPhase`). It needs gap maturity above 0 (`WayOutOpen`: something answered), not the first message, then the reasoner's checks, no recent high/critical incident, intent confidence 0.5, no "prefers advice" profile, and maturity under 0.8 (`ctx.ContextMaturity` was already the gap maturity). An open gap never reaches it: gaps are asked first.
- The reasoner's limit of 4 Socratic questions was dead (an empty list was passed). `socraticAsked` now counts the conversation's `layer8_socratic_q_` rows.
- Test: `agents/deepening_gates_test.go` (allowed at 0.5, not at 0, not at 0.9; the mutation that drops the gate fails it). No end-to-end scenario: the harness cannot reach the reasoner's complexity check.
- **Replaced by a model judgement (user decision):** `JudgeDeepening` (`agents/deepening_judge.go`) is one call, made last and only when every other gate passed: is there something the user may not have thought through (a real choice, a feeling, a trade-off, someone affected), and which constitution principles does it touch. Unreadable or failed = not worth (Moly helps). Only known principle ids are used, and they go to the Socratic question. The complexity score (`assessComplexity`, which needed a rich profile plus a leftover gap and so could never fire for a new user) and the principle word lists (`extractRelevantPrinciples`) are deleted. Tests: `agents/deepening_gates_test.go` (judgement read, unknown id dropped, unreadable and failed fail safe, not worth = no question; the mutation `return true` fails it).

## The old accomplishment maturity and the phase estimators are removed (2026-10-10, user decision)
- Deleted: `storage/maturity_service.go`, `tools/maturity_calculator.go` (4-factor score, `EstimateCurrentPhase`, severity gate), `models/phase_accomplishment.go`, `models/phase_definitions.go` (`ConversationMaturity`, phases, accomplishments), the `Context.Maturity` field, `LayerContext.MaturityContext`, `phaseLabel`, the `metadata.phase` object, and in `main.go` the loading, the `MarkAccomplished` calls, the phase persistence and the save of the old maturity.
- Layer 3 now only reads the gap maturity (`Layer3MaturityAssessmentAdapter` takes the clarification repo) and the orchestrator's replacement step is gone. The response carries `metadata.maturity` = gap maturity. The extension did not read `metadata.phase`.
- Kept on purpose: the `conversation_maturity` table in `schema.sql` and its delete in `conversation_delete.go` (existing databases; dropping tables was unsafe before). It is no longer written. The separate execution-state phase (`agents.ExecutionPhase`, gathering/processing/complete) is a different mechanism and is untouched; it still feeds `ConversationPhase`.

## Found in manual testing 2026-10-10: "Yes." was a withdrawal
- Scenario: vague request to a manager, name given, then Moly asked "Would you like to focus on discussing a fairer distribution of tasks with your manager?" (a confirmation of a goal the user had just stated; cause not yet known, needs the log: `Reply exit` and `[FactConfidence] goal:` lines). The user answered "Yes." and got the Layer 11 denial text ("I notice you might not want to dive deep... Instead, I'd suggest: We can take this at your pace").
- Root cause: `DenialDetector.DetectDenial` treated any message under 10 characters, after a prior user message, as a withdrawal. Layer 11 then outranks everything in `DecideReply`.
- Fix: the model judges withdrawal (`judgeWithdrawal`), given Moly's last message and the reply; only messages under 40 characters are checked, a cost limit and not the decision; no model or unreadable = not a denial. The denial reply no longer appends "Instead, I'd suggest: ..." (that made two sentences and a leaked label). Tests: `TestLayer11ShortAnswerIsNotWithdrawal`, and the withdrawal tests now use a model that says "withdrawing".
- Also seen: "What's Dana's professional role in your work?" was asked although the user had said Dana is the manager (a redundant gap from the model).

## testings.txt read, 2026-10-10 evening (manager/Dana scenario, real model)
Chat: "Hi Moly" -> "Help me write a message to my manager about a problem." -> name question -> "Her name is Dana. She keeps giving me extra tasks..." -> "What's Dana's professional role in your work?" -> "She is my manager. I want to ask her to share the extra tasks more fairly." -> "Would you like to focus on discussing a fairer distribution of tasks with your manager?" -> "Yes" -> Layer 11 denial text. Button disabled throughout. Findings from the log:
1. **"Would you like to focus on..." was not a goal confirmation.** It was the Layer 4 gap `goal_changed`, built by `change_to_clarification.go` from a text difference between goal lists ("I notice you no longer mention [Manage workload...]. Are you changing direction?"), severity high. The goal judge had said `refinement`. Fixed: the intention/goal text-diff gaps are removed; the model judges the relation (GoalRelation) and a real switch is confirmed (goal_switch.go). The meta-instruction conflict gap stays.
2. **"Yes" -> denial** (Layer 11 length rule): fixed earlier today (model judges withdrawal).
3. **Maturity never rose.** `[PendingQuestions] judgement unreadable (none closed)` on the answer "She is my manager...". The model's JSON is fine when asked directly (checked against Ollama), so the cause is likely a code fence or a sentence around it, which `SafeJSONParse` rejected, and every judge in the code (24 call sites) shared that. Fixed: `SafeJSONParse` takes the JSON out of a fence or chatter (`extractJSON`). The pending judge now logs whether the output was not JSON or had the wrong number of entries. Not proven to be the live cause: re-test and read `PendingQuestions` lines.
4. **Prompt leak again:** on "Yes" the extractor produced the contact "Girl from fet" from its own prompt example ("call her Girl from fet"). Example removed.
5. **The old conflict system still runs beside the new stages:** intention conflict `queue_for_approval` (pending conflict 27), `InlineConflictResolver` trying to read "Yes" as a resolution (conflict 28, `contact_ambiguous_generic_name`). Parallel mechanism, not touched; candidate for the next cleanup.
6. Redundant model gap: "The role of Dana in the user's professional life" after the user said "my manager". Not fixed.
7. Slowness: Layer 4 took 100-128 s per message, extraction minutes (deferred).

## Dana/manager: label lost, pronoun saved (2026-10-10, from testings.txt)
- The unnamed contact "manager" (id 63) was renamed to Dana, but the rename kept only the name; the extractor's category was "professional", so the gap model asked for Dana's role. **Fix:** `ApplyNameAnswer` keeps the old label as the contact's role ("manager"), and the gap prompt tells the model the role is known and must not be asked (`peopleStatusBlock`).
- `WhatWhoLinker` invented a role ("colleague/mentor") and dependencies ("has relevant expertise") from the category alone; the role was shown to the reply model as fact. **Removed.** (It is called from the identity stage, not a second writer as first thought.)
- On message 3 the identity stage saved the pronoun "her" as a second person (id 65): `JudgePersonReference` failed, and a failure counted as a new person. **Fix:** a failed or unreadable judgement is "unsure" -> doubt: nothing is saved and the reply asks. The likely cause of the failure is the same fenced-JSON problem fixed in `SafeJSONParse`.
- Tests: `TestNameAnswerKeepsTheOldLabelAsTheRole`, `TestUnjudgedLabelIsDoubtNotANewPerson`, `TestLinkerDoesNotInventARole`, `TestGapPromptKnowsAPersonsRole`; the first two fail when their fix is removed.

## More from testings.txt (second pass, 2026-10-10)
- **Safety evaluator over-flags a bare "Yes":** `Stakeholder Consideration (high, 0.90)` with evidence "Yes", confirmed by the second call. It is allowed through, but it feeds Layer 7 principle questions and keeps `canSkip` off. The evaluator already receives the recent exchange. Open: give the confirmation call the same exchange, or reject evidence that is the whole message of a bare reply. Not changed.
- **Layer 10 saved persistent questions that were never asked** (`persistent_q_*` as `active`). They only failed to save because `analysisCtx.UserID` was empty ("userId required"); with a user id they would have counted as open gaps and lowered the maturity. The save is removed.
- **Layer 4 gap response failed to parse** ("invalid character '[' after top-level value": two JSON values). `SafeJSONParse` now takes the first JSON value.
- **Cost:** a message takes 7-16 model calls and 7-12 minutes on this machine ("Hi Moly" alone: 7 calls, 8 minutes). A greeting runs extraction, intent, summary and more. Slowness is deferred, but the greeting path should skip most of this.
- "Early summary generation triggered at 10 messages" fired in a 2-message conversation (message counts include stored internals?). Not investigated.
- `AnalysisContextBuilder` loaded 0 relevant contacts on the "Yes" turn although Dana was the focus; relevant contacts are chosen by what the current message mentions.
