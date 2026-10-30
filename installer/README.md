# Installers

| Folder | What | State |
|---|---|---|
| `unix/` | `install.sh`, `uninstall.sh`: shared by Linux and macOS (per user, no root) | Linux tested. macOS paths and the quarantine/ad-hoc-signing step are written but **never run on a Mac**. |
| `linux/` | `build.sh` (tarball + checksum), `test-install.sh` (end-to-end test in a throw-away home), `host_probe.py` (plays the browser) | Done and tested. |
| `macos/` | `build.sh`: universal (arm64 + x86_64) tarball | Written; must run on a Mac or a macOS CI runner; **untested**. |
| `e2e/` | `run.sh`: **real Chromium** + the beta extension + native messaging + the installed Linux program, in Docker; `e2e.mjs` drives it with puppeteer | Done and tested: 5 situations, 23 checks. |
| `windows/` | `moly.iss` (Inno Setup, per user), `build.ps1` (on Windows), `test-wine.sh` (Docker + Wine test on Linux), `probe/` (browser stand-in as a Windows program) | Compiles; see the progress notes in `docs/PRODUCTION_PLAN.md` for what the Wine test proved. Never run on real Windows. |

`VERSION` is the release version. `extension-ids.txt` lists the store IDs the installers register: **fill it before a release is built** (the Windows build refuses an empty list).

**Release builds must be made on the oldest system you want to support.** The Linux program needs at least the C library of the machine that built it (a build on Ubuntu 26.04 needs glibc 2.38 and will not start on Ubuntu 22.04). Use a GitHub Actions runner or a container with an old base image.

Run the matching test after any change to an installer or to the host, `--register-host`, `--unregister-host`, `--stop` or `--doctor`:
`installer/linux/test-install.sh` and `installer/windows/test-wine.sh`.

**CI:** `.github/workflows/release.yml` runs the tests on every push, builds the three installers on a `v*` tag and prepares a draft release. The Linux job runs on Ubuntu 22.04 on purpose. The Linux installer test was also run in an `ubuntu:22.04` container: the program then needs glibc 2.34 at most.

**Beta extension ID.** The store ID does not exist until the Chrome publisher account is verified, so beta builds use a key of our own: `MOLY_BETA=1 npm run build` in `moly-extension` puts `installer/beta-extension-key.pub` into the manifest as `key`, which gives the fixed ID `fehjgleflcokbdifandadgmleomhjola` (the one in `extension-ids.txt`). A beta tester loads the unpacked beta build at `chrome://extensions`. The private half of the key is in `private/` (git-ignored); it is only needed to pack a `.crx`, never to load the extension. **The Web Store package must be built without `MOLY_BETA`.** When the store IDs exist, add them to `extension-ids.txt` and release a new installer; the beta ID can stay.

## Compatibility of the extension and the app

The extension and the app talk by native messages. Their protocol number (`hostVersion` in `moly-go/host.go`, `HOST_PROTOCOL` in `moly-extension/src/api/hostErrors.ts` and `background.ts`) is separate from the product version. It rises only when one side stops understanding the other. The extension sends its number with every message; the app refuses a different one, and the extension refuses an app that reports a different one, with the screen "The Moly app and the extension do not match". An app or extension from before this check sends no number and is taken as protocol 1.
