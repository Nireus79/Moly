#!/bin/sh
# Moly one-line installer for Linux and macOS:
#   curl -fsSL https://nireus79.github.io/Moly/install.sh | sh
# It finds the newest desktop release, downloads the package for this computer, checks its checksum, and runs the
# installer inside it (for your user only, no root). Nothing is installed before the checksum matches.
set -eu

REPO="Nireus79/Moly"
fail() { echo "Moly install: $*" >&2; exit 1; }

[ "$(id -u)" -ne 0 ] || fail "do not run this as root; Moly installs for one user."
command -v curl >/dev/null 2>&1 || fail "curl is needed."
command -v tar >/dev/null 2>&1 || fail "tar is needed."

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64|Linux-amd64) KIND="linux-amd64" ;;
  Darwin-*) KIND="macos-universal" ;;
  Linux-*) fail "there is no Linux package for $(uname -m) yet (only x86_64). Build from source: https://github.com/$REPO/blob/master/INSTALL.md" ;;
  *) fail "this installer is for Linux and macOS. On Windows, download moly-setup from https://nireus79.github.io/Moly/download.html" ;;
esac

# An earlier install made with sudo leaves a folder only root can write to; say how to fix it before anything is downloaded.
case "$(uname -s)" in
  Darwin) CHECK_DIR="${MOLY_PREFIX:-$HOME/Library/Application Support/moly-program}" ;;
  *)      CHECK_DIR="${MOLY_PREFIX:-$HOME/.local/share/moly}" ;;
esac
if [ -e "$CHECK_DIR" ] && [ ! -w "$CHECK_DIR" ]; then
  fail "the folder '$CHECK_DIR' belongs to another user (an old install made with sudo?). Take it over with:
  sudo chown -R \"\$USER\" '$CHECK_DIR'
then run this command again."
fi

if command -v sha256sum >/dev/null 2>&1; then SUM="sha256sum"; else SUM="shasum -a 256"; fi

# Releases come newest first. A release holds the desktop packages and the Android APK; only the *-$KIND.tar.gz name is picked here.
echo "Looking for the newest Moly release..."
URL=$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=30" \
  | grep -o "\"browser_download_url\": *\"[^\"]*/download/v[^/\"]*/moly-[^/\"]*-$KIND\.tar\.gz\"" \
  | head -n 1 | sed 's/.*: *"//; s/"$//') || true
[ -n "$URL" ] || fail "could not find a Moly package for your system on https://github.com/$REPO/releases (rate limit or no network?)."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM
FILE=$(basename "$URL")
echo "Downloading $FILE"
curl -fSL --progress-bar -o "$TMP/$FILE" "$URL"
curl -fsSL -o "$TMP/$FILE.sha256" "$URL.sha256" || fail "could not download the checksum."

WANT=$(cut -d ' ' -f 1 "$TMP/$FILE.sha256")
GOT=$($SUM "$TMP/$FILE" | cut -d ' ' -f 1)
[ -n "$WANT" ] && [ "$WANT" = "$GOT" ] || fail "the download does not match its checksum; nothing was installed."

tar -xzf "$TMP/$FILE" -C "$TMP"
DIR=$(find "$TMP" -mindepth 1 -maxdepth 1 -type d | head -n 1)
[ -x "$DIR/install.sh" ] || fail "the package has no install.sh."
"$DIR/install.sh"

# Packages before the snap check (see installer/unix/install.sh) do not warn about snap browsers; do it here for them.
if [ "$(uname -s)" = "Linux" ] && ! grep -q "(snap-check)" "$DIR/install.sh"; then
  for b in brave:Brave chromium:Chromium; do
    if [ -d "$HOME/snap/${b%%:*}" ]; then
      echo
      echo "NOTE: ${b##*:} is installed as a snap here. A snap browser cannot start Moly: its sandbox blocks the program."
      if [ "${b%%:*}" = brave ]; then echo "      Install Brave from Brave's own apt repository (a .deb, not the snap), or use Chrome or Firefox."
      else echo "      Use Chrome or Firefox, or Chromium from a source that is not a snap."; fi
    fi
  done
fi
