# Moly Validation Checklist

Complete checklist for testing and validating Moly installation and functionality.

## Phase 1: Pre-Installation Testing

### System Requirements Testing
- [ ] Test on Windows 10 (64-bit)
- [ ] Test on Windows 11 (64-bit)
- [ ] Test on macOS 10.15+
- [ ] Test on macOS with M1/M2 chip
- [ ] Test on Ubuntu 20.04+
- [ ] Test on Debian 10+
- [ ] Test on other Linux distributions
- [ ] Verify system requirements checker works correctly
- [ ] Verify error handling for insufficient RAM
- [ ] Verify error handling for insufficient disk space
- [ ] Verify error handling for unsupported OS

### Node.js Environment Testing
- [ ] Test with Node.js 16.x
- [ ] Test with Node.js 18.x
- [ ] Test with Node.js 20.x
- [ ] Verify npm version check
- [ ] Verify proper error if Node.js < 16

### Test Suite Validation
- [ ] Run `npm test` successfully
- [ ] Run E2E test: `node src/e2e-test.js`
- [ ] Verify all test components work
- [ ] Check system requirements test output
- [ ] Check download URLs availability test
- [ ] Check Ollama connectivity test
- [ ] Check CORS proxy connectivity test
- [ ] Check dependencies test

## Phase 2: Interactive Wizard Testing

### Provider Selection
- [ ] Test Ollama selection
- [ ] Test LM Studio selection
- [ ] Test Cloud-only selection
- [ ] Verify correct options shown for each choice
- [ ] Test navigation through wizard

### Model Selection
- [ ] Test model options for Ollama
- [ ] Test model options for LM Studio
- [ ] Test model options for Cloud-only
- [ ] Verify correct default model (Mistral 7B)
- [ ] Test model descriptions display correctly

### Configuration Options
- [ ] Test auto-start toggle
- [ ] Test custom path selection
- [ ] Test settings review screen
- [ ] Test confirmation before installation
- [ ] Verify user can cancel at any point

## Phase 3: Component Download Testing

### Download Manager
- [ ] Test Ollama download URL reachability
- [ ] Test LM Studio download URL reachability
- [ ] Test download progress bar functionality
- [ ] Test download verification after completion
- [ ] Test handling of failed downloads
- [ ] Test retry mechanism
- [ ] Test fallback to manual instructions

### CORS Proxy Installation
- [ ] Test npm install -g moly-proxy
- [ ] Verify proxy installs globally
- [ ] Test proxy can be executed from anywhere
- [ ] Verify proxy binary location correct

### Model Download
- [ ] Test Ollama model pull for Mistral
- [ ] Test Ollama model pull for Llama2
- [ ] Test LM Studio manual instructions appear
- [ ] Verify model download progress shown
- [ ] Verify model verification after download
- [ ] Test handling of download interruption

## Phase 4: Service Configuration Testing

### Linux (systemd)
- [ ] Test systemd service file creation
- [ ] Test service enable works
- [ ] Test service auto-start on boot
- [ ] Test service restart on failure
- [ ] Test journal logs accessible
- [ ] Test service stop/start/restart commands
- [ ] Verify service runs with correct user
- [ ] Test on Debian-based systems
- [ ] Test on Red Hat-based systems

### macOS (LaunchAgent)
- [ ] Test LaunchAgent plist creation
- [ ] Test agent loads on login
- [ ] Test agent restart on failure
- [ ] Test agent logs to files
- [ ] Test agent can be unloaded
- [ ] Test on Intel Macs
- [ ] Test on Apple Silicon Macs

### Windows (Task Scheduler)
- [ ] Test Task Scheduler registration
- [ ] Test task auto-start on logon
- [ ] Test task runs with correct user
- [ ] Test task can be disabled/enabled
- [ ] Test task logs in Event Viewer
- [ ] Test on Windows 10
- [ ] Test on Windows 11
- [ ] Verify admin privileges not always required

## Phase 5: Extension Integration Testing

### Auto-Detection
- [ ] Auto-detect Ollama on startup
- [ ] Auto-detect LM Studio on startup
- [ ] Auto-detect with proxy running
- [ ] Auto-detect without proxy (fallback)
- [ ] Respect user's manual selection
- [ ] Don't re-detect after user chose

### Provider Configuration
- [ ] Claude API key validation
- [ ] OpenAI API key validation
- [ ] Ollama URL configuration
- [ ] LM Studio URL configuration
- [ ] Test provider switching
- [ ] Test provider fallback

### Suggestion Generation
- [ ] Generate suggestions with Ollama
- [ ] Generate suggestions with LM Studio
- [ ] Generate suggestions with Claude
- [ ] Generate suggestions with OpenAI
- [ ] Verify provider name displayed
- [ ] Test different modes (Socratic/Direct)
- [ ] Test different contexts (Formal/Friendly/Dating)

