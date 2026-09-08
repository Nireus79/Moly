# Documentation Index

**Last Updated**: September 8, 2026  
**Purpose**: Quick reference guide to find what you need

---

## 🎯 Start Here

**First time?** → Read [`CLAUDE.md`](CLAUDE.md)  
**Want overview?** → Read [`DOCUMENTATION.md`](DOCUMENTATION.md)  
**In a hurry?** → See [`QUICKSTART.md`](QUICKSTART.md)

---

## 📚 Documentation by Category

### Architecture & Design (Authoritative)

| Document | Purpose | Status |
|----------|---------|--------|
| [`MOLY_V2_ARCHITECTURE/`](MOLY_V2_ARCHITECTURE/) | Complete system design (12 docs) | ✅ Current |
| [`MOLY_V2_1_ARCHITECTURE.md`](MOLY_V2_1_ARCHITECTURE.md) | V2.1 architecture overview | ✅ Current |

### Getting Started

| Document | Purpose | Status |
|----------|---------|--------|
| [`INSTALL.md`](INSTALL.md) | Installation instructions | ✅ Current |
| [`QUICKSTART.md`](QUICKSTART.md) | Quick start guide | ✅ Current |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Contributing guidelines | ✅ Current |

### Implementation Status & Roadmap

| Document | Purpose | Status |
|----------|---------|--------|
| [`GAPS_CLOSED_IMPLEMENTATION.md`](GAPS_CLOSED_IMPLEMENTATION.md) | All Phase 1.2 gaps closed (Sept 8) | ✅ Current |
| [`PHASE_1_2_FINAL_STATUS.md`](PHASE_1_2_FINAL_STATUS.md) | Phase 1.2 completion status | ✅ Current |
| [`PHASE_1_2_FINAL_SUMMARY.md`](PHASE_1_2_FINAL_SUMMARY.md) | Phase 1.2 summary | ✅ Current |
| [`V2_1_IMPLEMENTATION_ROADMAP.md`](V2_1_IMPLEMENTATION_ROADMAP.md) | Upcoming roadmap | ✅ Current |
| [`NEXT_STEPS_PHASE_1_1.md`](NEXT_STEPS_PHASE_1_1.md) | Next phase priorities | ✅ Current |

### Infrastructure & Configuration

| Document | Purpose | Status |
|----------|---------|--------|
| [`DEPLOYMENT.md`](DEPLOYMENT.md) | Deployment procedures | ✅ Current |
| [`SCHEMA_DEPLOYMENT_GUIDE.md`](SCHEMA_DEPLOYMENT_GUIDE.md) | Database schema deployment | ✅ Current |
| [`LLM_PROVIDER_SETUP.md`](LLM_PROVIDER_SETUP.md) | LLM configuration | ✅ Current |
| [`LLM_PROVIDER_COMPLETION_SUMMARY.md`](LLM_PROVIDER_COMPLETION_SUMMARY.md) | LLM setup completion status | ✅ Current |
| [`BACKGROUND_JOBS_SETUP.md`](BACKGROUND_JOBS_SETUP.md) | Background job configuration | ✅ Current |

### Operations & Maintenance

| Document | Purpose | Status |
|----------|---------|--------|
| [`TROUBLESHOOTING.md`](TROUBLESHOOTING.md) | Common issues & solutions | ✅ Current |
| [`ENHANCED_LOGGING_SUMMARY.md`](ENHANCED_LOGGING_SUMMARY.md) | Logging system overview | ✅ Current |
| [`QUICK_FIX_GUIDE.md`](QUICK_FIX_GUIDE.md) | Quick reference for common fixes | ✅ Current |
| [`INVESTIGATION_SUMMARY.md`](INVESTIGATION_SUMMARY.md) | Phase 1.2 investigation findings | ✅ Current |

### Reference & Guidelines

