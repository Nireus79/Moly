# Moly v2: Complete Architecture & Implementation Documentation

**Status**: Complete Specification (Sep 6, 2026)  
**Version**: 2.0 - Agent-Based Architecture  
**Audience**: Technical team, founders, investors

---

## Quick Navigation

### 📋 Start Here
1. **[MOLY_ARCHITECTURE_V2.md](./MOLY_ARCHITECTURE_V2.md)** — Core vision, philosophy, principles
   - What Moly is and isn't
   - 5-phase interaction flow
   - Red lines (never cross)
   - User contract

### 🏗️ System Design
2. **[BACKEND_AGENT_ARCHITECTURE.md](./BACKEND_AGENT_ARCHITECTURE.md)** — Complete system design
   - 4 specialized agents
   - Communication protocol
   - Data flow
   - Tools and responsibilities

3. **[AGENT_PROMPTS.md](./AGENT_PROMPTS.md)** — System prompts for all agents
   - Conversation Agent (orchestrator)
   - Learning Agent (pattern recognition)
   - Context Manager Agent (knowledge base)
   - Risk Monitoring Agent (safety with education)
   - Each with: reasoning framework + decision trees + examples

### 💻 Frontend Implementation
4. **[FRONTEND_ARCHITECTURE.md](./FRONTEND_ARCHITECTURE.md)** — Extension UI & state
   - Component hierarchy
   - Zustand stores (chat, settings, contacts)
   - API clients
   - Data flow example
   - Offline capability

### 🔌 API & Integration
5. **[API_SPECIFICATION.md](./API_SPECIFICATION.md)** — Complete API reference
   - All endpoints (generate, feedback, context, contacts, etc.)
   - Request/response formats
   - Error codes
   - Rate limiting
   - Data types

### 🗄️ Data & Storage
6. **[DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md)** — Database design
   - Core tables (users, about_me, contacts, conversations, messages)
   - Behavioral data (user_profiles, interaction_history)
   - Privacy enforcement (no contact surveillance)
   - Queries & indexes
   - Migrations

### 🚀 Operations
7. **[DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)** — Production deployment
   - Backend setup (Go, PostgreSQL)
   - Docker & Kubernetes
   - Scaling configuration
   - Monitoring & alerting
   - Backup & recovery
   - Troubleshooting

### 🔒 Privacy & Security
8. **[PRIVACY_SECURITY.md](./PRIVACY_SECURITY.md)** — Privacy specification
   - User behavior learning (✅ tracked)
   - Contact behavior (❌ never tracked)
   - Data minimization
   - Encryption strategies
   - GDPR/CCPA compliance
   - Incident response
   - Ethical guidelines

### 🛣️ Migration & Implementation
9. **[MIGRATION_PATH.md](./MIGRATION_PATH.md)** — Current → agents transition
   - 4-phase migration plan
   - Gradual rollout strategy
   - Data consistency
   - Fallback mechanisms
   - Timeline & success criteria

10. **[IMPLEMENTATION_ROADMAP.md](./IMPLEMENTATION_ROADMAP.md)** — 2-year plan
    - Q3 2026: Foundation (backend + agents)
    - Q4 2026: Scaling (1M+ users)
    - Q1 2027: Intelligence (advanced personalization)
    - Q2 2027: Expansion (multi-platform)

### ⚙️ Quality & Monitoring
11. **[ERROR_HANDLING.md](./ERROR_HANDLING.md)** — Resilience specification
    - Error hierarchy (safety → risk → warning → technical)
    - Common errors & recovery
    - Retry logic
    - Graceful degradation
    - Testing errors

12. **[SUCCESS_METRICS.md](./SUCCESS_METRICS.md)** — Analytics & measurement
    - North star metrics (authenticity, effectiveness, learning)
    - Product metrics (engagement, usage, retention)
    - Quality metrics (suggestions, safety, learning)
    - Monitoring dashboard
    - Success definition & OKRs

---

## Architecture Overview

