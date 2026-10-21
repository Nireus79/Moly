# Moly for Android

The phone app: the same Moly screens (`../moly-extension`) and the same Go engine (`../moly-go`), packaged with
[Capacitor](https://capacitorjs.com). The engine runs on the phone as a child process of a foreground service, with its
database key wrapped by the Android Keystore. Moly has no server: messages go from the phone to the AI provider you chose,
with your own key. It never reads other apps or web pages; you paste or share text to it.

Two store builds come from one code base:

| Flavor | Application ID | For |
|---|---|---|
| `foss` | `io.github.nireus79.moly` | GitHub releases, IzzyOnDroid, F-Droid. No ad or billing code. |
| `play` | `io.github.nireus79.moly.play` | Google Play, later. The only build that may contain proprietary libraries. |

## Build

Needs Go, Node, JDK 17, the Android SDK (platform 34) and NDK 26 (`ANDROID_HOME` set).

```bash
cd moly-mobile
scripts/build-engine.sh                 # libmolyengine.so for arm64-v8a (ABIS="arm64-v8a x86_64" adds the emulator)
scripts/build-apk.sh foss debug         # screens + sync + gradle; prints the APK path
scripts/build-apk.sh foss release       # unsigned release APK
KEYSTORE=... KEY_ALIAS=... KEY_PASS_FILE=... scripts/sign-apk.sh in.apk out.apk
```

The signing key and its password live outside the repository. The same key must sign every release: Android refuses an
update signed by another key, and uninstalling first deletes the person's chats.

## Layout

- `android/` the Android project (committed; the engine and the copied screens are build output and ignored).
  `app/src/main/java/.../moly/`: `EngineService` (owns the engine process, logs its life), `KeyHelper` (Keystore-wrapped
  database key), `MolyEnginePlugin` (the bridge the screens call), `Report` (text of "Report a problem"), `MainActivity`
  (share receiver). `src/foss` and `src/play`: the flavor-specific `AdSlot`.
- `shim.js` replaces `chrome.storage` and `chrome.runtime` for the screens and sets `window.molyPhone`.
- `scripts/` the build recipes above.

## CPU types

Only 64-bit ARM (`arm64-v8a`) is shipped. `ABIS="... x86_64"` adds the emulator build. `armeabi-v7a` (32-bit phones) compiles but has
never been run anywhere, so it is left out until it has been tested on a real 32-bit phone.

## Old Android web views

Android 8 phones can have a web view from 2017 (Chrome 61). The screens are built for ES2015, `shim.js` carries a few
polyfills, and the CSS under `.ai-phone` and the phone cards avoids flex `gap`, `inset` and `aspect-ratio`.

## Testing

Run the extension tests (`cd ../moly-extension && npx vitest run`) and the Go tests (`cd ../moly-go && go test ./...`). The
phone app itself is tested by hand on an emulator (Android 8.1 and 14) and on real phones; the engine honours
`MOLY_ANTHROPIC_BASE_URL` so a stand-in model server can be used (a debug build reads `files/test-env.txt`, a release build never does).
