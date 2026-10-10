# Orchestration map: what runs in which order, and what blocks what (2026-10-09)

Source: `moly-go/main.go`, `MessageProcessorHandler`. Line numbers are approximate.

## Stage order (one message)

1. Auth, validation, clarification-answer detection, **save user message** (~880–950)
2. Create conversation (~957); meta-instruction detection (~1003); execution state (~1042); maturity load (~1072)
3. **Extraction** `extractionPhase.Run` (~1273), then `extractedContext = epOutput.ExtractedContext` (~1284)
4. **Intent decided** (LLM, ~1290; new position) — drives everything below
5. **Name answer applied** (`ApplyNameAnswer`, ~1300) — only if the message is not a greeting
6. **Contact saved** (~1320s); then **name question marked** if the person has no name (`asked`)
7. Clarification capture, conflict answers, conversation history load (~1450–1850)
8. Build the agent context and analysis context (~1850–2400); goal/entity cleanup for greetings (~2160)
9. **Layers 1–11 orchestrator** (`ProcessMessage`, ~2420): Layer 2 safety, 3 maturity, 4 gaps, 5 conflicts, 6–11
10. Safety evaluation, maturity update, accomplishment marks (~2500–2700)
11. **Agent `Run`** (~2880): Layer 2 harm check → **name gate** (question only) → gap gate → Socratic → response
12. Reply resolved and **saved** (`resolveReplyText`, end of handler)

## What blocks what

- **A greeting** is decided at step 4. Before the fix it was decided at step 8, after the contact was saved at step 6, so a greeting saved "the user" as a contact.
- **The name answer** needs the extracted name, so it must run after step 3 and before step 6 (the save). It ran before step 3 until this fix, so it always saw no name.
- **The name question** must be marked after the save (step 6) and read by the agent (step 11). Marking ran before the save until this fix.
- **Gap and Socratic questions** wait for the name gate: the agent returns the name question first and stops.
- **Layer 2 safety** (step 9) runs before the agent (step 11). The name gate is placed after the Layer 2 check, so safety is never skipped.
- **The reply** is saved only at the end (step 12), so a failed run saves nothing.

## Known remaining ordering problems

- The message summary for message N is written before the layers run (FIX #10). No layer reads it for message N any more, but the write still happens early.
- The intent is decided without conversation history (the history is loaded at step 7). A greeting is still recognised; a "reacting" intent may be less accurate.
- A pasted profile is extracted as the user's own goal (step 3). Attribution of third-party text is not implemented.
- A second early contact-save block (~1155) runs only on retries, and the post-extraction block (~1320) does the normal save. The duplication is for the dead-code phase.
