#!/bin/sh
# Moly installer for Linux and macOS. Installs for the current user only (no root needed).
#   ./install.sh [--extension-ids ID[,ID...]] [--prefix DIR] [--no-symlink]
# The extension IDs default to extension-ids.txt next to this script, or MOLY_EXTENSION_IDS.
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
OS=$(uname -s)
case "$OS" in
  Linux)  DEFAULT_PREFIX="$HOME/.local/share/moly" ;;
  Darwin) DEFAULT_PREFIX="$HOME/Library/Application Support/moly-program" ;;
  *) echo "Moly install: this installer is for Linux and macOS." >&2; exit 1 ;;
esac
PREFIX="${MOLY_PREFIX:-$DEFAULT_PREFIX}"
IDS="${MOLY_EXTENSION_IDS:-}"
SYMLINK=1

while [ $# -gt 0 ]; do
  case "$1" in
    --extension-ids) IDS="$2"; shift 2 ;;
    --prefix) PREFIX="$2"; shift 2 ;;
    --no-symlink) SYMLINK=0; shift ;;
    -h|--help) sed -n '2,5p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

fail() { echo "Moly install: $*" >&2; exit 1; }

[ "$(id -u)" -ne 0 ] || fail "do not run it as root; Moly installs for one user."
[ -n "${HOME:-}" ] && [ -d "$HOME" ] || fail "HOME is not set."
[ -x "$HERE/moly" ] || fail "the moly program is missing next to this script."

if [ -z "$IDS" ] && [ -f "$HERE/extension-ids.txt" ]; then
  IDS=$(grep -v '^[[:space:]]*#' "$HERE/extension-ids.txt" | tr -s ' \t\n' ',' | sed 's/^,*//; s/,*$//')
fi
[ -n "$IDS" ] || fail "no extension ID known. Pass --extension-ids <id> (copy it from chrome://extensions)."

# The program must run on this machine before anything is changed.
"$HERE/moly" --version >/dev/null 2>&1 || fail "the program does not run on this system (wrong architecture or too old a C library?)."

# Safety: only ever replace a folder whose name starts with moly (on macOS a case-insensitive disk makes 'moly' and the config folder 'Moly' the same folder, so the program lives in moly-program).
case "$PREFIX" in
  ""|"/"|"$HOME"|"$HOME/") fail "refusing to use '$PREFIX' as the install folder." ;;
esac
case "$(basename "$PREFIX")" in moly|moly-*) ;; *) fail "the install folder name must start with 'moly' (got '$PREFIX')." ;; esac

# Stop a running Moly backend of this installation (asks it politely; the program hides from process lists on purpose).
if [ -x "$PREFIX/moly" ]; then
  "$PREFIX/moly" --stop >/dev/null 2>&1 || true
fi

if [ -e "$PREFIX" ] && [ ! -w "$PREFIX" ]; then
  fail "the folder '$PREFIX' belongs to another user (an old install made with sudo?). Take it over with: sudo chown -R \"\$USER\" '$PREFIX'  and run the installer again."
fi
mkdir -p "$PREFIX"
# Replace the program atomically, so a running browser never sees half a file.
cp "$HERE/moly" "$PREFIX/moly.new"
chmod 0755 "$PREFIX/moly.new"
mv -f "$PREFIX/moly.new" "$PREFIX/moly"
[ -f "$HERE/uninstall.sh" ] && cp "$HERE/uninstall.sh" "$PREFIX/uninstall.sh" && chmod 0755 "$PREFIX/uninstall.sh"
[ -f "$HERE/LICENSE" ] && cp "$HERE/LICENSE" "$PREFIX/LICENSE"
[ -f "$HERE/VERSION" ] && cp "$HERE/VERSION" "$PREFIX/VERSION"

if [ "$OS" = "Darwin" ]; then
  # A file the user unpacked from a download carries a quarantine mark; this installer was started by the user on purpose.
  xattr -dr com.apple.quarantine "$PREFIX" 2>/dev/null || true
  # Apple Silicon only runs signed code; an ad-hoc signature (no developer account) is enough for a program the user installed.
  codesign --force --sign - "$PREFIX/moly" >/dev/null 2>&1 || true
fi

if [ "$SYMLINK" -eq 1 ]; then
  mkdir -p "$HOME/.local/bin"
  ln -sf "$PREFIX/moly" "$HOME/.local/bin/moly"
fi

"$PREFIX/moly" --register-host --extension-ids "$IDS" --path "$PREFIX/moly"

echo
echo "Checking the installation:"
"$PREFIX/moly" --doctor || echo "(see the lines marked 'problem' above)"

# Snap-packaged browsers cannot start Moly: their sandbox has no access to hidden folders (~/.local) or the real data folder.
# (snap-check) The registration the installer wrote for them is never used, so say so instead of reporting success.
if [ "$(uname -s)" = "Linux" ]; then
  for b in brave:Brave chromium:Chromium; do
    if [ -d "$HOME/snap/${b%%:*}" ]; then
      echo
      echo "NOTE: ${b##*:} is installed as a snap here. A snap browser cannot start Moly: its sandbox blocks the program."
      if [ "${b%%:*}" = brave ]; then echo "      Install Brave from Brave's own apt repository (a .deb, not the snap), or use Chrome or Firefox."
      else echo "      Use Chrome or Firefox, or Chromium from a source that is not a snap."; fi
    fi
  done
fi

echo
echo "Moly is installed in $PREFIX"
echo "Open your browser and press 'Check again' in Moly (or just open Moly)."
echo "To remove it later: $PREFIX/uninstall.sh"
