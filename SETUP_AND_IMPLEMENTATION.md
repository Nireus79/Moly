# Moly V2 Setup & Implementation Guide

**Date**: September 7, 2026  
**Status**: Phase 1.1 - LLM Integration & Agent Implementation  
**Timeline**: 13 days to Sep 20 deadline  

---

## Quick Start (5 minutes)

### 1. Set Up API Key

```bash
# Export your Anthropic API key
export ANTHROPIC_API_KEY="sk-ant-YOUR_KEY_HERE"

# Or create a .env file in moly-go/
echo "ANTHROPIC_API_KEY=sk-ant-YOUR_KEY_HERE" > moly-go/.env
```

### 2. Build Backend

```bash
cd moly-go
go build -o moly-backend
```

### 3. Run Backend

```bash
./moly-backend
# Server listens on http://127.0.0.1:11436
```

### 4. Test LLM Integration

```bash
# Build test binary with integration tests
go test -v -run TestLLMClientInitialization
go test -v -run TestDatabaseInitialization
go test -v -run TestV2AgentSystemInitialization

# Full integration test (requires ANTHROPIC_API_KEY)
go test -v -run TestLLMClientWithAPIKey
```

---

## What's Complete (✅)

### Backend Infrastructure
- ✅ LLM Client (Claude, OpenAI, Ollama support)
- ✅ Database schema (SQLite with V1 + V2 tables)
- ✅ Database initialization & migrations
- ✅ HTTP API routes (V2 endpoints registered)
- ✅ CORS headers & preflight handling
- ✅ Suggestion generator with LLM integration
- ✅ Safety checker (basic implementation)
- ✅ Configuration management

### V2 Agents
- ✅ Conversation Agent structure (20% complete)
  - ✅ Context detection (AboutMe, Contact, Intention)
  - ✅ Socratic question generation
  - ✅ Phase routing (context_gathering vs suggestions_ready)
  - 🔄 LLM-based suggestion generation (NEW - just added)
- ✅ Learning Agent structure (10% complete)
- ✅ Context Manager structure (50% complete)
- ✅ Risk Monitor structure (5% complete)

### API Endpoints
- ✅ POST /api/v2/conversation/generate - Conversation flow
- ✅ POST /api/v2/conversation/feedback - Record user feedback
- ✅ GET /api/v2/context - Retrieve context
- ✅ GET /api/v2/contacts - List contacts
- ✅ POST /api/v2/about-me - Save About Me profile
- ✅ POST /api/v2/health - Health check

---

## Critical Path (Next 5 Days)

### Today (Sep 7): ✅ LLM Integration
- [x] Fix environment variable handling (ANTHROPIC_API_KEY)
- [x] Verify LLM client initialization
- [x] Add LLM-based suggestion generation to Conversation Agent
- [x] Build and compile successful
- [ ] **TODO**: Export ANTHROPIC_API_KEY and test real API calls

### Tomorrow (Sep 8): Database Learning Loop
- [ ] Implement Learning Agent.RecordSuggestionChoice() to persist to DB
- [ ] Implement Learning Agent.GetUserBehaviorProfile()
- [ ] Add user_behavioral_profile table if missing
- [ ] Test recording and retrieval

### Sep 9: Risk Monitor
- [ ] Implement risk detection using LLM
- [ ] Add risk alerts to conversation responses
- [ ] Test safety patterns

### Sep 10-13: Complete Agents & Tools
- [ ] Full context manager CRUD operations
- [ ] Question generator with LLM
- [ ] Constitution evaluator implementation
- [ ] Reflection extraction and storage

### Sep 14-19: Testing & Polish
- [ ] Unit tests for all agents
- [ ] Integration tests for full flow
- [ ] E2E testing with extension
- [ ] Bug fixes and optimization

---

## Architecture Overview

### Request Flow (POST /api/v2/conversation/generate)
```
1. Extension sends: {userId, conversationId, userMessage}
2. Backend receives at V2APIServer.ConversationGenerateHandler
3. Creates agent system (lazy init per user)
4. Builds context from AboutMe, Contacts, History
5. Calls ConversationAgent.Run(context)
6. Agent decides: gathering phase or suggestions phase
7. If gathering: Returns Socratic questions
8. If ready: Generates suggestions using LLMClient
9. Returns: {phase, questions/suggestions, safety_alerts}
```

### Agent Responsibilities

**ConversationAgent** (Main orchestrator)
- Analyzes context quality
- Decides phase: context_gathering → suggestions_ready
- Generates questions for missing context
- Generates suggestions when ready
- Handles safety checks
- Orchestrates reflection

**LearningAgent** (Behavioral analysis)
- Records user choices to database
- Analyzes patterns over time
- Builds behavioral profile
- Predicts user preferences

**ContextManager** (Knowledge store)
- Retrieves AboutMe profile
- Manages Contact list
- Stores conversation history
- Persists reflections
- Calculates context completeness

**RiskMonitor** (Safety)
- Detects crisis indicators
- Identifies harmful patterns
- Flags constitutional violations
- Provides education/resources

---

## Environment Variables

