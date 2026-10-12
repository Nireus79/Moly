# Claude Code guide — Moly

Read this first in every session, then `docs/ORCHESTRATOR_DESIGN.md` (decisions, findings, status) and `docs/ARCHITECTURE.md`.

## State (2026-10-10)
The orchestration was rebuilt Oct 9–10: one reply decision (`DecideReply`), one safety stage, one identity stage, one maturity (gap maturity), a skip button with pending questions, per-fact confidence, goals that must quote the user, model-judged Socratic deepening. The suite is green (`go vet ./... && go test -count=1 ./...` in `moly-go/`). Behaviour with the real model is checked by hand (`scripts/live_chat.sh`); open items are at the end of `docs/ORCHESTRATOR_DESIGN.md`.

## Product rules the user decided (keep applying)
- One question per reply, optionally one short sentence of understanding first. A greeting gets a short greeting, no goal needed. Help needs no goal.
- A named person is required to write to someone: "Can you give me a name for the {contact} you mentioned?".
- An unclear message is never a crisis: ask first. Clear self-harm or illegal requests get a refusal with **no help lines** ("I can't help with that. Please consider asking advice from a specialist." / "...a law expert."). Risk is checked however clear the request is.
- Doubt = do not save, confirm with one question. Goal shifts are judged by the model and need the user's yes.
- A clear, gap-free, risk-free request is answered at maturity 0. The skip button needs maturity above 0. Skipped questions are pending and return, ranked with new gaps.
- Use model judgement, not keyword lists; unreadable output fails safe.
- Moly's purpose is to understand the user, and a short reply ("Yes") cannot be read alone. Every model call that interprets the user's message gets the dialogue packet (`models.DialoguePacket`): the summary (Moly's memory in plain prose, rewritten after every stored turn by `ConversationSummaryManager.UpdateWithTurn`, in the background), the locked goal, the people with their role, the open questions, and Moly's last message (plus the user's previous message only while the summary lags). Facts the code depends on stay in the database; the summary may not override them. Scenarios S39, S40, S41 enforce it.
- A comment on the draft Moly just wrote is revised first; a pending question waits and returns on the next message (`ReplyFacts.Revision`, S48). A question Moly already asked is not asked again in other words (`DropRepeatedGaps`, S46). A pending question returns only while it is still worth asking, otherwise it is cancelled (S52). A goal switch is judged only when the reply really asked it (S53). A message may name several people; all are saved (S45).
- Every judge has labelled cases in `agents/judges_live_test.go` (docs/JUDGE_BATTERY.md); run it live after changing a judge prompt or the model. Add a case when a live run finds a miss.
- Judge prompts define every label they offer (yes/no/unclear, known/new/none). Measure a changed judge prompt on labelled cases with 3 replays before keeping it (the first versions of the goal-switch and person-reference judges were wrong in 9 of 24 and 10 of 24 cases on Haiku).
- The turn after a crisis refusal starts with the fixed check-in (S54); a crisis-rated message gets no name question (S55). The safety/bypass scenarios and `scripts/live_turn.sh` are local only (`private/`, git-ignored): never commit them or transcripts.
- A question about Moly itself is answered from `ProductKnowledge` and sets no goal (S56); keep that sheet true to the extension. Secrets (card numbers, passwords, IDs) are cut out before anything is stored (S57). The clarity decision is always the model's.
- A refused message saves no people, no summary line and no goal (S34, S49).
- Privacy (docs/PRIVACY.md): no user content in logs (the `privacy` scrubber masks what the server learns; `MOLY_LOG_CONTENT=1` shows it for debugging); secrets are cut out before anything is stored; profile deletion leaves nothing (`TestDeleteUserLeavesNothingInAnyTable`, S58). Register any new place that learns user content with `privacy.Register`. Keep full chat history for resume.
- Slowness is deferred.

## Working rules
- Commit and push only when asked. Keep pushes small: never binaries, logs or databases (`bin/`, `*.log`, `*.db` are ignored).
- Build and run the server only as needed: `cd moly-go && go build -o ../bin/moly . && ../bin/moly` (port 11436, ready after ~40–50 s). Stop it with `kill $(pgrep -x moly)`, never `pkill -f`.
- Add a scripted scenario (`moly-go/orchestration_test.go`) for each rule and check it fails without the rule.
- Record decisions and live findings in `docs/ORCHESTRATOR_DESIGN.md`; update this file when the rules change.
- Dead code: `~/go/bin/deadcode ./...` in `moly-go/`.

## Layout
```
moly-go/        backend (main.go = handler and turn flow; agents/ = stages and layers; database/; tools/; models/; config/)
moly-extension/ Chrome extension (src/sidebar/components/ChatInterface.tsx = chat and skip button)
moly-proxy/     native-host proxy
scripts/        live_chat.sh (interactive), live_check2.sh (rules), live_check.sh (older)
docs/           current docs; docs/archive/ = earlier plans, reports, superseded specs
```

## Where things are
| Topic | File |
|---|---|
| Reply decision | `agents/reply_policy.go`, `agents/reply_exits.go` |
| Safety | `agents/safety_stage.go`, `tools/constitutional_evaluator.go` |
| Goal / intent / identity | `agents/goal_stage.go`, `goal_switch.go`, `intent_stage.go`, `identity_stage.go` |
| Maturity, skip, pending | `agents/gap_maturity.go`, `skip_button.go`, `result_now.go`, `pending_questions.go` |
| Confidence | `agents/fact_confidence.go` |
| Socratic | `agents/deepening_judge.go`, `reply_exits.go` (`deepeningAllowed`) |
| Turn commit | `turn_commit.go` |
