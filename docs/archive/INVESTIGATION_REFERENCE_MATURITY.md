# Investigation: reference maturity logic (Socrates family)

Source: local PyCharm projects, read verbatim (`/home/nireus79/PycharmProjects/`):
- `socratic-agents` (git HEAD `70ac4a0`)
- `Socrates` (git HEAD `370aaf0`)
- `socratic-maturity` (git HEAD `e7a7794`)

A second Socrates working copy exists at `/home/nireus79/vs_projects/Socrates` (git HEAD `46ab234`) and differs in 30 places under `socratic_system/`. This document uses the PyCharm copy, which you named.

Purpose: Socrates is a **project-specification builder**. It asks questions until a software or business project is specified, phase by phase. Moly is a **communication thinking partner**. The purposes differ, so the logic transfers only in part (see section 5).

---

## 1. The live scoring rule (accumulator)

File: `socratic-agents/src/socratic_agents/quality_controller.py`, `_update_maturity_after_response` (lines 186–289).

```
score_before = project.phase_maturity_scores.get(project.phase, 0.0)
categorized  = calculator.categorize_insights(insights, project.phase)
answer_score = Σ spec.value × spec.confidence     (defaults: value 1.0, confidence 0.9)
score_after  = min(100.0, score_before + answer_score)
project.phase_maturity_scores[project.phase] = score_after
project.overall_maturity = project._calculate_overall_maturity()
project.progress = int(project.overall_maturity)
```

Stated purpose (code comment): "Instead of recalculating entire maturity, ADD this answer's score. This prevents previous answers' scores from being affected by new specs' confidence."

Other facts, verified:
- The stored score is the accumulator. `calculate_maturity` and `get_phase_maturity` return the saved score; the recalculated category breakdown is discarded (`_calculate_phase_maturity`).
- The API calls `update_after_response` (`Socrates/socrates-api/src/socrates_api/routers/projects_chat.py` ~1960), then `db.save_project(project)`. The accumulated value is what gets persisted.
- Thresholds (copied from the calculator): `READY 20`, `COMPLETE 100`, `WARNING 10`. Phase complete at 100 (`PHASE_READY_TO_ADVANCE` event). Advance-with-warning prompt below 20 (`socratic_counselor.py` `_advance_phase`).

## 2. Overall score

File: `socratic-agents/src/socratic_agents/models/project.py`, `_calculate_overall_maturity` (line 221).

```
overall = mean of phase scores where score > 0
```

The docstring says: "Instead of averaging (which penalizes starting new phases)… advancing to new phases doesn't decrease overall maturity", then gives the example "Discovery: 100%, Analysis: 30% → overall = (100 + 30) / 2 = 65%". The example contradicts the claim: starting a new phase lowers the overall score from 100 to 65.

## 3. Category model and targets (display only)

Files: `socratic-agents/src/socratic_agents/core/maturity_calculator.py`, `core/project_categories.py`.

- Each project type has four phases; each phase has categories with integer targets. **Every project type's phase totals to 90** (verified by evaluating the dictionaries: software, business, creative, research, marketing and educational, all 90 in every phase).
- The generic fallback (`GENERIC_PHASE_CATEGORIES`) totals 80 in discovery, analysis and design, and 90 in implementation. A phase on the generic fallback cannot reach 100, because the calculator's percentage is `sum / 90 × 100` and the maximum is 80/90 ≈ 89%. The fallback is used when a project type is unknown.
- `calculate_phase_maturity` computes, per category, `min(Σ value × confidence, target)`, sums them, and expresses the result as `sum / 90 × 100`. This is the category breakdown. **It does not feed the stored score**, so the category caps have no effect on the accumulator.

## 4. Socrates' other implementation (not called)

File: `Socrates/socratic_system/services/quality_service.py`.

