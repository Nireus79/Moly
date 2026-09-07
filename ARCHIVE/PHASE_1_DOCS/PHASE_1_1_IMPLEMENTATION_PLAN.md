# Phase 1.1: Backend Agent Implementation - Detailed Plan

**Date**: September 7, 2026  
**Phase**: 1.1 (Foundation)  
**Duration**: Sep 6-20, 2026 (2 weeks)  
**Status**: Ready for implementation

---

## Overview

Build the 4 agent system in parallel to existing code. No breaking changes. Clear isolation.

---

## Project Structure (New Agents)

```
moly-go/
├── (existing code - NO CHANGES)
│
├── agents/                        ← NEW FOLDER
│   ├── v2_agents.go              ← Main agent orchestration
│   ├── conversation_agent.go      ← Conversation Agent
│   ├── learning_agent.go          ← Learning Agent
│   ├── context_manager.go         ← Context Manager
│   └── risk_monitor.go            ← Risk Monitoring Agent
│
├── tools/                         ← NEW FOLDER
│   ├── suggestion_generator.go
│   ├── question_generator.go
│   ├── safety_checker.go
│   ├── constitution_evaluator.go
│   ├── context_extractor.go
│   └── llm_client.go
│
├── models/                        ← NEW FOLDER
│   ├── agent_types.go             ← Type definitions for agents
│   ├── conversation_types.go       ← Request/response types
│   └── context_types.go            ← Context & knowledge base types
│
├── agents_test/                   ← NEW FOLDER
│   ├── conversation_agent_test.go
│   ├── learning_agent_test.go
│   ├── integration_test.go
│   └── mocks.go
│
└── (existing code continues to work)
```

---

## Milestone Checklist

### Week 1: Core Infrastructure

#### 1.1.1: Project Setup (Day 1-2)
- [ ] Create `agents/` folder
- [ ] Create `tools/` folder
- [ ] Create `models/` folder
- [ ] Create `agents_test/` folder
- [ ] Create `.gitignore` entries (no conflicts)
- [ ] Add to `go.mod`: any new dependencies (anthropic SDK)

**Files to create**:
```
agents/v2_agents.go          (empty interface definitions)
models/agent_types.go         (type definitions)
models/conversation_types.go  (request/response types)
```

#### 1.1.2: Type Definitions (Day 2-3)
- [ ] Define all type structures in `models/agent_types.go`
- [ ] Define request/response types
- [ ] Create mock types for testing

**From docs**: See `02_BACKEND_AGENT_ARCHITECTURE.md` → Data Types section

**Files to create**:
```
models/agent_types.go
  ├── ConversationRequest
  ├── ConversationResponse
  ├── Suggestion
  ├── Reflection
  ├── SafetyCheckResult
  ├── RiskWarning
  ├── Message
  ├── Contact
  └── ... (all from API_SPECIFICATION.md)
```

#### 1.1.3: LLM Client (Day 3-4)
- [ ] Create `tools/llm_client.go`
- [ ] Implement Claude API wrapper
- [ ] Support extended thinking
- [ ] Error handling

**Config needed**:
```
CLAUDE_API_KEY (environment variable)
AGENT_MODEL = claude-opus-5 (configurable)
AGENT_MAX_TOKENS = 2000 (configurable)
```

**Tests**: 
```
tools/llm_client_test.go
  ├── TestClaudeConnection
  ├── TestPromptGeneration
  └── TestErrorHandling
```

### Week 2: Agent Implementation

#### 1.1.4: Conversation Agent (Day 5-7)
- [ ] Create `agents/conversation_agent.go`
- [ ] Implement decision tree (ANALYZE → CONTEXT → SAFETY → RISK → INTENTION → GENERATE → REFLECT)
- [ ] Implement phase logic
- [ ] Error handling with graceful fallback

**From docs**: See `03_AGENT_PROMPTS.md` → Conversation Agent section

