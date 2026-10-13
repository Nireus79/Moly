#!/bin/sh
# Builds dist/moly-<version>-linux-<arch>.tar.gz from the source tree. Run from anywhere.
# For a release, build on the oldest Linux you want to support: the program needs at least the C library it was built with.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
VERSION=$(tr -d ' \n' < "$ROOT/installer/VERSION")
ARCH=$(go env GOARCH)
NAME="moly-$VERSION-linux-$ARCH"
OUT="$ROOT/dist"
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

mkdir -p "$STAGE/$NAME" "$OUT"
( cd "$ROOT/moly-go" && CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$STAGE/$NAME/moly" . )
cp "$ROOT/installer/unix/install.sh" "$ROOT/installer/unix/uninstall.sh" "$STAGE/$NAME/"
cp "$ROOT/installer/extension-ids.txt" "$ROOT/installer/VERSION" "$ROOT/LICENSE" "$STAGE/$NAME/"
cat > "$STAGE/$NAME/README.txt" <<TXT
Moly $VERSION for Linux (beta)

1. Extract this folder, open a terminal in it.
2. Run:   ./install.sh
3. Open your browser (Chrome, Chromium, Edge or Brave) with the Moly extension installed.

Moly keeps its database key in your Secret Service keychain (GNOME Keyring, KWallet, KeePassXC). Without one, Moly asks on first start
how to protect the key: a passphrase (recommended) or, knowingly, a key file.
Remove with: ~/.local/share/moly/uninstall.sh   (add --purge to delete your data too)
TXT
chmod 0755 "$STAGE/$NAME/install.sh" "$STAGE/$NAME/uninstall.sh" "$STAGE/$NAME/moly"
tar -C "$STAGE" -czf "$OUT/$NAME.tar.gz" "$NAME"
( cd "$OUT" && sha256sum "$NAME.tar.gz" > "$NAME.tar.gz.sha256" )
echo "$OUT/$NAME.tar.gz"
cat "$OUT/$NAME.tar.gz.sha256"
