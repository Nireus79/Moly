#!/bin/bash
# End-to-end test of the Linux installer in a throw-away home folder: build, install, upgrade, refuse bad input, start the
# backend the way a browser does, uninstall (keep data), reinstall, uninstall --purge. Prints PASS or FAIL per check.
set -u
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
WORK=$(mktemp -d)
trap '"$FAKEHOME/.local/share/moly/moly" --stop >/dev/null 2>&1; rm -rf "$WORK"' EXIT
FAKEHOME="$WORK/home"; mkdir -p "$FAKEHOME"
FAILS=0
check() { if eval "$2" >/dev/null 2>&1; then echo "PASS  $1"; else echo "FAIL  $1"; FAILS=$((FAILS+1)); fi; }

echo "== build"
TARBALL=$("$ROOT/installer/linux/build.sh" | head -1) || { echo "FAIL  build"; exit 1; }
check "tarball and checksum exist" "[ -f '$TARBALL' ] && [ -f '$TARBALL.sha256' ] && (cd '$(dirname "$TARBALL")' && sha256sum -c '$(basename "$TARBALL").sha256')"
tar -C "$WORK" -xzf "$TARBALL"
PKG=$(ls -d "$WORK"/moly-*-linux-*)
check "package holds program, scripts, license, version" "[ -x '$PKG/moly' ] && [ -x '$PKG/install.sh' ] && [ -x '$PKG/uninstall.sh' ] && [ -f '$PKG/LICENSE' ] && [ -f '$PKG/VERSION' ]"

export HOME="$FAKEHOME" XDG_CONFIG_HOME="$FAKEHOME/.config"
export MOLY_DB_KEY=$(printf 'ab%.0s' $(seq 1 32)) MOLY_PORT=:11495 OLLAMA_ENDPOINT=http://127.0.0.1:1
unset MOLY_EXTENSION_IDS LLM_PROVIDER
ID=abcdefghijklmnopabcdefghijklmnop
PREFIX="$FAKEHOME/.local/share/moly"

echo "== bad input is refused and changes nothing"
# The package ships the beta extension ID; without it (and without --extension-ids) the installer must refuse.
mv "$PKG/extension-ids.txt" "$WORK/ids.bak"
check "no extension ID: refused with a clear message" "! '$PKG/install.sh' 2>&1 | tee '$WORK/o1' >/dev/null; grep -q 'no extension ID' '$WORK/o1'; [ ! -e '$PREFIX' ]"
mv "$WORK/ids.bak" "$PKG/extension-ids.txt"
check "install folder name not starting with moly: refused" "! '$PKG/install.sh' --extension-ids $ID --prefix '$WORK/other' 2>/dev/null; [ ! -e '$WORK/other' ]"
check "install folder = home: refused" "! '$PKG/install.sh' --extension-ids $ID --prefix '$FAKEHOME' 2>/dev/null"

echo "== install"
"$PKG/install.sh" --extension-ids $ID > "$WORK/install.out" 2>&1; INSTALL_RC=$?
check "install exits 0" "[ $INSTALL_RC -eq 0 ]"
VERSION=$(tr -d ' \n' < "$ROOT/installer/VERSION")
check "program installed and runs" "'$PREFIX/moly' --version | grep -q \"moly $VERSION\""
check "uninstall script installed" "[ -x '$PREFIX/uninstall.sh' ]"
check "symlink in ~/.local/bin" "[ \"\$(readlink '$FAKEHOME/.local/bin/moly')\" = '$PREFIX/moly' ]"
for b in google-chrome chromium microsoft-edge BraveSoftware/Brave-Browser; do
  M="$XDG_CONFIG_HOME/$b/NativeMessagingHosts/com.moly.backend_host.json"
  check "host manifest for $b" "grep -q '\"path\": \"$PREFIX/moly\"' '$M' && grep -q 'chrome-extension://$ID/' '$M' && ! grep -q allowed_extensions '$M'"
done
check "extension IDs recorded for the backend" "grep -qx $ID '$FAKEHOME/.moly/extension-ids.txt'"

echo "== upgrade over an existing install"
check "second install succeeds" "'$PKG/install.sh' --extension-ids $ID"

echo "== the browser starts the backend through the installed program"
python3 "$ROOT/installer/linux/host_probe.py" "$PREFIX/moly" "chrome-extension://$ID/" > "$WORK/probe.out" 2>&1; PROBE_RC=$?
check "host answers ping, start and status" "[ $PROBE_RC -eq 0 ] && grep -q '\"pong\": true' '$WORK/probe.out' && grep -q '\"running\": true' '$WORK/probe.out'"
check "the installed program reports a running backend" "'$PREFIX/moly' --doctor | grep -q 'answering on port 11495'"
check "backend listens on loopback only" "ss -ltn | grep ':11495 ' | grep -q '127.0.0.1:11495'"
check "data folder created with a private token" "[ \"\$(stat -c %a '$FAKEHOME/.moly/session.token')\" = 600 ]"
check "a different extension is refused by the host" "! python3 '$ROOT/installer/linux/host_probe.py' '$PREFIX/moly' chrome-extension://ponmlkjihgfedcbaponmlkjihgfedcba/"

echo "== uninstall keeps the data"
"$PREFIX/uninstall.sh" > "$WORK/un1.out" 2>&1
check "program folder gone" "[ ! -e '$PREFIX' ]"
check "symlink gone" "[ ! -e '$FAKEHOME/.local/bin/moly' ]"
check "host manifests gone" "! ls $XDG_CONFIG_HOME/*/NativeMessagingHosts/com.moly.backend_host.json $XDG_CONFIG_HOME/BraveSoftware/Brave-Browser/NativeMessagingHosts/com.moly.backend_host.json"
check "backend stopped (port free)" "! ss -ltn | grep -q ':11495 '"
check "user data kept" "[ -f '$FAKEHOME/.moly/moly.db' ]"

echo "== reinstall, then uninstall --purge"
"$PKG/install.sh" --extension-ids $ID >/dev/null 2>&1
"$PREFIX/uninstall.sh" --purge > "$WORK/un2.out" 2>&1
check "purge removes the data folder" "[ ! -e '$FAKEHOME/.moly' ]"
check "nothing left in the fake home but empty folders" "[ -z \"\$(find '$FAKEHOME' -type f)\" ]"
check "the browsers' folders we created are gone" "[ ! -e '$XDG_CONFIG_HOME/chromium' ] && [ ! -e '$XDG_CONFIG_HOME/BraveSoftware' ]"

echo
if [ $FAILS -eq 0 ]; then echo "ALL CHECKS PASSED"; else echo "$FAILS CHECK(S) FAILED"; echo "--- install output:"; cat "$WORK/install.out"; fi
exit $FAILS
