#!/bin/bash
# Inside the e2e container: installs the package, opens the extension in Chromium and has one real conversation with a
# small model served by the "moly-ollama" container. Checks that the whole pipeline answers; the quality of the answer
# of a tiny model is not judged.
set -u
export MOLY_DB_KEY=$(printf 'ab%.0s' $(seq 1 32)) EXT_ID OLLAMA_URL CHAT_MESSAGE
unset LLM_PROVIDER OLLAMA_ENDPOINT MOLY_EXTENSION_IDS
mkdir /tmp/pkg && tar -C /tmp/pkg -xzf /dist/moly-*-linux-*.tar.gz
/tmp/pkg/moly-*/install.sh > /tmp/install.out 2>&1 || { cat /tmp/install.out; exit 1; }
node /home/tester/e2e/e2e.mjs chat; RC=$?
echo "--- backend log (problems only):"
grep -iE "panic|fatal|error" ~/.moly/moly.log 2>/dev/null | grep -v "ServerConfig" | tail -15 | cut -c1-220
grep -c "PANIC" ~/.moly/moly.log 2>/dev/null | sed 's/^/panics in the log: /'
~/.local/share/moly/moly --stop >/dev/null
exit $RC
