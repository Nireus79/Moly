# Moly v2: Complete Agent-Based Architecture Documentation

**Date**: September 6, 2026  
**Status**: Complete & Ready for Implementation  
**Total Documentation**: ~270 KB across 13 comprehensive documents

---

## 📚 Documentation Index

Start here → [**00_INDEX.md**](./00_INDEX.md) — Master index and navigation guide

### Core Architecture Documents (Read in Order)

1. **[01_VISION_AND_PHILOSOPHY.md](./01_VISION_AND_PHILOSOPHY.md)** — The "why"
   - Moly's core philosophy and principles
   - What Moly is and isn't
   - Ethical framework
   - 5-phase interaction flow
   - Red lines (never cross)

2. **[02_BACKEND_AGENT_ARCHITECTURE.md](./02_BACKEND_AGENT_ARCHITECTURE.md)** — The "how"
   - System architecture diagram
   - 4 specialized agents and responsibilities
   - Communication protocol
   - Data flow and tool definitions
   - Implementation considerations

3. **[03_AGENT_PROMPTS.md](./03_AGENT_PROMPTS.md)** — Agent system prompts
   - Complete system prompt for each agent
   - Reasoning frameworks and decision trees
   - Concrete examples and workflows
   - Integration diagram

### Implementation Documents

4. **[04_FRONTEND_ARCHITECTURE.md](./04_FRONTEND_ARCHITECTURE.md)** — Extension UI
   - Component hierarchy
   - React + Zustand state management
   - Data flow
   - Offline capability

5. **[05_API_SPECIFICATION.md](./05_API_SPECIFICATION.md)** — API endpoints
   - Complete endpoint reference
   - Request/response formats
   - Error codes and handling
   - Rate limiting
   - Data types

6. **[06_DATABASE_SCHEMA.md](./06_DATABASE_SCHEMA.md)** — Database design
   - PostgreSQL schema
   - Tables, constraints, indexes
   - Queries and migrations
   - Privacy enforcement

### Operations & Quality

7. **[07_DEPLOYMENT_GUIDE.md](./07_DEPLOYMENT_GUIDE.md)** — Production deployment
   - Backend setup
   - Docker & Kubernetes
   - Scaling configuration
   - Monitoring & alerting
   - Troubleshooting

8. **[08_PRIVACY_SECURITY.md](./08_PRIVACY_SECURITY.md)** — Privacy spec
   - What Moly collects (user behavior only)
   - What Moly never collects (contact surveillance)
   - Encryption and compliance (GDPR, CCPA)
   - Incident response
   - Security checklist

9. **[10_ERROR_HANDLING.md](./10_ERROR_HANDLING.md)** — Resilience
   - Error hierarchy
   - Recovery strategies
   - Retry logic
   - Graceful degradation
   - Testing errors

10. **[11_SUCCESS_METRICS.md](./11_SUCCESS_METRICS.md)** — Analytics
    - North star metrics (authenticity, effectiveness, learning)
    - Product metrics (engagement, usage, retention)
    - Quality metrics
    - Monitoring dashboard
    - OKRs

### Planning & Strategy

11. **[09_MIGRATION_PATH.md](./09_MIGRATION_PATH.md)** — Current → v2 transition
    - 4-phase migration plan
    - Gradual rollout strategy
    - Data consistency
    - Fallback mechanisms

12. **[12_IMPLEMENTATION_ROADMAP.md](./12_IMPLEMENTATION_ROADMAP.md)** — 2-year plan
    - Q3 2026: Foundation phase
    - Q4 2026: Scaling phase
    - Q1 2027: Intelligence phase
    - Q2 2027: Expansion phase
    - Success criteria for each phase

---

## 🎯 Quick Start by Role

### For Developers
1. Read: 01_VISION_AND_PHILOSOPHY.md (30 min)
2. Read: 02_BACKEND_AGENT_ARCHITECTURE.md (45 min)
3. Read: 03_AGENT_PROMPTS.md (60 min)
4. Read: 12_IMPLEMENTATION_ROADMAP.md → Phase 1 tasks (30 min)

**Total**: ~2.5 hours to understand the full system

### For Infrastructure/DevOps
1. Read: 07_DEPLOYMENT_GUIDE.md (45 min)
2. Read: 06_DATABASE_SCHEMA.md (30 min)
3. Setup local environment following 07_DEPLOYMENT_GUIDE.md (1-2 hours)

### For Product/Founders
1. Read: 01_VISION_AND_PHILOSOPHY.md (30 min)
2. Read: 11_SUCCESS_METRICS.md (30 min)
3. Read: 12_IMPLEMENTATION_ROADMAP.md (45 min)
4. Read: 09_MIGRATION_PATH.md (30 min)

