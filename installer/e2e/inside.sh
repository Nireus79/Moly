#!/bin/bash
# Runs inside the container (see run.sh): installs the Linux package, then drives Chromium through five situations.
set -u
export MOLY_DB_KEY=$(printf 'ab%.0s' $(seq 1 32)) OLLAMA_ENDPOINT=http://127.0.0.1:1 EXT_ID
unset LLM_PROVIDER MOLY_EXTENSION_IDS
FAILS=0
run() { echo "== $1"; node /home/tester/e2e/e2e.mjs "$2" || FAILS=$((FAILS+1)); }
M=$HOME/.local/share/moly/moly

mkdir /tmp/pkg && tar -C /tmp/pkg -xzf /dist/moly-*-linux-*.tar.gz
/tmp/pkg/moly-*/install.sh > /tmp/install.out 2>&1 || { cat /tmp/install.out; exit 1; }

$M --unregister-host >/dev/null
run "no Moly app registered for the browser" missing

$M --register-host --extension-ids ponmlkjihgfedcbaponmlkjihgfedcba >/dev/null
run "app registered for a different extension" notallowed

$M --register-host --extension-ids "$EXT_ID" >/dev/null
$M --stop >/dev/null
run "first run: the app starts by itself, provider setup with a passphrase" firstrun

$M --stop >/dev/null; sleep 1
run "after a restart: the passphrase is asked for" unlock

# This container has no keychain. Without MOLY_DB_KEY the app must ask how to protect the database key.
$M --stop >/dev/null; sleep 1
rm -rf ~/.moly/moly.db* ~/.moly/keystore.json ~/.moly/db.key
unset MOLY_DB_KEY
run "no keychain: the person is asked how to protect the key" keystore-setup

$M --stop >/dev/null; sleep 1
run "no keychain, after a restart: the passphrase is asked for" keystore-unlock

$M --stop >/dev/null
echo; [ $FAILS -eq 0 ] && echo "REAL BROWSER CHECKS PASSED" || echo "$FAILS SCENARIO(S) FAILED"
[ $FAILS -eq 0 ] || { echo "--- moly.log"; tail -20 ~/.moly/moly.log 2>/dev/null | cut -c1-200; }
exit $FAILS