```
┌─────────────────────────────────────────────┐
│ USER (Browser)                              │
│                                             │
│  Chrome Extension (React + Zustand)        │
│  ├─ Sidebar (main UI)                      │
│  ├─ Settings (API key, model)              │
│  ├─ Conversations (history)                │
│  └─ Reflection Modal (learning)            │
│                                             │
└────────────┬────────────────────────────────┘
             │ HTTP/JSON (via Backend Manager)
┌────────────▼────────────────────────────────┐
│ BACKEND (Go + Claude API)                   │
│                                             │
│  ┌─ Conversation Agent (Orchestrator)      │
│  │  ├─ Phase-based decision tree           │
│  │  ├─ Context gathering                   │
│  │  ├─ Safety checks                       │
│  │  └─ Suggestion generation               │
│  │                                         │
│  ├─ Learning Agent (User Profile)          │
│  │  ├─ Communication patterns              │
│  │  ├─ Suggestion preferences              │
│  │  ├─ Success metrics                     │
│  │  └─ Growth tracking                     │
│  │                                         │
│  ├─ Context Manager (Knowledge Base)       │
│  │  ├─ About Me                            │
│  │  ├─ Contacts (user observations)        │
│  │  ├─ Conversation history                │
│  │  └─ Intelligent retrieval               │
│  │                                         │
│  ├─ Risk Monitor (Safety with Education)   │
│  │  ├─ Pattern detection                   │
│  │  ├─ Educational questions               │
│  │  ├─ Socratic approach                   │
│  │  └─ Principle-based guidance            │
│  │                                         │
│  └─ Shared Tools                           │
│     ├─ SafetyChecker                       │
│     ├─ ConstitutionEvaluator               │
│     ├─ SuggestionGenerator                 │
│     ├─ QuestionGenerator                   │
│     └─ ContextExtractor                    │
│                                             │
└────────────┬────────────────────────────────┘
             │
┌────────────▼────────────────────────────────┐
│ DATA LAYER (PostgreSQL)                     │
│                                             │
│  ├─ User Data (About Me, Contacts)         │
│  ├─ Conversations (History, Messages)      │
│  ├─ Behavioral Profiles (User only)        │
│  ├─ Risk Patterns (for education)          │
│  └─ Audit Log (compliance)                 │
│                                             │
│  NO: Contact surveillance, behavior        │
│  tracking, or monitoring data              │
│                                             │
└─────────────────────────────────────────────┘
```

---

## Core Principles

### Philosophy
- **User-Driven**: Moly responds, never auto-reads
- **Thinking Partner**: Helps user think, not decides for them
- **Authentic**: Suggestions feel like the user
- **Ethical**: Educates, never dictates
- **Learning**: Improves with each interaction

### What Moly Does
✅ Learn user's communication style  
✅ Generate personalized suggestions  
✅ Ask Socratic questions  
✅ Educate on ethical principles  
✅ Detect user risk patterns  
✅ Remember about contacts (user observations)  

### What Moly Never Does
❌ Monitor or track contacts  
❌ Collect surveillance data  
❌ Block user's choices  
❌ Dictate ethics  
❌ Use hardcoded templates  
❌ Assume contact behavior  

---

## Key Features by Phase

### MVP (Q3 2026)
- Core conversation flow
- Suggestion generation
- Contact management
- About Me profile
- Basic learning
- Safety checks

### v1.0 (Q4 2026)
- ✅ All MVP features
- Advanced personalization
- Reflection modal
- Risk pattern detection
- Offline mode
- Performance optimized

### v2.0 (Q1 2027)
- ✅ All v1.0 features
- Dynamic context detection
- Intention analysis
- Socratic conversation mode
- Educational safety layer
- User analytics dashboard

### v3.0 (Q2 2027)
- ✅ All v2.0 features
- Multi-platform (web, mobile)
- Premium features
- International support
- Advanced analytics
- Revenue model

---

## Getting Started

### For Developers
1. Read [MOLY_ARCHITECTURE_V2.md](./MOLY_ARCHITECTURE_V2.md) — understand the vision
2. Read [BACKEND_AGENT_ARCHITECTURE.md](./BACKEND_AGENT_ARCHITECTURE.md) — understand the system
3. Read [AGENT_PROMPTS.md](./AGENT_PROMPTS.md) — understand how agents think
4. Start with [IMPLEMENTATION_ROADMAP.md](./IMPLEMENTATION_ROADMAP.md) — Phase 1 tasks
5. Follow [MIGRATION_PATH.md](./MIGRATION_PATH.md) for rollout strategy

### For Infrastructure/DevOps
1. Read [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md) — deployment setup
2. Read [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md) — database design
3. Read [MONITORING_OBSERVABILITY.md](./MONITORING_OBSERVABILITY.md) — observability (linked from DEPLOYMENT)
4. Set up infrastructure before Phase 1.1