### For Security/Privacy
1. Read: 08_PRIVACY_SECURITY.md (45 min)
2. Review: 06_DATABASE_SCHEMA.md (30 min)
3. Review: 10_ERROR_HANDLING.md (20 min)

---

## 📊 Documentation Statistics

| Document | Purpose | Size | Status |
|----------|---------|------|--------|
| 00_INDEX | Navigation | 13 KB | ✅ Complete |
| 01_VISION_AND_PHILOSOPHY | Philosophy | 13 KB | ✅ Complete |
| 02_BACKEND_AGENT_ARCHITECTURE | System Design | 34 KB | ✅ Complete |
| 03_AGENT_PROMPTS | Agent Instructions | 54 KB | ✅ Complete |
| 04_FRONTEND_ARCHITECTURE | UI & State | 25 KB | ✅ Complete |
| 05_API_SPECIFICATION | API Reference | 15 KB | ✅ Complete |
| 06_DATABASE_SCHEMA | DB Design | 16 KB | ✅ Complete |
| 07_DEPLOYMENT_GUIDE | Operations | 13 KB | ✅ Complete |
| 08_PRIVACY_SECURITY | Security | 12 KB | ✅ Complete |
| 09_MIGRATION_PATH | Implementation | 10 KB | ✅ Complete |
| 10_ERROR_HANDLING | Resilience | 9.7 KB | ✅ Complete |
| 11_SUCCESS_METRICS | Analytics | 8.6 KB | ✅ Complete |
| 12_IMPLEMENTATION_ROADMAP | Planning | 9.9 KB | ✅ Complete |

**Total**: 268 KB of comprehensive documentation

---

## 🔑 Key Concepts

### 4 Specialized Agents

1. **Conversation Agent** (Orchestrator)
   - Runs the 5-phase interaction flow
   - Phase-based decision tree
   - Calls other agents as needed

2. **Learning Agent** (Pattern Recognition)
   - Builds user behavioral profile
   - Tracks communication patterns
   - Zero contact surveillance

3. **Context Manager Agent** (Knowledge Base)
   - Stores About Me, Contacts, Conversations
   - Intelligent context retrieval
   - Manages reflections

4. **Risk Monitoring Agent** (Safety with Education)
   - Detects concerning patterns
   - Educational Socratic questions
   - Pattern escalation tracking

### Core Principles

✅ **Do**:
- Learn user behavior
- Generate dynamic responses
- Ask Socratic questions
- Educate on ethics
- Respect autonomy

❌ **Never**:
- Monitor contacts
- Use hardcoding
- Block user choices
- Judge or shame
- Collect surveillance data

---

## 🚀 Next Steps

1. **Week 1**: Read all architecture documents (00-03)
2. **Week 2**: Review implementation docs (04-07)
3. **Week 3**: Start Phase 1.1 (Backend Agent Implementation)
   - See: 12_IMPLEMENTATION_ROADMAP.md → Phase 1.1
4. **Ongoing**: Reference specific docs as needed for implementation

---

## 📖 Reading Guide

**For first-time readers**: Follow the documentation in numerical order.

**For referencing specific topics**:
- Agents → 02_BACKEND_AGENT_ARCHITECTURE.md + 03_AGENT_PROMPTS.md
- API → 05_API_SPECIFICATION.md
- Database → 06_DATABASE_SCHEMA.md
- Deployment → 07_DEPLOYMENT_GUIDE.md
- Privacy → 08_PRIVACY_SECURITY.md
- Testing → 10_ERROR_HANDLING.md
- Metrics → 11_SUCCESS_METRICS.md
- Timeline → 12_IMPLEMENTATION_ROADMAP.md

---

## 🔄 Version Control

| Version | Date | Changes |
|---------|------|---------|
| 2.0 | Sep 6, 2026 | Complete agent-based architecture redesign |
| 1.0 | Sep 1, 2026 | Initial architecture (inline logic) |

---

## 📝 Contributing to Documentation

This documentation is frozen for implementation. Changes require:
1. Discussion with architecture team
2. Documentation update
3. Code review (if affecting implementation)

---

## ❓ Questions?

- **Architecture**: See 02_BACKEND_AGENT_ARCHITECTURE.md
- **Agents**: See 03_AGENT_PROMPTS.md
- **Implementation**: See 12_IMPLEMENTATION_ROADMAP.md
- **Privacy**: See 08_PRIVACY_SECURITY.md
- **Deployment**: See 07_DEPLOYMENT_GUIDE.md

---

**Last Updated**: September 6, 2026  
**Status**: Ready for Implementation  
**Next Review**: September 13, 2026 (after Phase 1.1)

---

Start with [00_INDEX.md](./00_INDEX.md) for a complete navigation guide.
