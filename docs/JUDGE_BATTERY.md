# Judge battery (live, labelled cases)

Every model judgement that steers Moly is measured on labelled cases against a real model, so that a change to a prompt or a switch of model can be compared with numbers, not impressions. The test is `moly-go/agents/judges_live_test.go` (`TestLiveJudgeBattery`); it is skipped unless `MOLY_LIVE_LLM=1`.

```
cd moly-go
MOLY_LIVE_LLM=1 LLM_PROVIDER=claude-cli AGENT_MODEL=haiku go test ./agents/ -run TestLiveJudgeBattery -v -timeout 3400s
MOLY_LIVE_ONLY="safety,intent answersQuestion"   # a subset of the judges
MOLY_LIVE_REPLAYS=5                              # replays per case (default 3)
```
Four calls run at a time. A full run takes about 5-6 minutes on Haiku through `claude-cli` (no temperature control: the noisiest setting). Add a case when a live run finds a miss.

## Results, Claude haiku, 2026-10-12 (right / replays)
| Judge | Cases | Result | Notes |
|---|---|---|---|
| safety (evaluator + confirmation) | 4 harmful, 10 harmless | 42/42 | explicit self-harm, forgery, poisoning, threat blocked; unclear distress, "Yes", "Never mind", a dilemma, an angry message allowed |
| withdrawal | 4 + 4 | 24/24 | |
| goal relation | 8 | 24/24 | only "different" is a shift |
| deepening (worth) | 3 + 3 | 18/18 | |
| clarity: can proceed / crisis | 3 + 4 | 9/9, 12/12 | |
| meta-instruction | 3 + 4 | 35/35 over 5 replays, model alone (was **0/9** on the positives) | see below |
| contradictory instructions | 3 + 5 | 40/40 | |
| principle reasoner (end to end) | 3 + 4 | 27/28 (was **0/12** on the positives) | see below; one false positive on the vague "boss problem" request |
| pending question still worth asking | 3 + 3 | 30/30 (5 replays) | |
| safety late in a conversation (short replies at maturity 0.8, exchange shown) | 6 | 30/30 | the old suspicion that a bare "Yes" is flagged is not reproduced |
| person reference (11 cases: roles, organizations, known names, tone words) | 11 | 54/55 on two runs | organizations (bank, company, council) are not people |
| persistence probe (property: one question, no brackets) | 4 | 16/16 | |
| summary writer (property: both people kept, prose, short) | 1 replay of 4 turns | 3/3 | |
| repeated gap | 5 | 25/25 twice (was 12/15) | definition sharpened: "the answer would be the same" |
| intent label | 6 | 26/30 | the miss is a message with facts and a stated feeling, labelled "sharing" |
| intent adjustsDraft | 6 | 30/30 twice | (a bad case of mine: the draft did not name Mark) |
| intent answersQuestion | 8 | 18/21 | misses only on "answer plus a new request"; covered by the cross-check below |

## Found and fixed by the battery
1. **Meta-instructions were never detected from the model's answer** (0 of 9 positives). `parseMetaInstructionResponse` searched the raw text for `"isMetaInstruction":true` (no space); Haiku writes `"isMetaInstruction": true`. Now parsed as JSON (test `TestMetaInstructionAnswerIsParsedAsJSON`). Before, the instruction was only caught by the linguistic and keyword tiers.
2. **An answer with a request of its own was taken for a plain answer** ("Her name is Dana, and also help me write to my brother": 4 of 5 replays), and a plain answer loses its goal. The verdict now stands only if the extraction does not state a quoted aim of its own (`IntentDecision.ConfirmAnswerOnly`; the extractor did state it 5 of 5 times).
3. **A repeated question in other words was kept** (reworded loan question: 0 of 3). The repeat definition is now "the answer would be the same".
4. The intent flags `answersQuestion` and `adjustsDraft` tolerate `"true"` as a string.

## Not in the battery yet
Persistence probes and the principle reasoner (question generators, not judges); the summary writer and the reply (free text: property checks in the scripted scenarios plus a read of the dynamic transcripts); the extraction form (`TestLiveExtractionInventsNothingFromAMessageWithNoGoal` and the 30-case comparison from 2026-10-11).

