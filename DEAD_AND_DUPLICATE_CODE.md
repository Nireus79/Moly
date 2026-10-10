# Dead and duplicate code inventory (2026-10-09)

Purpose: a list for later investigation and removal. Nothing here has been deleted. Each item is marked:
- **V** = verified in this session by reading callers, or by a probe.
- **C** = candidate from the automated scan (`DEAD_CODE_CANDIDATES.txt`); needs a manual check before deletion.

Scope: `moly-go` (Go backend) and the related Socrates family in `/home/nireus79/PycharmProjects`. The browser extension (`moly-extension`) was **not** scanned in this pass.

Method for the Go scan: every non-test function and method name, counted for textual occurrences outside definition lines and comments. Result: 325 names with no references at all, and 60 used only by tests. The scan cannot see reflection, string-keyed dispatch, or chains where a dead function calls another dead function. Those are listed under section 4.

---

## 1. Maturity (Moly) — verified

| ID | Item | Location | Status | Note |
|---|---|---|---|---|
| D-M1 | `calculateContextMaturity` | `main.go:5005` | V dead | No callers. Comment claims it gates safety; it does not. Floor 0.7. |
| D-M2 | Category model: `UpdateCategory`, eight categories, `IdentifyWeakCategories` | `tools/maturity_calculator.go` | V dead / meaningless | `UpdateCategory` has no callers, so categories stay 0; `IdentifyWeakCategories` marks all weak. |
| D-M3 | `BuildPhaseMaturity()` (no-argument) | `tools/maturity_calculator.go` | V dead | No callers. The live variant is `BuildPhaseMaturityWithFactors`. |
| D-M4 | `GetReadinessLevel`, `ShouldDeferEvaluation`, `GetCategoryImprovement`, `CreateMaturityEvent`, `GetPhaseCompletionPercentage` | `tools/maturity_calculator.go` | V dead | No live callers. |
| D-M5 | `CalculateProfileCompleteness` | `tools/maturity_calculator.go` | V placeholder | Returns constant 0.5 whenever the profile is non-nil. Live, but not a measurement. |
| D-M6 | `MarkAccomplishment` | `storage/maturity_service.go:65` | V dead | Duplicates `models.ConversationMaturity.MarkAccomplished`, which is what the handler calls. |
| D-M7 | Second phase estimator | `models/phase_accomplishment.go:127` | V duplicate | Same intent as `tools.MaturityCalculator.EstimateCurrentPhase`, different thresholds (0.25/0.5/0.75 and different names). |
| D-M8 | Handler phase mapping | `main.go:2551–2555`, `:2618–2622` | V duplicate | Third phase mapping (0.3/0.6/0.8). |
| D-M9 | Stuck detector output | `main.go:4210–4243` | V write-only | Writes `maturityStuck` metadata; nothing reads it. |
| D-M10 | Maturity cache shortcut | `agents/layer_adapters.go:422, 438`; `agents/layer4_gap_detector.go:76, 92`; same pattern in layers 2, 5, 8, 9, 11 | V duplicate and wrong | Repeated shortcut; see workflow investigation RC1. Remove, do not patch. |
| D-M11 | `calculateContextMaturity` duplicate notion | `main.go` vs `tools/maturity_calculator.go` vs `models/phase_accomplishment.go` | V duplicate | Four places compute "maturity". |
| D-M12 | Layer 3 maturity field `IsMessageOne` | `tools/layer_context.go:68`, `agents/unified_orchestrator.go:301–302` | V dead | Set but no longer read after the goal-lock fix (this session). |

## 2. Workflow and layers (Moly) — verified

