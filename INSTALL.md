# Installing Moly

Moly has two parts: a **browser extension** (Chrome, Chromium, Edge, Brave or Firefox) and a small **app** on your computer. The app keeps your data on your computer and starts and stops with your browser. You choose how it gets its AI: a model on your own computer (Ollama), or your own Claude or OpenAI key.

> **Status: beta.** Installers for Linux, Windows and Mac are on the [releases page](https://github.com/Nireus79/Moly/releases/latest). They are not code-signed yet. The extension is under review in the Chrome Web Store, Microsoft Edge Add-ons and Firefox Add-ons; until it is listed, use "The beta extension" below or "From source".

## The beta extension

While the extension is not in the Chrome Web Store, beta testers load it by hand: unzip `moly-extension-beta-<version>.zip` from the releases page, open `chrome://extensions` (or `edge://extensions`, `brave://extensions`), switch on Developer mode, choose "Load unpacked" and pick the unzipped folder. Its ID is always `fehjgleflcokbdifandadgmleomhjola`, which is what the installers expect.

## Firefox

Install the Moly add-on from Firefox Add-ons (link here when listed), then install the app as below. Open the sidebar with the Moly button or **View > Sidebar > Moly** (press Alt to show the menu bar). Firefox puts the sidebar on the left; to move it, use **Settings > General > Browser Layout > Show sidebar on the right**. If Firefox is installed as a snap or Flatpak, it may show a system dialog the first time Moly starts the app: allow it. Tested on Linux; the Windows and Mac installers register Firefox too, but have had less testing with it.

## Linux

Moly keeps its database key in your Secret Service keychain (GNOME Keyring, KWallet or KeePassXC). On a desktop without one (a minimal window manager, a server), Moly asks on first start how to protect the key: **a passphrase (recommended)**, which you type each time Moly starts, or, knowingly, a key file. A forgotten passphrase cannot be recovered.

```bash
tar xzf moly-<version>-linux-amd64.tar.gz
cd moly-<version>-linux-amd64
./install.sh
```

The installer works for your user only (no root), puts the app in `~/.local/share/moly`, and registers it for Chrome, Chromium, Edge, Brave and Firefox. Then open the browser with the Moly extension and, if it shows "Install the Moly app", press **Check again**.

Check an installation at any time: `~/.local/share/moly/moly --doctor`. It lists what works and what is missing, and contains no personal data, so you can send it with a question.

Remove Moly: `~/.local/share/moly/uninstall.sh` (your data in `~/.moly` stays) or `uninstall.sh --purge` (deletes the data too).

## Windows

Run `moly-setup-<version>.exe`. It installs for your user only (no administrator rights) in `%LOCALAPPDATA%\Moly` and connects Moly to Chrome, Chromium, Edge, Brave and Firefox. The beta installer is not signed yet, so Windows may say "Windows protected your PC": choose "More info", then "Run anyway". Remove Moly in Settings, Apps; it asks whether to delete your data too (a silent uninstall keeps it).

## Mac (beta)

Unpack the archive, open Terminal in the folder and run `./install.sh`. If macOS says it cannot verify the program, open System Settings, Privacy & Security, and choose "Open Anyway" for Moly. Remove with `"$HOME/Library/Application Support/moly-program/uninstall.sh"`.

## Android (beta)

Moly for Android is the same Moly in an app: no browser, no computer needed. It keeps your conversations on the phone, and it needs **your own Claude or OpenAI key** (there is no Ollama on the phone). It is free and has no ads.

1. On the phone, open the [releases page](https://github.com/Nireus79/Moly/releases) and download `moly-<version>.apk` from the latest release marked "Moly for Android". (Check it against `SHA256SUMS` if you like.)
2. Open the downloaded file. Android asks once to allow installs from your browser or file manager: allow it for that app only. Moly needs Android 7 or newer and a 64-bit phone (almost every phone made after 2017).
3. Open Moly, choose Claude or OpenAI, paste your key (the **Paste key** button reads your clipboard only when you tap it), tick the box and press **Save**. Create a separate key for Moly and set a spending limit at the provider.
4. Write to Moly, or select text in another app and use **Share, Moly**. A reply can take a minute or more; you can leave the app, and the reply is waiting when you come back.

Good to know:
- **Updates** are new APK files: install the new one over the old one. Your data stays.
- **Uninstalling deletes your conversations.** Use Settings, **Save a copy** first if you may want them back; after a reinstall enter your key first, then **Restore from a copy**.
- Some phones (Xiaomi, Huawei, Samsung and others) stop apps in the background to save battery. If a reply never arrives after you left the app, open the phone's battery settings and allow Moly to run in the background, and use Settings, **Report a problem** to tell us what happened.
- Moly for Android is a beta: it has been tried on a few phones only. Problems and questions: [Hermes_creative@proton.me](mailto:Hermes_creative@proton.me) or the issues page.

## From source (all systems, for developers)

```bash
git clone https://github.com/Nireus79/Moly.git
cd Moly/moly-go && go build -o ../bin/moly .       # needs Go and a C compiler (SQLCipher)
cd ../moly-extension && npm install && npm run build
```

Load `moly-extension/dist` as an unpacked extension at `chrome://extensions` (Developer mode), copy the extension's ID, then register the app for it:

```bash
bin/moly --register-host --extension-ids <that ID>
```

Open the Moly side panel: the app starts by itself. `bin/moly --unregister-host` undoes the registration.

**Development mode without the browser starting anything:** run `bin/moly` by hand (port 11436, open to any local program, no token) and load the extension; it falls back to that backend when no app is registered. Set `LLM_PROVIDER=ollama` (or `claude` with `ANTHROPIC_API_KEY`) if you want the environment to decide the AI provider.

## The AI provider

On first use Moly asks where its thinking should happen:

- **Ollama** (on your computer): install [Ollama](https://ollama.com), download a model (`ollama pull mistral`), and choose it in Moly.
- **Claude or OpenAI**: paste your own API key. Moly has been tested with Claude; **OpenAI support is experimental in this beta** (Moly uses the models your key lists and tells you in Settings when it has not been tested with them). Moly stores it in its encrypted database, protected by a passphrase you choose (you enter the passphrase once each time Moly starts). The text of your messages is then sent to that provider. Create a separate key for Moly and set a spending limit at the provider.

More: [docs/PRIVACY.md](docs/PRIVACY.md).

## Troubleshooting

| You see | Do this |
|---|---|
| "Install the Moly app" | Run the installer, then press **Check again**. |
| "Moly does not recognise this extension" | The app was set up for another copy of the extension. Install the latest app. |
| "Moly could not start" | Run `moly --doctor`; read `~/.moly/moly.log`. |
| Doctor: "database key" problem | The key is kept in the keychain, which cannot be reached now: log in to your desktop session or unlock the keychain, then press **Check again** in Moly. If the key is gone, restore the keychain or a backup. |
