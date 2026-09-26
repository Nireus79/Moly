# Claude Code - Moly Project Guide

**For all Claude Code sessions working on this project**

---

## Quick Start: Read These First (In Order)

1. **[MOLY_COMPLETE_VISION.md](./MOLY_COMPLETE_VISION.md)** — What Moly is, why it exists, core principles
2. **[MOLY_SECURITY_LAYERS.md](./MOLY_SECURITY_LAYERS.md)** — The 11-layer architecture (authoritative)
3. **[ARCHITECTURE.md](./ARCHITECTURE.md)** — System overview and data flow
4. **Session memory** — `/memory/MEMORY.md` tracks what was done across sessions

---

## Key Principles

### Moly Is NOT:
- A surveillance tool
- An enforcer
- An auto-responder
- A content reader
- A decision maker

### Moly IS:
- A thinking partner
- A Socratic coach
- A context keeper
- An ethical educator
- A communication advisor

**User stays in control. Moly helps user think, not tells user what to do.**

---

## Current State (Sept 26, 2026)

### ✅ Complete
- V2 architecture fully designed (11 layers, all wired)
- Type definitions & models complete
- 6 agent files with full implementation
- Contact API with full CRUD
- Database integration working
- All 11 security layers functional
- Safety/ethics checking operational
- Learning system integrated
- Build passing: `go build ./...` ✅

### 🎯 Production Ready
- No critical bugs
- All layers have implementation code
- Error handling with graceful degradation
- Response metadata carries layer results
- Build verified and tested

---

## When Making Changes

1. **Check vision first** → [MOLY_COMPLETE_VISION.md](./MOLY_COMPLETE_VISION.md)
   - Ensure change respects core principles
   - No surveillance, user autonomy maintained
   - Verify alignment with "thinking partner" philosophy

2. **Check architecture** → [MOLY_SECURITY_LAYERS.md](./MOLY_SECURITY_LAYERS.md)
   - Understand which layer you're touching
   - Verify data flow (extraction → storage → retrieval → use)
   - Check Layer X prerequisites/gates

3. **Check API contract** → [API.md](./API.md)
   - Responses must match spec
   - Request/response types defined

4. **Check memory** → `/memory/MEMORY.md`
   - What was discovered in prior sessions
   - What's blocked, what's working
   - Session history and context

---

## File Structure

```
Moly/
├── README.md                  ← Public overview
├── CLAUDE.md                  ← THIS FILE: Guide for Claude Code
├── MOLY_COMPLETE_VISION.md    ← What Moly is (product vision)
├── MOLY_SECURITY_LAYERS.md    ← Architecture: 11-layer model
├── ARCHITECTURE.md            ← System overview & data flow
├── API.md                     ← API reference
├── DEVELOPMENT.md             ← Dev workflow & testing
├── CONTRIBUTING.md            ← Contribution guidelines
├── INSTALL.md                 ← Setup & installation
├── moly-go/                   ← Backend (Go)
│   ├── main.go
│   ├── agents/
│   ├── models/
│   ├── database/
│   └── tools/
├── moly-extension/            ← Extension (TypeScript)
├── moly-proxy/                ← CORS proxy
└── memory/                    ← Session memory (auto-persistent)
    ├── MEMORY.md              ← Index of what we learned
    └── *.md                   ← Details about each discovery
```

---

## Common Tasks

### "I need to understand the system"
1. Read [MOLY_COMPLETE_VISION.md](./MOLY_COMPLETE_VISION.md) (what)
2. Read [MOLY_SECURITY_LAYERS.md](./MOLY_SECURITY_LAYERS.md) (how)
3. Read [ARCHITECTURE.md](./ARCHITECTURE.md) (system view)

### "I need to implement a feature"
1. Check [MOLY_SECURITY_LAYERS.md](./MOLY_SECURITY_LAYERS.md) — which layer does it belong to?
2. Check [API.md](./API.md) — what's the contract?
3. Check [DEVELOPMENT.md](./DEVELOPMENT.md) — testing & verification

### "I'm not sure if something is in scope"
→ Read [MOLY_COMPLETE_VISION.md](./MOLY_COMPLETE_VISION.md) — "Core Capabilities" section

### "I broke something, need to understand what it should do"
1. Check [API.md](./API.md) for contract
2. Check [MOLY_SECURITY_LAYERS.md](./MOLY_SECURITY_LAYERS.md) for layer behavior
3. Check `/memory/MEMORY.md` for recent changes

### "How do I test this?"
→ See [DEVELOPMENT.md](./DEVELOPMENT.md)

---

## Do NOT

❌ Assume architecture based on code state — check MOLY_SECURITY_LAYERS.md first  
❌ Break privacy principles — read MOLY_COMPLETE_VISION.md  
❌ Implement without checking API spec — read API.md  
❌ Skip understanding the vision — that's what makes code decisions  

---

## Session Notes

**Always update `/memory/MEMORY.md`** with:
- What you learned about the project
- What you changed (summarize commits)
- What's blocking progress
- What's next

This keeps knowledge persistent across sessions and helps future Claude Code agents understand context.

---

**Last Updated**: 2026-09-26  
**Version**: 2.0 (Cleaned up, references actual current docs)  
**For**: All Claude Code sessions on Moly project
