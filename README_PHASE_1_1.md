# Phase 1.1: Quick Start Guide

**Date**: September 7, 2026  
**Duration**: Sep 7-20, 2026 (2 weeks)  
**Ready**: ✅ YES

---

## Today's Tasks (Day 1)

### Morning
1. **Read** PHASE_1_1_IMPLEMENTATION_PLAN.md (30 min)
2. **Verify** project structure against plan (10 min)
3. **Create** folder structure (5 min)

### Afternoon
4. **Create** empty placeholder Go files (20 min)
5. **Update** go.mod with dependencies (10 min)
6. **Test** that existing code still builds (10 min)

### End of Day
- [ ] Verify with: `go build ./...` (should succeed)
- [ ] Verify with: `go test ./...` (existing tests pass)
- [ ] Commit: "Phase 1.1: Project structure setup"

---

## Folder Structure to Create

```bash
# Create directories
mkdir -p agents
mkdir -p tools
mkdir -p models
mkdir -p agents_test
mkdir -p database/migrations
```

## Files to Create (Empty Placeholders)

```bash
# Agents
touch agents/v2_agents.go
touch agents/conversation_agent.go
touch agents/learning_agent.go
touch agents/context_manager.go
touch agents/risk_monitor.go

# Tools
touch tools/llm_client.go
touch tools/suggestion_generator.go
touch tools/question_generator.go
touch tools/safety_checker.go
touch tools/constitution_evaluator.go
touch tools/context_extractor.go

# Models
touch models/agent_types.go
touch models/conversation_types.go
touch models/context_types.go

# Tests
touch agents_test/mocks.go
touch agents_test/conversation_agent_test.go
touch agents_test/learning_agent_test.go
touch agents_test/context_manager_test.go
touch agents_test/risk_monitor_test.go
touch agents_test/integration_test.go

# Create .env
cp .env.example .env.local

# Migrations directory
touch database/migrations/.gitkeep
```

## Placeholder File Content

Each file should start with:

```go
package agents  // (or tools, models, agents_test)

// TODO: Implement {AgentName} Agent
// See: /MOLY_V2_ARCHITECTURE/03_AGENT_PROMPTS.md
// Deadline: Sep 20, 2026
```

---

## Key References

| Document | Purpose | When to Read |
|----------|---------|--------------|
| PHASE_1_1_IMPLEMENTATION_PLAN.md | Detailed breakdown | Daily reference |
| IMPLEMENTATION_VERIFICATION.md | Verification checklist | After each section |
| MOLY_V2_ARCHITECTURE/02_BACKEND_AGENT_ARCHITECTURE.md | System design | When implementing agents |
| MOLY_V2_ARCHITECTURE/03_AGENT_PROMPTS.md | Agent instructions | When implementing each agent |
| MOLY_V2_ARCHITECTURE/05_API_SPECIFICATION.md | API details | When implementing handlers |
| MOLY_V2_ARCHITECTURE/06_DATABASE_SCHEMA.md | Database | When implementing database |

---

## Week-by-Week Milestones

### Week 1 (Sep 7-13)
- [ ] Day 1-2: Project setup
- [ ] Day 2-3: Type definitions
- [ ] Day 3-4: LLM client
- [ ] Day 5: Checkpoint review

### Week 2 (Sep 14-20)
- [ ] Day 5-7: Conversation Agent
- [ ] Day 7-8: Learning Agent
- [ ] Day 8-9: Context Manager
- [ ] Day 9-10: Risk Monitor
- [ ] Day 10-12: Integration tests
- [ ] Day 12: Final verification

---

## Daily Checklist Template

```
Date: Sep X, 2026

Morning:
- [ ] Read today's plan
- [ ] Review code from yesterday
- [ ] Start coding

Afternoon:
- [ ] Implement feature/test
- [ ] Review code changes
- [ ] Run tests

End of Day:
- [ ] All tests pass
- [ ] No compiler warnings
- [ ] Commit with message
```

---

## Success Criteria for Today (Sep 7)

By end of today:
- [ ] Folder structure created
- [ ] Placeholder files created
- [ ] go.mod updated
- [ ] Existing tests still pass
- [ ] No compiler errors
- [ ] No breaking changes to existing code

---

## Important Reminders

### ✅ DO:
- Use separate `/api/v2/` endpoints
- Put new agents in `agents/` folder
- Document as you go
- Test incrementally
- Commit frequently

### ❌ DON'T:
- Modify existing handlers
- Touch moly-extension code
- Change database schema (use migrations)
- Break existing tests
- Skip unit tests

---

## Commands

### Build
```bash
go build ./...
```

### Test
```bash
go test ./... -v
```

### Specific Test
```bash
go test ./agents -v
```

### Clean
```bash
go clean
```

### Format
```bash
go fmt ./...
```

---

## Git Workflow

### Commit Pattern
```bash
git add .
git commit -m "Phase 1.1: Day X - {Feature} implementation

- Implement {detail}
- Add tests for {feature}
- No breaking changes

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>"
```

### Daily Commit
```bash
git commit -m "Phase 1.1: Day 1 - Project structure setup

- Create agents/ folder
- Create tools/ folder  
- Create models/ folder
- Create agents_test/ folder
- Update go.mod
- Verify existing code builds

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>"
```

---

## Support

### If You Get Stuck

1. **Check**: PHASE_1_1_IMPLEMENTATION_PLAN.md → that day's section
2. **Read**: Relevant documentation in MOLY_V2_ARCHITECTURE/
3. **Verify**: IMPLEMENTATION_VERIFICATION.md → matching section
4. **Compare**: Naming conventions with project structure

### Common Issues

**Issue**: "Import not found"  
**Fix**: Check package name in placeholder file

**Issue**: "Existing tests fail"  
**Fix**: Don't modify existing code, only add new

**Issue**: "Unsure what to implement"  
**Fix**: Read the agent prompt in 03_AGENT_PROMPTS.md

---

## Timeline

| Date | Week | Status |
|------|------|--------|
| Sep 7-8 | 1 | Setup |
| Sep 9-10 | 1 | Types |
| Sep 11-13 | 1 | LLM Client |
| Sep 14-16 | 2 | Agents Part 1 |
| Sep 17-19 | 2 | Agents Part 2 |
| Sep 20 | 2 | Verification & Review |

---

## Day 1 Checklist (Today)

- [ ] Read this file (10 min)
- [ ] Read PHASE_1_1_IMPLEMENTATION_PLAN.md (30 min)
- [ ] Create folder structure (5 min)
- [ ] Create placeholder files (15 min)
- [ ] Update go.mod (5 min)
- [ ] Verify build works (5 min)
- [ ] Verify tests pass (5 min)
- [ ] First commit (3 min)

**Total**: ~90 minutes

---

## End of Day Sign-Off

When done today, verify:
```bash
# This should work
go build ./...
go test ./...
git status
```

Both should complete without errors.

---

**Ready to start Phase 1.1?** ✅

See you at the end of the day!