```bash
# API
ANTHROPIC_API_KEY=sk-ant-...      # Required for Claude
LLM_PROVIDER=claude               # claude|openai|ollama
AGENT_MODEL=claude-opus-5         # Model to use

# Behavior
AGENT_TEMPERATURE=0.7             # 0.0-1.0 (lower=deterministic)
AGENT_MAX_TOKENS=2000             # Max output length
AGENT_TIMEOUT_SECONDS=30          # API timeout

# Server
PORT=11436                        # Backend port
HOST=localhost                    # Bind address
LOG_LEVEL=info                    # Logging verbosity

# Ollama (if using local)
OLLAMA_ENDPOINT=http://127.0.0.1:11434
```

---

## Database Schema

### V2 Tables (for agents)
- `about_me` - User profile (communication style, values, preferences)
- `contacts` - People user interacts with
- `conversations` - Chat sessions
- `interactions` - Individual messages + analysis
- `behavioral_profile` - Learned patterns

### V1 Compatibility Tables (still used)
- `contacts` - Legacy contact management
- `behavior_patterns` - Default patterns
- (Others for backward compatibility)

---

## Testing Checklist

### Unit Tests
- [ ] ConversationAgent context detection
- [ ] ConversationAgent question generation
- [ ] ConversationAgent suggestion generation
- [ ] LearningAgent recording
- [ ] ContextManager CRUD
- [ ] RiskMonitor analysis

### Integration Tests
- [ ] Full conversation flow (no context → questions → suggestions)
- [ ] Database persistence (record choice → retrieve profile)
- [ ] Multi-turn conversation (context accumulation)
- [ ] Safety checks (crisis → alert → resources)

### E2E Tests with Extension
- [ ] Send message → receive questions
- [ ] Answer questions → save context
- [ ] Next message → receive suggestions
- [ ] Select suggestion → record feedback
- [ ] Verify learning (follow-up message gets personalized)

---

## Common Issues & Fixes

### Error: "ANTHROPIC_API_KEY not set"
```bash
# Fix: Export the key before running
export ANTHROPIC_API_KEY="sk-ant-..."
./moly-backend
```

### Error: "LLM call failed: 401 Unauthorized"
```bash
# Fix: Check API key is correct
echo $ANTHROPIC_API_KEY
# If empty or wrong, update it
```

### Error: "Database locked"
```bash
# Fix: SQLite gets locked if multiple processes access it
# Close all instances first, then restart
pkill moly-backend
./moly-backend
```

### Suggestions are generic, not personalized
```bash
# This means LLM isn't being called
# Check:
# 1. ANTHROPIC_API_KEY is set
# 2. ConversationAgent.llmClient is not nil
# 3. Check logs for LLM errors
```

---

## File Locations (Quick Reference)

```
moly-go/
├── main.go                          # Entry point
├── database.go                      # DB initialization
├── v2_handlers.go                   # API endpoints
├── tools/
│   ├── llm_client.go                # LLM provider routing
│   ├── suggestion_generator.go      # LLM-based generation
│   ├── safety_checker.go            # Safety checks
│   └── ...
├── agents/
│   ├── v2_agents.go                 # AgentSystem orchestrator
│   ├── conversation_agent.go        # Main agent (BEING IMPLEMENTED)
│   ├── learning_agent.go            # Behavioral learning
│   ├── context_manager.go           # Knowledge store
│   ├── risk_monitor.go              # Safety monitor
│   └── ...
├── models/                          # Type definitions
└── moly-backend                     # Compiled binary
```

---

## Next Immediate Actions

1. **Set ANTHROPIC_API_KEY**
   ```bash
   export ANTHROPIC_API_KEY="your_key_here"
   ```

2. **Rebuild and test**
   ```bash
   cd moly-go
   go build -o moly-backend
   go test -v ./...
   ```

3. **Run the backend**
   ```bash
   ./moly-backend
   # Now /api/v2/* endpoints are available
   ```

4. **Test conversation flow**
   ```bash
   curl -X POST http://localhost:11436/api/v2/conversation/generate \
     -H "Content-Type: application/json" \
     -d '{
       "userId": "test-user",
       "conversationId": "conv-1",
       "userMessage": "hello"
     }'
   # Should return context_gathering phase with questions
   ```

---

## Success Metrics

By Sep 20, you need:
- ✅ All 4 agents initialized and callable
- ✅ LLM integration working (real API calls)
- ✅ Database storing and retrieving data
- ✅ Conversation flow: questions → answers → suggestions
- ✅ Learning loop: recording choices and building profiles
- ✅ Safety checks working
- ✅ Tests passing (unit + integration)
- ✅ E2E test with extension successful
- ✅ No crashes on 24-hour run

---

## Git Workflow

All changes should be committed as:

```bash
git add <files>
git commit -m "Implement [component]

Description of changes.

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>"
```

---

## Resources

- **Architecture**: `MOLY_V2_ARCHITECTURE/02_BACKEND_AGENT_ARCHITECTURE.md`
- **API Spec**: `MOLY_V2_ARCHITECTURE/05_API_SPECIFICATION.md`
- **Privacy**: `MOLY_V2_ARCHITECTURE/08_PRIVACY_SECURITY.md`
- **Roadmap**: `MOLY_V2_ARCHITECTURE/12_IMPLEMENTATION_ROADMAP.md`

---

**Status**: Ready to implement. API key setup needed before testing.
