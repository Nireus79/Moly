#!/bin/sh
# Builds the .deb and checks what can be checked without root: the package's files and permissions, that the program inside runs, and that
# its post-install step writes the browser manifests (into a staging root, MOLY_SYSROOT) with the extension IDs from the file next to it.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
"$ROOT/installer/linux/build-deb.sh" >/dev/null
DEB=$(ls -t "$ROOT"/dist/moly_*.deb | head -n 1)
T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
ok=0
check() { if sh -c "$2"; then echo "ok   $1"; else echo "FAIL $1"; ok=1; fi; }

dpkg-deb -x "$DEB" "$T/root"
check "the program is in /opt/moly and runs" "[ -x '$T/root/opt/moly/moly' ] && '$T/root/opt/moly/moly' --version >/dev/null"
check "no folder is writable by others" "[ -z \"\$(find '$T/root' -type d -perm /022)\" ]"
check "the extension IDs travel with the program" "grep -qv '^[[:space:]]*\\(#\\|\$\\)' '$T/root/opt/moly/extension-ids.txt'"
check "the post-install step is a script that registers for every user" "dpkg-deb -e '$DEB' '$T/ctl' && grep -q -- '--register-host --system' '$T/ctl/postinst'"
check "the removal step unregisters" "grep -q -- '--unregister-host --system' '$T/ctl/prerm'"
HOME="$T/home" MOLY_SYSROOT="$T/sys" "$T/root/opt/moly/moly" --register-host --system --path /opt/moly/moly >/dev/null
check "Chrome and Firefox manifests are written under the shared folders" "[ -f '$T/sys/etc/opt/chrome/native-messaging-hosts/com.moly.backend_host.json' ] && [ -f '$T/sys/usr/lib/mozilla/native-messaging-hosts/com.moly.backend_host.json' ]"
check "the manifest points at the installed program and allows the store IDs" "grep -q '\"path\": \"/opt/moly/moly\"' '$T/sys/etc/opt/chrome/native-messaging-hosts/com.moly.backend_host.json' && grep -q 'chrome-extension://' '$T/sys/etc/opt/chrome/native-messaging-hosts/com.moly.backend_host.json'"
check "no user data folder was created" "[ ! -e '$T/home/.moly' ]"
exit $ok
