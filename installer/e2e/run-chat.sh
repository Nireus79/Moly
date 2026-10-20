#!/bin/bash
# One real conversation through the whole chain: Chromium + extension + native host + backend + YOUR Ollama (the one already
# running on this computer, reached through the host network). Needs docker and an Ollama with at least one model.
set -u
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OLLAMA=${OLLAMA:-http://127.0.0.1:11434}
curl -s -m 3 "$OLLAMA/api/tags" | grep -q '"models":\[{' || { echo "No model found at $OLLAMA: start Ollama and pull a model (ollama pull mistral)."; exit 1; }
WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
docker image inspect moly-e2e >/dev/null 2>&1 || docker build -q -t moly-e2e "$ROOT/installer/e2e" >/dev/null || { echo "image build failed"; exit 1; }

( cd "$ROOT/moly-extension" && MOLY_BETA=1 npm run build >/dev/null 2>&1 ) || { echo "extension build failed"; exit 1; }
cp -r "$ROOT/moly-extension/dist" "$WORK/ext"
( cd "$ROOT/moly-extension" && npm run build >/dev/null 2>&1 )
"$ROOT/installer/linux/build.sh" >/dev/null || { echo "package build failed"; exit 1; }
EXT_ID=$(grep -v '^[[:space:]]*#' "$ROOT/installer/extension-ids.txt" | head -1 | tr -d ' \n')

# --network host: the container sees this computer's 127.0.0.1, where Ollama listens.
docker run --rm --init --network host \
  -v "$WORK/ext":/ext:ro -v "$ROOT/dist":/dist:ro \
  -v "$ROOT/installer/e2e/e2e.mjs":/home/tester/e2e/e2e.mjs:ro -v "$ROOT/installer/e2e/chat-inside.sh":/chat-inside.sh:ro \
  -e EXT_ID="$EXT_ID" -e OLLAMA_URL="$OLLAMA" -e CHAT_WAIT_MS="${CHAT_WAIT_MS:-}" -e SCENARIO="${SCENARIO:-chat}" \
  -e CHAT_MESSAGE="I need to write to my landlord. The heating in my flat has been broken for two weeks and I am angry." \
  --entrypoint bash moly-e2e /chat-inside.sh