**Methods**:
```go
func (a *ConversationAgent) Run(ctx Context) (*Response, error)
func (a *ConversationAgent) analyzePhase(msg string) (Phase, error)
func (a *ConversationAgent) contextPhase(ctx Context) (Context, error)
func (a *ConversationAgent) safetyPhase(msg string) (SafetyResult, error)
func (a *ConversationAgent) riskPhase(msg string) (RiskResult, error)
func (a *ConversationAgent) intentionPhase(msg string) (Intention, error)
func (a *ConversationAgent) generatePhase(ctx Context) ([]Suggestion, error)
func (a *ConversationAgent) reflectPhase(msg string) (Reflection, error)
```

**Tests**:
```
agents_test/conversation_agent_test.go
  ├── TestAnalyzePhase
  ├── TestContextGathering
  ├── TestSafetyCheck
  ├── TestRiskDetection
  ├── TestIntentionDetection
  ├── TestSuggestionGeneration
  └── TestGracefulDegradation
```

#### 1.1.5: Learning Agent (Day 7-8)
- [ ] Create `agents/learning_agent.go`
- [ ] Implement user profile building
- [ ] Implement pattern detection
- [ ] NO contact surveillance

**From docs**: See `03_AGENT_PROMPTS.md` → Learning Agent section

**Methods**:
```go
func (a *LearningAgent) GetUserProfile(userId string) (*UserProfile, error)
func (a *LearningAgent) RecordInteraction(data InteractionData) error
func (a *LearningAgent) RecordSuggestionChoice(data ChoiceData) error
func (a *LearningAgent) BuildBehavioralProfile(userId string) (*BehavioralProfile, error)
func (a *LearningAgent) DetectPatterns(userId string) (*UserPatterns, error)
```

**Tests**:
```
agents_test/learning_agent_test.go
  ├── TestProfileBuilding
  ├── TestPatternDetection
  ├── TestCommunicationStyleLearning
  ├── TestSuccessMetrics
  └── TestNoContactSurveillance (important!)
```

#### 1.1.6: Context Manager Agent (Day 8-9)
- [ ] Create `agents/context_manager.go`
- [ ] Implement About Me storage/retrieval
- [ ] Implement Contact management
- [ ] Implement Conversation history
- [ ] Intelligent context retrieval

**From docs**: See `03_AGENT_PROMPTS.md` → Context Manager Agent section

**Methods**:
```go
func (a *ContextManager) GetAboutMe(userId string) (*AboutMe, error)
func (a *ContextManager) SetAboutMe(userId string, aboutMe *AboutMe) error
func (a *ContextManager) GetContact(userId, contactId string) (*Contact, error)
func (a *ContextManager) GetContacts(userId string) ([]Contact, error)
func (a *ContextManager) CreateContact(userId string, contact *Contact) (*Contact, error)
func (a *ContextManager) UpdateContact(userId, contactId string, updates Contact) error
func (a *ContextManager) GetRelevantContext(conversationId, userId string) (*Context, error)
func (a *ContextManager) SaveReflection(conversationId string, reflection *Reflection) error
func (a *ContextManager) ApproveReflection(conversationId string, reflection *Reflection) error
func (a *ContextManager) AppendMessage(conversationId string, message *Message) error
```

**Tests**:
```
agents_test/context_manager_test.go
  ├── TestAboutMeStorage
  ├── TestContactManagement
  ├── TestConversationHistory
  ├── TestReflectionWorkflow
  └── TestIntelligentRetrieval
```

#### 1.1.7: Risk Monitoring Agent (Day 9-10)
- [ ] Create `agents/risk_monitor.go`
- [ ] Implement pattern detection (manipulation, boundaries, scams, harm, insincerity)
- [ ] Implement educational questions
- [ ] Implement severity rating

**From docs**: See `03_AGENT_PROMPTS.md` → Risk Monitoring Agent section

**Methods**:
```go
func (a *RiskMonitor) AssessRisk(userId string, message string) (*RiskAssessment, error)
func (a *RiskMonitor) DetectPatterns(userId string) (*UserRiskProfile, error)
func (a *RiskMonitor) GenerateEducationalResponse(risk RiskAssessment) ([]string, error)
func (a *RiskMonitor) TrackPattern(userId string, pattern *RiskPattern) error
func (a *RiskMonitor) GetUserRiskProfile(userId string) (*UserRiskProfile, error)
```

**Tests**:
```
agents_test/risk_monitor_test.go
  ├── TestManipulationDetection
  ├── TestBoundaryViolationDetection
  ├── TestScamDetection
  ├── TestEducationalApproach
  └── TestPatternTracking
```

