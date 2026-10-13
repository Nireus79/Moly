#!/bin/bash
# Tests the Windows installer on Linux with Docker and Wine: cross-builds moly.exe, compiles the setup with Inno Setup,
# installs it silently in a Wine prefix, starts the backend the way a browser does, uninstalls it.
# Wine is not Windows: what this proves is the installer logic and the program's Windows build, not the Windows
# keychain, SmartScreen or the browsers. Needs: docker, network for the first run (images, Go modules).
set -u
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
DIST="$ROOT/dist"
VERSION=$(tr -d ' \n' < "$ROOT/installer/VERSION")
ID=abcdefghijklmnopabcdefghijklmnop
mkdir -p "$DIST"

echo "== build moly.exe (mingw-w64 in a Go container)"
docker run --rm -v "$ROOT":/src -v moly-gocache:/root/.cache -v moly-gomod:/go/pkg/mod -e HOSTUID="$(id -u)" -e HOSTGID="$(id -g)" -w /src/moly-go golang:1.26 sh -c '
  set -e
  apt-get update -qq >/dev/null && apt-get install -y -qq gcc-mingw-w64-x86-64 >/dev/null
  V=$(tr -d " \n" < /src/installer/VERSION)
  CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$V" -o /src/dist/moly.exe . 2>/dev/null
  cd /src/installer/windows/probe && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -buildvcs=false -o /src/dist/probe.exe .
  chown $HOSTUID:$HOSTGID /src/dist/moly.exe /src/dist/probe.exe' || { echo "FAIL  build"; exit 1; }

echo "== compile the installer (Inno Setup 6 under Wine)"
docker run --rm -v "$ROOT":/work -w /work/installer/windows amake/innosetup "/DAppVersion=$VERSION" "/DExtensionIds=$ID" '/DSourceExe=..\..\dist\moly.exe' moly.iss > "$DIST/iscc.log" 2>&1 \
  && echo "PASS  setup compiled" || { echo "FAIL  setup did not compile"; tail -20 "$DIST/iscc.log"; exit 1; }

echo "== install, run, uninstall in Wine (64-bit, image moly-wine64)"
docker image inspect moly-wine64 >/dev/null 2>&1 || docker build -q -t moly-wine64 "$ROOT/installer/windows/wine" >/dev/null
docker run --rm --init -v "$DIST":/dist:ro -v "$ROOT/installer/windows/wine-checks.sh":/wine-checks.sh:ro \
  -e ID=$ID -e VERSION="$VERSION" -e WINEDEBUG=-all -e MOLY_PORT=:11493 -e MOLY_DB_KEY="$(printf 'ab%.0s' $(seq 1 32))" -e OLLAMA_ENDPOINT=http://127.0.0.1:1 \
  --entrypoint xvfb-run moly-wine64 -a bash /wine-checks.sh
