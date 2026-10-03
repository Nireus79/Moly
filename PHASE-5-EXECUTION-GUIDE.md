# PHASE 5: Execution Guide

**Date:** October 3, 2026 (Session 24 Complete)  
**Status:** Ready for execution with proper environment setup  
**Prerequisite:** LLM (Ollama) must be running

---

## Prerequisites for Execution

### 1. LLM Service (Required)
Before starting Phase 5 tests, ensure Ollama is running:

```bash
# Check if Ollama is running
curl -s http://127.0.0.1:11434/api/tags

# If not running, start it
ollama serve  # (in another terminal or background)

# Verify mistral model is available
ollama list | grep mistral
```

### 2. Database Setup
Clean database for fresh test run:

```bash
# Remove old test database
rm -f /tmp/moly-v2.db /tmp/moly-v2-test.db
```

### 3. Port Availability
Ensure port 8080 is free:

```bash
# Check if port 8080 is in use
lsof -i :8080

# If in use, kill the process
pkill -f "bin/moly"
sleep 2
```

---

## Test 1: Phase Progression (M1→M2→M3)

### Step 1: Start Server

```bash
cd /home/nireus79/vs_projects/Moly/Moly/moly-go
./bin/moly
```

**Expected Output:**
```
[Database] Initialized at /tmp/moly-v2.db
[Moly] V2 API Server initialized
[LLMClient] Local Ollama detected at http://127.0.0.1:11434
[Moly] Server starting on http://localhost:8080
```

Wait for "Server starting on http://localhost:8080" before proceeding.

### Step 2: Create Test User

In another terminal:

```bash
TEST_EMAIL="phase5_user@test.example.com"
TEST_PASSWORD="Phase5TestPassword123!"

SIGNUP=$(curl -s -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")

echo "Signup response: $SIGNUP"

# Extract session token
SESSION_TOKEN=$(echo "$SIGNUP" | grep -o '"session_id":"[^"]*' | cut -d'"' -f4)
echo "Session token: $SESSION_TOKEN"

# Save for next messages
export SESSION_TOKEN
export TEST_EMAIL
```

If signup fails with 409 (user exists), try login instead:

```bash
LOGIN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")

SESSION_TOKEN=$(echo "$LOGIN" | grep -o '"session_id":"[^"]*' | cut -d'"' -f4)
export SESSION_TOKEN
```

### Step 3: Send Message M1 (Initial)

```bash
CONV_ID="test_phase5_001"

M1=$(curl -s -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"message\": \"I want to improve my relationship with John. He is my colleague.\",
    \"conversationId\": \"$CONV_ID\",
    \"aboutMe\": {
      \"communicationStyle\": \"\",
      \"coreValues\": [],
      \"tonePreference\": \"\"
    }
  }")

echo "=== M1 RESPONSE ==="
echo "$M1" | jq . 2>/dev/null | head -50
```

**Expected in response:**
- `metadata.phase.current = "initial"`
- `metadata.phase.maturity ≈ 0.2-0.3`
- `metadata.phase.transitioned = false`

### Step 4: Send Message M2 (Gathering)

```bash
M2=$(curl -s -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"message\": \"We often disagree on how to approach technical decisions. I try to be collaborative, but he seems dismissive of my ideas.\",
    \"conversationId\": \"$CONV_ID\",
    \"aboutMe\": {
      \"communicationStyle\": \"collaborative\",
      \"coreValues\": [\"respect\", \"clear_communication\"],
      \"tonePreference\": \"constructive\"
    }
  }")

echo "=== M2 RESPONSE ==="
echo "$M2" | jq . 2>/dev/null | head -50
```

**Expected in response:**
- `metadata.phase.current = "gathering"` (NOT "initial" - critical!)
- `metadata.phase.previous = "initial"`
- `metadata.phase.maturity ≈ 0.4-0.5`
- `metadata.phase.transitioned = true`

### Step 5: Send Message M3 (Analysis)

