#!/bin/bash

# Moly One-Click Startup - NOW STARTS BOTH GO BACKEND + CORS PROXY AUTOMATICALLY
# Services: Go Backend (11436) + CORS Proxy (11435) + Optional Ollama (11434)

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
GO_DIR="$SCRIPT_DIR/moly-go"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  MOLY STARTUP - One-Click Launch      ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""

# Cleanup any old processes
pkill -f "moly-go" 2>/dev/null || true
sleep 1

# Check if Go binary is built
if [ ! -f "$GO_DIR/moly" ]; then
    echo -e "${YELLOW}⏳ Go backend not built. Building...${NC}"
    cd "$GO_DIR"
    go build -o moly .
    cd - > /dev/null
    echo -e "${GREEN}✓ Go backend built${NC}"
fi

# Start Go Backend (which auto-starts CORS Proxy)
echo -e "${YELLOW}Starting Go Backend on 11436...${NC}"
cd "$GO_DIR"
./moly &
GO_PID=$!
sleep 2

if ps -p $GO_PID > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Go Backend started (PID $GO_PID)${NC}"
    echo -e "${GREEN}✓ CORS Proxy auto-started on 11435${NC}"
else
    echo -e "${RED}✗ Go Backend failed to start${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  MOLY READY FOR TESTING               ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo "Services running:"
echo -e "  ${GREEN}✓ Go Backend${NC}     http://127.0.0.1:11436"
echo -e "  ${GREEN}✓ CORS Proxy${NC}    http://127.0.0.1:11435"
echo "  ⚠ Ollama (11434)   - start manually if needed:"
echo "    ollama serve"
echo ""
echo "Next steps:"
echo "  1. Open chrome://extensions"
echo "  2. Load unpacked → moly-extension/dist/"
echo "  3. Click Moly icon to test"
echo "  4. Type a message"
echo "  5. See SafetyAlert + suggestions"
echo ""
echo "To stop services: Ctrl+C or pkill -f moly-go"
echo ""

wait
