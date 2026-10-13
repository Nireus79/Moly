#!/bin/bash
# Runs inside the moly-wine64 container (Debian + 64-bit Wine, under xvfb). See test-wine.sh. Needs /dist (moly.exe, probe.exe, setup).
FAILS=0
check() { if eval "$2" >/dev/null 2>&1; then echo "PASS  $1"; else echo "FAIL  $1"; FAILS=$((FAILS+1)); fi; }
U=$(whoami)
APP=/home/$U/.wine/drive_c/users/$U/AppData/Local/Moly
DATA=/home/$U/.wine/drive_c/users/$U/.moly
WINAPP="C:\\users\\$U\\AppData\\Local\\Moly"
SETUP="/dist/moly-setup-$VERSION.exe"
regq() { wine reg query "$1" 2>/dev/null | tr -d '\r'; }

wine "$SETUP" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART '/LOG=C:\setup.log' >/dev/null 2>&1
check "installed program exists" "[ -f $APP/moly.exe ]"
check "license copied" "[ -f $APP/LICENSE.txt ]"
check "installed program runs and reports the version" "wine \"$WINAPP\\moly.exe\" --version | tr -d '\r' | grep -q 'moly $VERSION'"
for k in 'Google\Chrome' 'Chromium' 'Microsoft\Edge' 'BraveSoftware\Brave-Browser'; do
  check "registry key for $k" "regq 'HKCU\\Software\\$k\\NativeMessagingHosts\\com.moly.backend_host' | grep -q 'com.moly.backend_host.json'"
done
MANIFEST=$DATA/com.moly.backend_host.json
check "manifest names the installed program" "grep -F 'Local\\\\Moly\\\\moly.exe' $MANIFEST"
check "manifest allows the extension, with no Firefox key" "grep -q 'chrome-extension://$ID/' $MANIFEST && ! grep -q allowed_extensions $MANIFEST"
check "extension IDs recorded for the backend" "grep -qx $ID $DATA/extension-ids.txt"
check "uninstaller installed" "ls $APP/unins*.exe"
check "doctor reports everything in place" "wine \"$WINAPP\\moly.exe\" --doctor | tr -d '\r' | grep -q 'Everything needed is in place'"

wine /dist/probe.exe "$WINAPP\\moly.exe" "chrome-extension://$ID/" > /tmp/probe.out 2>/tmp/probe.err; PRC=$?
check "host answers ping, start and status (Windows build)" "[ $PRC -eq 0 ] && grep -q '\"pong\":true' /tmp/probe.out && grep -q '\"running\":true' /tmp/probe.out"
check "session token created" "[ -s $DATA/session.token ]"
wine /dist/probe.exe "$WINAPP\\moly.exe" "chrome-extension://ponmlkjihgfedcbaponmlkjihgfedcba/" >/dev/null 2>&1
check "a different extension is refused by the host" "[ $? -ne 0 ]"
check "upgrade over a running installation succeeds" "wine \"$SETUP\" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART >/dev/null 2>&1 && [ -f $APP/moly.exe ]"

UNINS=$(ls $APP/unins*.exe 2>/dev/null | head -1)
wine "$UNINS" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART >/dev/null 2>&1
sleep 2
check "program removed" "[ ! -f $APP/moly.exe ]"
check "registry keys removed" "! regq 'HKCU\\Software\\Google\\Chrome\\NativeMessagingHosts\\com.moly.backend_host' | grep -q com.moly && ! regq 'HKCU\\Software\\BraveSoftware\\Brave-Browser\\NativeMessagingHosts\\com.moly.backend_host' | grep -q com.moly"
check "a silent uninstall keeps the user's data" "[ -f $DATA/moly.db ]"
echo
if [ $FAILS -eq 0 ]; then echo "WINE CHECKS PASSED"; else echo "$FAILS WINE CHECK(S) FAILED"; echo "--- probe output:"; head -20 /tmp/probe.out /tmp/probe.err 2>/dev/null; fi
exit $FAILS