- `calculate_phase_maturity` recalculates from all specs and **overwrites** the stored score (`repository.update_phase_maturity_score`, which sets `project.phase_maturity_scores[phase] = min(100, max(0, score))`).
- `update_maturity_after_response` in this file has **no callers** in Socrates. The live path is the `quality_controller` agent above.
- It also reads `score_after` from the in-memory project, which the repository write does not refresh, so the reported value would be stale. This is a timing defect in a dead path.
- Its `socratic_system/core/maturity_calculator.py` and `socratic-agents/src/socratic_agents/core/maturity_calculator.py` are both about 420 lines and differ in content.

## 5. Question loop and maturity

File: `socratic-agents/src/socratic_agents/socratic_counselor.py`.

- `_generate_question` (line 98): returns the next pending question, or generates one. **No maturity check.**
- `_check_phase_completion`: reads the score from `quality_controller` `get_phase_maturity`; "phase complete" message when `current_score >= 100`. Nothing stops asking.
- `_advance_phase`: below 20 with warnings, asks "Advance anyway? (yes/no)". No check requires 100 to advance.
- `max_questions_per_phase = 5` is set in `__init__` and never read.
- `_process_response`: maturity is updated only if conflict detection does not pend (early return).
- Conflict resolution ("replace with new specification", around lines 1090–1120) removes the old spec from the project context and does **not** call the quality controller. The stored score keeps the removed spec's points.

Files `question_queue_agent.py` and `core/question_selector.py` do not read maturity. `question_selector._get_covered_categories` (line 98) has a bug: for a list-valued spec it executes `covered.add(str(list(phase_specs.keys())[0]))`, which adds the first key of the phase dict instead of the spec's category, so categories are not marked covered. Verified from the local source; not executed.

## 6. Persistence layer

File: `Socrates/socratic_system/repositories/quality_repository.py` (update), `Socrates/socratic_system/services/repositories/maturity_repository.py` (a separate store).

- `maturity_repository.update_category_score` recalculates `overall_score` as an **unweighted mean** of category scores. A third definition of "overall" exists in the reference.

## 7. Is maturity decreasing in the reference?

By the accumulator's design, no: `min(100, before + answer)` only adds. Exceptions, all verified in code:
1. The overall score falls when a new phase starts (section 2).
2. A replaced spec does not reduce the score (section 5).
3. Negative `value` or `confidence` would reduce the score; these are not validated.
4. The Socrates `quality_service` path (section 4) overwrites the score with a recalculation, which can go down.

## 8. Comparison with Moly (current state)

| Aspect | Reference (live) | Moly (current) |
|---|---|---|
| Stored measure | per-phase accumulator, capped at 100 | single number recomputed each message (four-factor), plus accomplishment flags not persisted |
| Increments | value × confidence per spec | constant factors and boolean markers |
| Caps per facet | category targets computed, not applied to score | none in the score |
| Decrease rule | none in the score; the phase-average drop | Layer 3 keeps max(prev, new); accomplishment average can fall |
| Use for asking | none (phase message only) | decides everything (through a shortcut, RC1) |
| Gap detection | category coverage, rules in prompt | LLM gaps, skipped by the cache shortcut |

## 9. What transfers to Moly

- The accumulator and its stated reason: noise must not erase earlier evidence. This addresses M2 (the summary-driven drop).
- Per-phase persistence of the accumulator. This addresses M1.
- Facet caps from targets, as the calculator intended. This addresses M5/M2's domination risk.
- Explicit decreases through resolution of a contradiction (the counselor's replace path). Moly already has conflict detection (Layer 5).

## 10. What does not transfer

- "Maturity only reports progress; questions never depend on it." Moly's specification says maturity controls behaviour, so Moly keeps that rule and must make it correct.
- A project-spec vocabulary (requirements, tech stack). Moly's facets are person, goal, style, values and concerns.

## 11. Not verified

- `_calculate_overall_maturity` in the Socrates copy (`socratic_system/models/project.py`) matches the one above only by its docstring; I read the socratic-agents copy.
- The `value` and `confidence` ranges produced by the LLM categoriser (`InsightCategorizer`) are not read. The heuristic path uses confidence 0.7 and value 1.0.
- Whether any live caller runs `quality_service.py`. Grep found no caller of its update method. Other methods were not checked one by one.