## Phase 6: Edge Cases & Error Handling

### Port Conflicts
- [ ] Test port 11434 already in use
- [ ] Test port 11435 already in use
- [ ] Test port 8000 already in use
- [ ] Provide helpful error messages
- [ ] Offer alternative port configuration

### Installation Path Conflicts
- [ ] Test when Ollama already installed
- [ ] Test when LM Studio already installed
- [ ] Test when directory already exists
- [ ] Skip re-download if already present
- [ ] Provide option to reinstall

### Network Issues
- [ ] Test download with slow connection
- [ ] Test download interruption and resume
- [ ] Test offline during model pull
- [ ] Provide clear error messages
- [ ] Offer manual installation fallback

### Missing Dependencies
- [ ] Test without chalk installed
- [ ] Test without prompts installed
- [ ] Test without cli-progress installed
- [ ] Clear error message with npm install guidance

### Invalid Input
- [ ] Test canceling at each step
- [ ] Test going back in wizard
- [ ] Test invalid file paths
- [ ] Test special characters in paths
- [ ] Graceful error handling

## Phase 7: User Experience Testing

### UI/UX
- [ ] Terminal output is clear and readable
- [ ] Color coding is appropriate
- [ ] Progress bars update smoothly
- [ ] Error messages are helpful
- [ ] Instructions are clear
- [ ] No confusing jargon
- [ ] Emojis/icons display correctly

### Accessibility
- [ ] Works without color vision (contrast OK)
- [ ] Works with screen readers
- [ ] Keyboard navigation works
- [ ] Error messages are accessible

### Documentation
- [ ] README.md is complete
- [ ] QUICKSTART.md is easy to follow
- [ ] TROUBLESHOOTING.md covers common issues
- [ ] Code comments are clear
- [ ] Installation scripts are documented

## Phase 8: Performance Testing

### Installation Speed
- [ ] Overall installation < 30 minutes on average hardware
- [ ] Download progress is reported
- [ ] Model download doesn't block other operations
- [ ] Auto-start configuration is fast

### Runtime Performance
- [ ] Ollama suggestions in < 10 seconds
- [ ] CORS proxy has minimal latency
- [ ] Extension UI remains responsive
- [ ] Memory usage is reasonable

## Phase 9: Compatibility Testing

### Browser Compatibility
- [ ] Chrome 90+
- [ ] Chrome 100+
- [ ] Edge 90+
- [ ] Chromium-based browsers
- [ ] Test extension loads without errors
- [ ] Test sidebar appears correctly

### OS Combinations
- [ ] Windows 10 + Chrome
- [ ] Windows 11 + Edge
- [ ] macOS Intel + Chrome
- [ ] macOS M1/M2 + Chrome
- [ ] Ubuntu + Chrome
- [ ] Ubuntu + Chromium

### Provider Combinations
- [ ] Ollama + CORS Proxy
- [ ] Ollama without proxy (fallback)
- [ ] LM Studio alone
- [ ] LM Studio + Fallback to cloud
- [ ] Claude API key
- [ ] OpenAI API key
- [ ] Mixed local and cloud

## Phase 10: Security Testing

### Credential Handling
- [ ] API keys stored securely (Chrome storage)
- [ ] API keys never logged
- [ ] API keys not sent to wrong endpoints
- [ ] Local credentials never leave system
- [ ] No credentials in error messages

### Network Security
- [ ] CORS proxy only listens on localhost
- [ ] No sensitive data in proxy logs
- [ ] HTTPS for cloud API calls
- [ ] SSL certificate validation works

### Installation Security
- [ ] No arbitrary code execution
- [ ] No unverified downloads
- [ ] Safe file operations
- [ ] Proper permission handling

## Sign-Off Checklist

- [ ] All tests pass
- [ ] All edge cases handled
- [ ] Documentation complete
- [ ] User feedback incorporated
- [ ] Security review passed
- [ ] Performance acceptable
- [ ] Ready for public release

---

## Testing Report Template

```
Date: YYYY-MM-DD
Tester: Name
Platform: OS/Version
Node.js: Version
Installer: Version

System Requirements Check: PASS/FAIL
Interactive Wizard: PASS/FAIL
Download Manager: PASS/FAIL
Service Configuration: PASS/FAIL
Extension Integration: PASS/FAIL
Overall: PASS/FAIL

Issues Found:
1. [Issue description]
2. [Issue description]

Performance Notes:
- Download speed: XX MB/s
- Installation time: XX minutes
- Model download time: XX minutes

Recommendations:
- [Recommendation 1]
- [Recommendation 2]

Tester Signature: __________________
```

---

**Last Updated:** 2026-09-01
**Maintained By:** Moly Team
