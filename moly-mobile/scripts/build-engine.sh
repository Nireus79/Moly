#!/bin/bash
# Builds the Go engine (moly-go) as libmolyengine.so for each Android CPU type into the app's jniLibs.
# Needs: Go, and the Android NDK (ANDROID_NDK_HOME, or ANDROID_HOME/ndk/<version>).
# Default: arm64-v8a only (the phones). ABIS="arm64-v8a x86_64" adds the emulator build; armeabi-v7a (32-bit phones) builds but
# has never been run anywhere, so it is not shipped.
set -euo pipefail
cd "$(dirname "$0")/.."
NDK=${ANDROID_NDK_HOME:-$(ls -d "${ANDROID_HOME:?set ANDROID_HOME or ANDROID_NDK_HOME}"/ndk/* | sort -V | tail -1)}
BIN=$NDK/toolchains/llvm/prebuilt/linux-x86_64/bin
API=24
OUT=$PWD/android/app/src/main/jniLibs
ABIS=${ABIS:-"arm64-v8a"}
rm -rf "$OUT" # the APK carries exactly the engines built now

for abi in $ABIS; do
  case $abi in
    arm64-v8a)   export GOARCH=arm64 GOARM= CC=$BIN/aarch64-linux-android$API-clang ;;
    armeabi-v7a) export GOARCH=arm GOARM=7 CC=$BIN/armv7a-linux-androideabi$API-clang ;;
    x86_64)      export GOARCH=amd64 GOARM= CC=$BIN/x86_64-linux-android$API-clang ;;
    *) echo "unknown ABI $abi" >&2; exit 1 ;;
  esac
  mkdir -p "$OUT/$abi"
  echo "engine for $abi"
  (cd ../moly-go && GOOS=android CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o "$OUT/$abi/libmolyengine.so" .)
done