| ID | Item | Location | Status | Note |
|---|---|---|---|---|
| D-W1 | Greeting templates (hardcoded text, including "Hi. I'm Μώλυ. What's on your mind?") | `agents/response_template_manager.go:82–93` (context `new_user_greeting`, category `greeting`) | V dead | Seeded into the database on start-up. The only template read anywhere is `no_topic` (`conversation_agent.go:3533`). The greeting text is never used. |
| D-W2 | Other seeded templates | `agents/response_template_manager.go:99–118` | V dead | Same: seeded, never read. |
| D-W3 | `GenerateInitialGreeting` | `agents/` | C dead | No references. Likely the greeting generator of D-W1. |
| D-W4 | Placeholder reply | `agents/response_strategy.go:148` `BuildResponseFromExtraction` | V live but wrong | It is live, so it must be replaced, not just removed. Workflow RC3. |
| D-W5 | Strategy `maturity` variable | `agents/response_strategy.go:57` | V unused | Computed and stored, never used for a decision. |
| D-W6 | Intent-detector extraction chain: `ExtractEntitiesAndAnalyzeIntent`, `ExtractEntitiesWithClassification`, `SmartExtractEntities`, `ExtractAndLock`, `extractEntitiesWithLLM`, `validateAndParseEntities`, `parsePipeDelimitedEntities`, `validateParsedEntities` | `agents/intent_detector.go` (lines ~638–1480) | V dead cluster | No caller outside the cluster. `extraction_phase.go:121` says "REMOVED: SmartExtractEntities call". This cluster also holds the substring subject defaults (`"me"`, `"my"`, `"i "`). |
| D-W7 | Substring subject default | `agents/intent_detector.go:~781–792` | V dead (inside D-W6) | Would have misattributed subjects; the path is not live. |
| D-W8 | `vague_contact_name` remnant | `agents/layer4_gap_detector.go` `calculateGapImpact` (~792–799) | V dead rule | Rule removed by FIX #75; the impact function still names it. |
| D-W9 | `isVagueContactName` | `agents/layer4_gap_detector.go:665` | V test-only | Only tests call it. |
| D-W10 | `answeredGapTypes` parameter | `agents/layer4_gap_detector.go` `DetectGaps` and `filterGapsByGoal` | V dead parameter | Always passed `nil` after this session's change; type-level suppression is off by design. |
| D-W11 | Layer 5 — two implementations | `agents/layer5_conflict_handler.go` and `agents/layer5_unified_conflict.go` | V duplicate | Both wired. Decide one. |
| D-W12 | Conversation summary — two writers | `main.go` handler (~4290–4370) and `tools/conversation_summary_manager.go` (~146, ~237) | V duplicate | Both write one row. |
| D-W13 | Clarification answered — two writers | `main.go` answer endpoint (~4434) and `database/clarification_history.go` `MarkAsAnswered` | V duplicate | Both set `status = 'answered'`. |
| D-W14 | Type-based answered check chain: `HasBeenAnswered`, `ShouldAskClarification`, `HasBeenAsked` | `database/clarification_history.go:64`, `:~270–300` | V dead | `HasBeenAnswered` is called only from `ShouldAskClarification`, which has no callers (scan section A). Layer 4 uses `HasQuestionTextBeenAnswered` instead. |
| D-W15 | Gap gate block | `agents/conversation_agent.go:~1131` (`len(ctx.Gaps) >= 3`) | V unreachable for these messages | Runs after the placeholder early return. Not dead in general. |
| D-W16 | Agent maturity field | `ConversationContext.Maturity` | V never set | Read at `conversation_agent.go:1134`, never assigned by the handler. |
| D-W17 | Contact row scanning | `database/contact_repository.go` `GetByUserID`, `GetByID`, `GetByRelationship` | V duplicate | Three copies of the row scan. The mismatch in `GetByUserID` was RC8 (earlier report). `GetByRelationship` omits the WHAT columns and confidence. |
| D-W18 | Confidence column | `database/contact_repository.go` | V unloaded | Saved, never selected (RC8). |
| D-W19 | `MarkAsAnswered` (ID-based) and `MarkAsAsked`, `MarkAsSkipped` | `database/clarification_history.go` | C | No callers (scan section A). `MarkConversationQuestionsAnswered` (added this session) is the live path. Also the answer endpoint in `main.go` sets the same status (D-W13). |
| D-W20 | Old placeholder gap response | `agents/response_strategy.go:174` `BuildGoalAlignedGapResponse` | C | Check whether it is still reached after the strategy change. |

