# Moly Setup Guide

Complete step-by-step instructions for installing and using Moly.

---

## System Requirements

- **Chrome or Chromium browser** (version 90+)
- **Internet connection** (for downloading)
- **Administrator/sudo access** (for installation)

---

## Installation

Choose your operating system:

### Linux

**Step 1: Download the installer**

Open Terminal and run:
```bash
curl -L -o ~/Downloads/moly-install.sh \
  https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-install-linux.sh
```

Or manually download from:
https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-install-linux.sh

**Step 2: Make it executable**

```bash
chmod +x ~/Downloads/moly-install.sh
```

**Step 3: Run the installer**

```bash
~/Downloads/moly-install.sh
```

When prompted, enter your password and press Enter.

**Step 4: Wait for completion**

The installer will:
- Download the native host (~9 MB)
- Extract and install it
- Set up auto-start
- Show "Installation Complete!" message

---

### macOS

**Step 1: Download the installer**

Open Terminal and run:
```bash
curl -L -o ~/Downloads/moly-install.sh \
  https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-install-macos.sh
```

Or manually download from:
https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-install-macos.sh

**Step 2: Make it executable**

```bash
chmod +x ~/Downloads/moly-install.sh
```

**Step 3: Run the installer**

Option A - Terminal:
```bash
~/Downloads/moly-install.sh
```

Option B - Finder:
1. Open Finder → Downloads
2. Right-click `moly-install-macos.sh`
3. Select "Open With" → "Terminal"
4. Click "Open"

**Step 4: Enter password**

When prompted, enter your Mac password and press Enter.

**Step 5: Wait for completion**

The installer will:
- Detect your Mac architecture (Intel or Apple Silicon)
- Download the correct binary (~9 MB)
- Extract and install it
- Set up LaunchAgent for auto-start
- Show "Installation Complete!" message

---

### Windows

**Step 1: Download the installer**

Go to:
https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-install-windows.bat

Click the link and save to Downloads folder.

**Step 2: Run as administrator**

1. Open File Explorer (Windows key + E)
2. Go to Downloads folder
3. Right-click `moly-install-windows.bat`
4. Select "Run as administrator"
5. Click "Yes" if prompted by User Account Control

**Step 3: Wait for completion**

The installer will:
- Download the native host (~9 MB)
- Extract and install to Program Files
- Set up Task Scheduler for auto-start
- Show "Installation Complete!" message
- Close automatically

---

## Opening Terminal (if unfamiliar)

### Linux
Press `Ctrl+Alt+T` or search applications for "Terminal"

### macOS
Press `Cmd+Space`, type "terminal", press Enter

### Windows
Press `Windows+R`, type "cmd", press Enter (for Command Prompt)

---

## Next: Install Chrome Extension

After native host installation, you need to install the Moly extension.

**Step 1: Get the extension**

Option A - Chrome Web Store (coming soon):
- Will be available on Chrome Web Store

Option B - Load locally for testing:

1. Download Moly source: https://github.com/Nireus79/Moly
2. Extract the folder
3. Go to `chrome://extensions/`
4. Enable "Developer mode" (toggle in top right)
5. Click "Load unpacked"
6. Select the `moly-extension` folder
7. Click "Select Folder"

---

## First Use: Set Up Local Model

After installing both components:

**Step 1: Open Moly**
- Click Moly icon in Chrome toolbar
- Or go to chrome://extensions/ and click the extension

**Step 2: Open Settings**
- Click the gear/settings icon
- Select "Set Up Local Model"

**Step 3: Start Setup**
- Click "Configure Setup"
- Click "Start Setup"
- Extension will auto-detect the native host

**Step 4: Select a Model**
- Choose from recommended models:
  - Mistral 7B (recommended, fast)
  - Llama 2 7B (strong reasoning)
  - Neural Chat 7B (good balance)
- Or skip if you just want to use cloud (Claude/OpenAI)

**Step 5: Download and Extract Model**
- Model will download (this takes time - can be 5-30 minutes)
- Extension will show progress
- Model automatically extracts and loads

**Step 6: Start Coaching!**
- Open Moly sidebar
- Type a question or paste a message
- Get AI-powered suggestions

---

## Troubleshooting

### "Permission denied" error
```bash
chmod +x ~/Downloads/moly-install.sh
```
Then run again.

### "Command not found" error
Make sure you're in the Downloads folder:
```bash
cd ~/Downloads
./moly-install-linux.sh
```

### Extension says "Native host not available"
1. Run installer again - make sure it completes
2. Restart Chrome completely
3. Try again

### Installation script won't run (Windows)
1. Right-click the .bat file
2. Make sure you select "Run as administrator"
3. Click "Yes" if prompted by User Account Control

### Script asks for password repeatedly
Enter your password exactly as you would to log in to your computer.

### Still having issues?
Report at: https://github.com/Nireus79/Moly/issues

Include:
- Your operating system and version
- The complete error message
- Steps you took before the error

---

## Using Moly

### Basic Workflow

1. **Explain context** - Tell Moly who you're talking to and the situation
2. **Ask for help** - Moly asks guiding questions (Socratic mode)
3. **Paste message** - Copy the incoming message you need to respond to
4. **Get suggestions** - Moly generates 3-5 response options
5. **Copy and send** - Click "Copy" and paste into the chat

### Modes

**Socratic Mode** - Moly asks questions to help you develop your own response

**Direct Mode** - Moly generates ready-to-use responses

### Tones

**Formal** - Professional, respectful (business, authority figures)

**Friendly** - Warm, casual (friends, regular conversations)

**Dating** - Playful, engaging (romantic interests, flirting)

---

## Privacy & Data

- All conversations stored locally on your computer
- Not sent to Moly servers
- Encrypted with AES-256
- Only sent to your chosen LLM provider (Claude, OpenAI, or local)
- You can delete anytime: Settings → Clear All Data

---

## Getting Help

- **Quick questions**: Check README.md
- **Installation issues**: See INSTALLATION_ARCHITECTURE.md
- **Troubleshooting**: See TROUBLESHOOTING.md
- **Report bugs**: https://github.com/Nireus79/Moly/issues
- **Discussions**: https://github.com/Nireus79/Moly/discussions

---

## Support

- **Email**: efthimiosangelopoulos@gmail.com
- **GitHub**: https://github.com/Nireus79/Moly
- **Issues**: https://github.com/Nireus79/Moly/issues

---

*Last Updated: September 2026*
*Version: v1.0.0*
