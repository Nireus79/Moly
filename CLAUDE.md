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
- Unresolved data belongs to the conversation it came from (user decision, 2026-10-14): a person without a name, a name question, an unanswered conflict are visible and asked only there (`ContactRepository.InConversation`, `GetUnresolved(user, conversation)`). Moly uses resolved data everywhere. The name is asked first and compared with the saved people after (`NameContact`); a description ("my boss", "she") is never matched with a saved person by its kind; a pronoun or role points only to people the conversation has already brought up. Read `docs/ORCHESTRATOR_DESIGN.md`, section of 2026-10-14, before touching contacts, conflicts or the identity stage.
- Prompts name no person and no scenario; they describe the kind of thing in words (audit of 2026-10-14, `TestNoPersonalNamesOrScenarioWordsInStringLiterals`). An example is allowed only in a judge (it returns a label), in a neutral domain, and never as a sentence that is also a battery case: a prompt example that is a test case inflates the score (it did, for four judges). New battery cases are marked "held out:" and must not occur in any prompt. Anything stored or shown for one conversation (unconfirmed insights, past goal, pending clarifications, conflicts, unnamed people) is read only for that conversation.
- Privacy (docs/PRIVACY.md): no user content in logs (the `privacy` scrubber masks what the server learns; `MOLY_LOG_CONTENT=1` shows it for debugging); secrets are cut out before anything is stored; profile deletion leaves nothing (`TestDeleteUserLeavesNothingInAnyTable`, S58). Register any new place that learns user content with `privacy.Register`. Keep full chat history for resume.
- Slowness is deferred.
- Production mode (`MOLY_PRODUCTION=1`, `MOLY_EXTENSION_IDS`, optional `MOLY_IDLE_EXIT=1`; `security.go`, `lifecycle.go`) is for the native host and installer: loopback only, extension origins only, token header, idle exit. Development mode stays open on purpose; do not change it. Tests: `security_test.go`.
- A conversation starts as "Direct Message" and is named after its locked goal once there is one: a short title (at most five words, in the goal's language) written by the model in the background after the reply (`conversation_title.go`, scenario `TestScenarioConversationIsNamedAfterItsGoal`); a confirmed goal switch renames it; an unusable answer falls back to the first six words of the goal.
- Database key (`keystore/`, `keygate.go`): keychain first; with none the person chooses a passphrase (recommended) or a confirmed key file; never silently, never a new key over an existing database. While the key is missing the backend answers 423 on `/api/status` and serves only `/api/keystore/*`; the extension shows `KeystoreGate`. Tests: `keystore/keystore_test.go`, `keygate_test.go`, `KeystoreGate.test.tsx`, e2e `keystore-*` scenarios.
- AI provider and API keys BELONG TO THE USER (2026-10-14): `user_ai_settings` and `user_ai_keys` (`database/user_ai_settings.go`), read and unlocked only by the manager in `llm_settings.go`, optionally sealed with a passphrase (`secrets/`); never returned, logged or exported; the tables go with the profile. Every model call gets its provider from `tools.LLMSource.For(userID)`; the model-calling components are built per user (`llmParts`, `APIServer.forUser`, called by each handler right after authentication) and the server value for nobody fails every call. The environment never replaces or supplies a user's key; only `LLM_PROVIDER` (the operator's override) serves every user of a run. New code must not build a client, read a key, or call a vendor (`TestOneSourceOfTruthForProvidersAndKeys`), and the scenario suite fails if a model call is made for nobody. Never put a key in the extension's storage, in a log line, or in an API answer.
- A change in how a person is described is judged by the model against what was stored BEFORE the message (`agents.JudgeTraitChange`); never by comparing word lists, never after the message was merged. A real change is held back and asked ("should I update how I see this person?"); the judge that reads the answer is shown the question that was asked. Models are asked of the provider with the user's key (`tools/llm_models.go`); no model name is written in the code. A session the app rejects is forgotten by the extension (`setSessionRejectedHandler`). See `docs/ORCHESTRATOR_DESIGN.md`, section of the live findings of 2026-10-14.

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
