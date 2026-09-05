# Moly Installation - One-Time Setup

## What You're Installing

- Chrome extension (loads from dist folder - already built)
- Go backend (already built)
- Native host integration (allows extension to auto-start backend)

No complicated installation. Just 2 steps.

## Step 1: Set Up Native Host (One-Time Only)

This allows the extension to automatically start the backend when you click the icon.

```bash
# Linux/macOS
./NATIVE_HOST_SETUP.sh

# Windows (run PowerShell as Administrator)
.\NATIVE_HOST_SETUP.ps1
```

This creates a system-level configuration that links the extension to the Go backend binary.

Done. You'll never need to run this again.

## Step 2: Load Extension in Chrome

```
1. Open chrome://extensions
2. Enable "Developer mode" (toggle, top-right)
3. Click "Load unpacked"
4. Select: Moly/moly-extension/dist/
5. Done
```

## That's It. Now Use It.

```
1. Click Moly icon in toolbar
2. Extension checks if backend is running
3. If not, automatically starts it
4. Type a message
5. Get suggestions with safety/ethics analysis
```

No manual startup scripts. No systemd/LaunchAgent setup. No complicated installation process.

## How It Works Behind the Scenes

```
You click Moly icon
  ↓
Extension's BackendManager checks port 11436
  ↓
Backend not running? → Sends native host message
  ↓
Native host spawns Go backend binary
  ↓
Go backend starts and auto-spawns CORS Proxy (11435)
  ↓
Extension detects backend running → Shows ✓ Connected
  ↓
You type message → Goes to backend for safety/ethics check
  ↓
Extension requests LLM suggestions via CORS Proxy
  ↓
You see results with copy buttons
```

## What if Native Host Setup Fails?

If NATIVE_HOST_SETUP.sh fails:

1. Check Go binary exists: `ls -lh moly-go/moly`
2. Make it executable: `chmod +x moly-go/moly`
3. Try again: `./NATIVE_HOST_SETUP.sh`

If still failing:

- Run `./START_MOLY.sh` manually to start backend
- Then click extension icon
- Backend is already running, so it works

## Uninstall

1. Go to `chrome://extensions`
2. Click "Remove" on Moly
3. Done

Native host manifest can stay (doesn't hurt) or manually remove:
```bash
# Linux
rm ~/.config/google-chrome/NativeMessagingHosts/com.moly.backend_host.json

# macOS
rm ~/Library/Application\ Support/Google/Chrome/NativeMessagingHosts/com.moly.backend_host.json
```

## Troubleshooting

### Backend doesn't auto-start

1. Run native host setup: `./NATIVE_HOST_SETUP.sh`
2. Restart Chrome
3. Click icon again

### "Backend unavailable" in sidebar

- Backend isn't running
- Click again (triggers auto-start)
- Wait 2 seconds
- Should show ✓ Connected

### Nothing works

- Check console: F12 → Extensions → Inspect
- Check Go binary: `moly-go/moly` exists?
- Try manual start: `./START_MOLY.sh` then click icon

---

**Installation Time**: 2 minutes  
**Setup Complexity**: 2 commands  
**How often**: Once, ever
