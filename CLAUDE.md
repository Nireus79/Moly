# Claude Code - Moly Project Guide

**For all Claude Code sessions working on this project**

---

## Current Status (October 3, 2026)

✅ **Production Ready** - All systems operational  
✅ **Loop Pattern Implemented** - Clarifications properly handled  
✅ **All 11 Layers Operational** - Complete end-to-end wiring verified  
✅ **Zero Critical Bugs** - All tests passing

---

## Quick Navigation

1. **Understand the System:** Read [MOLY_11_LAYER_SYSTEM.md](./MOLY_11_LAYER_SYSTEM.md) first (authoritative)
2. **Check Architecture:** Review [ARCHITECTURE.md](./ARCHITECTURE.md) for data flow
3. **Current Session:** Check `/memory/MEMORY.md` for what was done recently
4. **Check API:** See [API.md](./API.md) for endpoints

---

## Key Files

**Essential Documentation:**
- `MOLY_11_LAYER_SYSTEM.md` - **Authoritative** orchestrator specification (11 layers, flow diagram, loop pattern)
- `MOLY_COMPLETE_VISION.md` - Product vision and core principles
- `ARCHITECTURE.md` - System design and data flow
- `MESSAGE_SUMMARY_SYSTEM.md` - **Cache optimization** (FIX #11, 67% performance improvement)
- `API.md` - Complete REST API reference
- `DEVELOPMENT.md` - Development workflow and testing
- `INSTALL.md` - Setup and installation
- `CONTRIBUTING.md` - Contribution guidelines

**Memory:**
- `/memory/MEMORY.md` - Session history and discoveries

---

## Core Principles (Updated Oct 3, 2026)

### Moly's Philosophy
- **User stays in control** - Moly helps user think, not decides for user
- **Every message is data gold** - Extract, evaluate, save, accumulate context
- **Maturity is THE key** - System behavior driven by context maturity, must improve each message
- **Goal-aligned gaps** - Ask gaps that block user's goal, not generic profile gaps
- **Natural conversation** - Follow user's intent and goal, not data extraction order
- **Clarify before blocking** - Layers 1-10 ask questions, Layer 11 only denies
- **No false positives** - Greetings never flagged, crisis detection <5% false positive rate

### Gap Prioritization (Critical for Natural Conversation)
```
DO NOT:
  Gap: "You mentioned preferring respectful. How does this apply?"
  ← User just explained how it applies!

DO:
  Gap: "In your first message, should you be direct about interests, or gradual?"
  ← Helps them write the actual message

Prioritization order:
  1. Goal-blocking gaps (HIGH) - What blocks achieving stated goal?
  2. Goal-supporting gaps (MEDIUM) - What helps achieve goal?
  3. Safety gaps (ALWAYS HIGH) - Any concerns?
  4. Profile gaps (LOW) - Nice-to-know context?

Never ask gaps about extracted data - use extracted data as context instead.
```

### Maturity Must Improve
```
Each message should increase maturity by contributing:
  - New extracted entities (high confidence)
  - Deeper context about goal
  - Clarification on previous ambiguities

If maturity stays same/decreases: System is broken (stuck loop).
Check: Does Layer 3 use ALL accumulated entities, not just current message?
```

### Architecture Pattern
```
First message:    [L1-11 full evaluation] → Ask prioritized gaps (aligned to goal)
Clarification:    [L1 extract] → [L3-11 with accumulated context] → Re-assess maturity
                                                                    → Ask remaining goal-aligned gaps or help
```

---

## Current Implementation Status

| Component | Status | Details |
|-----------|--------|---------|
| Layer 1-3 | ✅ | Extraction, principles, maturity |
| Layer 4-7 | ✅ | Gap detection, conflicts, ambiguity |
| Layer 8-11 | ✅ | Socratic, shifts, persistent questioning, denial |
| Loop Pattern | ✅ | Implemented Oct 3 |
| Data Flow | ✅ | All layer results → metadata |
| Wiring | ✅ | End-to-end verified |
| Build | ✅ | 21 MB, clean compilation |

---

## Critical Bug Fixes (Session 30 - October 5, 2026)

### FIXED: Message Summary Cache System (FIX #11, Commit beb3875)

**Three Critical Bugs Found & Fixed:**

1. **Type Conversion Bug**: Cache builder tried to convert MessageSummary as `map[string]interface{}` - type assertion always failed, cache was always empty
   
2. **Data Loss Bug**: Layers returned empty results when using cache, losing extracted data
   
3. **Missing Imports**: Added `models.MessageSummary` references but forgot imports in 6 layers

**Result**: 67% performance optimization (700ms → 0ms per cached message) is now fully functional

**See**: [MESSAGE_SUMMARY_SYSTEM.md](./MESSAGE_SUMMARY_SYSTEM.md) for complete details

---

## When Making Changes

1. **Check architecture first** → [MOLY_11_LAYER_SYSTEM.md](./MOLY_11_LAYER_SYSTEM.md)
   - Which layer does this belong to?
   - What are the prerequisites?
   - How does it flow through the system?

2. **Understand the vision** → [MOLY_COMPLETE_VISION.md](./MOLY_COMPLETE_VISION.md)
   - Respect core principles
   - Maintain user autonomy
   - No surveillance or decision-making

3. **Check the API** → [API.md](./API.md)
   - What's the contract?
   - What endpoints are affected?

4. **Verify data flow** → [ARCHITECTURE.md](./ARCHITECTURE.md)
   - How does it integrate end-to-end?
   - What layers depend on this?

5. **Update memory** → `/memory/MEMORY.md`
   - What changed?
   - What was discovered?
   - What's blocking?

---

## Development Workflow

### Building
```bash
cd /home/nireus79/vs_projects/Moly/Moly/moly-go
go build -o ../bin/moly .
```

### Testing
See [DEVELOPMENT.md](./DEVELOPMENT.md) for test suite and procedures.

### Committing
```bash
git add <files>
git commit -m "Clear description of what changed"
# Commit message will auto-include attribution
```

### Pushing
```bash
git push origin master
```

---

## File Structure

```
Moly/
├── README.md                      ← Public overview (current state)
├── CLAUDE.md                      ← THIS FILE: Claude Code guide
├── MOLY_COMPLETE_VISION.md        ← Product vision
├── MOLY_11_LAYER_SYSTEM.md        ← Orchestrator (authoritative)
├── ARCHITECTURE.md                ← System data flow
├── API.md                         ← API reference
├── DEVELOPMENT.md                 ← Dev workflow & testing
├── INSTALL.md                     ← Setup & installation
├── CONTRIBUTING.md                ← Contribution guidelines
├── moly-go/                       ← Backend (Go)
│   ├── main.go
│   ├── agents/
│   │   ├── unified_orchestrator.go    ← Loop pattern entry point
│   │   ├── conversation_agent.go      ← Layer data consumption
│   │   ├── layer4_gap_detector.go     ← Gap detection
│   │   └── (other layers)
│   ├── models/
│   ├── database/
│   ├── tools/
│   └── bin/moly                   ← Compiled binary
├── moly-extension/                ← Extension (TypeScript)
├── moly-proxy/                    ← CORS proxy
└── memory/                        ← Session memory
    └── MEMORY.md                  ← What we learned
```

---

## Loop Pattern Details (Oct 3, 2026)

### The Problem It Solves
Previously, every message (including clarifications) ran Layers 1-11 fresh, causing:
- Re-extraction of already-known information
- Re-detection of already-answered gaps
- Redundant questions to user

### The Solution
When a user provides clarification (detected by: has extracted entities + analysis context):
1. **Skip Layers 1-3** - No re-extraction
2. **Jump to Layer 4** - With accumulated context
3. **Re-evaluate gaps** - Against combined information
4. **Proceed forward** - If gaps resolved; ask only new gaps if remain

### Implementation
File: `agents/unified_orchestrator.go`, ProcessMessage function:
```go
// Detect if answering clarification
isAnsweringClarification := analysisCtx.CurrentMessage != "" && 
    analysisCtx.ExtractedConfidence > 0 && 
    len(analysisCtx.ExtractedEntities) > 0

// If clarification, skip to Layer 4
if isAnsweringClarification {
    startLayer = 3 // Layer 4 is at index 3
}

// Run from Layer 4 onwards
for i, layer := range uo.layers {
    if i < startLayer { continue } // Skip L1-3
    // Process L4-11...
}
```

---

## Common Tasks

### "System seems to be asking redundant questions"
→ Check if loop pattern is detecting clarifications properly
→ Verify `isAnsweringClarification` logic in `unified_orchestrator.go`
→ Check if gap analysis rejects already-answered items

### "Layer X output isn't flowing to response"
→ Check `conversation_agent.go` - all layers are read there
→ Verify metadata is being stored (`response.Metadata[...]`)
→ Check response generation uses the metadata

### "Gate isn't being enforced"
→ Verify early return logic (Layer 6, 11 have critical gates)
→ Check log messages for gate status
→ Ensure gate check happens BEFORE response generation

### "Build is failing"
```bash
cd moly-go && go build ./... 2>&1 | head -20
```
Should show no errors.

---

## Do NOT

❌ Assume architecture from code state - read MOLY_11_LAYER_SYSTEM.md  
❌ Break privacy principles - read MOLY_COMPLETE_VISION.md  
❌ Bypass the loop pattern - it's the architectural foundation  
❌ Add re-extraction logic on clarifications - use accumulated context  
❌ Skip end-to-end verification - test the full pipeline  

---

## Session Notes Template

When you finish work, update `/memory/MEMORY.md` with:
- **What was done** - Which layers, which fixes, what changed
- **What was discovered** - Root causes, patterns, insights
- **What's blocking** - Any remaining issues
- **What's next** - Recommended next steps

This keeps knowledge persistent across sessions.

---

## Quick Debugging

**Check compilation:**
```bash
go build ./...
```

**Check loop pattern:**
```bash
grep -n "isAnsweringClarification" agents/unified_orchestrator.go
```

**Check layer wiring:**
```bash
grep -n "Layer[2-9]" agents/conversation_agent.go | wc -l
```

**Check metadata:**
```bash
grep -n "response.Metadata" agents/conversation_agent.go | wc -l
```

---

## Architecture Layers Reference

| Layer | File | Responsibility |
|-------|------|-----------------|
| 1 | context_extractor.go | Extract contacts, intentions, values |
| 2 | constitutional_evaluator.go | Scan principles |
| 3 | maturity_calculator.go | Assess context quality |
| 4 | layer4_gap_detector.go | Detect missing info |
| 5 | conflict_detector.go | Find contradictions |
| 6 | ambiguity_handler.go | Clarify unclear requests |
| 7 | violation_clarifier.go | Evaluate ethical concerns |
| 8 | socratic_selector.go | Ask philosophical questions |
| 9 | subject_shift_detector.go | Detect topic changes |
| 10 | persistent_handler.go | Help reason through concerns |
| 11 | denial_handler.go | Absolute blocks only |

---

**Last Updated:** October 3, 2026  
**Version:** 2.2 (Production Ready, Loop Pattern Implemented)  
**For:** All Claude Code sessions on Moly project
