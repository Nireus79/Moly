# Moly v1.1.0 Release Guide

This guide walks through creating the GitHub release with all binaries.

## Status

- ✓ Code committed locally (commit 3d7d3a8)
- ✓ v1.1.0 tag created locally
- ⏳ Push to GitHub and create release (final step)

## Option 1: Push & Release via GitHub Web UI (Recommended)

### Step 1: Push changes to GitHub

```bash
cd ~/vs_projects/Moly/Moly

# Push all commits
git push origin master

# Push the v1.1.0 tag
git push origin v1.1.0
```

### Step 2: Create Release on GitHub

1. Go to https://github.com/Nireus79/Moly/releases
2. Click "Create a new release"
3. Select tag: `v1.1.0`
4. Title: `Release v1.1.0: Complete Setup System with Built-in CORS Proxy`
5. Description (copy from git tag):

```
MAJOR FEATURES:
✓ Python CORS proxy built into native host binary
✓ Zero npm/Node.js dependencies
✓ Single self-contained binary per platform
✓ Phase 1-3: Complete setup wizard flow
✓ Auto-detect and install missing components
✓ Cross-platform support (Linux, macOS, Windows)

IMPROVEMENTS:
- 80% smaller download (45MB → 9-15MB)
- Automatic CORS proxy startup
- Professional binary distribution
- Comprehensive documentation

DOWNLOAD:
Choose the appropriate binary for your OS:
- Linux: moly-native-host-linux-x64.tar.gz
- macOS ARM64: moly-native-host-macos-arm64.tar.gz
- macOS x64: moly-native-host-macos-x64.tar.gz
- Windows: moly-native-host-windows-x64.zip
```

### Step 3: Upload Binaries

Click "Attach binaries" and upload:

#### Currently Available:
- ✓ `moly-installer/native-host/releases/moly-native-host-linux-x64.tar.gz` (9.0 MB)

#### Need to Build:
- macOS ARM64: On Mac with M1/M2/M3, run: `bash build-macos.sh`
- macOS x64: On Intel Mac, run: `bash build-macos.sh`
- Windows: On Windows PC, run: `build-windows.bat`

### Step 4: Publish Release

Click "Publish release" button.

---

## Option 2: Complete Automation via GitHub CLI (Manual)

If you prefer to do it all from command line with GitHub CLI:

```bash
# Install GitHub CLI (if not installed)
# macOS: brew install gh
# Linux: sudo apt install gh
# Windows: choco install gh

# Authenticate with GitHub
gh auth login

# Push commits and tag
git push origin master
git push origin v1.1.0

# Create release with description
gh release create v1.1.0 \
  --title "Release v1.1.0: Complete Setup System with Built-in CORS Proxy" \
  --notes-file RELEASE_NOTES.md

# Upload binaries
gh release upload v1.1.0 \
  moly-installer/native-host/releases/moly-native-host-linux-x64.tar.gz
```

---

## Building Missing Binaries

### macOS (ARM64 or x64)

On a Mac, run:

```bash
cd moly-installer/native-host

# Create virtual environment
python3 -m venv build-env
source build-env/bin/activate

# Install PyInstaller
pip install pyinstaller

# Build (automatically detects your architecture)
bash build-macos.sh

# Binary will be in: releases/moly-native-host-macos-arm64 (or -x64)
```

### Windows (x64)

On a Windows PC:

```batch
cd moly-installer\native-host

REM Create virtual environment
python -m venv build-env
call build-env\Scripts\activate.bat

REM Install PyInstaller
pip install pyinstaller

REM Build
build-windows.bat

REM Binary will be in: releases\moly-native-host-windows-x64.exe
```

### After Building

Upload the new binaries to the GitHub release:

```bash
# macOS
gh release upload v1.1.0 \
  moly-installer/native-host/releases/moly-native-host-macos-arm64.tar.gz \
  moly-installer/native-host/releases/moly-native-host-macos-x64.tar.gz

# Windows
gh release upload v1.1.0 \
  moly-installer/native-host/releases/moly-native-host-windows-x64.zip
```

---

## Release Checklist

- [ ] Push commits: `git push origin master`
- [ ] Push tag: `git push origin v1.1.0`
- [ ] Create release on GitHub
- [ ] Add release notes/description
- [ ] Upload Linux binary ✓
- [ ] Build & upload macOS binaries
- [ ] Build & upload Windows binary
- [ ] Publish release
- [ ] Update README with download links
- [ ] Announce release

---

## Download Links (After Release)

Once released, users can download from:

```
https://github.com/Nireus79/Moly/releases/download/v1.1.0/moly-native-host-linux-x64.tar.gz
https://github.com/Nireus79/Moly/releases/download/v1.1.0/moly-native-host-macos-arm64.tar.gz
https://github.com/Nireus79/Moly/releases/download/v1.1.0/moly-native-host-macos-x64.tar.gz
https://github.com/Nireus79/Moly/releases/download/v1.1.0/moly-native-host-windows-x64.zip
```

---

## What's Included in This Release

### Code Changes
- CORS proxy rewritten in Python and built into native host
- Removed npm/Node.js dependency
- Added Phase 1-3 setup wizards
- Complete cross-platform support

### Binaries
- Self-contained executables for Linux, macOS, Windows
- No external dependencies required
- 80% smaller than npm-based approach

### Documentation
- BUILD_INSTRUCTIONS.md - How to build binaries
- CORS_PROXY_UPGRADE.md - Technical migration guide
- Complete API documentation

---

## Installer Scripts

The installers automatically download the appropriate binary from this release:

**Linux installer**: `moly-installer/native-host/install-linux.sh`
- Downloads: `v1.1.0/moly-native-host-linux-x64.tar.gz`
- Installs to: `/usr/local/bin/moly-native-host`

**macOS installer**: `moly-installer/native-host/install-macos.sh`
- Downloads: `v1.1.0/moly-native-host-macos-arm64.tar.gz` or `-x64`
- Installs to: `/usr/local/bin/moly-native-host`

**Windows installer**: `moly-installer/native-host/install-windows.bat`
- Downloads: `v1.1.0/moly-native-host-windows-x64.zip`
- Installs to: `C:\Program Files\Moly\moly-native-host.exe`

---

## Testing the Release

After publishing, test by running one of the installers or downloading a binary:

```bash
# Linux: Test the proxy in standalone mode
tar -xzf moly-native-host-linux-x64.tar.gz
./moly-native-host-linux-x64 --proxy-mode
```

The proxy should start and listen on `127.0.0.1:11435`.

---

## Troubleshooting

### Git push times out?
- Check your internet connection
- Large binary files might be slow
- Try pushing commits and tags separately

### GitHub API limits?
- Authenticate with `gh auth login` to get higher limits
- Wait a few minutes between multiple uploads

### Binary not showing up?
- Refresh the GitHub page
- Check Assets section at bottom of release

---

## Next Steps

1. Complete the steps above
2. Test installation on all platforms
3. Update README with download links
4. Update documentation with v1.1.0 references
5. Tag v1.2.0 with future improvements
