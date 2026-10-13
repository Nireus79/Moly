#!/bin/sh
# Removes Moly for the current user. Your data (~/.moly: conversations, contacts, settings) is kept unless you ask:
#   uninstall.sh [--purge]    --purge also deletes the data folder, including the encrypted database.
set -eu

case "$(uname -s)" in
  Darwin) DEFAULT_PREFIX="$HOME/Library/Application Support/moly-program" ;;
  *)      DEFAULT_PREFIX="$HOME/.local/share/moly" ;;
esac
PREFIX="${MOLY_PREFIX:-$DEFAULT_PREFIX}"
PURGE=0
for a in "$@"; do
  case "$a" in
    --purge) PURGE=1 ;;
    -h|--help) sed -n '2,3p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown option: $a" >&2; exit 2 ;;
  esac
done

case "$PREFIX" in ""|"/"|"$HOME"|"$HOME/") echo "Refusing to use '$PREFIX'." >&2; exit 1 ;; esac
case "$(basename "$PREFIX")" in moly|moly-*) ;; *) echo "The install folder name must start with 'moly' (got '$PREFIX')." >&2; exit 1 ;; esac

if [ -x "$PREFIX/moly" ]; then
  "$PREFIX/moly" --stop || true
  "$PREFIX/moly" --unregister-host || true
fi

[ "$(readlink "$HOME/.local/bin/moly" 2>/dev/null || true)" = "$PREFIX/moly" ] && rm -f "$HOME/.local/bin/moly"
rm -rf "$PREFIX"

if [ "$PURGE" -eq 1 ]; then
  rm -rf "$HOME/.moly" "${XDG_CONFIG_HOME:-$HOME/.config}/moly" "$HOME/Library/Application Support/Moly"
  echo "Moly and all its data were removed."
  echo "The database key may remain in your keychain (entry 'moly'); remove it in your keychain manager if you wish."
else
  echo "Moly was removed. Your data is still in $HOME/.moly (run with --purge to delete it)."
fi
