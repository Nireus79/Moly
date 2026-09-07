# Moly Documentation - Source of Truth

**Last Updated**: September 7, 2026  
**Current Phase**: V2 Implementation

---

## 📚 Primary Documentation

### ✅ V2 Architecture (AUTHORITATIVE)
All current development should reference these documents:

**Location**: `/MOLY_V2_ARCHITECTURE/`

1. **00_INDEX.md** — Quick navigation guide
2. **01_VISION_AND_PHILOSOPHY.md** — Core principles (READ THIS FIRST)
3. **02_BACKEND_AGENT_ARCHITECTURE.md** — Agent system design
4. **03_AGENT_PROMPTS.md** — LLM prompt templates
5. **04_FRONTEND_ARCHITECTURE.md** — Extension UI architecture
6. **05_API_SPECIFICATION.md** — API endpoints & contracts
7. **06_DATABASE_SCHEMA.md** — Database design
8. **07_DEPLOYMENT_GUIDE.md** — Deployment & setup
9. **08_PRIVACY_SECURITY.md** — Privacy & security model
10. **09_MIGRATION_PATH.md** — V1→V2 migration strategy
11. **10_ERROR_HANDLING.md** — Error handling patterns
12. **11_SUCCESS_METRICS.md** — What defines success
13. **12_IMPLEMENTATION_ROADMAP.md** — Phase breakdown & timeline

---

## 📋 Current Project Structure

### Main Project Root: `/Moly/`

**Core Directories:**
- `moly-go/` — Backend (Go + Claude API)
- `moly-extension/` — Browser extension (TypeScript/React)
- `moly-proxy/` — CORS proxy (Node.js)
- `moly-installer/` — Installation scripts

**Key Files to Use:**
- `Moly/README.md` — Main project overview (update as needed)
- `Moly/CONTRIBUTING.md` — Contribution guidelines
- `Moly/INSTALL.md` — Installation instructions
- `moly-extension/CLAUDE.md` — Extension dev notes
- `moly-extension/docs/` — Extension-specific docs

---

## 🔄 For Each Component

### Backend (`moly-go/`)
- Use `moly-go/README.md` for backend overview
- Use `MOLY_V2_ARCHITECTURE/02_BACKEND_AGENT_ARCHITECTURE.md` for agent design
- Use `MOLY_V2_ARCHITECTURE/05_API_SPECIFICATION.md` for API contracts
- Use `MOLY_V2_ARCHITECTURE/06_DATABASE_SCHEMA.md` for DB design

### Extension (`moly-extension/`)
- Use `moly-extension/README.md` for extension overview
- Use `MOLY_V2_ARCHITECTURE/04_FRONTEND_ARCHITECTURE.md` for UI architecture
- Use `MOLY_V2_ARCHITECTURE/05_API_SPECIFICATION.md` for API contracts
- Use `moly-extension/CLAUDE.md` for dev setup

### Privacy & Security
- Use `MOLY_V2_ARCHITECTURE/08_PRIVACY_SECURITY.md` for all privacy decisions
- Use `MOLY_V2_ARCHITECTURE/01_VISION_AND_PHILOSOPHY.md` for ethical principles

---

## 📦 Archive

**Location**: `/ARCHIVE/`

Old/obsolete documentation organized by category:
- `PHASE_1_DOCS/` — Phase 1.1 & 1.2 implementation docs
- `OLD_ROADMAPS/` — Previous roadmaps and summaries
- `TESTING_GUIDES/` — V1 testing & verification docs
- `DEPRECATED/` — Deprecated architecture & endpoints

**When to use Archive:**
- Historical context only
- Not for current implementation
- Reference only if V2 docs don't cover something

---

## ⚠️ What NOT to Use

❌ **DON'T USE:**
- `Moly/COMPLETE_SUMMARY.md` (archived)
- `Moly/ARCHITECTURE_COMPLETE.md` (archived, V1 only)
- Phase 1 planning documents
- Old testing guides
- `DEPRECATED_ENDPOINTS.md`

✅ **USE INSTEAD:**
- `MOLY_V2_ARCHITECTURE/` folder documents
- This file (DOCUMENTATION.md)
- Component-specific README files

---

## 🎯 For Claude Code Sessions

When working on Moly:

1. **First**: Read `MOLY_V2_ARCHITECTURE/01_VISION_AND_PHILOSOPHY.md`
2. **Then**: Reference the specific V2 architecture doc for your task
3. **For implementation**: Use `05_API_SPECIFICATION.md` and component architecture
4. **For decisions**: Consult `08_PRIVACY_SECURITY.md` and philosophy doc
5. **Never assume**: Always verify against V2 docs, not old phase docs

---

## 📝 Memory Reference

For future sessions, this file should be in memory as the authoritative structure. Reference `MOLY_V2_ARCHITECTURE/` for all implementation decisions.

---

**Status**: ✅ Documentation organized and current  
**Last organized**: 2026-09-07 14:33 UTC  
**Responsible**: Claude Code Session

