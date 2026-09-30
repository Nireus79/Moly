# Moly Project - Documentation Index

**Last Updated**: September 30, 2026  
**Status**: ✅ Production Ready | 2,430+ LOC | 50+ Tests | 100% Pass Rate

---

## 🚀 Quick Start

**New to the project?** Start here:
1. [README.md](README.md) - Project overview
2. [CLAUDE.md](CLAUDE.md) - Development guide for Claude Code
3. [MOLY_COMPLETE_VISION.md](MOLY_COMPLETE_VISION.md) - What Moly is and why

**Need to deploy?** Go to:
- [Production Deployment Guide](docs/production/PRODUCTION_DEPLOYMENT_GUIDE.md)

**Need architecture details?** Check:
- [MOLY_11_LAYER_SYSTEM.md](MOLY_11_LAYER_SYSTEM.md) - 11-layer orchestrator
- [ARCHITECTURE.md](ARCHITECTURE.md) - System overview

---

## 📚 Documentation Structure

### Root Directory (Active)
Core documentation for understanding and working on the project:

| Document | Purpose | Read When |
|----------|---------|-----------|
| [README.md](README.md) | Project intro & overview | First time |
| [CLAUDE.md](CLAUDE.md) | Claude Code development guide | Starting work |
| [MOLY_COMPLETE_VISION.md](MOLY_COMPLETE_VISION.md) | Product vision & principles | Understanding goals |
| [MOLY_11_LAYER_SYSTEM.md](MOLY_11_LAYER_SYSTEM.md) | 11-layer orchestrator design | Understanding architecture |
| [ARCHITECTURE.md](ARCHITECTURE.md) | System design overview | Implementation work |
| [API.md](API.md) | API reference | Integrating/calling APIs |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Development workflow | Setting up/testing |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution guidelines | Contributing code |
| [INSTALL.md](INSTALL.md) | Installation & setup | Getting started |
| [SELF_AWARENESS_OPTIMIZATION.md](SELF_AWARENESS_OPTIMIZATION.md) | Phase 5 technical details | Understanding Phase 5 |

### docs/active/
**Current work and implementation notes**

| Document | Purpose |
|----------|---------|
| [CURRENT_STATUS.md](docs/active/CURRENT_STATUS.md) | Latest session work & status |
| [IMPLEMENTATION_NOTES.md](docs/active/IMPLEMENTATION_NOTES.md) | Recent fixes & changes |
| [BUG_TRACKER.md](docs/active/BUG_TRACKER.md) | Known issues & fixes |

### docs/production/
**Deployment & operations**

| Document | Purpose |
|----------|---------|
| [PRODUCTION_DEPLOYMENT_GUIDE.md](docs/production/PRODUCTION_DEPLOYMENT_GUIDE.md) | How to deploy to production |
| [MONITORING.md](docs/production/MONITORING.md) | How to monitor the system |
| [ROLLBACK_PLAN.md](docs/production/ROLLBACK_PLAN.md) | Emergency rollback procedures |

### docs/design/
**Technical design documents**

| Document | Purpose |
|----------|---------|
| [CONSTITUTIONAL_SYSTEM.md](docs/design/CONSTITUTIONAL_SYSTEM.md) | Principle-based evaluation |
| [ORCHESTRATOR_DESIGN.md](docs/design/ORCHESTRATOR_DESIGN.md) | 11-layer system design |
| [DATABASE_SCHEMA.md](docs/design/DATABASE_SCHEMA.md) | Database model & schema |
| [API_DESIGN.md](docs/design/API_DESIGN.md) | API endpoint design |

### docs/reference/
**Reference materials**

| Document | Purpose |
|----------|---------|
| [GLOSSARY.md](docs/reference/GLOSSARY.md) | Terms & definitions |
| [PERFORMANCE_METRICS.md](docs/reference/PERFORMANCE_METRICS.md) | Performance benchmarks |
| [ERROR_CODES.md](docs/reference/ERROR_CODES.md) | Error reference guide |

### docs/archive/
**Historical documentation (kept for reference)**

