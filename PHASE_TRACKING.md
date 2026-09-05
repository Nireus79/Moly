# Moly Production Roadmap - Execution Tracking

**Goal**: Bring Moly from 2/10 to 8/10 production-ready  
**Commitment**: Complete ALL 7 phases, no half-done jobs  
**Status**: IN PROGRESS  

---

## PHASE 1: CRITICAL FIXES (Weeks 1-2)
**Target Completion**: 2 weeks  
**Status**: 🟡 IN PROGRESS (3/4 tasks complete: 75%)

### Task 1.1: Cross-Platform Database Paths
**Status**: ✅ COMPLETE  
**Files modified**: `moly-go/database.go`  
**Files created**: `moly-go/database_test.go`  

- [x] Add `runtime` import
- [x] Create `getConfigDir()` function
- [x] Replace hardcoded path with function call
- [x] Test on Linux (actual)
- [x] Cross-compile for Windows/macOS
- [x] Verify builds without errors
- [x] Unit tests created (3 tests)
- [x] Committed to git

**Verification Results**:
- ✓ Code changes applied correctly
- ✓ Compiles on Linux
- ✓ Cross-compiles for Windows
- ✓ Cross-compiles for macOS
- ✓ All 3 unit tests PASSING
- ✓ Backend runs on Linux
- ✓ Database created in ~/.config/moly/moly.db
- ✓ Working tree clean
- ✓ Git commits: feat, chore, docs

**Started**: Sep 5, 2026  
**Completed**: Sep 5, 2026 (same day)  

---

### Task 1.2: Configuration Management
**Status**: ✅ COMPLETE  
**Files created**: `moly-go/config.go`  
**Files modified**: `moly-go/main.go`, `moly-go/config_test.go`  

- [x] Create config.go with ServerConfig struct
- [x] Implement LoadConfig() function
- [x] Add env var support (MOLY_PORT, MOLY_HOST, MOLY_LOG_LEVEL, MOLY_CORS_PROXY_PORT)
- [x] Add config file support (moly.config.json)
- [x] Update main.go to use LoadConfig()
- [x] Remove hardcoded constants (Port, Host)
- [x] Test env vars work (verified with MOLY_PORT=:8888 MOLY_HOST=0.0.0.0)
- [x] Test config file path detection (platform-specific)
- [x] Unit tests created (9 comprehensive tests)
- [x] Committed to git

**Verification Results**:
- ✓ Code compiles without errors
- ✓ Environment variables override defaults (verified: 0.0.0.0:8888 when set)
- ✓ Config file path detection works on Linux
- ✓ DatabasePath automatically set to platform-specific location
- ✓ All 9 new LoadConfig tests PASSING
- ✓ Backend startup correctly applies environment variables
- ✓ Configuration precedence verified: env vars > file > defaults

**Started**: Sep 5, 2026  
**Completed**: Sep 5, 2026 (same day)  

---

### Task 1.3: CORS Proxy Path Robustness
**Status**: ✅ COMPLETE  
**Files modified**: `moly-go/main.go`  
**Files created**: `moly-go/proxy_test.go`  

- [x] Create findCORSProxyScript() function
- [x] Add MOLY_PROXY_PATH env var support (highest priority)
- [x] Check binary directory (../moly-proxy/bin/moly-proxy.js)
- [x] Check dev directory (moly-proxy/bin/moly-proxy.js, ../moly-proxy/bin/moly-proxy.js)
- [x] Check standard installation directories (OS-specific paths)
- [x] Graceful error handling with user-friendly messages
- [x] Update startCORSProxy() to use new function
- [x] Test with env var set (TestFindCORSProxyScriptEnvVarOverride)
- [x] Test with env var not set (TestFindCORSProxyScriptEnvVarNotSet)
- [x] Test relative paths in dev (TestFindCORSProxyScriptRelativeToDev)
- [x] Committed to git

**Verification Results**:
- ✓ All 5 unit tests PASSING
- ✓ MOLY_PROXY_PATH override works correctly
- ✓ Error messages guide users to MOLY_PROXY_PATH
- ✓ Platform-specific paths detected (Linux: /opt/moly, macOS: /Applications/Moly, Windows: %APPDATA%\Moly)
- ✓ Binary discovery searches correctly relative to executable
- ✓ Development paths (../moly-proxy/bin) checked properly

**Started**: Sep 5, 2026  
**Completed**: Sep 5, 2026 (same day)  

---

### Task 1.4: Remove Unused Endpoints
**Status**: ⏹ NOT STARTED  
**Files to modify**: `moly-go/main.go`  
**Files to create**: `DEPRECATED_ENDPOINTS.md`  

- [ ] Identify all unused endpoints
- [ ] Document removed endpoints
- [ ] Remove http.HandleFunc calls
- [ ] Remove handler functions
- [ ] Verify builds without errors
- [ ] Code compiles on all platforms
- [ ] Test that remaining endpoints still work
- [ ] Committed to git

**Expected start**: [After 1.3 complete]  
**Expected finish**: [1 day later]  

---

### Phase 1 Completion Checklist
- [ ] All 4 tasks complete
- [ ] Code compiles on Linux
- [ ] Cross-compiles for Windows/macOS without errors
- [ ] Unit tests created and passing
- [ ] Database path works on Linux
- [ ] Config loading works on Linux
- [ ] Proxy discovery robust on Linux
- [ ] Dead endpoints removed
- [ ] All commits pushed to git
- [ ] Ready for Phase 2

**Phase 1 Expected Completion**: [2 weeks from start]  

---

## PHASE 2: CROSS-PLATFORM INSTALLERS (Weeks 2-4)
**Status**: ⏹ NOT STARTED - Blocked on Phase 1

