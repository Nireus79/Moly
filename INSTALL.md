# Installing Moly

Moly has two parts: a **browser extension** (Chrome, Chromium, Edge or Brave) and a small **app** on your computer. The app keeps your data on your computer and starts and stops with your browser. You choose how it gets its AI: a model on your own computer (Ollama), or your own Claude or OpenAI key.

> **Status: beta.** Installers for Linux, Windows and Mac are written; none is published yet. Downloads will appear on the [releases page](https://github.com/Nireus79/Moly/releases/latest). Until then use "From source" below.

## The beta extension

While the extension is not in the Chrome Web Store, beta testers load it by hand: unzip `moly-extension-beta-<version>.zip` from the releases page, open `chrome://extensions` (or `edge://extensions`, `brave://extensions`), switch on Developer mode, choose "Load unpacked" and pick the unzipped folder. Its ID is always `fehjgleflcokbdifandadgmleomhjola`, which is what the installers expect.

## Linux

Moly keeps its database key in your Secret Service keychain (GNOME Keyring, KWallet or KeePassXC). On a desktop without one (a minimal window manager, a server), Moly asks on first start how to protect the key: **a passphrase (recommended)**, which you type each time Moly starts, or, knowingly, a key file. A forgotten passphrase cannot be recovered.

```bash
tar xzf moly-<version>-linux-amd64.tar.gz
cd moly-<version>-linux-amd64
./install.sh
```

The installer works for your user only (no root), puts the app in `~/.local/share/moly`, and registers it for Chrome, Chromium, Edge and Brave. Then open the browser with the Moly extension and, if it shows "Install the Moly app", press **Check again**.

Check an installation at any time: `~/.local/share/moly/moly --doctor`. It lists what works and what is missing, and contains no personal data, so you can send it with a question.

Remove Moly: `~/.local/share/moly/uninstall.sh` (your data in `~/.moly` stays) or `uninstall.sh --purge` (deletes the data too).

## Windows

Run `moly-setup-<version>.exe`. It installs for your user only (no administrator rights) in `%LOCALAPPDATA%\Moly` and connects Moly to Chrome, Chromium, Edge and Brave. The beta installer is not signed yet, so Windows may say "Windows protected your PC": choose "More info", then "Run anyway". Remove Moly in Settings, Apps; it asks whether to delete your data too (a silent uninstall keeps it).

## Mac (beta)

Unpack the archive, open Terminal in the folder and run `./install.sh`. If macOS says it cannot verify the program, open System Settings, Privacy & Security, and choose "Open Anyway" for Moly. Remove with `"$HOME/Library/Application Support/moly-program/uninstall.sh"`.

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
- **Claude or OpenAI**: paste your own API key. Moly stores it in its encrypted database, protected by a passphrase you choose (you enter the passphrase once each time Moly starts). The text of your messages is then sent to that provider. Create a separate key for Moly and set a spending limit at the provider.

More: [docs/PRIVACY.md](docs/PRIVACY.md).

## Troubleshooting

| You see | Do this |
|---|---|
| "Install the Moly app" | Run the installer, then press **Check again**. |
| "Moly does not recognise this extension" | The app was set up for another copy of the extension. Install the latest app. |
| "Moly could not start" | Run `moly --doctor`; read `~/.moly/moly.log`. |
| Doctor: "database key" problem | The key is kept in the keychain, which cannot be reached now: log in to your desktop session or unlock the keychain, then press **Check again** in Moly. If the key is gone, restore the keychain or a backup. |
