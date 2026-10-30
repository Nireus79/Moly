#!/usr/bin/env bash
# Writes the product version from installer/VERSION (MAJOR.MINOR.PATCH) into the extension manifest, the extension package, and the Android
# build file. versionCode = MAJOR*10000 + MINOR*100 + PATCH, so it only rises with the version. Run scripts/check-version.sh to verify.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
V=$(tr -d ' \n' < "$ROOT/installer/VERSION")
[[ "$V" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]] || { echo "installer/VERSION must be MAJOR.MINOR.PATCH, found '$V'" >&2; exit 1; }
CODE=$(( BASH_REMATCH[1] * 10000 + BASH_REMATCH[2] * 100 + BASH_REMATCH[3] ))
sed -i -E "s/^(  \"version\": \")[^\"]*(\",)/\1$V\2/" "$ROOT/moly-extension/manifest.json" "$ROOT/moly-extension/package.json"
sed -i -E "s/^( *versionCode )[0-9]+/\1$CODE/; s/^( *versionName \")[^\"]*(\")/\1$V\2/" "$ROOT/moly-mobile/android/app/build.gradle"
echo "version $V, versionCode $CODE"
