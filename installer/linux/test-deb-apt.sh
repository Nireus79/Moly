#!/bin/bash
# Installs the .deb with apt in a clean Debian container, checks what it did, removes it and checks that nothing is left.
# The package is built in a Go container of the same Debian release, so the C library matches. Needs docker and network.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OUT=$(mktemp -d); trap 'rm -rf "$OUT"' EXIT
docker run --rm -v "$ROOT":/src:ro -v "$OUT":/out golang:1.26 bash -c 'set -e; apt-get update -qq >/dev/null; apt-get install -y -qq fakeroot dpkg-dev >/dev/null; cp -r /src /work; rm -rf /work/dist /work/moly-extension/node_modules; cd /work && installer/linux/build-deb.sh >/dev/null && cp dist/moly_*.deb /out/' >/dev/null
mv "$OUT"/moly_*.deb "$OUT/moly.deb"
cat > "$OUT/check.sh" <<'CHECK'
set -u
ok() { echo "ok   $1"; }; bad() { echo "FAIL $1"; FAIL=1; }; FAIL=0
export DEBIAN_FRONTEND=noninteractive
apt-get install -y -qq /out/moly.deb >/tmp/apt.log 2>&1 && ok "apt installs the package" || { bad "apt install"; tail -5 /tmp/apt.log; }
dpkg -s moly | grep -q "Status: install ok installed" && ok "dpkg lists it as installed (version $(dpkg -s moly | sed -n "s/^Version: //p"))" || bad "dpkg status"
/opt/moly/moly --version | grep -q "moly 0.1.2" && ok "the program runs and reports 0.1.2" || bad "program version"
for f in /etc/opt/chrome/native-messaging-hosts /etc/chromium/native-messaging-hosts /etc/opt/edge/native-messaging-hosts /usr/lib/mozilla/native-messaging-hosts; do
  [ -f $f/com.moly.backend_host.json ] && ok "manifest in $f" || bad "manifest in $f"
done
grep -q '"path": "/opt/moly/moly"' /etc/opt/chrome/native-messaging-hosts/com.moly.backend_host.json && ok "manifest points at /opt/moly/moly" || bad "manifest path"
grep -q "chrome-extension://" /etc/opt/chrome/native-messaging-hosts/com.moly.backend_host.json && ok "manifest allows the store extension IDs" || bad "manifest origins"
useradd -m tester 2>/dev/null
DOC=$(su tester -c '/opt/moly/moly --doctor' 2>&1); echo "$DOC" | grep -E '^(ok|warning|problem)' | cut -c1-110
echo "$DOC" | grep -q '^problem' && bad 'the doctor reports a problem for a system install' || ok 'the doctor finds the system-wide registration (no problem)'
[ ! -e /home/tester/.moly/moly.db ] && ok "installing made no data in the user's folder" || bad "data folder"
apt-get remove -y -qq moly >/tmp/apt2.log 2>&1 && ok "apt remove works" || { bad "apt remove"; tail -5 /tmp/apt2.log; }
[ ! -e /opt/moly/moly ] && ok "the program is gone" || bad "program still there"
for f in /etc/opt/chrome/native-messaging-hosts /etc/chromium/native-messaging-hosts /etc/opt/edge/native-messaging-hosts /usr/lib/mozilla/native-messaging-hosts; do
  [ ! -f $f/com.moly.backend_host.json ] && ok "manifest removed from $f" || bad "manifest left in $f"
done
exit $FAIL
CHECK
docker run --rm -v "$OUT":/out debian:trixie-slim bash /out/check.sh
