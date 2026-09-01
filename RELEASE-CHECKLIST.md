# Moly Release Checklist

Complete checklist for pre-release validation and publication.

---

## Phase 1: Build & Compilation

### Native Host Binaries
- [ ] Build moly-native-host for macOS (Intel x64)
- [ ] Build moly-native-host for macOS (Apple Silicon ARM64)
- [ ] Build moly-native-host for Linux x64
- [ ] Build moly-native-host for Windows x64
- [ ] Verify binaries work on each platform
- [ ] Test native messaging on each platform

**Build Scripts Location**:
- macOS: `./moly-installer/native-host/build-macos.sh`
- Linux: `./moly-installer/native-host/build-linux.sh`
- Windows: `moly-installer/native-host/build-windows.bat`

### GitHub Release
- [ ] Tag version (e.g., `v1.0.0`)
- [ ] Create GitHub release
- [ ] Upload all binaries to release
- [ ] Write detailed release notes

**Command**:
```bash
git tag -a v1.0.0 -m "Release v1.0.0: Complete one-click installer"
git push origin v1.0.0
```

---

## Phase 2: Functional Testing (All Platforms)

### macOS (Intel)
- [ ] Installer downloads successfully
- [ ] Binary runs and self-installs
- [ ] Native messaging manifest created
- [ ] Service control buttons work
- [ ] Auto-start configured
- [ ] Services restart after reboot
- [ ] Extension detects Ollama
- [ ] Can generate suggestions

### macOS (Apple Silicon)
- [ ] Installer downloads successfully
- [ ] ARM64 binary runs correctly
- [ ] Native messaging manifest created
- [ ] Service control buttons work
- [ ] Auto-start configured
- [ ] Services restart after reboot
- [ ] Extension detects Ollama
- [ ] Can generate suggestions

### Linux (Ubuntu 20.04+)
- [ ] Installer downloads successfully
- [ ] Binary runs and self-installs
- [ ] Native messaging manifest created
- [ ] Service control buttons work
- [ ] Systemd service configured
- [ ] Services restart after reboot
- [ ] CORS proxy starts automatically
- [ ] Extension detects Ollama
- [ ] Can generate suggestions

### Windows 10/11
- [ ] Installer downloads successfully
- [ ] Binary runs and self-installs
- [ ] Native messaging manifest created
- [ ] Service control buttons work
- [ ] Task Scheduler entry created
- [ ] Services restart after reboot
- [ ] CORS proxy starts automatically
- [ ] Extension detects Ollama
- [ ] Can generate suggestions

---

## Phase 3: Feature Validation

### Model Management
- [ ] Model list displays correctly
- [ ] Can select different models
- [ ] "Add Another Model" attempts automation
- [ ] Fallback to manual command works
- [ ] Model pulling doesn't crash extension

### Service Control
- [ ] Start button works
- [ ] Stop button works
- [ ] Status reflects actual state
- [ ] No crashes on network errors
- [ ] Error messages are helpful

### Provider Support
- [ ] Claude API key validation works
- [ ] OpenAI API key validation works
- [ ] Ollama detection works
- [ ] Provider switching works
- [ ] Fallback chain works correctly

### Suggestion Generation
- [ ] Generate suggestions with Ollama
- [ ] Generate suggestions with Claude
- [ ] Generate suggestions with OpenAI
- [ ] Socratic mode works
- [ ] Direct mode works
- [ ] Formal context works
- [ ] Friendly context works
- [ ] Dating context works

---

## Phase 4: Edge Cases & Error Handling

### Network Issues
- [ ] Handles slow connections gracefully
- [ ] Timeouts don't hang extension
- [ ] Offline mode has clear messages
- [ ] Recovery works after reconnection

### Port Conflicts
- [ ] Detects port 11434 already in use
- [ ] Detects port 11435 already in use
- [ ] Provides clear error messages
- [ ] Suggests solutions

### Missing Dependencies
- [ ] Clear error if Ollama not found
- [ ] Clear error if CORS proxy fails
- [ ] Clear error if native host missing
- [ ] All error messages are actionable

### Invalid Input
- [ ] Bad API keys show clear errors
- [ ] Invalid URLs handled gracefully
- [ ] Special characters don't break anything
- [ ] No crashes on unexpected input