| Document | Purpose | Status |
|----------|---------|--------|
| [`README.md`](README.md) | Main project readme | ✅ Current |
| [`SECURITY_AUDIT.md`](SECURITY_AUDIT.md) | Security analysis | ✅ Reference |
| [`RELEASE_GUIDE.md`](RELEASE_GUIDE.md) | Release procedures | ✅ Reference |
| [`UNIFIED_INSTALLER_PLAN.md`](UNIFIED_INSTALLER_PLAN.md) | Installer design | ✅ Reference |
| [`SIDEPANEL_KNOWN_ISSUES.md`](SIDEPANEL_KNOWN_ISSUES.md) | Known issues list | ✅ Current |
| [`TESTING_MVP.md`](TESTING_MVP.md) | MVP testing guide | ✅ Reference |
| [`CLEANUP_REPORT.md`](CLEANUP_REPORT.md) | Archive & dead code cleanup | ✅ Current |

---

## 📦 Archived Documentation

**Location**: [`ARCHIVE/`](ARCHIVE/)

### Organization

```
ARCHIVE/
├── OBSOLETE_SESSION_DOCS/  ← Moved session status docs (27 files)
├── DEAD_CODE/              ← Archived dead code files (2 files)
├── PHASE_1_DOCS/           ← Phase 1 implementation docs
├── DEPRECATED/             ← Deprecated features/APIs
├── TESTING_GUIDES/         ← Old testing documentation
└── OLD_ROADMAPS/           ← Previous roadmaps
```

**Note**: Historical documentation is preserved but no longer maintained. Refer to current docs for accurate information.

---

## 🔍 Finding What You Need

### By Task

**I need to...**

- **Set up Moly**
  - Start: [`QUICKSTART.md`](QUICKSTART.md)
  - Details: [`INSTALL.md`](INSTALL.md)

- **Deploy to production**
  - See: [`DEPLOYMENT.md`](DEPLOYMENT.md)
  - Schema: [`SCHEMA_DEPLOYMENT_GUIDE.md`](SCHEMA_DEPLOYMENT_GUIDE.md)

- **Configure LLM providers**
  - See: [`LLM_PROVIDER_SETUP.md`](LLM_PROVIDER_SETUP.md)

- **Fix a problem**
  - See: [`TROUBLESHOOTING.md`](TROUBLESHOOTING.md)
  - Quick fix: [`QUICK_FIX_GUIDE.md`](QUICK_FIX_GUIDE.md)

- **Understand the architecture**
  - Start: [`MOLY_V2_ARCHITECTURE/01_VISION_AND_PHILOSOPHY.md`](MOLY_V2_ARCHITECTURE/01_VISION_AND_PHILOSOPHY.md)
  - API contract: [`MOLY_V2_ARCHITECTURE/05_API_SPECIFICATION.md`](MOLY_V2_ARCHITECTURE/05_API_SPECIFICATION.md)

- **Understand the current status**
  - See: [`GAPS_CLOSED_IMPLEMENTATION.md`](GAPS_CLOSED_IMPLEMENTATION.md)

- **Contribute to the project**
  - See: [`CONTRIBUTING.md`](CONTRIBUTING.md)

- **Look up something from a previous session**
  - Check: [`ARCHIVE/`](ARCHIVE/) (organized by category)

### By Audience

**If you're a...**

- **Developer**: Start with [`CLAUDE.md`](CLAUDE.md) then [`MOLY_V2_ARCHITECTURE/02_BACKEND_AGENT_ARCHITECTURE.md`](MOLY_V2_ARCHITECTURE/02_BACKEND_AGENT_ARCHITECTURE.md)

- **DevOps/SRE**: Start with [`DEPLOYMENT.md`](DEPLOYMENT.md) and [`TROUBLESHOOTING.md`](TROUBLESHOOTING.md)

