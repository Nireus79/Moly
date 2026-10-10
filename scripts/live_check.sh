#!/usr/bin/env bash
# Step 7: one live check of scenario S1 against a running server (see ORCHESTRATOR_DESIGN.md).
# Usage:  ./live_check.sh            (server already running on :11436)
# It registers a throwaway user, sends the three S1 messages in one new conversation,
# prints each reply, and checks the same things TestScenarioS1ThreeMessages checks.
# Replies take minutes on the local model; the timeout per message is 15 minutes.
set -u
BASE="${MOLY_URL:-http://localhost:11436}"
STAMP="$(date +%s)"
EMAIL="livecheck_${STAMP}@example.invalid"
PASS="livecheck-${STAMP}"
FAILS=0

fail() { echo "  FAIL: $*"; FAILS=$((FAILS+1)); }
ok()   { echo "  ok:   $*"; }

curl -s -m 5 "$BASE/api/status" >/dev/null || { echo "Server not reachable at $BASE. Start bin/moly first."; exit 2; }
curl -s -m 5 localhost:11434/api/tags >/dev/null || { echo "Ollama not reachable on :11434."; exit 2; }

REG=$(curl -s -m 30 -X POST "$BASE/api/auth/register" -H 'Content-Type: application/json' \
  -d "{\"name\":\"Live Check\",\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")
TOKEN=$(echo "$REG" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("token",""))')
[ -n "$TOKEN" ] || { echo "Register failed: $REG"; exit 2; }

CONV=""
send() { # $1 = message ; sets REPLY, CONV, CODE
  local body out
  body=$(python3 -c 'import json,sys; print(json.dumps({"message":sys.argv[1],"conversationId":sys.argv[2],"browserSessionId":"live-check"}))' "$1" "$CONV")
  out=$(curl -s -m 900 -w '\n%{http_code}' -X POST "$BASE/api/message-processor" \
    -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" -d "$body")
  CODE=$(echo "$out" | tail -n1)
  local json; json=$(echo "$out" | sed '$d')
  REPLY=$(echo "$json" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("response",""))' 2>/dev/null)
  CONV=$(echo "$json" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("conversationId",""))' 2>/dev/null)
}
contacts() { curl -s -m 30 "$BASE/api/contacts" -H "Authorization: Bearer $TOKEN" |
  python3 -c 'import sys,json
d=json.load(sys.stdin); d=d.get("contacts",d) if isinstance(d,dict) else d
for c in d or []: print(c.get("name"), "|", c.get("nameStatus", c.get("name_status","")))'; }

NAMEQ="Can you give me a name for the girl you mentioned?"

echo "== 1. greeting"
send "Hi Moly"; echo "  reply: $REPLY"
[ "$CODE" = 200 ] && [ -n "$REPLY" ] && ok "status 200, non-empty reply" || fail "status $CODE, reply empty?"
[ "$REPLY" != "$NAMEQ" ] && ok "no name question" || fail "greeting asked for a name"
[ -z "$(contacts)" ] && ok "no person saved" || fail "a greeting saved a person: $(contacts)"

echo "== 2. goal message"
send "I want to write a first message to a girl I saw on fetlife."; echo "  reply: $REPLY"
[ "$CODE" = 200 ] && ok "status 200" || fail "status $CODE"
case "$REPLY" in
  "Can you give me a name for the "*" you mentioned?") ok "reply is the name question (label chosen by the model)";;
  *) fail "reply is not the name question";;
esac
echo "  contacts: $(contacts | tr '\n' ';')"

echo "== 3. name given"
send "Her name is Christine."; echo "  reply: $REPLY"
[ "$CODE" = 200 ] && ok "status 200" || fail "status $CODE"
[ "$REPLY" != "$NAMEQ" ] && ok "name question not asked again" || fail "name question asked again"
echo "  contacts: $(contacts | tr '\n' ';')"
[ "$(contacts | wc -l)" = 1 ] && contacts | grep -q '^Christine' && ok "one person, named Christine" || fail "expected one person named Christine"

echo
echo "Compare by eye: the live model decides the goal wording and the gap/principle questions, so the replies in 1 and 3 will differ from the scripted ones. Only the checks above are rules."
echo "Throwaway user: $EMAIL (conversation $CONV). Remove it with DELETE /api/user/delete if you want."
[ "$FAILS" = 0 ] && echo "RESULT: all checks passed" || echo "RESULT: $FAILS check(s) failed"
exit $((FAILS>0))