```bash
M3=$(curl -s -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"message\": \"I have tried being direct about my concerns, but it does not seem to work. Should I escalate to management or try a different approach?\",
    \"conversationId\": \"$CONV_ID\",
    \"aboutMe\": {
      \"communicationStyle\": \"collaborative\",
      \"coreValues\": [\"respect\", \"clear_communication\", \"problem_solving\"],
      \"tonePreference\": \"constructive\"
    }
  }")

echo "=== M3 RESPONSE ==="
echo "$M3" | jq . 2>/dev/null | head -50
```

**Expected in response:**
- `metadata.phase.current = "analysis"`
- `metadata.phase.previous = "gathering"`
- `metadata.phase.maturity ≈ 0.6-0.7`
- `metadata.phase.transitioned = true`

---

## Test 2: Database Verification

After sending all three messages:

```bash
sqlite3 /tmp/moly-v2.db << 'SQL'
.mode column
.headers on

SELECT 
  conversation_id, 
  phase, 
  ROUND(maturity, 2) as maturity,
  datetime(updated_at/1000, 'unixepoch') as updated_at
FROM conversation_execution_state 
WHERE conversation_id='test_phase5_001'
ORDER BY updated_at ASC;
SQL
```

**Expected output:**
```
conversation_id | phase     | maturity | updated_at
test_phase5_001 | initial   | 0.2      | 2026-10-03 ...
test_phase5_001 | gathering | 0.45     | 2026-10-03 ...
test_phase5_001 | analysis  | 0.65     | 2026-10-03 ...
```

**Critical check:** No "initial→gathering→initial" pattern (that would indicate reset bug)

---

## Test 3: Response Metadata Validation

For each message (M1, M2, M3), verify the response contains complete metadata:

```bash
# Example: Check M2 response for required fields
echo "$M2" | jq '.metadata.phase' 2>/dev/null
```

**Expected structure:**
```json
{
  "phase": {
    "current": "gathering",
    "previous": "initial",
    "maturity": 0.45,
    "transitioned": true,
    "accomplishments": {
      "completed": 2,
      "total": 4,
      "maturity": 0.5
    }
  }
}
```

**Verification checklist:**
- [ ] `current` field present (string: initial/gathering/analysis/help)
- [ ] `previous` field present (string)
- [ ] `maturity` field present (number 0.0-1.0)
- [ ] `transitioned` field present (boolean)
- [ ] `accomplishments` object present
- [ ] `accomplishments.completed` is integer
- [ ] `accomplishments.total` is integer
- [ ] `accomplishments.maturity` is number
- [ ] No null or undefined values

---

## Success Criteria

### ✅ Test 1 PASS:
- M1 phase = "initial"
- M2 phase = "gathering" (NOT reset!)
- M3 phase = "analysis"
- Maturity increases M1→M2→M3

### ✅ Test 2 PASS:
- Database has 3 rows
- Phases progress: initial→gathering→analysis
- Maturity values increasing
- No phase resets

### ✅ Test 3 PASS:
- All phase fields present
- All field types correct
- Values match database state
- No missing or null fields

---

## Troubleshooting

### Issue: Server won't start

**Symptom:** "address already in use"

```bash
# Kill existing process
pkill -f "bin/moly"
sleep 2

# Try again
./bin/moly
```

**Symptom:** "connection refused" to Ollama

```bash
# Check if Ollama is running
curl http://127.0.0.1:11434/api/tags

# If fails, start Ollama
ollama serve
```

### Issue: Authentication fails

**Symptom:** Signup returns 409 or login fails

```bash
# Try different email
TEST_EMAIL="phase5_user_$(date +%s)@test.example.com"

# Re-run signup
SIGNUP=$(curl -s -X POST http://localhost:8080/api/v1/auth/signup ...)
```

### Issue: Phase resets to initial on M2

**Symptom:** M2 returns phase="initial" instead of "gathering"

**This indicates the Phase 4 bug fix didn't apply properly.**

```bash
# Verify fix is in place
grep -n "currentPhase already set from maturityCalc" moly-go/main.go

# If not found, rebuild
cd moly-go
go build -o ../bin/moly .
```

### Issue: Metadata missing from response