### For Product/Founders
1. Read [MOLY_ARCHITECTURE_V2.md](./MOLY_ARCHITECTURE_V2.md) — philosophy
2. Read [SUCCESS_METRICS.md](./SUCCESS_METRICS.md) — how we measure success
3. Read [IMPLEMENTATION_ROADMAP.md](./IMPLEMENTATION_ROADMAP.md) — timeline & milestones
4. Review [MIGRATION_PATH.md](./MIGRATION_PATH.md) — rollout strategy

### For Security/Privacy
1. Read [PRIVACY_SECURITY.md](./PRIVACY_SECURITY.md) — complete security spec
2. Read [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md) — constraints & enforcement
3. Review [ERROR_HANDLING.md](./ERROR_HANDLING.md) — incident response

---

## Document Statistics

| Document | Purpose | Length | Status |
|----------|---------|--------|--------|
| MOLY_ARCHITECTURE_V2 | Vision & Principles | ~3K lines | ✅ Complete |
| BACKEND_AGENT_ARCHITECTURE | System Design | ~4K lines | ✅ Complete |
| AGENT_PROMPTS | Agent Instructions | ~8K lines | ✅ Complete |
| FRONTEND_ARCHITECTURE | UI & State | ~3K lines | ✅ Complete |
| API_SPECIFICATION | API Reference | ~2K lines | ✅ Complete |
| DATABASE_SCHEMA | DB Design | ~2K lines | ✅ Complete |
| DEPLOYMENT_GUIDE | Operations | ~2K lines | ✅ Complete |
| PRIVACY_SECURITY | Security Spec | ~2K lines | ✅ Complete |
| MIGRATION_PATH | Implementation | ~2K lines | ✅ Complete |
| ERROR_HANDLING | Resilience | ~2K lines | ✅ Complete |
| SUCCESS_METRICS | Analytics | ~2K lines | ✅ Complete |
| IMPLEMENTATION_ROADMAP | 2-Year Plan | ~2K lines | ✅ Complete |

**Total**: ~32K lines of comprehensive documentation

---

## Key Decisions & Rationale

### Agent-Based Architecture
**Why**: No hardcoding, adaptive behavior, learning over time  
**Trade-off**: More complex, higher latency, higher API costs  
**Mitigation**: Caching, agent pooling, optimization

### User-Only Learning (No Contact Surveillance)
**Why**: Privacy principle, user autonomy, ethical foundation  
**Trade-off**: Less predictive power  
**Benefit**: Trust, user confidence, regulatory compliance

### Gradual Migration Strategy
**Why**: Minimize risk, gather feedback, enable rollback  
**Trade-off**: Takes longer, dual maintenance  
**Benefit**: Safety, data validation, real-world testing

### Socratic Safety Approach
**Why**: Educate user, respect autonomy, build judgment  
**Trade-off**: Slower than blocking, user might ignore  
**Benefit**: User learns, relationships deepen, ethical alignment

---

## Success Criteria

**Moly is successful when:**
1. Users trust suggestions (pick rate > 70%, satisfaction > 3.5/4)
2. Users improve (positive response rate > 70%)
3. Relationships deepen (conversation continuation > 60%)
4. Ethics work (user overrides 15-20%, no complaints)
5. Scale works (1M+ users, < 1s latency, < 0.5% error)
6. Business works (retention > 40%, LTV/CAC > 3)

---

## Contact & Questions

- **Architecture**: Technical lead
- **Product**: Product manager
- **Operations**: DevOps lead
- **Privacy**: Security lead

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 2.0 | Sep 6, 2026 | Complete agent-based architecture redesign |
| 1.0 | Sep 1, 2026 | Initial architecture (inline logic) |

---

**Last Updated**: September 6, 2026  
**Status**: Ready for implementation  
**Next Review**: September 13, 2026 (after Phase 1.1)

---

## Quick Links

- GitHub Repo: (to be created)
- Slack Channel: #moly-architecture
- Project Board: (Linear/GitHub Projects)
- Demo: (staging environment)
- Roadmap: [IMPLEMENTATION_ROADMAP.md](./IMPLEMENTATION_ROADMAP.md)

---

This comprehensive documentation provides everything needed to build, deploy, and scale Moly from concept to 10M+ users.