## Model answers are read as JSON; the meta-instruction detector is the model alone (2026-10-12)
- The detector no longer tries a grammar parser (`LinguisticParser`) or a "you are Moly" phrase check before the model. Both are deleted, with their tests. The model alone scores 35/35 (7 cases x 5 replays).
- The intent analyzer no longer searches an unreadable answer for the words "asking", "sharing" ...: an answer that is not JSON is intent unknown. The reaction-context and incoming-message-type judges read their `*_engaged` flags with `jsonFlags` (JSON), not by searching the text.
- Removed because they were never called: the subject-shift detector (a field and a log line), `tools/extraction_orchestrator.go` (an empty shell), `tools/group_detector.go`.
- The 11 phrase regexes of `context_change_tracker.go` ("keep it short", "don't give advice" ...) are replaced by the model: the meta-instruction call also answers `contradictory` (true only when one message gives instructions about Moly that cannot both be followed). Layer 4 turns that into one question, "Which should I follow?" (scenario S50; battery 40/40). The regex tracker, its persisted flags, `ChangeToClarification`, the log-only Layer 11 block and `ResponseAdapter.GetMetaInstructionRespect` are deleted.
- Intent label: the prompt's "venting" definition and the guideline that preferred "sharing" were sharpened (plain venting 1/5 -> 5/5). A message with facts and a stated feeling ("furious and exhausted") is still "sharing" 4 of 5 times: the two labels overlap there.
- Still rule-based (decide what a user message means with patterns): 4 regexes in `tools/context_extractor.go`, 2 in `agents/incoming_message_analyzer.go`. `placeholder_lines.go` checks Moly's own reply text for placeholders (a format check, not a reading of the user).

## Principle reasoner and the extra "reacting" call (2026-10-12)
- **The principle reasoner never produced a question** (0 of 12 on the cases that need one). Its prompt listed the principles by display name ("Stakeholder Consideration") and asked for `principle_name`; the code turned the answer into a question by matching ids (`stakeholder_consideration`). The model wrote display names, no id matched, the question was empty and the concern was dropped silently. This is why no principle question appeared in the D11 run. Now: the prompt lists ids and defines `needsClarification`, the answer is normalised by `principleID` (names and ids both work), and a dropped concern is logged with the reason. End to end 27/28; the principle question still comes after any open gap question (by design), and a plain request is not affected (checked live).
- **The "reacting" intent cost a second model call** (`getReactionContextLLM`) whose result, `ReactionTarget`, nothing read. Removed: a reaction ("Yes", "ok") now costs one call (test `TestReactingIntentCostsOneModelCall`).

## Pending questions are cancelled when they stop being relevant (2026-10-12)
`JudgePendingStillWorthAsking` runs when pending (ignored) questions are about to come back as candidate gaps, and only then. It sees the dialogue, the user's newest message and the questions. A question the user has since answered, that later words made moot, or that the conversation has left behind is set to `cancelled` (kept in the history, no longer open, no longer counted in maturity); an unreadable or failed answer keeps every question. Scenario S52; battery 30/30.

## Format parsers that remain (not judgements about the user)
`tools/context_extractor.go` (bullet lists in a text section) and `agents/incoming_message_analyzer.go` (a "From: Name" header on a pasted message) use regexes on structure. They do not read what the user means.

## Care checks (2026-10-12)
- The turn after Moly refused a crisis statement starts with one fixed check-in (S54); the refusal records `safetyAlertType` in the stored reply.
- A message the model rates as a crisis is not answered with a name question (S55). `MessageClarityAnalyzer.ModelPriority` asks the model only; the heuristic shortcut cannot see danger in a short message. Battery "clarity crisis": 55/55.
- The principle reasoner's `needsClarification` was too wide (a third of ordinary requests with the dialogue in view): it now needs a plan that could wrong someone, hide something, or decide for them. In context 20/20.

## Product knowledge, secrets, and the clarity decision (2026-10-12)
- **Questions about Moly** ("how do I use you?", "what does the Skip button do?", "who can read my messages?"): the intent call reports `aboutMoly` (battery 60/60 over 12 cases). Such a message sets no goal, saves no person, asks nothing back, and is answered from `ProductKnowledge` (agents/product_knowledge.go), a fact sheet checked against the side panel and the backend and included in the reply prompt only for these messages. Layer 1 does not look for a goal in them (before, "Learn how to use Moly" became the locked goal and the next request asked "switch focus?"). Keep the sheet in step with the extension.
- **Secrets are not stored.** Card numbers (Luhn) and IBANs (mod 97) are cut out by their shape before any model call, summary or save (`tools.RedactStructured`). Passwords, PINs, ID numbers and keys are listed by the extraction (`sensitive`, exactly as written) and cut out of the message and of everything derived from it before the intent, the reply and every save (`tools.RedactItems`, `agents.ScrubSensitive`). Limits: the safety judgement, the meta-instruction call and the extraction itself read the raw text of that one message; the safety incident row of a refused message holds only the shape-redacted text; the long extraction prompt (MOLY_EXTRACT_PROMPT=long) does not ask for `sensitive`.
- **Clarity is always decided by the model.** The length-and-punctuation heuristic that skipped the model is deleted.

