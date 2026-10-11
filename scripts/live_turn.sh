#!/usr/bin/env bash
# Dynamic live test: one turn at a time, so that whoever drives it (a person, or Claude playing the user) reads Moly's reply and
# decides the next message. The conversation is kept in a state file between calls.
#
#   scripts/live_turn.sh new                 register a throwaway user and start a new conversation
#   scripts/live_turn.sh say "message"       send a user message, print Moly's reply, canSkip, reply kind, maturity, log markers
#   scripts/live_turn.sh skip                press the Skip button
#   scripts/live_turn.sh memory              what Moly remembers (summary, goal, people, open questions)
#   scripts/live_turn.sh people              the saved contacts
#
# MOLY_URL (default http://localhost:11436), MOLY_SERVER_LOG (stage markers from the server log), MOLY_LIVE_STATE (state file).
set -u
BASE="${MOLY_URL:-http://localhost:11436}"
STATE="${MOLY_LIVE_STATE:-/tmp/moly_live_state.json}"
cmd="${1:-}"; shift || true

get() { python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(d.get(sys.argv[2],""))' "$STATE" "$1" 2>/dev/null; }
put() { python3 - "$STATE" "$1" "$2" <<'PY'
import json,sys,os
p,k,v=sys.argv[1:4]
d=json.load(open(p)) if os.path.exists(p) else {}
d[k]=v
json.dump(d,open(p,"w"))
PY
}
markers() {
  [ -n "${MOLY_SERVER_LOG:-}" ] || return 0
  local from; from="$(get logmark)"; from="${from:-0}"
  tail -n +"$((from+1))" "$MOLY_SERVER_LOG" | grep -E "Reply exit|GapMaturity|FactConfidence|Name requested|Goal switch|Skip pressed|candidate gaps again|Goal in doubt|Person in doubt|Answer only|Goal relation judged|Safety verdict|PendingQuestions|Summary rewritten|Summary update failed|Identity\]" | sed 's/^[0-9\/: ]* /    log: /' | cut -c1-200
}
mark() { [ -n "${MOLY_SERVER_LOG:-}" ] && put logmark "$(wc -l < "$MOLY_SERVER_LOG")" || true; }

case "$cmd" in
new)
  curl -s -m 5 "$BASE/api/status" >/dev/null || { echo "Server not reachable at $BASE"; exit 2; }
  stamp="$(date +%s)"; email="dyn_${stamp}@example.invalid"
  reg=$(curl -s -m 30 -X POST "$BASE/api/auth/register" -H 'Content-Type: application/json' -d "{\"name\":\"Dynamic Test\",\"email\":\"$email\",\"password\":\"dyn-$stamp\"}")
  token=$(echo "$reg" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("token",""))')
  [ -n "$token" ] || { echo "Register failed: $reg"; exit 2; }
  rm -f "$STATE"; put token "$token"; put conv ""; put email "$email"; mark
  echo "new user $email, new conversation"
  ;;
say|skip)
  token="$(get token)"; [ -n "$token" ] || { echo "run: live_turn.sh new"; exit 2; }
  if [ "$cmd" = skip ]; then msg="Go ahead with what you have."; skipflag=skip; else msg="${1:?message required}"; skipflag=""; fi
  mark
  body=$(python3 -c 'import json,sys; print(json.dumps({"message":sys.argv[1],"conversationId":sys.argv[2],"browserSessionId":"dynamic-test","skipQuestions":sys.argv[3]=="skip"}))' "$msg" "$(get conv)" "$skipflag")
  start=$(date +%s)
  out=$(curl -s -m 1500 -w '\n%{http_code}' -X POST "$BASE/api/message-processor" -H 'Content-Type: application/json' -H "Authorization: Bearer $token" -d "$body")
  code=$(echo "$out" | tail -n1); json=$(echo "$out" | sed '$d')
  python3 - "$json" "$code" "$(( $(date +%s) - start ))" "$STATE" <<'PY'
import json,sys
raw,code,secs,state=sys.argv[1:5]
try: d=json.loads(raw)
except Exception: print("status",code,"unreadable:",raw[:300]); sys.exit(0)
md=d.get("metadata") or {}
conv=d.get("conversationId") or ""
if conv:
    s=json.load(open(state)); s["conv"]=conv; json.dump(s,open(state,"w"))
print("MOLY>", d.get("response",""))
print(f"  [status {code}, {secs}s, canSkip={str(md.get('canSkip',False)).lower()}, kind={md.get('replyExit','?')}, maturity={md.get('maturity','?')}]")
PY
  markers
  ;;
memory)
  token="$(get token)"; conv="$(get conv)"
  curl -s -m 30 "$BASE/api/conversations/$conv/memory" -H "Authorization: Bearer $token" | python3 -c '
import sys,json
d=json.load(sys.stdin)
print("SUMMARY:", d.get("summary") or "(none yet)", "| covers", d.get("summaryCoversMessages"), "messages")
print("GOAL:", d.get("goal") or "(none)")
print("PEOPLE:", "; ".join(d.get("people") or []) or "(none)")
print("OPEN:", "; ".join(d.get("openQuestions") or []) or "(none)")'
  ;;
people)
  token="$(get token)"
  curl -s -m 30 "$BASE/api/contacts" -H "Authorization: Bearer $token" | python3 -c '
import sys,json
d=json.load(sys.stdin); d=d.get("contacts",d) if isinstance(d,dict) else d
for c in d or []: print("-", c.get("name"), "|", c.get("relationship"))'
  ;;
*) sed -n 2,13p "$0"; exit 1 ;;
esac
