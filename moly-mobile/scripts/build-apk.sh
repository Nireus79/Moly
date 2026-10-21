#!/bin/bash
# Builds the APK:  scripts/build-apk.sh [foss|play] [debug|release]   (default: foss debug)
# Needs: JDK 17, ANDROID_HOME (SDK 34), Node. Run build-engine.sh first. The release APK is unsigned; scripts/sign-apk.sh signs it.
set -euo pipefail
cd "$(dirname "$0")/.."
FLAVOR=${1:-foss}; TYPE=${2:-debug}
[ -d node_modules ] || npm ci
scripts/build-www.sh
npx cap sync android >/dev/null
(cd android && ./gradlew --no-daemon -q "assemble$(tr '[:lower:]' '[:upper:]' <<<"${FLAVOR:0:1}")${FLAVOR:1}$(tr '[:lower:]' '[:upper:]' <<<"${TYPE:0:1}")${TYPE:1}")
ls -1 android/app/build/outputs/apk/"$FLAVOR"/"$TYPE"/*.apk
