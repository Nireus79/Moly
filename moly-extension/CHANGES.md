# Extension changes

## Next version (0.1.2), not released

- **Card messages from the website.** The "Sponsored" card text comes from `site/cards.json` (fetched at most once a day, cached, validated, built-in card as fallback; on the phone too). Changing the card = edit `site/cards.json` and push; no store review. Code: `src/api/cardFeed.ts`, `src/components/SponsorCard.tsx`; commit `078ff31`.
- **Before submitting 0.1.2:**
  - push `078ff31` first, so `cards.json` exists when the extension asks for it;
  - raise the version in `manifest.json` and `package.json`;
  - build the zips (Chrome/Edge: `npm run build`; Firefox: `npm run build:firefox`; the source zip for Mozilla: `git archive` of `moly-extension`);
  - privacy: the privacy page already mentions the daily card request; the Chrome privacy tab and the reviewer notes need the sentence in `docs/STORE_LISTING.md` ("Added for 0.1.2");
  - release notes for the stores: "The card message can now be updated without a new version."

## 0.1.1 (submitted 2026-10-25)

- One install button, "Download latest version": Windows gets the newest installer, Linux and Mac a one-line command with Copy; the screen continues by itself when the app is ready.
- Instructions and a button for getting a Claude or OpenAI API key.
- Privacy: the one request to api.github.com (newest installer file name, Windows only, on button press) is disclosed.

## Notes for the next desktop release (no version bump now)

- `installer/unix/install.sh` (inside the Linux and Mac packages) now explains a root-owned install folder (left by the old `moly-install-linux.sh` run with sudo) and prints the `sudo chown` fix, instead of failing with `cp: Permission denied`. Commit `6fb1eea`. It reaches users only with the next desktop release. The one-line installer (`site/install.sh`) already has the same check and is live.
