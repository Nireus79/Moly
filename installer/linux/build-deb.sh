#!/bin/sh
# Builds dist/moly_<version>_<arch>.deb: the same program as the tar.gz, installed for every user of the computer by a double-click in the
# software installer (or: sudo apt install ./moly_*.deb). The program goes to /opt/moly and the browser link is registered for Chrome,
# Chromium, Edge and Firefox in their shared folders (moly --register-host --system); data stays in each user's own ~/.moly.
# A copy installed per user by install.sh is found first by the browsers; remove it with ~/.local/share/moly/uninstall.sh.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
VERSION=$(tr -d ' \n' < "$ROOT/installer/VERSION")
DEBVERSION=$(printf '%s' "$VERSION" | sed 's/-/~/')   # 0.1.0-beta.4 sorts before 0.1.0 in dpkg as 0.1.0~beta.4
ARCH=$(go env GOARCH)
OUT="$ROOT/dist"
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
PKG="$STAGE/pkg"

mkdir -p "$PKG/DEBIAN" "$PKG/opt/moly" "$PKG/usr/share/doc/moly" "$OUT"
( cd "$ROOT/moly-go" && CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$PKG/opt/moly/moly" . )
cp "$ROOT/installer/extension-ids.txt" "$ROOT/installer/VERSION" "$PKG/opt/moly/"
cp "$ROOT/LICENSE" "$PKG/usr/share/doc/moly/copyright"
chmod 0644 "$PKG/usr/share/doc/moly/copyright"
find "$PKG" -type d -exec chmod 0755 {} +
chmod 0755 "$PKG/opt/moly/moly"
chmod 0644 "$PKG/opt/moly/extension-ids.txt" "$PKG/opt/moly/VERSION"

SIZE=$(du -sk "$PKG/opt" | cut -f1)
cat > "$PKG/DEBIAN/control" <<CTL
Package: moly
Version: $DEBVERSION
Architecture: $ARCH
Maintainer: Hermes Software <Hermes_creative@proton.me>
Installed-Size: $SIZE
Depends: libc6
Section: utils
Priority: optional
Homepage: https://nireus79.github.io/Moly/
Description: Moly, a communication coach (the app for the browser extension)
 Moly keeps your conversations on your computer, encrypted, and talks to the AI
 model you choose. This package is the small app that the Moly browser extension
 starts. Install the extension from your browser's store.
CTL
cat > "$PKG/DEBIAN/postinst" <<'SH'
#!/bin/sh
set -e
case "$1" in configure) /opt/moly/moly --register-host --system || echo "Moly: the browser link could not be registered; see /opt/moly/moly --doctor" >&2 ;; esac
exit 0
SH
cat > "$PKG/DEBIAN/prerm" <<'SH'
#!/bin/sh
set -e
case "$1" in remove|purge) /opt/moly/moly --unregister-host --system || true ;; esac
exit 0
SH
chmod 0755 "$PKG/DEBIAN/postinst" "$PKG/DEBIAN/prerm"

NAME="moly_${DEBVERSION}_${ARCH}.deb"
fakeroot dpkg-deb --build -Zxz "$PKG" "$OUT/$NAME" >/dev/null
( cd "$OUT" && sha256sum "$NAME" > "$NAME.sha256" )
echo "$OUT/$NAME"
