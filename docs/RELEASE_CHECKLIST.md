# Release checklist 0.1.2 (started 2026-10-31; every item from "What else is missing", none dropped)

Status: [ ] open, [x] done and checked, [U] only the user can do it (instructions in "Hand checks").

## A. I do now
- [x] A1 Upgrade of old data: an old (beta.4) binary makes a local profile with conversations; the new binary lets the first profile take it over; nothing lost, password needed afterwards
- [x] A2 Change password in Settings (needs the old password) + lockout after repeated wrong passwords (backend + extension + tests)
- [x] A3 .deb installed with apt in a Debian container (postinst, manifests, remove)
- [x] A4 Workflows linted (actionlint) and the release logic read once more
- [x] A5 Sign-in screens checked in dark mode (screenshots)
- [x] A6 (code already hardened; real run stays with the tester) OpenAI: what the code does now that the old model is gone; reduce the risk, list what the tester must run
- [x] A7 (measured again: 0 of 60, not reproduced; fix stays unproven) Greek "name used as address": one more measurement
- [x] A8 Desktop release notes (the 0.1.2 notes are Android-only), CHANGES.md, NEXT_VERSION.md, STORE_LISTING.md
- [x] A9 (written; the Console wording stays with the user) Play data-safety "sharing" question: reasoned answer written down
- [x] A10 (rehearsed in a clone; plan in ../merge-prep/MERGE_PLAN.md) Merge preparation: compare the main tree's uncommitted PRIVACY.md / F-Droid / .gitignore edits with the branch (no merge, no push)
- [x] A11 (docs/HAND_CHECKS.md) Hand-check instructions file for the user (Windows, Mac, phone, Chrome button, uninstall page)
- [x] A12 (all tests, e2e, packages rebuilt; deb is container-built so test-deb.sh was not rerun on it, apt test passed on the same recipe) Final: all tests, e2e in Chromium, rebuild every package, checksums

## U. Only the user can do (listed again at the end)
- [U] Windows real machine; Mac (.pkg); phone (APK); .deb on the real machine
- [U] Publish the site (uninstall.html, download.html) and check the browser opens it after removing the extension
- [U] GitHub secrets (MOBILE_KEYSTORE_*, CWS_*); the first tag run in CI
- [U] Store submissions (Chrome, Firefox, Edge), Play data-safety form, F-Droid entry
- [U] OpenAI real-key run (the tester); manual safety runs on "go"
- [U] Rotate keys in ~/.bashrc; delete the claude-builders-bounty fork; the Google ticket; yank decision for core 0.1.4; card feed location; testers' Play opt-in link; WordPress text
- [U] Merge to master, tag, publish (on the user's go)

## Excluded on purpose
- P5 (interface in other languages), self-update through the app (step 3), signing certificates
