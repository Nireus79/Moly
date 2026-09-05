#!/bin/bash

# Moly Complete Setup - Native Host + Extension Load
# One script does everything

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
GO_BINARY="$SCRIPT_DIR/moly-go/moly"
EXTENSION_PATH="$SCRIPT_DIR/moly-extension/dist"
HOST_MANIFEST="$SCRIPT_DIR/moly-go/com.moly.backend_host.json"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  MOLY INSTALLATION & SETUP            ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Step 0: Check for Chrome
echo -e "${YELLOW}Step 0: Checking for Chrome...${NC}"
CHROME_FOUND=false
if command -v google-chrome &> /dev/null; then
    CHROME_FOUND=true
    CHROME_CMD="google-chrome"
elif command -v chromium &> /dev/null; then
    CHROME_FOUND=true
    CHROME_CMD="chromium"
elif command -v chromium-browser &> /dev/null; then
    CHROME_FOUND=true
    CHROME_CMD="chromium-browser"
fi

if [ "$CHROME_FOUND" = false ]; then
    echo -e "${RED}✗ Chrome/Chromium not found${NC}"
    echo ""
    echo "Moly is a Chrome extension and requires:"
    echo "  • Chrome: google.com/chrome"
    echo "  • Chromium: chromium.woolyss.com"
    echo "  • Edge: microsoft.com/edge (Chromium-based)"
    echo ""
    echo "Install one of these browsers, then run this script again."
    exit 1
fi
echo -e "${GREEN}✓ Chrome/Chromium found${NC}"

# Step 1: Verify Go binary
echo -e "${YELLOW}Step 1: Checking Go backend...${NC}"
if [ ! -f "$GO_BINARY" ]; then
    echo -e "${YELLOW}Building Go backend...${NC}"
    cd "$SCRIPT_DIR/moly-go"
    go build -o moly .
    if [ $? -ne 0 ]; then
        echo -e "${RED}✗ Go build failed${NC}"
        exit 1
    fi
    cd - > /dev/null
fi
echo -e "${GREEN}✓ Go backend ready${NC}"

# Step 2: Verify extension
echo -e "${YELLOW}Step 2: Checking extension...${NC}"
if [ ! -f "$EXTENSION_PATH/manifest.json" ]; then
    echo -e "${RED}✗ Extension not built at $EXTENSION_PATH${NC}"
    echo "Build it: cd moly-extension && npm run build"
    exit 1
fi
echo -e "${GREEN}✓ Extension ready at $EXTENSION_PATH${NC}"

# Step 3: Set up native host
echo -e "${YELLOW}Step 3: Setting up native host...${NC}"

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    MANIFEST_DIR="$HOME/.config/google-chrome/NativeMessagingHosts"
    mkdir -p "$MANIFEST_DIR"
    sed "s|MOLY_BACKEND_PATH|$GO_BINARY|g" "$HOST_MANIFEST" > "$MANIFEST_DIR/com.moly.backend_host.json"
    echo -e "${GREEN}✓ Native host installed${NC}"
    
elif [[ "$OSTYPE" == "darwin"* ]]; then
    MANIFEST_DIR="$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts"
    mkdir -p "$MANIFEST_DIR"
    sed "s|MOLY_BACKEND_PATH|$GO_BINARY|g" "$HOST_MANIFEST" > "$MANIFEST_DIR/com.moly.backend_host.json"
    echo -e "${GREEN}✓ Native host installed${NC}"
else
    echo -e "${YELLOW}⚠ Unsupported OS for automatic setup${NC}"
fi

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  SETUP COMPLETE                       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo "Extension path: $EXTENSION_PATH"
echo ""
echo -e "${BLUE}NEXT: Load extension in Chrome${NC}"
echo ""
echo "1. Open Chrome and go to:"
echo "   ${BLUE}chrome://extensions${NC}"
echo ""
echo "2. Enable 'Developer mode' (top-right toggle)"
echo ""
echo "3. Click 'Load unpacked'"
echo ""
echo "4. Select this folder:"
echo "   ${BLUE}$EXTENSION_PATH${NC}"
echo ""
echo "5. Click Moly icon and start typing"
echo ""
echo -e "${GREEN}That's it! Backend will auto-start on first click.${NC}"
echo ""
echo "Opening chrome://extensions in your browser..."
sleep 1

# Try to open Chrome (detect which is available)
if command -v google-chrome &> /dev/null; then
    google-chrome "chrome://extensions" &
elif command -v chromium &> /dev/null; then
    chromium "chrome://extensions" &
elif command -v chromium-browser &> /dev/null; then
    chromium-browser "chrome://extensions" &
elif [[ "$OSTYPE" == "darwin"* ]]; then
    open -a "Google Chrome" "chrome://extensions"
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    start "C:\Program Files\Google\Chrome\Application\chrome.exe" "chrome://extensions"
else
    echo "Please manually open Chrome and go to chrome://extensions"
fi

echo ""
echo -e "${GREEN}Installation complete!${NC}"
