# Moly Installer - Cross-Platform Testing Guide

Complete testing checklist for installer functionality on Mac, Linux, and Windows.

## Pre-Testing Setup

### Get Extension ID
1. Open `chrome://extensions/`
2. Find "Moly - Messaging Coach"
3. Copy the Extension ID
4. Save for later use

### Build Native Host (Optional but Recommended)
```bash
# For testing, optional step - installers can work without native host
cd moly-installer/native-host/

# Choose your platform:
./build-macos.sh    # macOS
./build-linux.sh    # Linux
build-windows.bat   # Windows
```

## Testing Scenarios

### Scenario 1: No Local Model Installed
**Expected Behavior**: Show setup dialog on "Start Setup" button

#### Test Steps:
1. Open Moly Settings
2. Go to "Local Models Status" section
3. If no local model detected, click "Start Setup"
4. InstallerDialog should appear with platform-specific instructions
5. Verify:
   - [ ] Platform name is correct (macOS/Linux/Windows)
   - [ ] Setup instructions are platform-appropriate
   - [ ] All buttons are visible and clickable

#### Platform Verification:

**macOS:**
- [ ] Instructions mention .dmg file
- [ ] Instructions include "Drag to Applications"
- [ ] Release page opens in new tab
- [ ] Download starts moly-installer-macos.dmg

**Linux:**
- [ ] Instructions mention chmod +x
- [ ] Instructions show terminal commands
- [ ] Release page opens correctly
- [ ] Download starts moly-installer-linux-x64

**Windows:**
- [ ] Instructions mention .exe file
- [ ] Instructions mention User Account Control prompt
- [ ] Release page opens correctly
- [ ] Download starts moly-installer-windows-x64.exe

### Scenario 2: Native Host Installed
**Expected Behavior**: "Launch Installer" button should work directly

#### Test Steps:
1. Follow NATIVE-HOST-SETUP.md to install native host for your platform
2. Restart Chrome
3. Open Moly Settings → "Local Models Status"
4. Click "Start Setup"
5. Click "Launch Installer" button
6. Verify:
   - [ ] Installer application launches
   - [ ] "Installer launched!" message appears
   - [ ] Message clears after 2 seconds
   - [ ] Dialog closes after success

#### Platform Verification:

**macOS:**
- [ ] App opens from Applications folder
- [ ] Moly Installer.app is recognized
- [ ] Native messaging host is executable

**Linux:**
- [ ] Binary launches from /usr/local/bin
- [ ] Terminal window opens (if applicable)
- [ ] Native messaging host has execute permission

**Windows:**
- [ ] .exe launches successfully
- [ ] UAC prompt may appear (acceptable)
- [ ] PowerShell can write registry (admin required)

### Scenario 3: Download Installer
**Expected Behavior**: Browser downloads installer file

#### Test Steps:
1. Open Moly Settings → "Local Models Status"
2. Click "Start Setup"
3. Click "Download Installer" button
4. Verify:
   - [ ] Download starts immediately
   - [ ] Message shows "Download started!"
   - [ ] File appears in Downloads folder
   - [ ] File has correct name for platform
   - [ ] File size is reasonable (not empty)

#### Platform File Names:
- macOS: `moly-installer-macos.dmg`
- Linux: `moly-installer-linux-x64`
- Windows: `moly-installer.exe`

### Scenario 4: Release Page Link
**Expected Behavior**: Opens GitHub releases page

#### Test Steps:
1. Open Moly Settings → "Local Models Status"
2. Click "Start Setup"
3. Click "Release Page" button
4. Verify:
   - [ ] New tab opens
   - [ ] GitHub releases page loads
   - [ ] Correct release version shown
   - [ ] Download links visible

### Scenario 5: Manual Installation
**Expected Behavior**: Using manual download + setup

#### macOS:
1. Download moly-installer-macos.dmg
2. Double-click to mount
3. Follow on-screen instructions
4. Verify:
   - [ ] Disk image mounts successfully
   - [ ] Installer app is visible
   - [ ] Can drag to Applications
   - [ ] Can launch from Applications

#### Linux:
1. Download moly-installer-linux-x64
2. In terminal: `chmod +x moly-installer-linux-x64`
3. Run: `./moly-installer-linux-x64`
4. Verify:
   - [ ] File becomes executable
   - [ ] Installer starts without errors
   - [ ] Wizard interface appears
   - [ ] Can complete setup

