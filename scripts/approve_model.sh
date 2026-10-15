#!/bin/bash
# Measures a model before it goes into tools/approved_models.json (plan "models by task").
#
#   scripts/approve_model.sh fast  haiku     the live judge battery on the model (the floors are in the test and docs/JUDGE_BATTERY.md)
#   scripts/approve_model.sh reply sonnet    the battery on the model, then a replay of the recorded conversation with the model as
#                                            the reply model and the fast model for the rest, so that the replies can be read
#
# It runs through the Claude CLI (the developer's own login: prompts leave the machine, use no real data). The model name is the
# CLI's ("haiku", "sonnet", "opus"). It does NOT edit the table: when the numbers hold and the replies read well, add the model
# there by hand with today's date and a note. The Claude CLI is not the API: an approved model should also be tried once with a key.
#
# Run it with NOTHING else using the Claude CLI: a failed call is counted as a wrong answer (the first Sonnet run, made while other
# jobs used the CLI, showed dozens of misses that vanished when the same judges were run alone).
set -u
ROLE=${1:-}; MODEL=${2:-}
[ -n "$ROLE" ] && [ -n "$MODEL" ] || { sed -n 2,13p "$0"; exit 2; }
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT/moly-go" || exit 2

echo "== live judge battery on $MODEL (3 replays per case)"
MOLY_LIVE_LLM=1 LLM_PROVIDER=claude-cli AGENT_MODEL="$MODEL" go test ./agents/ -run TestLiveJudgeBattery -v -timeout 3000s 2>&1 | grep -E "judges_live_test.go|^(--- |ok|FAIL)" | sed 's/^ *//'
RESULT=${PIPESTATUS[0]}

if [ "$ROLE" = "reply" ]; then
  echo
  echo "== replay with $MODEL as the reply model (fast model: haiku); read the replies yourself"
  go build -o "$ROOT/bin/moly" . || exit 2
  SCRATCH=${TMPDIR:-/tmp}/approve_model_$$
  mkdir -p "$SCRATCH/home"
  ln -s "$HOME/.claude" "$SCRATCH/home/.claude"; ln -s "$HOME/.claude.json" "$SCRATCH/home/.claude.json"
  HOME="$SCRATCH/home" MOLY_DB_KEY=$(printf 'cd%.0s' $(seq 1 32)) MOLY_PORT=:11498 LLM_PROVIDER=claude-cli AGENT_MODEL=haiku \
    MOLY_MODEL_FAST=haiku MOLY_MODEL_REPLY="$MODEL" OLLAMA_ENDPOINT=http://127.0.0.1:1 "$ROOT/bin/moly" > "$SCRATCH/server.log" 2>&1 &
  PID=$!
  for i in $(seq 1 90); do curl -s -m 1 http://127.0.0.1:11498/api/status >/dev/null && break; sleep 1; done
  python3 "$ROOT/scripts/replay_conversation.py" http://127.0.0.1:11498
  kill $PID
  echo "(server log: $SCRATCH/server.log)"
fi
echo
[ "$RESULT" = "0" ] && echo "battery: PASSED" || echo "battery: FAILED (a judge is under its floor): do not approve"
exit "$RESULT"
