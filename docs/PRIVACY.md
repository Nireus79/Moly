# Privacy: what Moly stores, where, and how it is removed

Audited 2026-10-12 (tests named below keep it true).

## What is stored, and where
- **Database**: `~/.moly/moly.db`, encrypted with SQLCipher (AES-256). The key lives in the operating system's keychain (or `MOLY_DB_KEY`); it is never written to a file. The directory is `0700` and the database files `0600`, and the process umask is `0077`, so new files (write-ahead log, shared memory) are private from the start (`database.HardenDataDir`).
- **Per user**: account (email, bcrypt password hash), sessions (24 h), conversations and chat messages, per-message summaries, conversation summaries, people and their traits, goals and facts (`context_attributes`), clarification questions and answers, reflections, safety incidents, metrics. Every table that can hold user data has a `user_id` (or a `conversation_id` or a foreign key that leads to one).
- **Browser**: the session and the cached About Me profile in `localStorage`, error reports in `localStorage`, extension data in `chrome.storage.local`.
- **Third parties**: to write a reply, the text of the turn is sent to the model the user configured: a local Ollama model, or Anthropic, or OpenAI. Nothing else leaves the machine.

## What is never stored
- **Secrets** the user types. Card numbers (Luhn) and IBANs (mod 97) are cut out by their shape before any model call, summary or save. Passwords, PINs, ID numbers and keys are listed by the extraction (`sensitive`) and cut out of the message and of everything derived from it before the intent, the reply and every save (`tools/redact.go`, `agents.ScrubSensitive`; scenario S57).
- **Anything learned from a refused message**: no people, no goal, no per-message summary, no facts, no saved entities (S49, S51). Kept for audit: the two chat lines and the safety incident.
- **User content in the log.** The standard logger writes through a scrubber (`privacy` package): the message, the goal, names, traits, quotes, gaps, questions, summaries and Moly's replies are registered as the server learns them and masked in every later log line. `MOLY_LOG_CONTENT=1` turns the masking off for debugging. Test `TestLogsHoldNoUserContent`; a live run with unusual words found none of them in 893 log lines.

## Deleting a profile
`DELETE /api/user/delete` with the password (bcrypt check). In one transaction it deletes every row of every table with a `user_id`, every row of tables keyed only by conversation, and the rows that hang from the user's rows through foreign keys, then the user. A table added later is covered because the tables are found from the schema. Afterwards it truncates the write-ahead log and compacts the file (`VACUUM`), `secure_delete` is on for every connection, the user's sessions are gone with the user, and the server drops the user's in-memory state (saved conversation states, skip offers, learning agent, the log scrubber's list). The extension clears `chrome.storage.local`, `localStorage` and `sessionStorage`. On logout it removes the cached About Me profile and the error reports.
Tests: `TestDeleteUserLeavesNothingInAnyTable` (fills every table for a user and a bystander, deletes, scans every cell), `TestScenarioS58ProfileDeletionRemovesEverything` (through the handler: wrong password deletes nothing, the right one leaves no row mentioning the user, the old token fails, memory is empty, another user is untouched), `TestSecureDeleteIsOnForEveryConnection`.

## Known gaps (decisions open)
- No export of a user's data on request.
- The safety judgement, the meta-instruction call and the extraction itself read the raw text of a message once (they are how secrets are found); a refused message's incident row holds the shape-redacted text.
- The long extraction prompt (`MOLY_EXTRACT_PROMPT=long`) does not list secrets.
- Raw refused text stays in the safety incident table until the profile is deleted (no retention limit yet).
- Production items: the server listens on all interfaces, with `Access-Control-Allow-Origin: *` and no TLS (local use only for now); sessions are plain random tokens in the `sessions` table.
- Old plaintext database copies on a developer machine are outside the program's control (see the audit note for 2026-10-12: `~/.moly/*.bak*`, `*-test-lifecycle.db`).
