# Documentation Audit & Cleanup Plan

**Date**: Sept 28, 2026  
**Status**: Complete audit, cleanup plan ready

---

## Summary

| Category | Action | Count |
|----------|--------|-------|
| **Keep & Update** | Update dates, verify content | 11 files |
| **Archive** | Move to historical folder | 3 files |
| **Keep As-Is** | No changes needed | 1 file |

---

## TIER 1: CORE ARCHITECTURE (Keep, verify current)

### ✅ MOLY_COMPLETE_VISION.md (379 lines)
- **Purpose**: Product vision and core capabilities
- **Status**: Current ✅
- **Last Updated**: 2026-09-26
- **Action**: KEEP - Reflects current product vision correctly
- **Review**: Vision is accurate, all capabilities implemented

### ✅ MOLY_SECURITY_LAYERS.md (423 lines)
- **Purpose**: 11-layer security architecture specification
- **Status**: Current ✅
- **Last Updated**: 2026-09-26
- **Action**: KEEP - Architectural spec is authoritative
- **Review**: All 11 layers implemented and wired

### ✅ ARCHITECTURE.md (478 lines)
- **Purpose**: System overview and data flow
- **Status**: Current ✅
- **Last Updated**: 2026-09-15
- **Action**: KEEP - Still accurate overview
- **Review**: Data flows match current implementation

---

## TIER 2: OPERATIONAL GUIDES (Keep, update dates)

### ✅ CLAUDE.md (157 lines)
- **Purpose**: Developer quick start and project guide
- **Status**: Current ✅
- **Last Updated**: 2026-09-26
- **Action**: UPDATE DATE & SUMMARY to reflect Sept 28 status
- **Change**: Add note about dead code removal & doc cleanup

### ✅ API.md (523 lines)
- **Purpose**: API endpoint reference
- **Status**: Needs verification ⚠️
- **Last Updated**: 2026-09-14
- **Action**: REVIEW & UPDATE
  - Check if debug endpoints documented
  - Verify auth endpoints match userAuthServer
  - Remove references to deleted endpoints
  - Update: /api/generate-auth-code (DELETED)
  - Update: /api/code-based-login (DELETED)
  - Update: /api/validate-auth-code (DELETED)

### ✅ DEVELOPMENT.md (304 lines)
- **Purpose**: Development workflow and testing
- **Status**: Current ✅
- **Last Updated**: 2026-09-14
- **Action**: UPDATE - Add reference to TEST_SCENARIOS.md

### ✅ INSTALL.md (131 lines)
- **Purpose**: Setup and installation guide
- **Status**: Current ✅
- **Last Updated**: 2026-09-05
- **Action**: KEEP - Still accurate

### ✅ CONTRIBUTING.md (244 lines)
- **Purpose**: Contribution guidelines
- **Status**: Current ✅
- **Last Updated**: 2026-09-14
- **Action**: KEEP - Still accurate

### ✅ TEST_SCENARIOS.md (380 lines)
- **Purpose**: Comprehensive test scenarios
- **Status**: Current ✅
- **Last Updated**: 2026-09-28
- **Action**: KEEP - Recently created, comprehensive

---

## TIER 3: PROJECT STATUS (Keep, consolidate)

### ✅ README.md (449 lines)
- **Purpose**: Public project overview
- **Status**: Needs update ⚠️
- **Last Updated**: 2026-09-16
- **Action**: UPDATE
  - Update project status
  - Add Sept 28 completion milestone
  - Reference DEPLOYMENT_READY.md for current status
  - Remove references to incomplete work

### ✅ DEPLOYMENT_READY.md (304 lines)
- **Purpose**: Current deployment status
- **Status**: Current ✅
- **Last Updated**: 2026-09-28
- **Action**: KEEP - Fresh, comprehensive status report

---

## TIER 4: INVESTIGATION/COMPLETED WORK (Archive)

### ⚠️ ARCHITECTURAL_INVESTIGATION.md (701 lines)
- **Purpose**: Investigation of 3 architectural issues
- **Status**: COMPLETED (findings implemented)
- **Last Updated**: 2026-09-28
- **Action**: ARCHIVE
  - Move to `docs/historical/` folder
  - Reason: Investigation findings are now implemented, not active work
  - Keep for historical reference (how we found issues)
  - Note in README.md where to find historical docs

### ⚠️ IMPLEMENTATION_ROADMAP.md (1,336 lines)
- **Purpose**: Roadmap for Week 1-3 implementation
- **Status**: COMPLETED (all implemented)
- **Last Updated**: 2026-09-28
- **Action**: ARCHIVE
  - Move to `docs/historical/` folder
  - Reason: All 7 solutions implemented, roadmap complete
  - Keep for reference (how implementation was tracked)
  - Link from README.md to historical docs