## 3. Test-only and test helper duplicates — verified

| ID | Item | Note |
|---|---|---|
| D-T1 | `testKey`, `setupTestDB`, `setupTestAttributeRepo`, `seedUserAndConversation`, `newSummaryFixture` | Helper duplicates across packages. Consolidate into a shared test helper. |
| D-T2 | `runLayer1WithIntention` | Test helper that duplicates the Layer 1 reuse path. |
| D-T3 | `TestLayer4NoVagueNameRule` | Asserts a removed rule is absent; keep only if FIX #75's contract is kept. |
| D-T4 | Tests of removed rules (`TestLayer4NoConfidenceRule`, `TestLayer4NoMaturityRule`) | Test removed behaviour. |

## 4. Candidates from the automated scan (C)

The full lists are in `DEAD_CODE_CANDIDATES.txt`:
- **Section A — no references anywhere (325 names).** Includes `calculateContextMaturity`, `BuildPhaseMaturity`, `GetReadinessLevel`, `ShouldDeferEvaluation`, `CreateMaturityEvent`, `GetCategoryImprovement`, `MarkAccomplishment`, `GenerateInitialGreeting`, `BuildProfile`, `ExtractContext`, `ValidateJSON`, several `Log*` helpers (`LogLayer`, `LogRequestStart`, …), `Stats`, `Health`, `HealthCheck`, `Shutdown`, many `New*` constructors and `Get*` accessors, and about 40 `Validate*` functions. Each must be checked before deletion, because some may be called by reflection or by tooling.
- **Section B — referenced only from tests (60 names).** Production dead code kept alive by tests. Includes `IsExpired`, `GetCache`, `GetTopHitters`, `ListLayers`, `AddLayer`, `SetDebugMode`, `NewProfileParser`, and `MergeProfiles`.

## 5. Socrates family (outside Moly) — verified

| ID | Item | Location | Status |
|---|---|---|---|
| D-S1 | Two working copies of Socrates | `/home/nireus79/PycharmProjects/Socrates` (HEAD `370aaf0`) and `/home/nireus79/vs_projects/Socrates` (HEAD `46ab234`), differing in 30 places under `socratic_system/` | duplicate |
| D-S2 | Maturity calculator in two packages | `Socrates/socratic_system/core/maturity_calculator.py` and `socratic-agents/src/socratic_agents/core/maturity_calculator.py` | duplicate, diverged |
| D-S3 | Quality service (recalculating, overwrites score) | `Socrates/socratic_system/services/quality_service.py` | dead: `update_maturity_after_response` has no callers; the live path is the `quality_controller` agent |
| D-S4 | Two maturity stores | `Socrates/socratic_system/services/repositories/maturity_repository.py` and `Socrates/socratic_system/repositories/quality_repository.py` | duplicate |
| D-S5 | `max_questions_per_phase` | `socratic-agents/src/socratic_agents/socratic_counselor.py` `__init__` | dead: set, never read |
| D-S6 | `orchestrator.py.bak` | `socratic-agents/src/socratic_agents/` | dead backup file |
| D-S7 | `models_old/`, `utils_old/` | `socratic-agents/src/socratic_agents/` | dead legacy copies of `models/` and `utils/` |
| D-S8 | `_get_covered_categories` bug | `socratic-agents/src/socratic_agents/core/question_selector.py:98` | defect in live code (first key of the phase dict is added) |
| D-S9 | Unused `socratic_system/maturity/__init__.py` import | `Socrates/socratic_system/maturity/__init__.py` | re-export of the external library |
| D-S10 | `socratic-maturity` library | `PycharmProjects/socratic-maturity` | the reference; its calculator and workflows duplicate concepts in `socratic-agents` |

## 6. Process notes

- Each item needs a single decision: delete, merge, or keep. Do not delete in the same change as a behaviour fix (see the plan).
- Verify each C item with a reference search and a test run before deletion.
- Do not rely on the scan for functions called through interfaces. Check the interface definition first.