### Task 2.1: Linux Setup Script
- [ ] Create moly-installer/install.sh
- [ ] Detect OS
- [ ] Create config directory
- [ ] Copy binary
- [ ] Setup native messaging
- [ ] Verify installation
- [ ] Test on Linux

### Task 2.2: Windows Setup Script
- [ ] Create moly-installer/install.ps1
- [ ] PowerShell implementation
- [ ] Registry setup for native messaging
- [ ] Verify builds

### Task 2.3: macOS Setup Script
- [ ] Create moly-installer/install-macos.sh
- [ ] Create proper directories
- [ ] Setup native messaging
- [ ] Verify builds

### Task 2.4: Native Messaging Registration
- [ ] Windows registry paths configured
- [ ] macOS paths configured
- [ ] Chrome support
- [ ] Brave support

---

## PHASE 3: PRODUCTION LOGGING (Weeks 4-6)
**Status**: ⏹ NOT STARTED - Blocked on Phase 1

### Task 3.1: Structured Logging
- [ ] Add logrus dependency
- [ ] Create logger.go
- [ ] JSON format output
- [ ] Log levels
- [ ] File logging

### Task 3.2: Frontend Error Reporting
- [ ] Create errorReporter.ts
- [ ] Store errors locally
- [ ] Format timestamps, context

### Task 3.3: Log Rotation
- [ ] Add lumberjack
- [ ] Configure rotation
- [ ] Test rotation

---

## PHASE 4: TESTING & CI/CD (Weeks 6-8)
**Status**: ⏹ NOT STARTED - Blocked on Phase 1

### Task 4.1: GitHub Actions
- [ ] Create .github/workflows/build.yml
- [ ] Multi-platform build
- [ ] Automated testing
- [ ] Artifact generation

### Task 4.2: Unit Tests
- [ ] database_test.go
- [ ] config_test.go
- [ ] API tests
- [ ] Target: > 60% coverage

### Task 4.3: Integration Tests
- [ ] Backend startup tests
- [ ] Database creation tests
- [ ] Configuration tests

---

## PHASE 5: DOCUMENTATION (Weeks 8-10)
**Status**: ⏹ NOT STARTED - Blocked on Phase 1

### Task 5.1: User Installation Guide
- [ ] Windows steps
- [ ] macOS steps
- [ ] Linux steps
- [ ] Troubleshooting

### Task 5.2: Privacy Policy & Terms
- [ ] PRIVACY_POLICY.md
- [ ] TERMS_OF_SERVICE.md
- [ ] Legal review

### Task 5.3: Developer Documentation
- [ ] ARCHITECTURE.md
- [ ] API.md
- [ ] CONTRIBUTING.md
- [ ] BUILD.md

---

## PHASE 6: SECURITY HARDENING (Weeks 10-12)
**Status**: ⏹ NOT STARTED - Blocked on Phase 1

### Task 6.1: Input Validation
- [ ] Create validation.go
- [ ] Validate all endpoints
- [ ] Parameterized queries

### Task 6.2: Rate Limiting
- [ ] Add tollbooth
- [ ] Configure limits
- [ ] Test rate limiting

### Task 6.3: Security Audit
- [ ] SQL injection check
- [ ] XSS prevention
- [ ] Header security
- [ ] Input validation complete

---

## PHASE 7: DISTRIBUTION (Weeks 12-14)
**Status**: ⏹ NOT STARTED - Blocked on Phase 1

### Task 7.1: Package Creation
- [ ] Linux .deb package
- [ ] Linux .rpm package
- [ ] Windows MSI installer
- [ ] macOS DMG installer

### Task 7.2: Chrome Web Store Submission
- [ ] Developer account
- [ ] Upload extension
- [ ] Screenshots
- [ ] Description
- [ ] Submit for review

### Task 7.3: Brave Web Store Submission
- [ ] Same assets as Chrome
- [ ] Submit to Brave
- [ ] Approval

---

## SUCCESS CRITERIA

### Before Launch:
- [ ] Builds on Windows, macOS, Linux
- [ ] Database in correct OS-specific location
- [ ] Setup automation works
- [ ] No hardcoded paths
- [ ] Unused endpoints removed
- [ ] Structured logging in place
- [ ] CI/CD green on all platforms
- [ ] Test coverage > 60%
- [ ] Documentation complete
- [ ] Privacy policy & ToS published
- [ ] Security audit passed
- [ ] Manual testing on all 3 OSes
- [ ] Beta testing with 20+ users

### Timeline:
- Phase 1: Weeks 1-2
- Phase 2: Weeks 2-4
- Phase 3: Weeks 4-6
- Phase 4: Weeks 6-8
- Phase 5: Weeks 8-10
- Phase 6: Weeks 10-12
- Phase 7: Weeks 12-14
- **Total: 14 weeks**

### With 2 Developers (parallel work):
- **Realistic: 7-10 weeks**

---

## RULES

✋ **STOP SIGN**: Don't move to next task until current task is:
- ✓ Fully implemented
- ✓ Tested on Linux
- ✓ Tested on cross-compilation
- ✓ Unit tests created
- ✓ Committed to git
- ✓ Documented

🔒 **NO HALF-DONE JOBS**: Every task either 100% complete or 0% started

📝 **TRACKING**: Update this file after EACH task completion

🚀 **NO SKIPPING PHASES**: Do them in order (1 → 2 → 3 → ... → 7)

---

## CURRENT STATUS

**Today's Date**: [NOW]  
**Phase 1 Progress**: Starting Task 1.1  
**Next Milestone**: Complete Task 1.1 (today)  
**Launch Target**: Week 12-14  

