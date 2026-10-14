#!/bin/bash
# Builds the beta extension (fixed extension ID) and makes the folder the browser loads an exact copy of it, so that files of
# older builds (the chunk files get a new name at every build) do not pile up. Then rebuilds the normal dist/.
# Usage: scripts/refresh-beta.sh [folder]   (default ~/moly-beta-extension). Reload the extension in the browser afterwards.
set -eu
ROOT=$(cd "$(dirname "$0")/.." && pwd)
TARGET=${1:-$HOME/moly-beta-extension}

# The folder is emptied of anything that is not in the new build, so it must be a folder of a Moly build (or new/empty).
if [ -e "$TARGET" ] && [ -n "$(ls -A "$TARGET" 2>/dev/null)" ] && ! grep -q '"manifest_version"' "$TARGET/manifest.json" 2>/dev/null; then
  echo "refusing: $TARGET is not empty and holds no extension manifest" >&2
  exit 1
fi

cd "$ROOT/moly-extension"
MOLY_BETA=1 npm run build >/dev/null
mkdir -p "$TARGET"
rsync -a --delete dist/ "$TARGET"/
npm run build >/dev/null
echo "beta extension copied to $TARGET ($(ls "$TARGET" | wc -l) entries); reload it in the browser"
