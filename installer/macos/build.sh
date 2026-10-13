#!/bin/sh
# Builds dist/moly-<version>-macos-universal.tar.gz. MUST run on a Mac (or a macOS CI runner): the database library needs
# the Apple C compiler. Not yet run on a Mac by the author: treat the first run as a test.
set -eu
[ "$(uname -s)" = "Darwin" ] || { echo "This script runs on macOS only." >&2; exit 1; }
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
VERSION=$(tr -d ' \n' < "$ROOT/installer/VERSION")
NAME="moly-$VERSION-macos-universal"
OUT="$ROOT/dist"
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$STAGE/$NAME" "$OUT"

for ARCH in arm64 amd64; do
  ( cd "$ROOT/moly-go" && CGO_ENABLED=1 GOOS=darwin GOARCH=$ARCH \
      CGO_CFLAGS="-arch $( [ $ARCH = amd64 ] && echo x86_64 || echo arm64 ) -mmacosx-version-min=11.0" \
      CGO_LDFLAGS="-arch $( [ $ARCH = amd64 ] && echo x86_64 || echo arm64 )" \
      go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$STAGE/moly-$ARCH" . )
done
lipo -create -output "$STAGE/$NAME/moly" "$STAGE/moly-arm64" "$STAGE/moly-amd64"

cp "$ROOT/installer/unix/install.sh" "$ROOT/installer/unix/uninstall.sh" "$STAGE/$NAME/"
cp "$ROOT/installer/extension-ids.txt" "$ROOT/installer/VERSION" "$ROOT/LICENSE" "$STAGE/$NAME/"
cat > "$STAGE/$NAME/README.txt" <<TXT
Moly $VERSION for Mac (beta)

1. Unpack this folder and open Terminal in it.
2. Run:   ./install.sh
3. If macOS says it cannot verify the program, open System Settings > Privacy & Security and choose "Open Anyway" for Moly.
4. Open your browser (Chrome, Edge or Brave) with the Moly extension installed.

Remove with: "$HOME/Library/Application Support/moly-program/uninstall.sh"   (add --purge to delete your data too)
TXT
chmod 0755 "$STAGE/$NAME/install.sh" "$STAGE/$NAME/uninstall.sh" "$STAGE/$NAME/moly"
tar -C "$STAGE" -czf "$OUT/$NAME.tar.gz" "$NAME"
( cd "$OUT" && shasum -a 256 "$NAME.tar.gz" > "$NAME.tar.gz.sha256" )
echo "$OUT/$NAME.tar.gz"