#### Windows:
1. Download moly-installer-windows-x64.exe
2. Double-click the file
3. If UAC prompt appears, click "Yes"
4. Verify:
   - [ ] File runs without antivirus blocking
   - [ ] Installer window opens
   - [ ] Wizard can proceed to next steps
   - [ ] Installation directory selection works

## Integration Testing

### Test 1: Auto-Detection After Install
1. Install Ollama using installer
2. Restart browser
3. Return to Moly Settings
4. Verify:
   - [ ] "Local Models Status" shows Ollama running
   - [ ] Models are detected and listed
   - [ ] LocalModelSetup no longer shows setup panel
   - [ ] ModelManagement component shows available models

### Test 2: Model Selection After Install
1. After successful Ollama install:
2. Verify installer automatically pulled a model
3. In ModelManagement panel:
   - [ ] Model is listed
   - [ ] Can click to select it
   - [ ] Selected model appears in Settings dropdown
   - [ ] Can send test message with that model

### Test 3: Multiple Models
1. Install Ollama with Mistral model
2. Use terminal to pull another model: `ollama pull llama2`
3. In Moly Settings → ModelManagement:
   - [ ] Both models are listed
   - [ ] Can switch between them
   - [ ] Currently selected shows checkmark
   - [ ] Can add more models via "Add Another Model"

## Browser Compatibility

### Test Each Browser:

#### Chrome
- [ ] Extension loads without warnings
- [ ] All buttons functional
- [ ] Download works
- [ ] Native messaging works (if host installed)

#### Chromium (if available)
- [ ] Same as Chrome
- [ ] Native messaging configuration may differ

#### Edge
- [ ] Extension loads
- [ ] UI renders correctly
- [ ] Download works
- [ ] Native messaging works (if host installed)

#### Brave (if available)
- [ ] Extension loads
- [ ] All features functional

## Error Cases

### Test Each Error Scenario:

#### Network Error During Download
1. Disconnect internet
2. Click "Download Installer"
3. Verify:
   - [ ] Error message appears
   - [ ] No corrupted files created
   - [ ] Can retry when online

#### Invalid Extension ID (Native Host)
1. Modify extension ID in host config to random string
2. Click "Launch Installer"
3. Verify:
   - [ ] Gracefully falls back to download
   - [ ] Error message is helpful
   - [ ] No crash or hang

#### Installer Not Found (Native Host)
1. Uninstall native host
2. Uninstall Ollama (remove from Applications/Program Files)
3. Click "Launch Installer"
4. Verify:
   - [ ] Shows "Native installer not found" message
   - [ ] Download button is available
   - [ ] Can still proceed manually

#### Corrupted Download
1. Interrupt download mid-way
2. Retry download
3. Verify:
   - [ ] Can retry without errors
   - [ ] Completes successfully on retry
   - [ ] Downloaded file is valid

## Performance Testing

### Test Installation Speed:

#### macOS:
- [ ] DMG downloads in under 30 seconds
- [ ] Installer launches within 2 seconds
- [ ] Ollama downloads within 5 minutes

#### Linux:
- [ ] Binary downloads in under 30 seconds
- [ ] Installer starts within 2 seconds
- [ ] Installation wizard responsive

#### Windows:
- [ ] EXE downloads in under 30 seconds
- [ ] Installer launches within 3 seconds (UAC adds delay)
- [ ] GUI responsive throughout

### Test Native Messaging Response Time:

1. Install native host
2. Open Chrome DevTools (F12)
3. Console should show message timing
4. Verify:
   - [ ] Launch attempt completes within 2 seconds
   - [ ] Timeout handling works if host hangs
   - [ ] Error handling doesn't delay UI

## UI/UX Testing

### Dialog Appearance
- [ ] Modal overlay is visible
- [ ] Dialog is centered on screen
- [ ] Text is readable (font size OK)
- [ ] All buttons are visible
- [ ] Close button (X) works
- [ ] Can click outside to close

### Button States
- [ ] Buttons enabled initially
- [ ] Buttons disabled during action
- [ ] Button text changes appropriately ("Launching...", "Downloading...")
- [ ] Buttons re-enable after completion

### Messages and Feedback
- [ ] Success messages are clear
- [ ] Error messages are helpful
- [ ] Messages auto-clear after 3 seconds
- [ ] Manual close (X) button works

### Mobile Responsiveness (if applicable)
- [ ] Dialog fits on smaller screens
- [ ] Buttons are still clickable on touch devices
- [ ] Text is readable on all screen sizes

