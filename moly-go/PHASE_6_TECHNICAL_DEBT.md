# Phase 6: Technical Debt Registry

**Date:** October 8, 2026  
**Status:** POST-LAUNCH CLEANUP GUIDE

This document identifies safe-to-remove code that should be cleaned up after production monitoring confirms no impact.

---

## Code Candidates for Removal (Phase 7+)

### 1. Unused Manager Files (LOW RISK)
**Status:** Declared but never called in active orchestrator

- [ ] `agents/contact_manager.go` - Created in main.go but never used
- [ ] `agents/context_attribute_manager.go` - Created in main.go but never used
- [ ] `agents/response_template_manager.go` - Called once to initialize, never used after
- [ ] `agents/context_manager.go` - Created in agent_system but unclear if used

**Removal Process:**
1. Remove initialization from main.go (lines 138-139)
2. Remove field declarations from APIServer struct
3. Remove manager files themselves
4. Update agent_system.go if context_manager is unused
5. Test conversation_agent edge cases

**Risk Level:** MEDIUM (embedded in structures, could affect future code)

### 2. Layer 5 Legacy Code (MEDIUM RISK)

- [ ] `agents/layer5_conflict_handler.go` - Replaced by unified_orchestrator
- [ ] `layer5_conflict_handler` field in APIServer (line 120)
- [ ] Initialization in main.go (lines 250-258)

**Status:** Not called by orchestrator, kept for backwards compatibility

**Removal Process:**
1. Verify no calls to its methods in active code
2. Remove initialization from main.go
3. Remove from APIServer struct
4. Keep tests for regression safety
5. Remove file if no test dependencies

**Risk Level:** LOW (confirmed not in orchestrator pipeline)

### 3. Test Files with Dead Code (LOW RISK)

- [ ] `agents/phase2_integration_test.go` - References layer5_conflict_handler extensively
- [ ] May have other test-only code

**Status:** Can be cleaned up after confirming no test regressions

---

## Database Tables Already Removed

✅ **Phase 5 removed from schema:**
- audit_log
- behavior_patterns
- clarification_answers
- response_templates
- structured_context
- suggestion_choices
- user_interactions

✅ **Migrations created:**
- Migration 045: Drop user_contacts
- Migration 046: Drop 7 minimal-use tables

---

## What NOT to Remove

❌ **conversation_agent.go** - Still in active use
❌ **answerProcessor** - Used in clarification flow
❌ **incomingMessageAnalyzer** - Used in message processing
❌ **conflictDetector** - Used in orchestrator
❌ **extractionPhase** - Core to system

---

## Recommended Phase 7 Schedule

After production monitoring confirms no issues:

**Week 1-2:** Remove unused managers (contact, context_attribute)
**Week 2-3:** Complete Layer 5 consolidation
**Week 3-4:** Remove test-only dead code
**Week 4+:** Document final system state

---

## Verification Checklist for Each Removal

For EVERY code removal, before committing:

- [ ] Search entire codebase for references
- [ ] Check test files for dependencies
- [ ] Run full test suite
- [ ] Verify build succeeds
- [ ] Manually test feature that used the code
- [ ] Document why it's safe to remove

---

## Current System State

✅ Production Ready  
✅ Security Hardened  
✅ Schema Optimized  
✅ Build Clean  

**Safe to deploy immediately. Phase 7 cleanup can happen post-launch.**
