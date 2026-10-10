# Moly architecture (2026-10-10)

Authoritative detail and the reasons behind each choice: [ORCHESTRATOR_DESIGN.md](ORCHESTRATOR_DESIGN.md). The earlier 11-layer description is archived in `archive/ARCHITECTURE_2026-10-08.md` and `archive/MOLY_11_LAYER_SYSTEM.md`.

## Pieces

| Part | Where |
|---|---|
| Extension (TypeScript/React sidebar) | `moly-extension/` |
| Backend (Go, port 11436) | `moly-go/` — `main.go` is the HTTP handler and turn flow |
| Local model | Ollama on :11434 (`mistral`) |
| Native-host proxy | `moly-proxy/` |
| Store | SQLite, encrypted, `moly-go/database/` (schema: `database/schema.sql`) |

## One message, in order

1. **Safety stage** (`agents/safety_stage.go`) — runs first, on every message. Fails closed. Clear direct harm → instant short refusal. Ambiguous → allowed through to the question stages.
2. **Extraction** (`agents/context_extractor.go`) — intention (with a quoted `intentionEvidence`), contact, traits, with a basis (stated / implied / guessed).
3. **Fact confidence** (`agents/fact_confidence.go`) — per fact. A fact in doubt is not saved; the reply confirms it.
4. **Intent and goal stages** (`agents/intent_stage.go`, `goal_stage.go`, `goal_switch.go`) — greeting, answer-only, goal lock, goal switches (the model judges relation; a switch needs the user's yes).
5. **Identity stage** (`agents/identity_stage.go`) — the only writer of contacts; a named person is required; a pronoun points to the known person.
6. **Pending-question judge** (`agents/pending_questions.go`) — one call: does this message answer a question still open?
7. **Layers 3–11** (`agents/unified_orchestrator.go`, `layer*_*.go`) — maturity, gap detection (ranked by severity against the goal, pending questions included), conflicts, ambiguity, principles, Socratic, topic shift, persistence, denial. They supply facts.
8. **Reply decision** (`agents/reply_policy.go` `DecideReply`, `reply_exits.go`) — a pure function of the facts. Order: deny, safety alert, greeting, name, confirm fact, clarity, open gap, principle, persistent, topic shift, contact, intent, Socratic, help. Exactly one question per reply (`single_question.go`).
9. **Output gate** — a flagged draft is replaced by one question (`blocked_reply.go`).
10. **Turn commit** (`turn_commit.go`) — one transaction per turn; a failed turn keeps nothing.

## Maturity

One number: `answered / (answered + open)`, open = asked-and-waiting plus skipped (`agents/gap_maturity.go`). A clear message can have zero gaps. The old accomplishment/phase maturity is gone.

## Skip button

`metadata.canSkip` is true only when the reply is a skippable question (`agents/skip_button.go`), maturity is above 0, and no conflict waits. The request carries `skipQuestions: true`; the server accepts it only if the last reply offered it. A press makes waiting questions `skipped` (pending) and answers from the conversation (`agents/result_now.go`). Risk checks still apply. An unaccepted press is ignored and carries no goal or person.

## Tests

`moly-go/orchestration_test.go` runs scenarios S1–S37 end to end with a scripted model and a real temporary database; unit tests sit beside the code. Live behaviour is checked with `scripts/`.