- **phase5/** - Phase 5 implementation work
- **weeks_1-4/** - Foundation weeks (1-4) implementation
- **historical/** - Earlier architectural investigations

---

## 📊 Implementation Status

### Latest Session (Sept 30, 2026)

**Fixes & Improvements**:
- ✅ Factory pattern for ConversationAgent initialization (enforced order)
- ✅ Removed 3 orphaned fields from conversationAgent
- ✅ Fixed 2 CRITICAL FK constraint timing bugs
- ✅ Fixed nil error handling in response channel
- ✅ Started rows.Err() fixes (2/11 completed)

**Current Metrics**:
- Total LOC: 2,430+
- Total Tests: 50+
- Test Pass Rate: 100%
- Build Status: ✅ Clean
- Known Bugs: 9 rows.Err() checks needed (tracked in BUG_TRACKER.md)

---

## 🔍 Key Navigation Paths

### By Role

**Project Manager**:
1. [README.md](README.md) - Overview
2. [CURRENT_STATUS.md](docs/active/CURRENT_STATUS.md) - What's being done
3. [MOLY_COMPLETE_VISION.md](MOLY_COMPLETE_VISION.md) - Goals

**Developer**:
1. [CLAUDE.md](CLAUDE.md) - Development guide
2. [DEVELOPMENT.md](DEVELOPMENT.md) - Workflow
3. [MOLY_11_LAYER_SYSTEM.md](MOLY_11_LAYER_SYSTEM.md) - Architecture
4. [BUG_TRACKER.md](docs/active/BUG_TRACKER.md) - Known issues

**DevOps/Operations**:
1. [INSTALL.md](INSTALL.md) - Setup
2. [PRODUCTION_DEPLOYMENT_GUIDE.md](docs/production/PRODUCTION_DEPLOYMENT_GUIDE.md) - Deploy
3. [MONITORING.md](docs/production/MONITORING.md) - Operate
4. [ROLLBACK_PLAN.md](docs/production/ROLLBACK_PLAN.md) - Emergency response

**Architect**:
1. [MOLY_COMPLETE_VISION.md](MOLY_COMPLETE_VISION.md) - Vision
2. [MOLY_11_LAYER_SYSTEM.md](MOLY_11_LAYER_SYSTEM.md) - Design
3. [ARCHITECTURE.md](ARCHITECTURE.md) - Overview
4. [docs/design/](docs/design/) - Detailed designs

---

## 🎯 By Task

**"I want to get started quickly"**:
→ [README.md](README.md) → [INSTALL.md](INSTALL.md) → [DEVELOPMENT.md](DEVELOPMENT.md)

**"I need to understand the system"**:
→ [MOLY_COMPLETE_VISION.md](MOLY_COMPLETE_VISION.md) → [MOLY_11_LAYER_SYSTEM.md](MOLY_11_LAYER_SYSTEM.md) → [ARCHITECTURE.md](ARCHITECTURE.md)

**"I need to deploy this"**:
→ [PRODUCTION_DEPLOYMENT_GUIDE.md](docs/production/PRODUCTION_DEPLOYMENT_GUIDE.md) → [MONITORING.md](docs/production/MONITORING.md)

**"I need to fix a bug"**:
→ [BUG_TRACKER.md](docs/active/BUG_TRACKER.md) → [DEVELOPMENT.md](DEVELOPMENT.md) → [ARCHITECTURE.md](ARCHITECTURE.md)

---

## 📈 Statistics

| Metric | Value |
|--------|-------|
| Total Documentation Files | 37 |
| Root Documents | 10 |
| Active Work Docs | 3 |
| Production Docs | 3 |
| Design Docs | 4 |
| Reference Docs | 3 |
| Archive Docs | 11 |
| Lines of Code | 2,430+ |
| Tests | 50+ |
| Test Pass Rate | 100% |
| Known Bugs | 9 (rows.Err() checks) |

---

## 🔄 Documentation Maintenance

**Last Reorganized**: September 30, 2026
**Organization Status**: In Progress (Phase 2 of 2)

**Phase 1 (Complete)**: Master index & structure defined
**Phase 2 (In Progress)**: Moving & consolidating docs to new structure

---

## 📝 Related Resources

- **Memory System**: `/home/nireus79/.claude/projects/-home-nireus79-vs-projects-Moly/memory/MEMORY.md`
- **Source Code**: `moly-go/` directory
- **Tests**: `moly-go/**/*_test.go`
- **Configuration**: `config/` directory

---

**Questions?** Check [GLOSSARY.md](docs/reference/GLOSSARY.md) or refer to [CLAUDE.md](CLAUDE.md) for development help.
