#!/bin/sh
# Builds dist/moly-<version>-macos-universal.pkg from the program that build.sh put into the tar.gz (run build.sh first). A double-click opens
# the macOS installer; the program goes to /usr/local/lib/moly and the browser link is registered for every user (moly --register-host --system).
# The package is not signed with an Apple developer certificate, so the first time macOS asks to open it anyway (right-click, Open).
# MUST run on a Mac (pkgbuild). Not yet run on a Mac by the author: treat the first run as a test.
set -eu
[ "$(uname -s)" = "Darwin" ] || { echo "This script runs on macOS only." >&2; exit 1; }
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
VERSION=$(tr -d ' \n' < "$ROOT/installer/VERSION")
NAME="moly-$VERSION-macos-universal"
TAR="$ROOT/dist/$NAME.tar.gz"
[ -f "$TAR" ] || { echo "Run installer/macos/build.sh first ($TAR is missing)." >&2; exit 1; }
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$STAGE/unpack" "$STAGE/root/usr/local/lib/moly" "$STAGE/scripts"
tar -xzf "$TAR" -C "$STAGE/unpack"
SRC="$STAGE/unpack/$NAME"
cp "$SRC/moly" "$SRC/extension-ids.txt" "$SRC/VERSION" "$SRC/LICENSE" "$STAGE/root/usr/local/lib/moly/"

cat > "$STAGE/root/usr/local/lib/moly/uninstall.sh" <<'SH'
#!/bin/sh
# Removes the Moly app installed by the .pkg. Your conversations (~/.moly) stay; delete that folder yourself to remove them.
set -eu
[ "$(id -u)" -eq 0 ] || { echo "Run it with sudo." >&2; exit 1; }
/usr/local/lib/moly/moly --unregister-host --system || true
rm -rf /usr/local/lib/moly
pkgutil --forget io.github.nireus79.moly >/dev/null 2>&1 || true
echo "Moly removed."
SH
chmod 0755 "$STAGE/root/usr/local/lib/moly/uninstall.sh" "$STAGE/root/usr/local/lib/moly/moly"

cat > "$STAGE/scripts/postinstall" <<'SH'
#!/bin/sh
# Runs as root after the files are in place.
set -e
D=/usr/local/lib/moly
xattr -dr com.apple.quarantine "$D" 2>/dev/null || true
# Apple Silicon only runs signed code; an ad-hoc signature (no developer account) is enough for a program the user installed.
codesign --force --sign - "$D/moly" >/dev/null 2>&1 || true
"$D/moly" --register-host --system || echo "Moly: the browser link could not be registered; run $D/moly --doctor" >&2
exit 0
SH
chmod 0755 "$STAGE/scripts/postinstall"

PKG="$ROOT/dist/$NAME.pkg"
pkgbuild --root "$STAGE/root" --scripts "$STAGE/scripts" --identifier io.github.nireus79.moly --version "$VERSION" --install-location / "$PKG"
( cd "$ROOT/dist" && shasum -a 256 "$NAME.pkg" > "$NAME.pkg.sha256" )
echo "$PKG"
