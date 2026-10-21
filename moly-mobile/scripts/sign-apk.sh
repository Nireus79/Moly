#!/bin/bash
# Aligns and signs a release APK:  KEYSTORE=... KEY_ALIAS=... KEY_PASS_FILE=... scripts/sign-apk.sh in.apk out.apk
# The key and its password stay outside the repository; the same key must sign every later release.
set -euo pipefail
IN=${1:?unsigned apk}; OUT=${2:?output apk}
BT=$(ls -d "${ANDROID_HOME:?}"/build-tools/* | sort -V | tail -1)
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
"$BT/zipalign" -f -p 4 "$IN" "$TMP/aligned.apk"
"$BT/apksigner" sign --ks "${KEYSTORE:?}" --ks-key-alias "${KEY_ALIAS:?}" --ks-pass "file:${KEY_PASS_FILE:?}" --out "$OUT" "$TMP/aligned.apk"
"$BT/apksigner" verify --verbose "$OUT" | grep -E "^Verifies|v2 scheme|v3 scheme"