## Full dynamic pass (2026-10-12) and what it changed
D1-D35 on Claude haiku (about 100 turns). Safety held in every scenario. Fixed from it: the name question waits until the user states an aim that involves the person (chat, abuse disclosure, a figure of speech no longer get "can you give me a name for ..."; `PersonInput.NoNameQuestion`, S59); an organization is not a person (extractor and person judge: a bank gave a nonsense "is the bank a person?" question); the "just to confirm" question is written with the dialogue in view; replies and check-ins never contain a hotline number (a check-in once ended with "please call 988"). Still open: a "help" reply that asks a question has no Skip (the question can come back unchanged); Moly sometimes misreads who a pronoun-less detail belongs to ("Elena's sister").

## Judge map (2026-10-15)

Every model call that steers Moly, with what happens when the call fails or cannot be read. "Small" is the fast model (`MOLY_MODEL_FAST`), "reply" the reply model. The right column is the battery group in `agents/judges_live_test.go` (or the live test) that measures it; a blank means no labelled cases yet (step 2 of `docs/QUALITY_PLAN.md`: add them).

| Judge (file) | Model | Decides | When it fails | Measured by |
|---|---|---|---|---|
| extraction (`context_extractor.go`) | small | intention, people, traits, relations, secrets | empty extraction (nothing stored), then form checks (`extraction_sanitize.go`) | extraction battery, names, pasted text, prompt leak |
| intent + meta-instruction (`message_reading.go`, `intent_detector.go`, `meta_instruction_detector.go`) | small | what the user is doing; instructions about Moly | the two judges are asked separately; then "unknown" | intent label, adjustsDraft, answersQuestion, meta-instruction, contradictory |
| safety (`tools/constitutional_evaluator.go`) | small | block, warn or allow | no block on a failed call; findings under 0.6 confidence are dropped | safety, safety late in a conversation |
| principle reasoner (`conversation_agent.go`, `layer7_*`) | small / reply | whether a principle question is worth asking, and the question | canned question text | principle reasoner (3 groups) |
| clarity (`message_clarity_analyzer.go`) | small | can Moly proceed; crisis | proceeds | clarity: can proceed / crisis |
| gap generator (`layer4_gap_detector.go`) | reply | what is missing for the goal | no gaps (the result is given) | held-out cases only |
| repeated question (`gap_novelty.go`, `principle_repeat.go`) | reply | whether a question repeats an earlier one | all questions kept | repeated gap (87% on the small model, paraphrases pass on the reply model) |
| pending question (`pending_questions.go`) | small | answered / still worth asking | kept open | pending question |
| goal relation (`goal_relation.go`), goal switch (`goal_switch.go`) | small | same / refinement / substep / different goal; yes-or-no to a switch | "same", "unclear": no switch question | goal relation; switch answer |
| person reference (`person_label.go`) | small | known / new / none for a phrase | "unsure": the person is not saved and the reply asks | person reference (81 cases) |
| name answer (`name_answer_judge.go`) | small | given / declined / other | question stays open | name answer |
| trait change (`trait_change_judge.go`) | small | a real change or a rewording | "no change" | trait change |
| withdrawal (`layer11_denial_protocol.go`) | small | gives up or refuses the topic | not a denial | withdrawal |
| deepening (`deepening_judge.go`) | small | is a reflective question worth asking | "not worth" | deepening |
| persistence (`conversation_agent.go`, `layer10_*`) | small | the user keeps to a course after a concern | not persistent / static probe | persistence probe |
| reply premise check (`premise_check.go`) | small / reply | the reply takes something for granted | the reply is kept | premise (live) |
| one-question rewrite (`single_question.go`) | reply | rewrites a reply to one question | first question of the draft | property checks |
| link notice (`link_notice.go`) | small | the reply says Moly does not read pages | a fixed sentence | link notice (live, 21/21) |
| question writers (`confirm_question.go`, `name_question.go`, `blocked_reply.go`, `greeting_reply.go`) | reply | the wording of a question or greeting | fixed text | S-scenarios, D-scenarios |
| summary (`tools/conversation_summarizer.go`) | small | the conversation summary | old summary kept | summary writer |
| reply (`conversation_agent.go`) | reply | the help text and drafts | error to the user, nothing saved | dynamic scenarios |

Gaps found while writing this table: the gap generator, the premise check and the question writers have no labelled battery of their own (the dynamic scenarios are their only measure), and the repeat judge is the least accurate on the small model.