- **Product Manager**: Start with [`MOLY_V2_ARCHITECTURE/01_VISION_AND_PHILOSOPHY.md`](MOLY_V2_ARCHITECTURE/01_VISION_AND_PHILOSOPHY.md) and [`V2_1_IMPLEMENTATION_ROADMAP.md`](V2_1_IMPLEMENTATION_ROADMAP.md)

- **Security Reviewer**: Start with [`SECURITY_AUDIT.md`](SECURITY_AUDIT.md) and [`MOLY_V2_ARCHITECTURE/08_PRIVACY_SECURITY.md`](MOLY_V2_ARCHITECTURE/08_PRIVACY_SECURITY.md)

---

## 📋 Document Status Legend

| Status | Meaning |
|--------|---------|
| ✅ Current | Actively maintained, reflects current state |
| 🔄 Reference | Stable reference, not frequently updated |
| 📦 Archived | Historical, maintained in ARCHIVE folder |
| ⚠️ Deprecated | Outdated, do not use |

---

## 🗂️ Root Directory Contents

### Core Project Files
- `README.md` - Main readme
- `INSTALL.md` - Installation
- `QUICKSTART.md` - Quick start
- `CONTRIBUTING.md` - Contributing
- `CLAUDE.md` - Developer guide

### Architecture & Design
- `MOLY_V2_ARCHITECTURE/` - Authoritative architecture (12 docs)
- `MOLY_V2_1_ARCHITECTURE.md` - V2.1 overview

### Current Status
- `GAPS_CLOSED_IMPLEMENTATION.md` - Sept 8: All gaps closed
- `PHASE_1_2_FINAL_STATUS.md` - Phase 1.2 complete
- `PHASE_1_2_FINAL_SUMMARY.md` - Phase 1.2 summary
- `PHASE_1_2_CHECKLIST.md` - Phase 1.2 checklist
- `INVESTIGATION_SUMMARY.md` - Investigation findings
- `CLEANUP_REPORT.md` - Documentation & code cleanup

### Implementation Guides
- `DEPLOYMENT.md` - How to deploy
- `SCHEMA_DEPLOYMENT_GUIDE.md` - Database setup
- `LLM_PROVIDER_SETUP.md` - LLM configuration
- `BACKGROUND_JOBS_SETUP.md` - Job scheduling
- `QUICK_FIX_GUIDE.md` - Common fixes
- `TROUBLESHOOTING.md` - Problem solving

### Operational
- `ENHANCED_LOGGING_SUMMARY.md` - Logging overview
- `SECURITY_AUDIT.md` - Security analysis
- `RELEASE_GUIDE.md` - Release procedures

### Navigation
- `DOCUMENTATION.md` - Documentation overview (you may be looking for this)
- `DOCUMENTATION_INDEX.md` - **You are here**

### Project Directories
- `MOLY_V2_ARCHITECTURE/` - Architecture specification
- `ARCHIVE/` - Historical & obsolete docs
- `moly-go/` - Go backend source
- `moly-extension/` - Browser extension source
- `moly-proxy/` - CORS proxy source

---

## ✨ Quick Tips

1. **Bookmark this page** - Use DOCUMENTATION_INDEX.md to find things quickly

2. **Start with CLAUDE.md** - It explains the project structure and philosophy

3. **Check creation dates** - Newer docs supersede older ones (see ARCHIVE)

4. **Follow the roadmap** - Read V2_1_IMPLEMENTATION_ROADMAP.md to understand priorities

5. **When stuck** - Try TROUBLESHOOTING.md first, then QUICK_FIX_GUIDE.md

6. **For history** - Check ARCHIVE/ for previous sessions' findings

---

## Last Cleanup

**Date**: September 8, 2026  
**Action**: Archived 27 obsolete documents, removed 2 dead code files  
**Result**: Repository is clean and well-organized  
**See**: [`CLEANUP_REPORT.md`](CLEANUP_REPORT.md)

---

**Status**: 🟢 DOCUMENTATION ORGANIZED & INDEXED
