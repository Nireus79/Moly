#!/bin/bash
# Inside the Firefox e2e container (see run-chat-firefox.sh): installs the package, registers the app for the Firefox add-on
# and has one real conversation in Firefox.
set -u
export CHAT_WAIT_MS MOLY_DB_KEY=$(printf 'ab%.0s' $(seq 1 32)) EXT_ID OLLAMA_URL CHAT_MESSAGE BROWSER=firefox MOZ_DISABLE_CONTENT_SANDBOX=1 MOZ_DISABLE_GMP_SANDBOX=1
unset LLM_PROVIDER OLLAMA_ENDPOINT MOLY_EXTENSION_IDS
mkdir /tmp/pkg && tar -C /tmp/pkg -xzf "$(ls -t /dist/moly-*-linux-*.tar.gz | head -1)"
/tmp/pkg/moly-*/install.sh > /tmp/install.out 2>&1 || { cat /tmp/install.out; exit 1; }
M=$HOME/.local/share/moly/moly
$M --register-host --extension-ids "$EXT_ID"
cat ~/.mozilla/native-messaging-hosts/com.moly.backend_host.json
xvfb-run -a node /home/tester/e2e/e2e.mjs "${SCENARIO:-chat}"; RC=$?
echo "--- backend log (pipeline, last lines):"
grep -vE "GetConnection|AboutMe" ~/.moly/moly.log 2>/dev/null | tail -40 | cut -c1-200
echo "--- backend log (problems only):"
grep -iE "panic|fatal|error|attach|shutting" ~/.moly/moly.log 2>/dev/null | grep -v "ServerConfig" | tail -15 | cut -c1-220
$M --stop >/dev/null
exit $RC