**Symptom:** Response has no metadata.phase field

```bash
# Check if response has metadata at all
echo "$M1" | jq '.metadata' 

# If null or empty, check main.go lines 3223-3237
grep -A 10 "Response metadata" moly-go/main.go
```

---

## Running the Full Test Suite

Create a script to automate all tests:

```bash
#!/bin/bash

# phase5_full_test.sh

set -e

echo "🧪 PHASE 5 FULL TEST SUITE"
echo "=========================="

# Setup
TEST_EMAIL="phase5_test_$(date +%s)@example.com"
TEST_PASSWORD="Phase5TestPassword123!"
CONV_ID="test_phase5_001"

# Create user
echo "1️⃣  Creating test user..."
SIGNUP=$(curl -s -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")

SESSION_TOKEN=$(echo "$SIGNUP" | jq -r '.session_id // empty')
if [ -z "$SESSION_TOKEN" ]; then
  echo "❌ Failed to create user"
  exit 1
fi
echo "✅ User created"

# Send M1
echo "2️⃣  Sending M1 (initial)..."
M1=$(curl -s -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"Help with my relationship\",\"conversationId\":\"$CONV_ID\"}")

M1_PHASE=$(echo "$M1" | jq -r '.metadata.phase.current // "ERROR"')
M1_MATURITY=$(echo "$M1" | jq -r '.metadata.phase.maturity // 0')
echo "  Phase: $M1_PHASE (expected: initial)"
echo "  Maturity: $M1_MATURITY (expected: ~0.2)"

# Send M2
echo "3️⃣  Sending M2 (gathering)..."
M2=$(curl -s -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"We disagree often\",\"conversationId\":\"$CONV_ID\"}")

M2_PHASE=$(echo "$M2" | jq -r '.metadata.phase.current // "ERROR"')
M2_MATURITY=$(echo "$M2" | jq -r '.metadata.phase.maturity // 0')
echo "  Phase: $M2_PHASE (expected: gathering - NOT initial!)"
echo "  Maturity: $M2_MATURITY (expected: ~0.45)"

# Send M3
echo "4️⃣  Sending M3 (analysis)..."
M3=$(curl -s -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"Should I escalate?\",\"conversationId\":\"$CONV_ID\"}")

M3_PHASE=$(echo "$M3" | jq -r '.metadata.phase.current // "ERROR"')
M3_MATURITY=$(echo "$M3" | jq -r '.metadata.phase.maturity // 0')
echo "  Phase: $M3_PHASE (expected: analysis)"
echo "  Maturity: $M3_MATURITY (expected: ~0.65)"

# Results
echo ""
echo "📊 TEST RESULTS"
echo "=============="
if [ "$M1_PHASE" = "initial" ] && [ "$M2_PHASE" = "gathering" ] && [ "$M3_PHASE" = "analysis" ]; then
  echo "✅ TEST 1 PASS: Phase progression correct"
else
  echo "❌ TEST 1 FAIL: Phase progression incorrect"
fi

# Database check
echo ""
echo "Checking database..."
sqlite3 /tmp/moly-v2.db "SELECT COUNT(*) FROM conversation_execution_state WHERE conversation_id='$CONV_ID';" | grep -q "3" && echo "✅ TEST 2 PASS: Database has 3 records" || echo "❌ TEST 2 FAIL: Database check failed"

echo ""
echo "🎉 Phase 5 Test Suite Complete"
```

Save as `phase5_full_test.sh` and run:

```bash
chmod +x phase5_full_test.sh
./phase5_full_test.sh
```

---

## Next Steps After Phase 5

### If all tests pass:
- Document results in SESSION-24-PHASE-5-RESULTS.md
- Proceed to Phase 6 (Extended testing & optimization)

### If tests fail:
- Identify which test failed
- Check troubleshooting section above
- Refer to code sections provided
- Fix and re-run

---

**Ready to Execute Phase 5**

Follow the steps above in order. Start with "Test 1: Phase Progression" and work through each step carefully.

**Estimated total time:** 45-60 minutes

Good luck! 🚀
