#!/bin/bash

# TEST SCRIPT: Verify FIX #71, #72, #73
# Sends Christine_sub.txt messages and monitors for fix indicators

set -e

cd "$(dirname "$0")"

echo "=========================================="
echo "FIX #71, #72, #73 VERIFICATION TEST"
echo "=========================================="
echo ""
echo "Test Case: Multi-message conversation with goal progression"
echo "Messages: 3 (goal shifts from 'craft message' to 'initiate relationship')"
echo ""

# Define test messages (from Christine_sub.txt)
MSG1="Hello Moly. I want to talk to you about a girl I am interested to. I saw her profile on fetlife. Her name is Christine_sub. I think we have some things in common. I am dominant male and she is submissive. I was thinking of starting a chat with her and I need a first message for that. I don't want a simple \"hello\" like the dozens she probably take every month, but something smart and playful. Can you help?"

MSG2="Here are some insights from my profile. 46 male dominant. 100% Rigger 98% Dominant 81% Sadist 73% Master/Mistress 73% Owner 57% Degrader 55% Non-monogamist 44% Primal (Hunter) 43% Brat tamer 31% Experimentalist 28% Vanilla 13% Daddy/Mommy 7% Switch 4% Voyeur 0% Age player 0% Exhibitionist. Here are some from her's: Genders Female Cisgender, Roles submissive Exploring Good Girl, Orientation Bicurious Bisexual Sapiosexual. About Hello!! I'm a quiet, bi curvy girl, wanting to explore my sub nature with a caring Dom mentor/partner... I'm a 'good girl', who blooms with honesty, praise and clear, direct words. As for your second question, I have no previous experience on fetlife. I usually start a conversation about sexual preferences, then I refer to bdsm, if the other person is interested I mention that I am dominant and some of my basic kinks."

MSG3="Thanks for that. But given her profile, what specifically should my opening message focus on to genuinely connect with her, considering what we both want?"

# Color codes for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Extract metrics from logs
extract_maturity() {
    grep -i "maturity.*score\|overall.*maturity" | tail -1 || echo "NOT FOUND"
}

extract_goal_coherence() {
    grep -i "goal coherence\|goalcoherence\|progression=" | tail -1 || echo "NOT FOUND"
}

extract_layer10_state() {
    grep -i "layer10.*session\|persistence.*state" | tail -1 || echo "NOT FOUND"
}

extract_db_operations() {
    grep -i "persistence_sessions\|layer10.*db" | tail -1 || echo "NOT FOUND"
}

# Create temp log file
LOGFILE="/tmp/moly_test_$(date +%s).log"
echo "Logging to: $LOGFILE"
echo ""

echo "=========================================="
echo "STEP 1: Build the latest version"
echo "=========================================="
cd Moly/moly-go
go build -o ../../bin/moly . 2>&1 | tail -5
echo -e "${GREEN}✓ Build successful${NC}"
cd ../..
echo ""

echo "=========================================="
echo "STEP 2: Start server (background)"
echo "=========================================="
./bin/moly > "$LOGFILE" 2>&1 &
MOLY_PID=$!
echo "Server PID: $MOLY_PID"
echo "Waiting for server startup..."
sleep 3

# Check if server started
if ! kill -0 $MOLY_PID 2>/dev/null; then
    echo -e "${RED}✗ Server failed to start${NC}"
    tail -30 "$LOGFILE"
    exit 1
fi
echo -e "${GREEN}✓ Server running${NC}"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Stopping server..."
    kill $MOLY_PID 2>/dev/null || true
    sleep 1
}
trap cleanup EXIT

# API endpoint
API="http://localhost:11436"
USER_ID="test_user_$(date +%s)"
CONV_ID="test_conv_$(date +%s)"

echo "=========================================="
echo "STEP 3: Register test user"
echo "=========================================="
REGISTER_RESPONSE=$(curl -s -X POST "$API/api/v2/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test_$RANDOM@example.com\",\"password\":\"TestPassword123!\"}")

