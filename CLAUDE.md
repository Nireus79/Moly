# Claude Code - Moly Project Guide

**For all Claude Code sessions working on this project**

---

## Quick Navigation

- **Read this first**: `DOCUMENTATION.md` (root of project)
- **Then read**: `MOLY_V2_ARCHITECTURE/01_VISION_AND_PHILOSOPHY.md`
- **Implementation reference**: `MOLY_V2_ARCHITECTURE/` (12 documents)
- **Archived docs**: `ARCHIVE/` (ignore unless historical context needed)

---

## Key Principles

### Moly is NOT:
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

## Current State (Sep 7, 2026)

### ✅ Complete
- V2 architecture designed (12 documents in MOLY_V2_ARCHITECTURE/)
- Type definitions & models
- 6 tool files (LLM wrapper, suggestion generator, safety checker, etc.)
- 5 agent files (conversation, learning, context manager, risk monitor)
- API handlers scaffolding
- Extension UI components

### 🚀 In Progress
- Wiring agents to handlers (conversation generation working with context-aware suggestions)
- Backend initialization without LLM API key (graceful degradation)
- Database integration (TODOs in agent files)
- V1 fallback implementation

### ⏳ TODO
- LLM integration (real Claude API calls)
- Database persistence
- Learning loop (tracking user patterns)
- Full E2E testing

---

## When Making Changes

1. **Check philosophy first**: `01_VISION_AND_PHILOSOPHY.md`
   - Ensure change respects core principles
   - No surveillance of contacts
   - User autonomy maintained

2. **Check architecture**: Relevant doc from `MOLY_V2_ARCHITECTURE/`
   - Understand the design intent
   - Follow established patterns

3. **Check API contract**: `05_API_SPECIFICATION.md`
   - Responses must match spec
   - Request/response types defined

4. **Update memory**: Save what you learned for next session

---

## File Locations

```
~/vs_projects/Moly/
├── DOCUMENTATION.md          ← START HERE
├── CLAUDE.md                 ← THIS FILE
├── MOLY_V2_ARCHITECTURE/     ← AUTHORITATIVE DOCS (12 files)
│   ├── 00_INDEX.md
│   ├── 01_VISION_AND_PHILOSOPHY.md
│   ├── 02_BACKEND_AGENT_ARCHITECTURE.md
│   ├── ... (9 more)
│   └── README.md
├── ARCHIVE/                  ← OLD DOCS (ignore for development)
│   ├── PHASE_1_DOCS/
│   ├── OLD_ROADMAPS/
│   ├── TESTING_GUIDES/
│   └── DEPRECATED/
└── Moly/                     ← PROJECT SOURCE
    ├── moly-go/              ← Backend
    ├── moly-extension/       ← Extension
    ├── moly-proxy/           ← CORS proxy
    ├── README.md
    ├── INSTALL.md
    └── CONTRIBUTING.md
```

---

## Common Tasks

### "I need to understand the system"
→ Read `MOLY_V2_ARCHITECTURE/01_VISION_AND_PHILOSOPHY.md`
→ Then read `02_BACKEND_AGENT_ARCHITECTURE.md`

### "I need to implement an endpoint"
→ Check `05_API_SPECIFICATION.md` for contract
→ Reference `02_BACKEND_AGENT_ARCHITECTURE.md` for agent flow
→ Implement according to spec

### "I need to understand privacy rules"
→ Read `01_VISION_AND_PHILOSOPHY.md` (red lines section)
→ Read `08_PRIVACY_SECURITY.md` for implementation details

### "I'm not sure if something is in scope"
→ Check `01_VISION_AND_PHILOSOPHY.md` for "What Moly Does/Doesn't Do"
→ Check `12_IMPLEMENTATION_ROADMAP.md` for phase breakdown

### "I broke something, need to understand what it should do"
→ Reference `05_API_SPECIFICATION.md` for contract
→ Reference component architecture doc for flow
→ Never assume - always verify against V2 docs

---

## Do NOT

❌ Use old phase documentation for implementation
❌ Assume architecture based on code state
❌ Add features not in V2 roadmap
❌ Break privacy principles (read philosophy doc)
❌ Implement without consulting API spec

---

## Session Notes

Each session should update `MEMORY.md` in `/home/nireus79/.claude/projects/-home-nireus79-vs-projects-Moly/memory/` with:
- What you learned about the project
- What you changed
- What's blocking progress
- What's next

This keeps knowledge persistent across sessions.

---

**Last Updated**: 2026-09-07  
**Version**: 1.0  
**For**: All Claude Code sessions on Moly project