---

## Phase 5: Performance & Security

### Performance
- [ ] Extension loads in < 2 seconds
- [ ] Settings page responsive
- [ ] Suggestions generate in reasonable time
- [ ] No memory leaks over time
- [ ] UI responsive during operations

### Security
- [ ] API keys stored securely
- [ ] No credentials in logs
- [ ] HTTPS for cloud APIs
- [ ] No arbitrary code execution
- [ ] Proper permission handling

---

## Phase 6: Compatibility

### Browser Support
- [ ] Chrome 90+
- [ ] Edge 90+
- [ ] Chromium-based browsers

### Operating Systems
- [ ] Windows 10 & 11 (x64)
- [ ] macOS 10.13+ (Intel)
- [ ] macOS 11+ (Apple Silicon)
- [ ] Ubuntu 20.04+ (x64)
- [ ] Other Linux distributions

### Provider Combinations
- [ ] Ollama + CORS Proxy
- [ ] Ollama without proxy (fallback)
- [ ] LM Studio
- [ ] Claude API
- [ ] OpenAI API
- [ ] Mixed local and cloud

---

## Phase 7: Documentation & Polish

### Documentation
- [ ] README.md is complete and accurate
- [ ] QUICKSTART.md is easy to follow
- [ ] TROUBLESHOOTING.md covers common issues
- [ ] INSTALLATION_ARCHITECTURE.md is correct
- [ ] All code is commented appropriately
- [ ] No broken links in docs

### User Experience
- [ ] Error messages are clear
- [ ] Instructions are unambiguous
- [ ] No confusing technical jargon
- [ ] Progress indicators show status
- [ ] Success/failure messages clear

### Chrome Web Store Listing
- [ ] Extension icon created (128x128 PNG)
- [ ] Short description written (≤132 chars)
- [ ] Full description written
- [ ] Category: Productivity
- [ ] Screenshots captured (1280x800 or 640x400)
- [ ] Privacy policy prepared
- [ ] Support email configured

---

## Phase 8: Chrome Web Store Submission

### Account Setup
- [ ] Chrome Developer account created
- [ ] Payment method on file
- [ ] Developer identity verified

### Extension Upload
- [ ] Extension built (`npm run build`)
- [ ] manifest.json verified
- [ ] ZIP file created
- [ ] Uploaded to Web Store
- [ ] Waiting for approval (1-3 days typical)

### Post-Approval
- [ ] Get extension ID from Web Store
- [ ] Update configuration files:
  - [ ] moly-installer/src/cli.js
  - [ ] moly-installer/.env
- [ ] Update download URLs in code
- [ ] Rebuild and test with real ID

---

## Phase 9: Pre-Launch

### Final Validation
- [ ] All tests pass
- [ ] All platforms tested
- [ ] No critical bugs
- [ ] Documentation complete
- [ ] Security review passed
- [ ] Performance acceptable

### Release Preparation
- [ ] Write release announcement
- [ ] Prepare support plan
- [ ] Set up issue monitoring
- [ ] Brief support team
- [ ] Have rollback plan ready

---

## Phase 10: Launch & Monitoring

### Publishing
- [ ] Create GitHub release with binaries
- [ ] Publish announcement
- [ ] Web Store shows "Published"
- [ ] Extension available for download

### Post-Launch
- [ ] Monitor GitHub issues
- [ ] Monitor Web Store reviews
- [ ] Fix critical bugs immediately
- [ ] Prepare patch releases
- [ ] Gather user feedback

---

## Sign-Off

**Code Quality**: ✓ Verified
**Testing**: ✓ Complete
**Documentation**: ✓ Complete
**Security**: ✓ Verified
**Performance**: ✓ Acceptable
**User Experience**: ✓ Polish complete

**Ready for Release**: [ ] YES [ ] NO

**Release Date**: _______________

**Released By**: _______________

---

## Post-Release Monitoring

- [ ] Track Web Store rating
- [ ] Monitor support tickets
- [ ] Watch GitHub issues
- [ ] Performance metrics normal
- [ ] No critical user reports
- [ ] Plan v1.1 improvements

---

*Use this checklist for every release to ensure quality and consistency.*
