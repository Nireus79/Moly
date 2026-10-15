#!/bin/bash
# Real-browser test of the whole chain on Linux: Chromium + the beta extension build + native messaging + the installed
# program. Needs docker and network (first run builds an image with Chromium). Takes a few minutes.
set -u
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
docker image inspect moly-e2e >/dev/null 2>&1 || docker build -q -t moly-e2e "$ROOT/installer/e2e" >/dev/null || { echo "image build failed"; exit 1; }

echo "== build the beta extension and the Linux package"
( cd "$ROOT/moly-extension" && MOLY_BETA=1 npm run build >/dev/null 2>&1 ) || { echo "extension build failed"; exit 1; }
cp -r "$ROOT/moly-extension/dist" "$WORK/ext"
( cd "$ROOT/moly-extension" && npm run build >/dev/null 2>&1 )   # leave the normal build in dist/
"$ROOT/installer/linux/build.sh" >/dev/null || { echo "package build failed"; exit 1; }
EXT_ID=$(grep -v '^[[:space:]]*#' "$ROOT/installer/extension-ids.txt" | head -1 | tr -d ' \n')

docker run --rm --init \
  -v "$WORK/ext":/ext:ro -v "$ROOT/dist":/dist:ro \
  -v "$ROOT/installer/e2e/e2e.mjs":/home/tester/e2e/e2e.mjs:ro -v "$ROOT/installer/e2e/fake-vendors.mjs":/home/tester/e2e/fake-vendors.mjs:ro -v "$ROOT/installer/e2e/inside.sh":/inside.sh:ro \
  -e EXT_ID="$EXT_ID" --entrypoint bash moly-e2e /inside.sh
