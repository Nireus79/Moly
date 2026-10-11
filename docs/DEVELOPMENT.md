# Development guide

## Prerequisites
Go 1.21+, Node 18+, Ollama with `mistral`. SQLite is bundled.

## Build and run
```bash
cd moly-go
go build -o ../bin/moly .          # binaries are never committed (bin/ is ignored)
../bin/moly                        # port 11436; /api/status answers after ~40–50 s
kill $(pgrep -x moly)              # stop it (do not use pkill -f)
cd ../moly-extension && npm install && npm run build   # load dist/ as an unpacked extension
```

## Test
```bash
cd moly-go
go vet ./... && go test -count=1 ./...     # ~30 s; scripted scenarios + unit tests
```
Scenarios live in `orchestration_test.go` (scripted model, real temp DB). When you add a rule, add a scenario and check it fails without the rule (a mutation).

## Live checks (real model; each message can take minutes)
```bash
export MOLY_SERVER_LOG=/path/to/server.log      # prints stage markers; start the server with its output redirected there
scripts/live_check2.sh [A B C D E F]            # rule checks; notes are for you to read
scripts/live_chat.sh                            # interactive: you type; /skip presses the button
```
Run one at a time. The skip button cannot be tested with canned answers.

## Dynamic tests (someone plays the user)
```bash
export MOLY_SERVER_LOG=/path/to/server.log
scripts/live_turn.sh new                  # throwaway user, new conversation
scripts/live_turn.sh say "message"        # one turn: Moly's reply, canSkip, reply kind, maturity, log markers
scripts/live_turn.sh skip                 # press the Skip button
scripts/live_turn.sh memory               # what Moly remembers (summary, goal, people, open questions)
scripts/live_turn.sh people               # saved contacts
```
The driver (a person, or Claude playing the user) reads each reply and writes the next message, so the test follows what Moly really asks. `GET /api/conversations/{id}/memory` returns the user's own memory data (nothing is logged).

### Testing with a strong model: `claude-cli`
```bash
LLM_PROVIDER=claude-cli AGENT_MODEL=haiku ../bin/moly     # haiku | sonnet | opus
```
Every model call runs `claude -p` (Claude Code in print mode, no tools) on **your claude.ai login**. The child process is started without `ANTHROPIC_API_KEY`, so it never bills an API account. A turn takes about a minute instead of ten. **The prompts leave the machine: use throwaway test users only.** Temperature and token limits cannot be set. A strong model hides weaknesses of small ones, so use it to find logic errors and Mistral to check robustness.

## Dead code
```bash
go install golang.org/x/tools/cmd/deadcode@latest
cd moly-go && ~/go/bin/deadcode ./...
```

## Conventions
- No message text or email in logs. Keep full chat history (resume needs it).
- Model judgements, not word lists. Unreadable model output fails safe.
- Commit and push only when asked; pushes stay small (no binaries, logs, databases).
- Decisions and findings are recorded in `ORCHESTRATOR_DESIGN.md`.
