#!/usr/bin/env bash
# Upload the store zip to the Chrome Web Store as a draft. It never publishes: "Submit for review" stays a click in the dashboard.
# Needs CWS_CLIENT_ID, CWS_CLIENT_SECRET, CWS_REFRESH_TOKEN and CWS_EXTENSION_ID in the environment
# (or in ~/.config/moly/cws.env, which the script reads). Usage: scripts/cws_upload.sh [path/to/moly-extension-store-X.zip]
set -euo pipefail

# The values come from the environment, or from the private file the person made once (see below).
ENV_FILE=${MOLY_CWS_ENV:-$HOME/.config/moly/cws.env}
if [ -z "${CWS_REFRESH_TOKEN:-}" ] && [ -f "$ENV_FILE" ]; then . "$ENV_FILE"; fi

for v in CWS_CLIENT_ID CWS_CLIENT_SECRET CWS_REFRESH_TOKEN CWS_EXTENSION_ID; do
  [ -n "${!v:-}" ] || { echo "$v is not set." >&2; exit 1; }
done
command -v jq >/dev/null || { echo "jq is needed." >&2; exit 1; }

ROOT=$(cd "$(dirname "$0")/.." && pwd)
ZIP=${1:-$(ls -t "$ROOT"/dist/moly-extension-store-*.zip 2>/dev/null | head -1)}
[ -f "${ZIP:-}" ] || { echo "No store zip found. Build it first (dist/moly-extension-store-*.zip)." >&2; exit 1; }
echo "Uploading $ZIP"

TOKEN=$(curl -sS --fail-with-body https://oauth2.googleapis.com/token \
  -d client_id="$CWS_CLIENT_ID" -d client_secret="$CWS_CLIENT_SECRET" \
  -d refresh_token="$CWS_REFRESH_TOKEN" -d grant_type=refresh_token | jq -r .access_token)
[ -n "$TOKEN" ] && [ "$TOKEN" != null ] || { echo "No access token: check the refresh token and that the consent screen is published." >&2; exit 1; }

RESP=$(curl -sS --fail-with-body -X PUT \
  -H "Authorization: Bearer $TOKEN" -H "x-goog-api-version: 2" \
  -T "$ZIP" "https://www.googleapis.com/upload/chromewebstore/v1.1/items/$CWS_EXTENSION_ID")
echo "$RESP" | jq .
STATE=$(echo "$RESP" | jq -r .uploadState)
[ "$STATE" = SUCCESS ] || { echo "Upload state is $STATE, not SUCCESS." >&2; exit 1; }
echo "Uploaded as a draft. Open the dashboard and press Submit for review."
