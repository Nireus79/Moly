#!/bin/bash
# Like run-chat.sh, but in Firefox with the Firefox build of the extension. Needs docker and a running Ollama with a model.
set -u
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OLLAMA=${OLLAMA:-http://127.0.0.1:11434}
curl -s -m 3 "$OLLAMA/api/tags" | grep -q '"models":\[{' || { echo "No model found at $OLLAMA: start Ollama and pull a model (ollama pull mistral)."; exit 1; }
docker image inspect moly-e2e >/dev/null 2>&1 || docker build -q -t moly-e2e "$ROOT/installer/e2e" >/dev/null || { echo "image build failed"; exit 1; }
docker build -q -t moly-e2e-firefox -f "$ROOT/installer/e2e/Dockerfile.firefox" "$ROOT/installer/e2e" >/dev/null || { echo "firefox image build failed"; exit 1; }
( cd "$ROOT/moly-extension" && npm run build:firefox >/dev/null 2>&1 ) || { echo "extension build failed"; exit 1; }
"$ROOT/installer/linux/build.sh" >/dev/null || { echo "package build failed"; exit 1; }
EXT_ID=moly@nireus79.github.io
docker run --rm --init --network host \
  -v "$ROOT/moly-extension/dist-firefox":/ext:ro -v "$ROOT/dist":/dist:ro \
  -v "$ROOT/installer/e2e/e2e.mjs":/home/tester/e2e/e2e.mjs:ro -v "$ROOT/installer/e2e/chat-inside-firefox.sh":/chat-inside.sh:ro \
  -e EXT_ID="$EXT_ID" -e OLLAMA_URL="$OLLAMA" -e CHAT_WAIT_MS="${CHAT_WAIT_MS:-}" -e SCENARIO="${SCENARIO:-chat}" \
  -e CHAT_MESSAGE="I need to write to my landlord. The heating in my flat has been broken for two weeks and I am angry." \
  --entrypoint bash moly-e2e-firefox /chat-inside.sh