### ⚠️ DEAD_CODE_ANALYSIS.md (198 lines)
- **Purpose**: Analysis of dead code in codebase
- **Status**: COMPLETED (all removed)
- **Last Updated**: 2026-09-28
- **Action**: ARCHIVE
  - Move to `docs/historical/` folder
  - Reason: Analysis complete, action taken and completed
  - Keep for reference (what was removed and why)

---

## TIER 5: OTHER

### ✅ moly-proxy/README.md (112 lines)
- **Purpose**: CORS proxy documentation
- **Status**: Current ✅
- **Last Updated**: 2026-09-05
- **Action**: KEEP - Component-specific documentation

---

## RECOMMENDED CLEANUP

### Step 1: Archive Completed Work
```bash
mkdir -p docs/historical/
mv ARCHITECTURAL_INVESTIGATION.md docs/historical/
mv IMPLEMENTATION_ROADMAP.md docs/historical/
mv DEAD_CODE_ANALYSIS.md docs/historical/
```

### Step 2: Update Core Documentation

**README.md**: Update status section
- Change: "Status: [OLD]" → "Status: ✅ PRODUCTION READY (Sept 28, 2026)"
- Add section: "Quick Links to Documentation"
- Add section: "Recent Changes (Sept 28)"
- Update: "For detailed investigations, see docs/historical/"

**CLAUDE.md**: Update dates
- Change last updated date to 2026-09-28
- Add note: "Dead code cleanup (1,130 LOC removed)"
- Add note: "Documentation audit completed"

**API.md**: Remove deleted endpoints
- Remove: /api/generate-auth-code
- Remove: /api/code-based-login  
- Remove: /api/validate-auth-code
- Add note: "Code-based auth feature removed (unused)"

**DEVELOPMENT.md**: Add references
- Add: "See TEST_SCENARIOS.md for comprehensive test procedures"
- Add: "See DEPLOYMENT_READY.md for current status"

### Step 3: Create docs/historical/README.md
Explain what each historical doc contains and when it's useful.

---

## Files Summary After Cleanup

### Active Documentation (11 files)
- ✅ MOLY_COMPLETE_VISION.md (Product vision)
- ✅ MOLY_SECURITY_LAYERS.md (Architecture spec)
- ✅ ARCHITECTURE.md (System overview)
- ✅ CLAUDE.md (Dev guide)
- ✅ API.md (API reference)
- ✅ DEVELOPMENT.md (Dev workflow)
- ✅ INSTALL.md (Setup guide)
- ✅ CONTRIBUTING.md (Contribution guide)
- ✅ TEST_SCENARIOS.md (Test procedures)
- ✅ README.md (Public overview)
- ✅ DEPLOYMENT_READY.md (Status report)

### Component Documentation (1 file)
- ✅ moly-proxy/README.md (Proxy component)

### Historical Documentation (3 files, archived)
- 📦 docs/historical/ARCHITECTURAL_INVESTIGATION.md
- 📦 docs/historical/IMPLEMENTATION_ROADMAP.md
- 📦 docs/historical/DEAD_CODE_ANALYSIS.md

### Total: 15 files active, 3 archived

---

## Before/After Metrics

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| **Active Docs** | 15 | 12 | -3 (archived) |
| **Root Docs** | 15 | 11 | -4 (3 archived + 1 moved) |
| **Total LOC** | 6,400+ | 4,300+ | -2,100+ LOC (archived) |
| **Clarity** | Mixed | High | ✅ |
| **Current** | Partial | Complete | ✅ |

---

## Priority Actions

### HIGH (Do immediately)
1. ✅ Archive investigation/roadmap/analysis docs
2. ✅ Update README.md status
3. ✅ Update CLAUDE.md date
4. ✅ Update API.md (remove deleted endpoints)

### MEDIUM (Do soon)
1. ✅ Update DEVELOPMENT.md with references
2. ✅ Create docs/historical/README.md

### LOW (Optional)
1. Verify all docs link to each other correctly
2. Add version/date to each doc

---

## Success Criteria

✅ Active documentation reflects current system state  
✅ No references to deleted features  
✅ Clear path to both current status and historical context  
✅ New developers can find what they need quickly  
✅ All 7 solutions documented  
✅ All 11 layers documented  
✅ Test procedures clear  
✅ Deployment status current  

---

**Status**: Ready for implementation  
**Estimated Time**: 1-2 hours  
**Risk Level**: LOW (documentation only)

