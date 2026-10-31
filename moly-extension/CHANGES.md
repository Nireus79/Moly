# Extension changes

## 0.1.2 (prepared 2026-10-31, not yet submitted)

- **Sign-in on a computer.** Moly always asks who is using it: the first start makes a profile (name and password, recovery code shown once), later starts ask for the password; sessions live in the browser's session area, shared by the side panel and Settings and emptied when the browser closes. Settings: make another profile, change password, sign out. The phone keeps its single profile.
- **Install and update button.** "Download latest version" downloads the installer of the newest release (Windows setup, Linux .deb, Mac .pkg) and opens it, with the new OPTIONAL permissions `downloads` and `downloads.open` asked for at the button press; without them an ordinary download link is used. Settings, "Moly app": installed version and "Update to x".
- **The app removed while Moly is open** turns the screen into the install screen by itself; every failure screen offers "Reinstall the Moly app".
- **Removal help.** Settings, "Remove Moly", and a page (`site/uninstall.html`) that the browser opens when the extension is removed (`runtime.setUninstallURL`, no permission).
- **Card messages from the website.** The "Sponsored" card text comes from `site/cards.json` (fetched at most once a day, cached, validated, built-in card as fallback; on the phone too).
- Cleanups: no message text in the browser console, unused code and tables removed.
- **Before submitting:** build the zips (`npm run build`, `npm run build:firefox`, and the source zip for Mozilla: `git archive` of `moly-extension`); the Chrome privacy tab and reviewer notes need the sentences in `docs/STORE_LISTING.md` ("Added for 0.1.2"); store release note: "Sign-in with a password on your computer, an install and update button, and the card message can be updated without a new version."

## 0.1.1 (submitted 2026-10-25)

- One install button, "Download latest version": Windows gets the newest installer, Linux and Mac a one-line command with Copy; the screen continues by itself when the app is ready.
- Instructions and a button for getting a Claude or OpenAI API key.
- Privacy: the one request to api.github.com (newest installer file name, Windows only, on button press) is disclosed.

## Notes for the next desktop release (no version bump now)

- `installer/unix/install.sh` (inside the Linux and Mac packages) now explains a root-owned install folder (left by the old `moly-install-linux.sh` run with sudo) and prints the `sudo chown` fix, instead of failing with `cp: Permission denied`. Commit `6fb1eea`. It reaches users only with the next desktop release. The one-line installer (`site/install.sh`) already has the same check and is live.
