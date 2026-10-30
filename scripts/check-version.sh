#!/usr/bin/env bash
# Fails when a place that carries the product version differs from installer/VERSION. With a tag name as argument (v0.1.2 or mobile-v0.1.2),
# also checks that the tag matches.
set -uo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
V=$(tr -d ' \n' < "$ROOT/installer/VERSION")
[[ "$V" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]] || { echo "installer/VERSION must be MAJOR.MINOR.PATCH, found '$V'" >&2; exit 1; }
CODE=$(( BASH_REMATCH[1] * 10000 + BASH_REMATCH[2] * 100 + BASH_REMATCH[3] ))
bad=0
check() { [ "$2" = "$3" ] || { echo "$1: found '$2', expected '$3'" >&2; bad=1; }; }
check "moly-extension/manifest.json" "$(sed -n -E 's/^  "version": "([^"]*)".*/\1/p' "$ROOT/moly-extension/manifest.json")" "$V"
check "moly-extension/package.json" "$(sed -n -E 's/^  "version": "([^"]*)".*/\1/p' "$ROOT/moly-extension/package.json")" "$V"
check "build.gradle versionName" "$(sed -n -E 's/^ *versionName "([^"]*)".*/\1/p' "$ROOT/moly-mobile/android/app/build.gradle")" "$V"
check "build.gradle versionCode" "$(sed -n -E 's/^ *versionCode ([0-9]+).*/\1/p' "$ROOT/moly-mobile/android/app/build.gradle")" "$CODE"
if [ -n "${1:-}" ]; then
  case "$1" in
    mobile-v*) check "tag" "${1#mobile-v}" "$V" ;;
    v*) check "tag" "${1#v}" "$V" ;;
    *) echo "tag '$1' is neither vX.Y.Z nor mobile-vX.Y.Z" >&2; bad=1 ;;
  esac
fi
[ "$bad" = 0 ] && echo "version $V, versionCode $CODE: all places agree"
exit $bad
