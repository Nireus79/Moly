# Moly Pre-Release Checklist

**Status**: Code Complete - Awaiting Build & Web Store Publication

## Build & Release Preparation

### 1. Build Native Host Binaries
- [ ] Build moly-native-host for macOS (Intel x64)
- [ ] Build moly-native-host for macOS (Apple Silicon ARM64)
- [ ] Build moly-native-host for Linux x64
- [ ] Build moly-native-host for Windows x64
- [ ] Verify binaries work on each platform
- [ ] Test native messaging on each platform

**Location**: `moly-installer/native-host/moly-host.py`

**Build Scripts**:
- macOS: `./moly-installer/native-host/build-macos.sh`
- Linux: `./moly-installer/native-host/build-linux.sh`
- Windows: `moly-installer/native-host/build-windows.bat`

### 2. Create GitHub Release
- [ ] Tag version (e.g., `v1.0.0`)
- [ ] Upload native host binaries to release
- [ ] Upload installer executables/packages
- [ ] Create detailed release notes

**Command**:
```bash
git tag -a v1.0.0 -m "Release v1.0.0: Complete one-click installer"
git push origin v1.0.0
```

### 3. Test Installer End-to-End
- [ ] Test on macOS (Intel)
- [ ] Test on macOS (Apple Silicon)
- [ ] Test on Ubuntu 20.04+
- [ ] Test on Fedora 35+
- [ ] Test on Windows 10
- [ ] Test on Windows 11

**Test Checklist per OS**:
- [ ] Installer downloads successfully
- [ ] System requirements check works
- [ ] Wizard runs without errors
- [ ] Ollama downloads and installs
- [ ] CORS proxy installs
- [ ] Native host downloads and installs
- [ ] Model pulls automatically
- [ ] Auto-start configures properly
- [ ] Services start on reboot
- [ ] ServiceManager UI controls work
- [ ] Extension detects Ollama
- [ ] Can generate suggestions

### 4. Submit Extension to Chrome Web Store
- [ ] Create Chrome developer account (if not already done)
- [ ] Prepare store listing:
  - [ ] Extension icon (128x128 PNG)
  - [ ] Short description (132 chars max)
  - [ ] Full description
  - [ ] Category: Productivity
  - [ ] Screenshots (1280x800 or 640x400)
  - [ ] Privacy policy: `./docs/PRIVACY_POLICY.md`
  - [ ] Support email
- [ ] Upload extension.zip to Web Store
- [ ] Wait for approval (typically 1-3 days)
- [ ] Get extension ID from Web Store URL

**Example URL**: `https://chromewebstore.google.com/detail/moly-messaging-coach/[EXTENSION_ID]`

### 5. Update Configuration
After Web Store approval, update these files:

**moly-installer/src/cli.js**:
```javascript
// Line ~230: Update extension ID
const extensionId = 'your-actual-extension-id';

// Line ~240: Update Web Store URL
console.log(chalk.dim(`   Chrome Web Store: https://chromewebstore.google.com/detail/moly-messaging-coach/YOUR_ID`));
```

**moly-installer/.env**:
```bash
MOLY_EXTENSION_ID=your-actual-extension-id
MOLY_WEBSTORE_URL=https://chromewebstore.google.com/detail/moly-messaging-coach/your-id
```

### 6. Create Release Documentation
- [ ] Update README.md with installation instructions
- [ ] Create GETTING_STARTED.md guide
- [ ] Test all documentation links
- [ ] Verify screenshots match current UI

### 7. Release Management
- [ ] Create GitHub release (tag v1.0.0)
- [ ] Upload final binaries to release
- [ ] Publish announcement
- [ ] Monitor for issues
- [ ] Prepare bug fix release process

## File Updates Required

### moly-installer/src/downloadManager.js
Currently uses real GitHub URLs:
```javascript
url = 'https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-native-host-macos';
```

✓ **Status**: Updated with real repository path

### moly-installer/src/cli.js
Uses environment variables for extension ID and Web Store URL:
```javascript
const extensionId = process.env.MOLY_EXTENSION_ID || 'placeholder';
const webStoreUrl = process.env.MOLY_WEBSTORE_URL || 'https://...';
```

✓ **Status**: Updated to support env vars

### moly-installer/.env.example
Created as template for configuration.

✓ **Status**: Complete

## Timeline

**Week 1**: Build native host binaries
**Week 2**: Test on all platforms
**Week 3**: Submit to Chrome Web Store
**Week 4**: Get approval, update config
**Week 5**: Release!

## Known Blockers

### None currently
All critical features are implemented. Only blocker is building platform-specific binaries.

## Post-Release

### Version 1.0.1 (Bug Fixes)
- Monitor GitHub issues
- Gather user feedback
- Fix critical bugs
- Re-release as patch

### Version 1.1 (Enhancements)
- GPU acceleration support
- More model options
- Performance improvements
- Firefox support

### Version 2.0 (Major)
- Cloud sync (encrypted)
- Safari support
- Mobile app
- Team collaboration

## Contact

**Project**: Moly
**Author**: Nireus79 (efthimiosangelopoulos@gmail.com)
**Repository**: https://github.com/Nireus79/Moly
**Issues**: https://github.com/Nireus79/Moly/issues

---

**Last Updated**: 2026-09-01  
**Status**: READY FOR BUILD & RELEASE
