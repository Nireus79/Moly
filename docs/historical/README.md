# Historical Documentation

This folder contains completed investigations, roadmaps, and analyses that are no longer actively maintained but kept for historical reference.

## Contents

### ARCHITECTURAL_INVESTIGATION.md
**Date**: Sept 28, 2026  
**Purpose**: Investigation of 3 major architectural issues discovered during system audit

Covers:
- Issue 1: Workflow priority bug (intent detection overridden by gap clarification)
- Issue 2: Cascading LLM calls (9-15 calls per message, timeouts)
- Issue 3: Self-awareness gap (no reserved contact for Moly)

**Status**: ✅ COMPLETE - All findings implemented in sessions C-30n Session 2-4

**When to Read**: Understanding how architectural issues were identified and analyzed.

---

### IMPLEMENTATION_ROADMAP.md
**Date**: Sept 28, 2026  
**Purpose**: Week-by-week implementation roadmap for 7 critical solutions

Covers:
- Week 1: Critical solutions (Intent-first, Tiered timeouts, Graceful fallback)
- Week 2: Performance solutions (Phase-aware gaps, Result caching)
- Week 3: Self-awareness solutions (System contact, Self-reference prompts)
- Week 4+: Future architecture improvements

**Status**: ✅ COMPLETE - All 7 solutions implemented and wired

**When to Read**: Understanding the implementation strategy and how solutions were prioritized.

---

### DEAD_CODE_ANALYSIS.md
**Date**: Sept 28, 2026  
**Purpose**: Comprehensive analysis of dead code in the codebase

Covers:
- mode_transition.go (870 LOC) - Completely unused
- Incomplete auth handlers (260 LOC) - Replaced by proper auth system
- Debug endpoints (250 LOC) - Kept as developer tools
- Impact analysis and removal plan

**Status**: ✅ COMPLETE - All dead code identified and removed

**When to Read**: Understanding what dead code existed and why it was removed.

---

## Using Historical Documentation

These documents are archived because:
1. **Investigations**: Questions they answer have been resolved
2. **Roadmaps**: Implementation plans are complete
3. **Analyses**: Issues have been addressed

However, they remain valuable for:
- Understanding project history and how decisions were made
- Learning about the investigation process
- Reference when similar issues arise
- Onboarding new team members on architectural context

---

## Current Documentation

For current, active documentation, see:
- **MOLY_COMPLETE_VISION.md** - Product vision
- **MOLY_11_LAYER_SYSTEM.md** - 11-layer architecture
- **DEPLOYMENT_READY.md** - Current deployment status
- **TEST_SCENARIOS.md** - Testing procedures
- **API.md** - API reference
- **CLAUDE.md** - Developer guide

---

**Last Updated**: Sept 28, 2026