### Week 2.5: Tools Implementation

#### 1.1.8: Shared Tools (Day 5-10, parallel with agents)
- [ ] Create `tools/suggestion_generator.go`
- [ ] Create `tools/question_generator.go`
- [ ] Create `tools/safety_checker.go`
- [ ] Create `tools/constitution_evaluator.go`
- [ ] Create `tools/context_extractor.go`

Each tool should:
- Have clear input/output types
- Call LLM client
- Have comprehensive error handling
- Be independently testable

**Tests**: `tools/` folder with `*_test.go` for each tool

---

## Database Schema (Parallel)

These MUST be implemented before agent testing:

```sql
-- From 06_DATABASE_SCHEMA.md, create migrations:

migration_001_create_base_schema.sql
  ├── users table
  ├── about_me table
  ├── contacts table
  ├── conversations table
  └── messages table

migration_002_behavioral_profiles.sql
  ├── user_profiles table
  ├── interaction_history table
  └── risk_patterns table

migration_003_indexes_and_constraints.sql
  └── Indexes, FKs, check constraints
```

**Implementation**:
- Create `database/migrations/` folder
- Use database migration tool (e.g., migrate, sql-migrate)
- Test migrations locally

---

## Integration Testing

#### 1.1.9: Integration Tests (Day 10-12)
- [ ] Create `agents_test/integration_test.go`
- [ ] Test complete flow: User message → Agent processing → Response
- [ ] Test graceful degradation
- [ ] Test error recovery

**Test scenarios**:
```
├── Happy path (user sends message, gets suggestions)
├─- Missing context (user needs About Me first)
├── Safety alert (crisis/illegal content)
├── Risk warning (manipulation pattern)
├── Timeout recovery
├── Partial failures (one agent fails, others succeed)
└── Learning on feedback
```

---

## Configuration & Environment

#### 1.1.10: Configuration Setup (Day 1)
- [ ] `.env.example` with all required variables
- [ ] `config.go` updates for v2 agents
- [ ] Environment variable documentation

**Required variables**:
```
CLAUDE_API_KEY=sk-ant-...
AGENT_MODEL=claude-opus-5
AGENT_TEMPERATURE=0.7
AGENT_MAX_TOKENS=2000
DATABASE_URL=postgres://user:password@localhost:5432/moly
AGENT_POOL_SIZE=10
LOG_LEVEL=info
```

---

## API Endpoints (Parallel)

