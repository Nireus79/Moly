# Hand checks before publishing 0.1.2 (things only you can do; nothing here was run on a real Windows, Mac or phone)

Packages are in /home/nireus79/vs_projects/Moly/Moly-next/dist (checksums in SHA256SUMS-extra.txt). Test with a clean browser profile.

## Windows
1. Run moly-setup-0.1.2.exe. Expect "Windows protected your PC": More info, Run anyway. Finish.
2. Load the extension (unpacked build, dist/moly-extension-beta-0.1.2.zip unzipped). Expect: "Create your profile", then the recovery code, then Moly.
3. Close the browser, reopen: Moly asks for the password. A wrong one five times: a wait message.
4. Settings: Moly app shows 0.1.2; Change password works; Remove Moly lists the steps.
5. Uninstall in Settings > Installed apps. With the panel open it should turn into the install screen within seconds, with a button.

## Mac (.pkg, never run on a Mac)
Right-click moly-0.1.2-macos-universal.pkg, Open. Then as for Windows from step 2. Remove: sudo /usr/local/lib/moly/uninstall.sh.

## Linux .deb on your real machine
sudo apt install ./moly_0.1.2_amd64.deb (x86-64). Reload the extension, check as above. Remove: sudo apt remove moly. The data in ~/.moly stays.

## Chrome install button
Remove the app, open the panel: install screen. Press "Download latest version": the browser asks for the downloads permission once. This works only when the GitHub release exists (draft releases do not count: publish first). If refused, a plain link appears.

## Uninstall page
After the site is published, remove the extension: the browser should open https://nireus79.github.io/Moly/uninstall.html.

## Phone (APK)
Install the signed 0.1.2 APK over beta: it should open without a login and keep conversations. Check the daily card, the stop button, a long message.

## Not done by me
Publish site; GitHub secrets (MOBILE_KEYSTORE_*, CWS_*); first tag run; store submissions; Play data-safety form (draft in STORE_LISTING.md); OpenAI real run; merge, tag, publish.