## Documentation Verification

### Check Documentation:
- [ ] NATIVE-HOST-SETUP.md is complete
- [ ] Instructions are accurate for each OS
- [ ] Registry paths are correct (Windows)
- [ ] File permissions are correct (Linux)
- [ ] App bundle structure is correct (macOS)

### Check Code Comments:
- [ ] installerLauncher.ts has clear comments
- [ ] InstallerDialog.tsx is well documented
- [ ] Function purposes are clear
- [ ] Error scenarios are documented

## Post-Installation Verification

### After Successful Installation:

1. **Check Auto-Start:**
   - [ ] macOS: Ollama starts on login (LaunchAgent installed)
   - [ ] Linux: Ollama starts on boot (systemd service enabled)
   - [ ] Windows: Ollama starts on login (Task Scheduler setup)

2. **Check Model Installation:**
   - [ ] Ollama has at least 1 model available
   - [ ] Model is listed in Moly Settings
   - [ ] Can select and use the model

3. **Check CORS Proxy:**
   - [ ] If available, proxy runs on port 11435
   - [ ] Extension can communicate through proxy
   - [ ] Falls back to direct connection if needed

4. **Check End-to-End:**
   - [ ] Open Moly sidebar
   - [ ] Type a message
   - [ ] Send to installed local model
   - [ ] Get response within reasonable time
   - [ ] Response quality is acceptable

## Regression Testing

### Test That Existing Features Still Work:

- [ ] Cloud provider setup (Claude/OpenAI) still works
- [ ] Settings page loads without errors
- [ ] Chat functionality works with installed model
- [ ] History is preserved
- [ ] Storage permissions work

## Sign-Off Checklist

When ALL tests pass on a platform, check off:

### macOS Sign-Off
- [ ] Platform detection: PASS
- [ ] Setup dialog: PASS
- [ ] Download installer: PASS
- [ ] Launch installer: PASS (if native host)
- [ ] Manual installation: PASS
- [ ] Auto-detection after install: PASS
- [ ] Model selection: PASS
- [ ] End-to-end test: PASS
- [ ] No security warnings
- [ ] No performance issues
- **Overall Status: READY FOR RELEASE**

### Linux Sign-Off
- [ ] Platform detection: PASS
- [ ] Setup dialog: PASS
- [ ] Download installer: PASS
- [ ] Launch installer: PASS (if native host)
- [ ] Manual installation: PASS
- [ ] Auto-detection after install: PASS
- [ ] Model selection: PASS
- [ ] End-to-end test: PASS
- [ ] No security warnings
- [ ] No performance issues
- **Overall Status: READY FOR RELEASE**

### Windows Sign-Off
- [ ] Platform detection: PASS
- [ ] Setup dialog: PASS
- [ ] Download installer: PASS
- [ ] Launch installer: PASS (if native host)
- [ ] Manual installation: PASS
- [ ] Auto-detection after install: PASS
- [ ] Model selection: PASS
- [ ] End-to-end test: PASS
- [ ] No UAC issues
- [ ] No antivirus blocking
- [ ] No performance issues
- **Overall Status: READY FOR RELEASE**

## Known Issues and Workarounds

### macOS Apple Silicon (ARM64)
**Issue**: Some tools may not run on ARM64
**Workaround**: Distribute both ARM64 and Intel x64 versions

### Linux Distribution Variations
**Issue**: Install paths vary across distributions
**Workaround**: Detect common paths, provide manual path entry

### Windows Antivirus
**Issue**: Downloaded executable may be blocked
**Workaround**: Sign executable with certificate, document whitelist steps

## Reporting Issues

When tests fail, report with:
- Platform and OS version
- Browser and version
- Extension ID
- Steps to reproduce
- Expected vs actual behavior
- Screenshots if applicable
- Browser console errors (F12)
- Network tab results

## Release Readiness

Once all platforms pass all tests:

1. [ ] Build final installers for all platforms
2. [ ] Test downloaded installers (not build versions)
3. [ ] Create release notes
4. [ ] Update documentation
5. [ ] Prepare Chrome Web Store listing
6. [ ] Submit to Chrome Web Store
7. [ ] Monitor for user-reported issues
8. [ ] Have rollback plan ready

---

**Last Updated**: 2026-09-01
**Status**: Testing Framework Complete, Awaiting Platform Testing