#### 1.1.11: New API Routes (Day 8-12)
- [ ] Create `v2_handlers.go` (new handlers, don't touch existing)
- [ ] Implement `POST /api/v2/conversation/generate`
- [ ] Implement `POST /api/v2/conversation/feedback`
- [ ] Implement `GET /api/v2/context`
- [ ] Mount at `/api/v2/` (don't conflict with existing `/api/`)

**From docs**: See `05_API_SPECIFICATION.md` for exact request/response formats

**Key principle**: `/api/` (old) and `/api/v2/` (new) run in parallel until ready to migrate.

---

## Testing Strategy

### Unit Tests
- Each agent: isolated tests
- Each tool: isolated tests
- Coverage target: 80%+

### Integration Tests
- Complete flow: message → response
- Error scenarios
- Performance (latency < 2s)

### Manual Testing
- Via Postman/curl
- Test with Claude API key
- Local database

---

## Success Criteria

### Day 12 (End of Phase 1.1) - ALL MUST PASS

- [ ] ✅ All 4 agents compile without errors
- [ ] ✅ All unit tests pass (coverage > 80%)
- [ ] ✅ Integration tests pass (happy path + error cases)
- [ ] ✅ Database migrations run successfully
- [ ] ✅ API endpoints `/api/v2/` respond correctly
- [ ] ✅ Graceful error handling (no panics)
- [ ] ✅ Claude API integration working
- [ ] ✅ NO changes to existing `/api/` routes
- [ ] ✅ README updated with new agent documentation
- [ ] ✅ Code follows existing Go style guide

### Performance Targets
- Suggestion generation: < 2s p95
- Safety check: < 200ms p50
- Questions: < 500ms p50

### Code Quality
- [ ] Zero compiler warnings
- [ ] All tests pass
- [ ] No new linting errors
- [ ] Clear comments on complex logic
- [ ] Type definitions exported where needed

---

## Files to Create (Complete List)

### Agents (5 files)
```
agents/v2_agents.go                    (orchestration)
agents/conversation_agent.go           (main agent)
agents/learning_agent.go               (learning)
agents/context_manager.go              (knowledge base)
agents/risk_monitor.go                 (risk detection)
```

### Tools (5 files)
```
tools/llm_client.go                    (Claude API wrapper)
tools/suggestion_generator.go          (suggestion generation)
tools/question_generator.go            (question generation)
tools/safety_checker.go                (safety checks)
tools/constitution_evaluator.go        (ethics checks)
tools/context_extractor.go             (insight extraction)
```

### Models (3 files)
```
models/agent_types.go                  (agent types)
models/conversation_types.go           (request/response)
models/context_types.go                (context types)
```

### Tests (6 files)
```
agents_test/mocks.go                   (test mocks)
agents_test/conversation_agent_test.go
agents_test/learning_agent_test.go
agents_test/context_manager_test.go
agents_test/risk_monitor_test.go
agents_test/integration_test.go
```

### Tools Tests (6 files)
```
tools/llm_client_test.go
tools/suggestion_generator_test.go
tools/question_generator_test.go
tools/safety_checker_test.go
tools/constitution_evaluator_test.go
tools/context_extractor_test.go
```

### Database (3 migration files)
```
database/migrations/001_create_base_schema.sql
database/migrations/002_behavioral_profiles.sql
database/migrations/003_indexes_constraints.sql
```

### API (1 file)
```
v2_handlers.go                         (new endpoints at /api/v2/)
```

### Configuration (1 file)
```
.env.example                           (environment variables)
```

**Total new files**: 24 + documentation

---

## Naming Conventions (NO CONFLICTS)

**Agents** (interface + implementation):
```go
type ConversationAgent interface { ... }
type conversationAgent struct { ... }
func NewConversationAgent(...) ConversationAgent { ... }
```

**Files**:
- Agent files: `agents/{name}_agent.go`
- Tool files: `tools/{name}.go`
- Test files: `{package}/{name}_test.go`
- Model files: `models/{name}_types.go`
- Migration files: `database/migrations/{number}_{name}.sql`

**Packages**:
- `agents` — agent implementations
- `tools` — shared tools
- `models` — type definitions
- `database` — migrations and queries (namespace existing)
- Main handler: `v2_handlers.go` (new file, v2 namespace)

**No naming conflicts**: Existing code in root (main.go, chat.go, etc.) stays unchanged.

---

## Checklist for Day 1

Before any coding:

- [ ] Create folder structure (agents/, tools/, models/, agents_test/)
- [ ] Create empty placeholder files (so import paths work)
- [ ] Update go.mod with anthropic/claude SDK
- [ ] Create .env.example
- [ ] Verify existing code still builds
- [ ] Run existing tests (should all pass)
- [ ] Create this checklist in a task tracking system

---

## Risk Mitigation

**Risk**: Breaking existing code  
**Mitigation**: New agents in separate `/agents` folder, new endpoints at `/api/v2/`, no changes to existing code

**Risk**: Database conflicts  
**Mitigation**: Clear migration naming (001_, 002_, 003_), separate schema namespace if needed

**Risk**: Import circular dependencies  
**Mitigation**: Clear package structure (agents → tools → models), no reverse imports

**Risk**: Claude API costs  
**Mitigation**: Use API monitoring, set rate limits, test with small payloads first

---

## Communication Points

- **Daily standups**: Report folder/file creation progress
- **Mid-week sync**: Review agent implementations
- **End of week**: Integration test results
- **Day 12 review**: All success criteria met?

---

## Next Phase

After Phase 1.1 complete:
- Phase 1.2: Extension integration (Week 3-4)
- Phase 2: Gradual rollout (Week 5-8)
- See: IMPLEMENTATION_ROADMAP.md

---

**START DATE**: September 7, 2026  
**END DATE**: September 20, 2026  
**READY**: Yes, all specs documented