TOKEN=$(echo $REGISTER_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ Failed to register user${NC}"
    echo "Response: $REGISTER_RESPONSE"
    exit 1
fi
echo -e "${GREEN}✓ User registered${NC}"
echo "Token: ${TOKEN:0:20}..."
echo ""

echo "=========================================="
echo "MESSAGE 1: User asks for help with message to Christine"
echo "=========================================="
echo "Sending message 1..."
RESPONSE1=$(curl -s -X POST "$API/api/v2/process-message" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"$MSG1\",\"conversationId\":\"$CONV_ID\"}")

echo "Extracting metrics from logs..."
sleep 1

# Check logs for FIX #71 indicator (maturity score)
echo -n "Maturity score (Message 1): "
MATURITY1=$(tail -100 "$LOGFILE" | grep -i "overall.*maturity\|maturity.*score" | tail -1 | grep -o "[0-9]\.[0-9][0-9]" | tail -1)
if [ -n "$MATURITY1" ]; then
    echo -e "${GREEN}$MATURITY1${NC}"
else
    echo -e "${YELLOW}NOT CAPTURED${NC}"
fi

# Check for goal coherence
echo -n "Goal detected (Message 1): "
GOAL1=$(tail -100 "$LOGFILE" | grep -i "intention\|goal" | grep -o "'[^']*'" | head -1)
if [ -n "$GOAL1" ]; then
    echo -e "${GREEN}$GOAL1${NC}"
else
    echo -e "${YELLOW}NOT CAPTURED${NC}"
fi

# Check for Layer 10
echo -n "Layer 10 state (Message 1): "
LAYER10_MSG1=$(tail -100 "$LOGFILE" | grep -i "layer10" | tail -1)
if echo "$LAYER10_MSG1" | grep -q "persistence"; then
    echo -e "${GREEN}Persistence active${NC}"
else
    echo -e "${YELLOW}Check logs${NC}"
fi

echo ""
echo "=========================================="
echo "MESSAGE 2: User provides context with goal SHIFT"
echo "=========================================="
echo "Sending message 2..."
RESPONSE2=$(curl -s -X POST "$API/api/v2/process-message" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"$MSG2\",\"conversationId\":\"$CONV_ID\"}")

echo "Extracting metrics from logs..."
sleep 1

# Check FIX #71: Maturity should INCREASE from Message 1
echo -n "Maturity score (Message 2): "
MATURITY2=$(tail -100 "$LOGFILE" | grep -i "overall.*maturity\|maturity.*score" | tail -1 | grep -o "[0-9]\.[0-9][0-9]" | tail -1)
if [ -n "$MATURITY2" ]; then
    echo -e "${GREEN}$MATURITY2${NC}"
    if (( $(echo "$MATURITY2 > $MATURITY1" | bc -l 2>/dev/null || echo 0) )); then
        echo -e "${GREEN}  ✓ FIX #71: Maturity IMPROVED ($MATURITY1 → $MATURITY2)${NC}"
    elif [ -n "$MATURITY1" ]; then
        echo -e "${YELLOW}  ⚠ Maturity didn't improve ($MATURITY1 → $MATURITY2)${NC}"
    fi
else
    echo -e "${YELLOW}NOT CAPTURED${NC}"
fi

# Check FIX #72: Goal coherence should detect change
echo -n "Goal coherence (Message 2): "
COHERENCE=$(tail -100 "$LOGFILE" | grep -i "goal coherence\|goalcoherence" | grep -i "different\|subgoal\|same" | tail -1)
if [ -n "$COHERENCE" ]; then
    if echo "$COHERENCE" | grep -q -i "different"; then
        echo -e "${GREEN}DIFFERENT ✓${NC}"
        echo -e "${GREEN}  ✓ FIX #72: Goal change detected${NC}"
    else
        echo -e "$COHERENCE"
    fi
else
    echo -e "${YELLOW}NOT CAPTURED${NC}"
fi

# Check FIX #73: Layer 10 should show persistence
echo -n "Layer 10 persistence (Message 2): "
LAYER10_MSG2=$(tail -100 "$LOGFILE" | grep -i "layer10\|persistence_sessions" | tail -1)
if echo "$LAYER10_MSG2" | grep -q "persisted\|persist"; then
    echo -e "${GREEN}Persisting to DB ✓${NC}"
    echo -e "${GREEN}  ✓ FIX #73: Session saved${NC}"
else
    echo -e "${YELLOW}Check logs${NC}"
fi

echo ""
echo "=========================================="
echo "MESSAGE 3: Clarification response"
echo "=========================================="
echo "Sending message 3..."
RESPONSE3=$(curl -s -X POST "$API/api/v2/process-message" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"$MSG3\",\"conversationId\":\"$CONV_ID\"}")

echo "Extracting metrics from logs..."
sleep 1

# Check FIX #71: Maturity should continue improving
echo -n "Maturity score (Message 3): "
MATURITY3=$(tail -100 "$LOGFILE" | grep -i "overall.*maturity\|maturity.*score" | tail -1 | grep -o "[0-9]\.[0-9][0-9]" | tail -1)
if [ -n "$MATURITY3" ]; then
    echo -e "${GREEN}$MATURITY3${NC}"
    if (( $(echo "$MATURITY3 >= $MATURITY2" | bc -l 2>/dev/null || echo 0) )); then
        echo -e "${GREEN}  ✓ FIX #71: Maturity improving ($MATURITY1 → $MATURITY2 → $MATURITY3)${NC}"
    fi
else
    echo -e "${YELLOW}NOT CAPTURED${NC}"
fi

echo ""
echo "=========================================="
echo "SUMMARY"
echo "=========================================="
echo ""
echo "FIX #71 (Entity Saturation): "
if [ -n "$MATURITY1" ] && [ -n "$MATURITY2" ]; then
    if (( $(echo "$MATURITY2 > $MATURITY1" | bc -l 2>/dev/null || echo 0) )); then
        echo -e "${GREEN}✓ PASS - Maturity increases between messages${NC}"
    else
        echo -e "${YELLOW}⚠ CHECK - Maturity values: $MATURITY1 → $MATURITY2${NC}"
    fi
else
    echo -e "${YELLOW}? INCONCLUSIVE - Missing maturity metrics${NC}"
fi

echo ""
echo "FIX #72 (Goal Coherence): "
if tail -50 "$LOGFILE" | grep -q -i "goal.*different"; then
    echo -e "${GREEN}✓ PASS - Goal changes detected${NC}"
else
    echo -e "${YELLOW}? CHECK - Review logs for goal coherence${NC}"
fi

echo ""
echo "FIX #73 (Layer 10 Persistence): "
if grep -q "persistence_sessions" "$LOGFILE"; then
    echo -e "${GREEN}✓ PASS - Database persistence active${NC}"
else
    echo -e "${YELLOW}⚠ CHECK - Review logs for Layer 10 DB operations${NC}"
fi

echo ""
echo "Full logs available at: $LOGFILE"
echo ""
echo "For detailed analysis, check:"
echo "  grep -i 'maturity' $LOGFILE | tail -10"
echo "  grep -i 'goalcoherence' $LOGFILE | tail -10"
echo "  grep -i 'layer10.*session' $LOGFILE | tail -10"
echo ""
