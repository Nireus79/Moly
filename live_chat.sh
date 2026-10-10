#!/usr/bin/env bash
# Interactive live chat: YOU type every message, nothing is scripted. Shows Moly's reply, whether the skip button is offered
# (canSkip), and the stage markers from the server log.
# Usage:  MOLY_SERVER_LOG=/path/to/server.log ./live_chat.sh
#   type a message and press Enter;  /skip = press the skip button (only works when canSkip is true);  /new = new conversation;
#   empty line or /quit = end.
# Live check 2: the behaviour added after step 7 (see ORCHESTRATOR_DESIGN.md, "Decisions of 2026-10-10").
# Usage:  ./live_check2.sh [SECTIONS]            e.g. ./live_check2.sh C E   (default: all; C needs B, so B runs first)
#         ./live_check2.sh                       (server already running on :11436, Ollama on :11434)
#         MOLY_SERVER_LOG=/path/to/server.log ./live_check2.sh    (also prints the stage markers from the server log)
#
# Rules fail the run. Notes are what the local model decided; read them, they do not fail the run.
# Each message can take minutes on the local model (timeout 15 minutes per message).
# It registers a throwaway user; the user stays in the database.
set -u
SECTIONS=""
case " $SECTIONS " in *" C "*) case " $SECTIONS " in *" B "*) ;; *) SECTIONS="B $SECTIONS";; esac;; esac
want() { case " $SECTIONS " in *" $1 "*) return 0;; esac; return 1; }
BASE="${MOLY_URL:-http://localhost:11436}"
STAMP="$(date +%s)"
EMAIL="livecheck2_${STAMP}@example.invalid"
PASS="livecheck-${STAMP}"
FAILS=0

fail() { echo "  FAIL: $*"; FAILS=$((FAILS+1)); }
ok()   { echo "  ok:   $*"; }
note() { echo "  note: $*"; }

curl -s -m 5 "$BASE/api/status" >/dev/null || { echo "Server not reachable at $BASE. Start bin/moly from moly-go/ first."; exit 2; }
curl -s -m 5 localhost:11434/api/tags >/dev/null || { echo "Ollama not reachable on :11434."; exit 2; }

REG=$(curl -s -m 30 -X POST "$BASE/api/auth/register" -H 'Content-Type: application/json' \
  -d "{\"name\":\"Live Check 2\",\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")
TOKEN=$(echo "$REG" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("token",""))')
[ -n "$TOKEN" ] || { echo "Register failed: $REG"; exit 2; }

CONV=""
send() { # $1 = message, $2 = "skip" to press the skip button ; uses CONV (empty = new conversation) ; sets REPLY, CONV, CODE, CANSKIP
  local body out json
  body=$(python3 -c 'import json,sys; print(json.dumps({"message":sys.argv[1],"conversationId":sys.argv[2],"browserSessionId":"live-check-2","skipQuestions":sys.argv[3]=="skip"}))' "$1" "$CONV" "${2:-}")
  out=$(curl -s -m 900 -w '\n%{http_code}' -X POST "$BASE/api/message-processor" \
    -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" -d "$body")
  CODE=$(echo "$out" | tail -n1)
  json=$(echo "$out" | sed '$d')
  REPLY=$(echo "$json" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("response",""))' 2>/dev/null)
  CONV=$(echo "$json" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("conversationId","") or "")' 2>/dev/null)
  CANSKIP=$(echo "$json" | python3 -c 'import sys,json; print(str((json.load(sys.stdin).get("metadata") or {}).get("canSkip", False)).lower())' 2>/dev/null)
}
questions() { printf '%s' "$REPLY" | tr -cd '?' | wc -c; }
words()     { printf '%s' "$REPLY" | wc -w; }
people()    { curl -s -m 30 "$BASE/api/contacts" -H "Authorization: Bearer $TOKEN" |
  python3 -c 'import sys,json
d=json.load(sys.stdin); d=d.get("contacts",d) if isinstance(d,dict) else d
for c in d or []: print(c.get("name"))'; }
one_question_rule() { # the reply asks at most one thing
  [ "$(questions)" -le 1 ] && ok "at most one question in the reply" || fail "the reply asks $(questions) questions"
}
no_help_lines_rule() {
  if printf '%s' "$REPLY" | grep -qiE '988|741741|116 123|helpline|hotline|crisis line|samaritans'; then
    fail "the reply carries help lines"
  else ok "no help lines"; fi
}
mark() { [ -n "${MOLY_SERVER_LOG:-}" ] && LOGMARK=$(wc -l < "$MOLY_SERVER_LOG") || LOGMARK=0; }
markers() { # print the stage markers written since mark()
  [ -n "${MOLY_SERVER_LOG:-}" ] || return 0
  tail -n +"$((LOGMARK+1))" "$MOLY_SERVER_LOG" | grep -E "Reply exit|GapMaturity|FactConfidence|Name requested|Goal switch|Skip pressed|skipped, not answered|candidate gaps again|Pending question answered|pending question|Goal in doubt|Person in doubt|Answer only|Goal relation judged|Goal changed|Safety verdict" | sed 's/^[0-9\/: ]* /    log: /' | cut -c1-200
}


echo "Throwaway user: $EMAIL   (type /skip to press the button, /new for a new conversation, /quit to end)"
CONV=""
while true; do
  printf '\nyou> '
  IFS= read -r LINE || break
  case "$LINE" in ""|"/quit") break;; "/new") CONV=""; echo "(new conversation)"; continue;; esac
  mark
  if [ "$LINE" = "/skip" ]; then send "Go ahead with what you have." skip; else send "$LINE"; fi
  echo
  echo "moly> $REPLY"
  echo "      [status $CODE, canSkip=$CANSKIP]"
  markers
done
